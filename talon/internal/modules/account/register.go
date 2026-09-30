package account

import (
	"context"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/rpc"
)

func RegisterHandlers(registry *rpc.ModuleRegistry) {
	registry.Register("account.account", &rpc.ModuleHandler{
		Search:   SearchAccount,
		Validate: ValidateAccount,
	})

	registry.Register("account.fiscalyear", &rpc.ModuleHandler{
		Validate: ValidateFiscalyear,
	})

	registry.Register("account.period", &rpc.ModuleHandler{
		Validate: ValidatePeriod,
	})

	registry.Register("account.move", &rpc.ModuleHandler{
		ButtonActions: map[string]func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error){
			"post":  ButtonMovePost,
			"draft": ButtonMoveDraft,
		},
		Search:   SearchMove,
		Validate: ValidateMove,
	})

	registry.Register("account.move.line", &rpc.ModuleHandler{
		OnChange: OnChangeMoveLine,
		Validate: ValidateMoveLine,
	})

	registry.Register("account.journal", &rpc.ModuleHandler{
		Validate: ValidateJournal,
	})
}
