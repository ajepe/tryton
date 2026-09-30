package wizard

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/ajepe/talon/internal/rpc"
)

const EndState = "end"

type TransitionFunc func(ctx context.Context, instance *Instance, values map[string]interface{}) (string, error)

type Definition struct {
	Name        string
	Initial     string
	Transitions map[string]map[string]TransitionFunc
}

func NewDefinition(name, initial string) *Definition {
	return &Definition{
		Name:        name,
		Initial:     initial,
		Transitions: map[string]map[string]TransitionFunc{},
	}
}

func (d *Definition) On(state, action string, fn TransitionFunc) *Definition {
	if d.Transitions[state] == nil {
		d.Transitions[state] = map[string]TransitionFunc{}
	}
	d.Transitions[state][action] = fn
	return d
}

func (d *Definition) Actions(state string) []string {
	actions := make([]string, 0, len(d.Transitions[state]))
	for action := range d.Transitions[state] {
		actions = append(actions, action)
	}
	return actions
}

type Instance struct {
	ID      string                 `json:"id"`
	Name    string                 `json:"name"`
	State   string                 `json:"state"`
	Values  map[string]interface{} `json:"values"`
	Session string                 `json:"session,omitempty"`
}

type Engine struct {
	mu        sync.RWMutex
	defs      map[string]*Definition
	instances map[string]*Instance
}

func NewEngine() *Engine {
	return &Engine{
		defs:      map[string]*Definition{},
		instances: map[string]*Instance{},
	}
}

func (e *Engine) Register(def *Definition) error {
	if def == nil || def.Name == "" {
		return fmt.Errorf("wizard definition requires a name")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, exists := e.defs[def.Name]; exists {
		return fmt.Errorf("wizard %s already registered", def.Name)
	}
	e.defs[def.Name] = def
	return nil
}

func (e *Engine) Get(name string) (*Definition, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	def, ok := e.defs[name]
	return def, ok
}

func (e *Engine) Start(name, session string, values map[string]interface{}) (*Instance, error) {
	def, ok := e.Get(name)
	if !ok {
		return nil, fmt.Errorf("wizard %s not found", name)
	}
	if values == nil {
		values = map[string]interface{}{}
	}
	instance := &Instance{
		ID:      newID(),
		Name:    name,
		State:   def.Initial,
		Values:  values,
		Session: session,
	}
	e.mu.Lock()
	e.instances[instance.ID] = instance
	e.mu.Unlock()
	return instance, nil
}

func (e *Engine) Execute(ctx context.Context, id, action string, values map[string]interface{}) (*Instance, error) {
	e.mu.Lock()
	instance, ok := e.instances[id]
	if !ok {
		e.mu.Unlock()
		return nil, fmt.Errorf("wizard session %s not found", id)
	}
	def, ok := e.defs[instance.Name]
	e.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("wizard %s not found", instance.Name)
	}

	transitions, ok := def.Transitions[instance.State]
	if !ok {
		return nil, fmt.Errorf("wizard %s has no transitions from state %s", instance.Name, instance.State)
	}
	fn, ok := transitions[action]
	if !ok {
		return nil, fmt.Errorf("wizard %s does not support action %s in state %s", instance.Name, action, instance.State)
	}

	for name, value := range values {
		instance.Values[name] = value
	}
	next, err := fn(ctx, instance, instance.Values)
	if err != nil {
		return nil, err
	}
	instance.State = next

	if next == EndState {
		e.mu.Lock()
		delete(e.instances, id)
		e.mu.Unlock()
	}
	return instance, nil
}

func (e *Engine) Delete(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.instances[id]; !ok {
		return fmt.Errorf("wizard session %s not found", id)
	}
	delete(e.instances, id)
	return nil
}

func (e *Engine) Instance(id string) (*Instance, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	instance, ok := e.instances[id]
	return instance, ok
}

func (e *Engine) RegisterRPC(srv *rpc.Server, name string) {
	srv.Register("wizard."+name+".create", func(ctx context.Context, params []interface{}) (interface{}, error) {
		var values map[string]interface{}
		if len(params) > 0 {
			values, _ = params[0].(map[string]interface{})
		}
		return e.Start(name, sessionFromContext(ctx), values)
	})

	srv.Register("wizard."+name+".execute", func(ctx context.Context, params []interface{}) (interface{}, error) {
		if len(params) < 2 {
			return nil, &rpc.UserError{Message: "execute requires id and action"}
		}
		id, _ := params[0].(string)
		action, _ := params[1].(string)
		var values map[string]interface{}
		if len(params) > 2 {
			values, _ = params[2].(map[string]interface{})
		}
		return e.Execute(ctx, id, action, values)
	})

	srv.Register("wizard."+name+".delete", func(ctx context.Context, params []interface{}) (interface{}, error) {
		if len(params) < 1 {
			return nil, &rpc.UserError{Message: "delete requires id"}
		}
		id, _ := params[0].(string)
		if err := e.Delete(id); err != nil {
			return nil, &rpc.UserError{Message: err.Error()}
		}
		return nil, nil
	})
}

func sessionFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if session, ok := ctx.Value("session").(string); ok {
		return session
	}
	return ""
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString(b[:])
	}
	return hex.EncodeToString(b[:])
}
