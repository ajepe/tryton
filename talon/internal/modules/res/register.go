package res

import (
	"context"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/rpc"
)

func RegisterHandlers(registry *rpc.ModuleRegistry) {
	registry.Register("res.user", &rpc.ModuleHandler{
		OnChange: OnChangeUser,
		Search:   SearchUser,
		Validate: ValidateUser,
	})

	registry.Register("res.group", &rpc.ModuleHandler{
		Validate: ValidateGroup,
	})

	registry.Register("res.notification", &rpc.ModuleHandler{
		OnChange: OnChangeNotification,
		ButtonActions: map[string]func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error){
			"mark_read": ButtonMarkRead,
		},
		Validate: ValidateNotification,
	})

	registry.Register("res.user.application", &rpc.ModuleHandler{
		ButtonActions: map[string]func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error){
			"validate": ButtonApplicationTransition("validate"),
			"cancel":   ButtonApplicationTransition("cancel"),
		},
		Validate: ValidateApplication,
	})
}
