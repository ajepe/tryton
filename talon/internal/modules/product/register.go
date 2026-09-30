package product

import (
	"github.com/ajepe/talon/internal/rpc"
)

func RegisterHandlers(registry *rpc.ModuleRegistry) {
	registry.Register("product.template", &rpc.ModuleHandler{
		Search:   SearchTemplate,
		Validate: ValidateTemplate,
	})

	registry.Register("product.product", &rpc.ModuleHandler{
		OnChange: OnChangeProduct,
		Search:   SearchProduct,
	})

	registry.Register("product.category", &rpc.ModuleHandler{
		Search:   SearchCategory,
		Validate: ValidateCategory,
	})

	registry.Register("product.uom", &rpc.ModuleHandler{
		OnChange: OnChangeUom,
		Search:   SearchUom,
		Validate: ValidateUom,
	})

	registry.Register("product.uom.category", &rpc.ModuleHandler{})
}
