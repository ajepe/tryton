package sale

import (
	"context"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/rpc"
)

func RegisterHandlers(registry *rpc.ModuleRegistry) {
	registry.Register("sale.sale", &rpc.ModuleHandler{
		OnChange: OnChangeSale,
		ButtonActions: map[string]func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error){
			"quote":   buttonTransition("quote"),
			"confirm": buttonTransition("confirm"),
			"cancel":  buttonTransition("cancel"),
			"draft":   buttonTransition("draft"),
			"process": buttonTransition("process"),
		},
		Search:   SearchSale,
		Validate: ValidateSale,
	})

	registry.Register("sale.line", &rpc.ModuleHandler{
		OnChange: OnChangeLine,
		Validate: ValidateSaleLine,
	})
}
