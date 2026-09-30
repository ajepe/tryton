package ir

import (
	"context"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/rpc"
)

func RegisterHandlers(registry *rpc.ModuleRegistry) {
	registry.Register("ir.sequence", &rpc.ModuleHandler{
		OnChangeWith: OnChangeSequence,
		ButtonActions: map[string]func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error){
			"get": ButtonGetSequence,
		},
		Validate: ValidateSequence,
	})

	registry.Register("ir.cron", &rpc.ModuleHandler{
		Validate: ValidateCron,
	})

	registry.Register("ir.attachment", &rpc.ModuleHandler{
		OnChange: OnChangeAttachment,
		Search:   SearchAttachment,
	})

	registry.Register("ir.translation", &rpc.ModuleHandler{
		Search: SearchTranslation,
	})

	registry.Register("ir.ui.menu", &rpc.ModuleHandler{
		Search: SearchMenu,
	})

	registry.Register("ir.ui.view", &rpc.ModuleHandler{
		Validate: ValidateView,
	})

	registry.Register("ir.lang", &rpc.ModuleHandler{
		Validate: ValidateLang,
	})

	registry.Register("ir.module", &rpc.ModuleHandler{
		ButtonActions: map[string]func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error){
			"activate":   ButtonModuleTransition("activate"),
			"upgrade":    ButtonModuleTransition("upgrade"),
			"deactivate": ButtonModuleTransition("deactivate"),
			"cancel":     ButtonModuleTransition("cancel"),
		},
	})
}
