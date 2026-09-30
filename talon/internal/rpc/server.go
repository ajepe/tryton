package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type Request struct {
	ID     int             `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type Response struct {
	ID     int         `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

type Handler func(ctx context.Context, params []interface{}) (interface{}, error)

type Authenticator func(r *http.Request, method string) (context.Context, error)

type Server struct {
	mu            sync.RWMutex
	handlers      map[string]Handler
	codec         *Codec
	cache         *Cache
	authenticator Authenticator
}

func NewServer() *Server {
	return &Server{
		handlers: map[string]Handler{},
		codec:    NewCodec(),
		cache:    NewCache(),
	}
}

func (s *Server) SetAuthenticator(a Authenticator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authenticator = a
}

func (s *Server) Register(method string, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = h
}

func (s *Server) lookup(method string) (Handler, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if h, ok := s.handlers[method]; ok {
		return h, "", true
	}
	for pattern, h := range s.handlers {
		if target, ok := matchPlaceholder(pattern, method); ok {
			return h, target, true
		}
	}
	return nil, "", false
}

func matchPlaceholder(pattern, method string) (string, bool) {
	start := strings.IndexByte(pattern, '<')
	if start < 0 {
		return "", false
	}
	end := strings.IndexByte(pattern[start:], '>')
	if end < 0 {
		return "", false
	}
	end += start + 1

	prefix := pattern[:start]
	suffix := pattern[end:]
	if !strings.HasPrefix(method, prefix) || !strings.HasSuffix(method, suffix) {
		return "", false
	}
	if len(method) <= len(prefix)+len(suffix) {
		return "", false
	}
	return method[len(prefix) : len(method)-len(suffix)], true
}

func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 0, &UserError{Message: "invalid JSON-RPC request"})
		return
	}

	handler, model, ok := s.lookup(req.Method)
	if !ok {
		s.writeError(w, req.ID, &UserError{Message: fmt.Sprintf("method %s not found", req.Method)})
		return
	}

	var params []interface{}
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.writeError(w, req.ID, &UserError{Message: "invalid params"})
			return
		}
	}

	ctx := r.Context()
	if s.authenticator != nil {
		authCtx, err := s.authenticator(r, req.Method)
		if err != nil {
			s.writeError(w, req.ID, err)
			return
		}
		if authCtx != nil {
			ctx = authCtx
		}
	}
	ctx = context.WithValue(ctx, "method", req.Method)
	if model != "" {
		ctx = context.WithValue(ctx, "model", model)
	}
	if len(params) > 0 {
		if ctxMap, ok := params[len(params)-1].(map[string]interface{}); ok {
			ctx = context.WithValue(ctx, "context", ctxMap)
		}
	}

	result, err := handler(ctx, params)
	if err != nil {
		s.writeError(w, req.ID, err)
		return
	}

	s.writeResult(w, req.ID, result)
}

func (s *Server) writeResult(w http.ResponseWriter, id int, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     id,
		"result": s.codec.Encode(result),
	})
}

func (s *Server) writeError(w http.ResponseWriter, id int, err error) {
	w.Header().Set("Content-Type", "application/json")
	var errData interface{}
	status := http.StatusOK
	switch e := err.(type) {
	case *UserError:
		errData = []interface{}{"UserError", []interface{}{e.Message, e.Description, e.Domain}}
	case *UserWarning:
		errData = []interface{}{"UserWarning", []interface{}{e.Name, e.Message, e.Description}}
		status = http.StatusConflict
	case *ConcurrencyException:
		errData = []interface{}{"ConcurrencyException", []interface{}{e.Message}}
		status = http.StatusLocked
	case *LoginException:
		errData = []interface{}{"LoginException", []interface{}{e.Name, e.Message, e.Type}}
		switch e.Type {
		case "rate_limit":
			status = http.StatusTooManyRequests
		default:
			status = http.StatusForbidden
		}
	default:
		errData = []interface{}{"Error", []interface{}{err.Error()}}
	}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    id,
		"error": errData,
	})
}

func (s *Server) Call(ctx context.Context, method string, params []interface{}) (interface{}, error) {
	s.mu.RLock()
	handler, ok := s.handlers[method]
	s.mu.RUnlock()
	if !ok {
		return nil, &UserError{Message: fmt.Sprintf("method %s not found", method)}
	}
	return handler(ctx, params)
}

func (s *Server) RegisterModelMethods(model string, methods map[string]Handler) {
	for name, h := range methods {
		s.Register(fmt.Sprintf("model.%s.%s", model, name), h)
	}
}

func (s *Server) RegisterCommonMethods(methods map[string]Handler) {
	for name, h := range methods {
		s.Register(fmt.Sprintf("common.%s", name), h)
	}
}

func (s *Server) RegisterWizardMethods(wizard string, methods map[string]Handler) {
	for name, h := range methods {
		s.Register(fmt.Sprintf("wizard.%s.%s", wizard, name), h)
	}
}

func (s *Server) RegisterReportMethods(report string, h Handler) {
	s.Register(fmt.Sprintf("report.%s.execute", report), h)
}

func (s *Server) MethodPrefix(method string) string {
	parts := strings.SplitN(method, ".", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func (s *Server) MethodName(method string) string {
	parts := strings.SplitN(method, ".", 3)
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func (s *Server) ModelName(method string) string {
	parts := strings.SplitN(method, ".", 3)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
