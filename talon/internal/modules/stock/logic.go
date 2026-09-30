package stock

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajepe/talon/internal/orm"
)

var moveTransitions = map[string]map[string]string{
	"draft": {
		"staging":  "draft",
		"assigned": "draft",
	},
	"assign": {
		"draft":    "assigned",
		"staging":  "assigned",
		"assigned": "assigned",
	},
	"do": {
		"draft":    "done",
		"assigned": "done",
	},
	"cancel": {
		"staging":  "cancelled",
		"draft":    "cancelled",
		"assigned": "cancelled",
		"done":     "cancelled",
	},
}

var shipmentOutTransitions = map[string]map[string]string{
	"wait": {
		"draft":   "waiting",
		"packed":  "waiting",
		"shipped": "waiting",
	},
	"assign": {
		"waiting": "assigned",
	},
	"pick": {
		"waiting":  "picked",
		"assigned": "picked",
	},
	"pack": {
		"assigned": "packed",
		"picked":   "packed",
	},
	"ship": {
		"packed": "shipped",
	},
	"do": {
		"packed":  "done",
		"shipped": "done",
	},
	"cancel": {
		"draft":    "cancelled",
		"waiting":  "cancelled",
		"assigned": "cancelled",
		"picked":   "cancelled",
		"packed":   "cancelled",
		"shipped":  "cancelled",
		"done":     "cancelled",
	},
	"draft": {
		"cancelled": "draft",
	},
}

var shipmentInTransitions = map[string]map[string]string{
	"receive": {
		"draft": "received",
	},
	"do": {
		"received": "done",
	},
	"cancel": {
		"draft":    "cancelled",
		"received": "cancelled",
		"done":     "cancelled",
	},
	"draft": {
		"cancelled": "draft",
	},
}

var inventoryTransitions = map[string]map[string]string{
	"confirm": {
		"draft": "done",
	},
	"cancel": {
		"draft": "cancelled",
	},
}

func NextState(action string, transitions map[string]map[string]string, current string) (string, error) {
	if current == "" {
		current = "draft"
	}
	byAction, ok := transitions[action]
	if !ok {
		return "", fmt.Errorf("unknown action %s", action)
	}
	next, ok := byAction[current]
	if !ok {
		return "", fmt.Errorf("cannot %s in state %q", action, current)
	}
	return next, nil
}

func buttonTransition(transitions map[string]map[string]string, action string) func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
	return func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
		ids := []interface{}{}
		for _, r := range records {
			next, err := NextState(action, transitions, orm.AsString(valueAt(r, "state")))
			if err != nil {
				return nil, err
			}
			r.Set("state", next)
			if err := r.Save(); err != nil {
				return nil, err
			}
			ids = append(ids, r.ID())
		}
		return ids, nil
	}
}

func OnChangeMove(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if uom, ok := orm.AsInt(values["uom"]); ok && uom != 0 {
		return result, nil
	}
	productID, ok := orm.AsInt(values["product"])
	if !ok || productID == 0 {
		return result, nil
	}
	if uom := productUom(ctx, group, productID); uom != 0 {
		result["uom"] = uom
	}
	return result, nil
}

func OnChangeInventoryLine(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	return OnChangeMove(ctx, group, values, fields)
}

func productUom(ctx context.Context, group *orm.Group, productID int) int {
	db := group.DB()
	if db == nil {
		return 0
	}
	var uom int
	err := db.QueryRowContext(ctx,
		`SELECT COALESCE(t.uom, 0)
		 FROM product_product p
		 JOIN product_template t ON t.id = p.template
		 WHERE p.id = $1`, productID).Scan(&uom)
	if err != nil {
		return 0
	}
	return uom
}

func ValidateMove(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if quantity, ok := orm.AsFloat(valueAt(record, "quantity")); ok && quantity < 0 {
		return fmt.Errorf("quantity must not be negative")
	}
	if from, ok := orm.AsInt(valueAt(record, "from_location")); ok {
		if to, ok := orm.AsInt(valueAt(record, "to_location")); ok && from != 0 && from == to {
			return fmt.Errorf("from_location and to_location must differ")
		}
	}
	switch orm.AsString(valueAt(record, "state")) {
	case "", "staging", "draft", "assigned", "done", "cancelled":
	default:
		return fmt.Errorf("invalid state")
	}
	return nil
}

func ValidateShipmentOut(ctx context.Context, group *orm.Group, record *orm.Record) error {
	customer, ok := record.Get("customer")
	if id, isInt := orm.AsInt(customer); !ok || !isInt || id == 0 {
		return fmt.Errorf("customer is required")
	}
	switch orm.AsString(valueAt(record, "state")) {
	case "", "draft", "waiting", "assigned", "picked", "packed", "shipped", "done", "cancelled":
	default:
		return fmt.Errorf("invalid state")
	}
	return nil
}

func ValidateShipmentIn(ctx context.Context, group *orm.Group, record *orm.Record) error {
	supplier, ok := record.Get("supplier")
	if id, isInt := orm.AsInt(supplier); !ok || !isInt || id == 0 {
		return fmt.Errorf("supplier is required")
	}
	switch orm.AsString(valueAt(record, "state")) {
	case "", "draft", "received", "done", "cancelled":
	default:
		return fmt.Errorf("invalid state")
	}
	return nil
}

func ValidateInventory(ctx context.Context, group *orm.Group, record *orm.Record) error {
	location, ok := record.Get("location")
	if id, isInt := orm.AsInt(location); !ok || !isInt || id == 0 {
		return fmt.Errorf("location is required")
	}
	switch orm.AsString(valueAt(record, "state")) {
	case "", "draft", "done", "cancelled":
	default:
		return fmt.Errorf("invalid state")
	}
	return nil
}

func ValidateInventoryLine(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if quantity, ok := orm.AsFloat(valueAt(record, "quantity")); ok && quantity < 0 {
		return fmt.Errorf("quantity must not be negative")
	}
	product, ok := record.Get("product")
	if id, isInt := orm.AsInt(product); !ok || !isInt || id == 0 {
		return fmt.Errorf("product is required")
	}
	return nil
}

func SearchShipmentOut(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	return searchColumn(ctx, group, "stock_shipment_out", "number", text)
}

func SearchShipmentIn(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	return searchColumn(ctx, group, "stock_shipment_in", "number", text)
}

func SearchLocation(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	return searchColumn(ctx, group, "stock_location", "name", text)
}

func searchColumn(ctx context.Context, group *orm.Group, table, column, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	query := fmt.Sprintf("SELECT id FROM %s WHERE %s ILIKE $1 ORDER BY id", table, column)
	rows, err := db.QueryContext(ctx, query, strings.TrimSpace("%"+text+"%"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func valueAt(r *orm.Record, name string) interface{} {
	v, _ := r.Get(name)
	return v
}
