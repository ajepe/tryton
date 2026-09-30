package rpc

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ajepe/talon/internal/orm"
)

type ModuleHandler struct {
	OnChange      func(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error)
	OnChangeWith  func(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error)
	ButtonActions map[string]func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error)
	Search        func(ctx context.Context, group *orm.Group, text string) ([]int, error)
	Validate      func(ctx context.Context, group *orm.Group, record *orm.Record) error
	Preprocess    func(ctx context.Context, group *orm.Group, values map[string]interface{}, mode string) (map[string]interface{}, error)
}

type ModuleRegistry struct {
	mu       sync.RWMutex
	handlers map[string]*ModuleHandler
}

func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		handlers: map[string]*ModuleHandler{},
	}
}

func (r *ModuleRegistry) Register(model string, h *ModuleHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[model] = h
}

func (r *ModuleRegistry) Get(model string) (*ModuleHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[model]
	return h, ok
}

func (r *ModuleRegistry) OnChange(ctx context.Context, model string, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	h, ok := r.Get(model)
	if !ok || h.OnChange == nil {
		return map[string]interface{}{}, nil
	}
	return h.OnChange(ctx, group, values, fields)
}

func (r *ModuleRegistry) OnChangeWith(ctx context.Context, model string, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	h, ok := r.Get(model)
	if !ok || h.OnChangeWith == nil {
		return map[string]interface{}{}, nil
	}
	return h.OnChangeWith(ctx, group, values, fields)
}

func (r *ModuleRegistry) ButtonAction(ctx context.Context, model, action string, group *orm.Group, records []*orm.Record) (interface{}, error) {
	h, ok := r.Get(model)
	if !ok || h.ButtonActions == nil {
		return nil, fmt.Errorf("no button action %s for model %s", action, model)
	}
	fn, ok := h.ButtonActions[action]
	if !ok {
		return nil, fmt.Errorf("no button action %s for model %s", action, model)
	}
	return fn(ctx, group, records)
}

func (r *ModuleRegistry) Search(ctx context.Context, model string, group *orm.Group, text string) ([]int, error) {
	h, ok := r.Get(model)
	if !ok || h.Search == nil {
		return nil, fmt.Errorf("no search handler for model %s", model)
	}
	return h.Search(ctx, group, text)
}

func (r *ModuleRegistry) Validate(ctx context.Context, model string, group *orm.Group, record *orm.Record) error {
	h, ok := r.Get(model)
	if !ok || h.Validate == nil {
		return nil
	}
	return h.Validate(ctx, group, record)
}

func (r *ModuleRegistry) Preprocess(ctx context.Context, model string, group *orm.Group, values map[string]interface{}, mode string) (map[string]interface{}, error) {
	h, ok := r.Get(model)
	if !ok || h.Preprocess == nil {
		return values, nil
	}
	return h.Preprocess(ctx, group, values, mode)
}

func (r *ModuleRegistry) RegisterAll(srv *Server) {
	r.registerButtonMethods(srv)

	srv.Register("module.<model>.on_change", func(ctx context.Context, params []interface{}) (interface{}, error) {
		model := modelNameFromContext(ctx)
		group, err := getGroupFromContext(ctx, model)
		if err != nil {
			return nil, err
		}
		values, _ := params[0].(map[string]interface{})
		fieldsArr, _ := params[1].([]interface{})
		fields := make([]string, len(fieldsArr))
		for i, f := range fieldsArr {
			fields[i], _ = f.(string)
		}
		return r.OnChange(ctx, model, group, values, fields)
	})

	srv.Register("module.<model>.on_change_with", func(ctx context.Context, params []interface{}) (interface{}, error) {
		model := modelNameFromContext(ctx)
		group, err := getGroupFromContext(ctx, model)
		if err != nil {
			return nil, err
		}
		values, _ := params[0].(map[string]interface{})
		fieldsArr, _ := params[1].([]interface{})
		fields := make([]string, len(fieldsArr))
		for i, f := range fieldsArr {
			fields[i], _ = f.(string)
		}
		return r.OnChangeWith(ctx, model, group, values, fields)
	})

	srv.Register("module.<model>.button", func(ctx context.Context, params []interface{}) (interface{}, error) {
		model := modelNameFromContext(ctx)
		action, _ := params[0].(string)
		group, err := getGroupFromContext(ctx, model)
		if err != nil {
			return nil, err
		}
		records := getRecordsFromContext(ctx, group)
		return r.ButtonAction(ctx, model, action, group, records)
	})

	srv.Register("module.<model>.search", func(ctx context.Context, params []interface{}) (interface{}, error) {
		model := modelNameFromContext(ctx)
		text, _ := params[0].(string)
		group, err := getGroupFromContext(ctx, model)
		if err != nil {
			return nil, err
		}
		ids, err := r.Search(ctx, model, group, text)
		if err != nil {
			return nil, err
		}
		return ids, nil
	})

	srv.Register("module.<model>.validate", func(ctx context.Context, params []interface{}) (interface{}, error) {
		model := modelNameFromContext(ctx)
		group, err := getGroupFromContext(ctx, model)
		if err != nil {
			return nil, err
		}
		record := getRecordFromContext(ctx, group)
		return nil, r.Validate(ctx, model, group, record)
	})
}

func (r *ModuleRegistry) registerButtonMethods(srv *Server) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for model, handler := range r.handlers {
		for action := range handler.ButtonActions {
			model, action := model, action
			srv.Register("model."+model+"."+action, func(ctx context.Context, params []interface{}) (interface{}, error) {
				group, err := getGroupFromContext(ctx, model)
				if err != nil {
					return nil, err
				}
				if len(params) > 1 {
					if ids := idsFromParams(params[0]); len(ids) > 0 {
						if err := group.Load(ids); err != nil {
							return nil, err
						}
					}
				}
				return r.ButtonAction(ctx, model, action, group, group.Records())
			})
		}
	}
}

func idsFromParams(param interface{}) []int {
	raw, ok := param.([]interface{})
	if !ok {
		return nil
	}
	ids := make([]int, 0, len(raw))
	for _, value := range raw {
		if id, ok := orm.AsInt(value); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func modelNameFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if model, ok := ctx.Value("model").(string); ok && model != "" {
		return model
	}
	if method, ok := ctx.Value("method").(string); ok {
		parts := strings.SplitN(method, ".", 3)
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}

func getGroupFromContext(ctx context.Context, model string) (*orm.Group, error) {
	pool := orm.DefaultPool()
	return pool.NewGroup(model)
}

func getRecordsFromContext(ctx context.Context, group *orm.Group) []*orm.Record {
	return group.Records()
}

func getRecordFromContext(ctx context.Context, group *orm.Group) *orm.Record {
	records := group.Records()
	if len(records) > 0 {
		return records[0]
	}
	return nil
}
