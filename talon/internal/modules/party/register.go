package party

import (
	"context"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/rpc"
)

func RegisterHandlers(registry *rpc.ModuleRegistry) {
	registry.Register("party.party", &rpc.ModuleHandler{
		OnChange:     OnChangeName,
		OnChangeWith: OnChangeWithCategory,
		ButtonActions: map[string]func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error){
			"duplicate":  ButtonDuplicate,
			"erase":      ButtonErase,
			"activate":   ButtonActivate,
			"deactivate": ButtonDeactivate,
		},
		Search:   SearchByName,
		Validate: ValidateParty,
	})

	registry.Register("party.address", &rpc.ModuleHandler{
		OnChange: OnChangeAddress,
	})

	registry.Register("party.contact_mechanism", &rpc.ModuleHandler{
		OnChange: OnChangeContactMechanism,
	})
}
