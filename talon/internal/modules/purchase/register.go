package purchase

import (
	"context"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/rpc"
)

func RegisterHandlers(registry *rpc.ModuleRegistry) {
	registry.Register("purchase.purchase", &rpc.ModuleHandler{
		OnChange: OnChangePurchase,
		ButtonActions: map[string]func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error){
			"quote":   buttonTransition("quote"),
			"confirm": buttonTransition("confirm"),
			"cancel":  buttonTransition("cancel"),
			"draft":   buttonTransition("draft"),
			"process": buttonTransition("process"),
		},
		Search:   SearchPurchase,
		Validate: ValidatePurchase,
	})

	registry.Register("purchase.line", &rpc.ModuleHandler{
		OnChange: OnChangeLine,
		Validate: ValidatePurchaseLine,
	})
}
