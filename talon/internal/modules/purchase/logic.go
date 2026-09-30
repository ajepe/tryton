package purchase

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajepe/talon/internal/orm"
)

var purchaseTransitions = map[string]map[string]string{
	"quote": {
		"draft": "quotation",
	},
	"confirm": {
		"quotation": "confirmed",
	},
	"cancel": {
		"draft":     "cancelled",
		"quotation": "cancelled",
	},
	"draft": {
		"cancelled": "draft",
		"quotation": "draft",
		"confirmed": "draft",
	},
	"process": {
		"confirmed":  "processing",
		"processing": "processing",
	},
}

func LineAmount(quantity, unitPrice float64) float64 {
	return orm.Round(quantity*unitPrice, 2)
}

func SumLineAmounts(lines []map[string]interface{}) float64 {
	total := 0.0
	for _, line := range lines {
		if amount, ok := orm.AsFloat(line["amount"]); ok {
			total += amount
			continue
		}
		quantity, _ := orm.AsFloat(line["quantity"])
		unitPrice, _ := orm.AsFloat(line["unit_price"])
		total += LineAmount(quantity, unitPrice)
	}
	return orm.Round(total, 2)
}

func OnChangeLine(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	triggered := map[string]bool{}
	for _, f := range fields {
		triggered[f] = true
	}
	if triggered["product"] {
		if id, ok := orm.AsInt(values["product"]); ok && id != 0 {
			_, price, description := productPurchaseInfo(ctx, group, id)
			if orm.AsString(values["description"]) == "" {
				result["description"] = description
			}
			if _, ok := orm.AsFloat(values["unit_price"]); !ok || price != 0 {
				result["unit_price"] = price
			}
		}
	}
	quantity, _ := orm.AsFloat(values["quantity"])
	unitPrice, _ := orm.AsFloat(values["unit_price"])
	if v, ok := result["unit_price"]; ok {
		unitPrice, _ = orm.AsFloat(v)
	}
	if _, ok := values["quantity"]; ok {
		result["amount"] = LineAmount(quantity, unitPrice)
	} else if _, ok := values["unit_price"]; ok {
		result["amount"] = LineAmount(quantity, unitPrice)
	}
	return result, nil
}

func productPurchaseInfo(ctx context.Context, group *orm.Group, productID int) (name string, price float64, description string) {
	db := group.DB()
	if db == nil {
		return "", 0, ""
	}
	err := db.QueryRowContext(ctx,
		`SELECT COALESCE(p.name, t.name, ''), COALESCE(t.cost_price, 0), COALESCE(p.description, '')
		 FROM product_product p
		 JOIN product_template t ON t.id = p.template
		 WHERE p.id = $1`, productID).Scan(&name, &price, &description)
	if err != nil {
		return "", 0, ""
	}
	return name, price, description
}

func OnChangePurchase(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if lines := linesFromValues(values); lines != nil {
		result["amount"] = SumLineAmounts(lines)
	}
	return result, nil
}

func linesFromValues(values map[string]interface{}) []map[string]interface{} {
	raw, ok := values["lines"]
	if !ok {
		return nil
	}
	items, ok := raw.([]interface{})
	if !ok {
		if lines, ok := raw.([]map[string]interface{}); ok {
			return lines
		}
		return nil
	}
	lines := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if line, ok := item.(map[string]interface{}); ok {
			lines = append(lines, line)
		}
	}
	return lines
}

func ValidatePurchase(ctx context.Context, group *orm.Group, record *orm.Record) error {
	supplier, ok := record.Get("supplier")
	if id, isInt := orm.AsInt(supplier); !ok || !isInt || id == 0 {
		return fmt.Errorf("supplier is required")
	}
	switch orm.AsString(valueAt(record, "state")) {
	case "", "draft", "quotation", "confirmed", "processing", "done", "cancelled":
	default:
		return fmt.Errorf("invalid state")
	}
	return nil
}

func ValidatePurchaseLine(ctx context.Context, group *orm.Group, record *orm.Record) error {
	quantity, okQuantity := orm.AsFloat(valueAt(record, "quantity"))
	unitPrice, okUnitPrice := orm.AsFloat(valueAt(record, "unit_price"))
	amount, okAmount := orm.AsFloat(valueAt(record, "amount"))
	if okQuantity && okUnitPrice && okAmount {
		if !orm.EqualRound(amount, LineAmount(quantity, unitPrice), 2) {
			return fmt.Errorf("amount does not match quantity and unit_price")
		}
	}
	return nil
}

func NextState(action string, current string) (string, error) {
	if current == "" {
		current = "draft"
	}
	byAction, ok := purchaseTransitions[action]
	if !ok {
		return "", fmt.Errorf("unknown action %s", action)
	}
	next, ok := byAction[current]
	if !ok {
		return "", fmt.Errorf("cannot %s purchase in state %q", action, current)
	}
	return next, nil
}

func buttonTransition(action string) func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
	return func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
		ids := []interface{}{}
		for _, r := range records {
			next, err := NextState(action, orm.AsString(valueAt(r, "state")))
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

func RecName(number string) string {
	if number != "" {
		return number
	}
	return ""
}

func SearchPurchase(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	rows, err := db.QueryContext(ctx,
		`SELECT id FROM purchase_purchase WHERE number ILIKE $1 ORDER BY id`, strings.TrimSpace("%"+text+"%"))
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
