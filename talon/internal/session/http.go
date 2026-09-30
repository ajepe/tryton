package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ajepe/talon/internal/rpc"
)

const CookieName = "tryton_session"

func RequiresSession(method string) bool {
	for _, prefix := range []string{"model.", "module.", "wizard.", "report."} {
		if strings.HasPrefix(method, prefix) {
			return true
		}
	}
	return false
}

func FromRequest(r *http.Request, mgr *Manager) *Session {
	if header := r.Header.Get("Authorization"); strings.HasPrefix(header, "session ") {
		encoded := strings.TrimSpace(strings.TrimPrefix(header, "session "))
		if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil {
			if sess := lookupValue(mgr, string(decoded)); sess != nil {
				return sess
			}
		}
	}
	if cookie, err := r.Cookie(CookieName); err == nil {
		if sess := lookupValue(mgr, cookie.Value); sess != nil {
			return sess
		}
	}
	return nil
}

func lookupValue(mgr *Manager, value string) *Session {
	parts := strings.SplitN(value, ":", 3)
	if len(parts) != 3 {
		return nil
	}
	sess, ok := mgr.Get(parts[2])
	if !ok {
		return nil
	}
	return sess
}

func RegisterRPC(s *rpc.Server, mgr *Manager, database string) {
	s.Register("common.db.login", func(ctx context.Context, params []interface{}) (interface{}, error) {
		sess, err := LoginParams(ctx, mgr, database, params)
		if err != nil {
			return nil, err
		}
		return []interface{}{sess.UserID(), ""}, nil
	})

	s.Register("common.db.logout", func(ctx context.Context, params []interface{}) (interface{}, error) {
		if token, ok := ctx.Value("session").(string); ok {
			mgr.Logout(token)
		}
		return nil, nil
	})
}

func LoginParams(ctx context.Context, mgr *Manager, database string, params []interface{}) (*Session, error) {
	if len(params) < 2 {
		return nil, &rpc.UserError{Message: "login requires username and password"}
	}
	login, _ := params[0].(string)
	password := ""
	switch credentials := params[1].(type) {
	case map[string]interface{}:
		password, _ = credentials["password"].(string)
	case string:
		password = credentials
	}
	sess, err := mgr.Login(ctx, database, login, password)
	if err != nil {
		var loginErr *LoginError
		if errorsAs(err, &loginErr) {
			return nil, &rpc.LoginException{
				Name:    "LoginException",
				Message: loginErr.Message,
				Type:    loginErr.Type,
			}
		}
		return nil, &rpc.UserError{Message: err.Error()}
	}
	return sess, nil
}

func LoginHandler(mgr *Manager, database string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		params := []interface{}{}
		if len(request.Params) > 0 {
			if err := json.Unmarshal(request.Params, &params); err != nil {
				http.Error(w, "invalid params", http.StatusBadRequest)
				return
			}
		}

		sess, err := LoginParams(r.Context(), mgr, database, params)
		if err != nil {
			writeSessionError(w, request.ID, err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     CookieName,
			Value:    sess.CookieValue(),
			Path:     "/" + database,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Secure:   r.TLS != nil,
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     request.ID,
			"result": []interface{}{sess.UserID(), ""},
		})
	}
}

func LogoutHandler(mgr *Manager, database string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if sess := FromRequest(r, mgr); sess != nil {
			mgr.Logout(sess.Token())
		}
		http.SetCookie(w, &http.Cookie{
			Name:     CookieName,
			Value:    "",
			Path:     "/" + database,
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Secure:   r.TLS != nil,
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     0,
			"result": nil,
		})
	}
}

func writeSessionError(w http.ResponseWriter, id int, err error) {
	status := http.StatusOK
	var errData interface{}
	switch e := err.(type) {
	case *rpc.LoginException:
		errData = []interface{}{"LoginException", []interface{}{e.Name, e.Message, e.Type}}
		switch e.Type {
		case "rate_limit":
			status = http.StatusTooManyRequests
		default:
			status = http.StatusForbidden
		}
	case *rpc.UserError:
		errData = []interface{}{"UserError", []interface{}{e.Message, e.Description, e.Domain}}
	default:
		errData = []interface{}{"Error", []interface{}{err.Error()}}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    id,
		"error": errData,
	})
}

func errorsAs(err error, target **LoginError) bool {
	loginErr, ok := err.(*LoginError)
	if !ok {
		return false
	}
	*target = loginErr
	return true
}

func RPCAuthenticator(mgr *Manager, database string) rpc.Authenticator {
	return func(r *http.Request, method string) (context.Context, error) {
		if !RequiresSession(method) {
			return r.Context(), nil
		}
		sess := FromRequest(r, mgr)
		if sess == nil {
			return nil, &rpc.LoginException{
				Name:    "LoginException",
				Message: "session expired",
				Type:    "session",
			}
		}
		mgr.Renew(sess.Token())
		ctx := context.WithValue(r.Context(), "session", sess.Token())
		ctx = context.WithValue(ctx, "user", sess.UserID())
		ctx = context.WithValue(ctx, "database", database)
		return ctx, nil
	}
}
