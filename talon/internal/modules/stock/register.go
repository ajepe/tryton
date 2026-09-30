package stock

import (
	"context"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/rpc"
)

func RegisterHandlers(registry *rpc.ModuleRegistry) {
	registry.Register("stock.location", &rpc.ModuleHandler{
		Search: SearchLocation,
	})

	registry.Register("stock.move", &rpc.ModuleHandler{
		OnChange: OnChangeMove,
		ButtonActions: map[string]func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error){
			"draft":  buttonTransition(moveTransitions, "draft"),
			"assign": buttonTransition(moveTransitions, "assign"),
			"do":     buttonTransition(moveTransitions, "do"),
			"cancel": buttonTransition(moveTransitions, "cancel"),
		},
		Validate: ValidateMove,
	})

	registry.Register("stock.shipment.out", &rpc.ModuleHandler{
		ButtonActions: map[string]func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error){
			"wait":   buttonTransition(shipmentOutTransitions, "wait"),
			"assign": buttonTransition(shipmentOutTransitions, "assign"),
			"pick":   buttonTransition(shipmentOutTransitions, "pick"),
			"pack":   buttonTransition(shipmentOutTransitions, "pack"),
			"ship":   buttonTransition(shipmentOutTransitions, "ship"),
			"do":     buttonTransition(shipmentOutTransitions, "do"),
			"cancel": buttonTransition(shipmentOutTransitions, "cancel"),
			"draft":  buttonTransition(shipmentOutTransitions, "draft"),
		},
		Search:   SearchShipmentOut,
		Validate: ValidateShipmentOut,
	})

	registry.Register("stock.shipment.in", &rpc.ModuleHandler{
		ButtonActions: map[string]func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error){
			"receive": buttonTransition(shipmentInTransitions, "receive"),
			"do":      buttonTransition(shipmentInTransitions, "do"),
			"cancel":  buttonTransition(shipmentInTransitions, "cancel"),
			"draft":   buttonTransition(shipmentInTransitions, "draft"),
		},
		Search:   SearchShipmentIn,
		Validate: ValidateShipmentIn,
	})

	registry.Register("stock.inventory", &rpc.ModuleHandler{
		ButtonActions: map[string]func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error){
			"confirm": buttonTransition(inventoryTransitions, "confirm"),
			"cancel":  buttonTransition(inventoryTransitions, "cancel"),
		},
		Validate: ValidateInventory,
	})

	registry.Register("stock.inventory.line", &rpc.ModuleHandler{
		OnChange: OnChangeInventoryLine,
		Validate: ValidateInventoryLine,
	})
}
