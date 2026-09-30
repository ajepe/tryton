package product

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajepe/talon/internal/orm"
)

const uomConversionDecimals = 12

func RecName(code, name string) string {
	if code != "" {
		return fmt.Sprintf("[%s] %s", code, name)
	}
	return name
}

func OnChangeProduct(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	template, ok := orm.AsInt(values["template"])
	if !ok || template == 0 {
		return result, nil
	}
	db := group.DB()
	if db == nil {
		return result, nil
	}

	var name, code string
	err := db.QueryRowContext(ctx,
		`SELECT COALESCE(name, ''), COALESCE(code, '') FROM product_template WHERE id = $1`,
		template).Scan(&name, &code)
	if err != nil {
		return nil, err
	}
	if orm.AsString(values["name"]) == "" {
		result["name"] = name
	}
	if orm.AsString(values["code"]) == "" {
		result["code"] = code
	}
	return result, nil
}

func UomRateFromFactor(factor float64) float64 {
	if factor == 0 {
		return 0
	}
	return orm.Round(1.0/factor, uomConversionDecimals)
}

func UomFactorFromRate(rate float64) float64 {
	if rate == 0 {
		return 0
	}
	return orm.Round(1.0/rate, uomConversionDecimals)
}

func OnChangeUom(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	triggered := map[string]bool{}
	for _, f := range fields {
		triggered[f] = true
	}
	switch {
	case triggered["factor"]:
		factor, _ := orm.AsFloat(values["factor"])
		result["factor"] = factor
		result["rate"] = UomRateFromFactor(factor)
	case triggered["rate"]:
		rate, _ := orm.AsFloat(values["rate"])
		result["rate"] = rate
		result["factor"] = UomFactorFromRate(rate)
	default:
		if factor, ok := orm.AsFloat(values["factor"]); ok && factor != 0 {
			result["rate"] = UomRateFromFactor(factor)
		} else if rate, ok := orm.AsFloat(values["rate"]); ok && rate != 0 {
			result["factor"] = UomFactorFromRate(rate)
		}
	}
	return result, nil
}

func ValidateTemplate(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if strings.TrimSpace(orm.AsString(valueAt(record, "name"))) == "" {
		return fmt.Errorf("name is required")
	}
	if v, ok := record.Get("type"); ok {
		switch orm.AsString(v) {
		case "", "goods", "assets", "service":
		default:
			return fmt.Errorf("invalid type")
		}
	}
	for _, name := range []string{"sale_price", "cost_price"} {
		v, ok := record.Get(name)
		if !ok || v == nil {
			continue
		}
		if f, ok := orm.AsFloat(v); ok && f < 0 {
			return fmt.Errorf("%s must not be negative", name)
		}
	}
	return nil
}

func ValidateCategory(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if strings.TrimSpace(orm.AsString(valueAt(record, "name"))) == "" {
		return fmt.Errorf("name is required")
	}
	if parent, ok := record.Get("parent"); ok {
		if id, ok := orm.AsInt(parent); ok && id != 0 && id == record.ID() {
			return fmt.Errorf("a category cannot be its own parent")
		}
	}
	return nil
}

func ValidateUom(ctx context.Context, group *orm.Group, record *orm.Record) error {
	name := strings.TrimSpace(orm.AsString(valueAt(record, "name")))
	if name == "" {
		return fmt.Errorf("name is required")
	}
	rate, _ := orm.AsFloat(valueAt(record, "rate"))
	factor, _ := orm.AsFloat(valueAt(record, "factor"))
	if rate == 0 && factor == 0 {
		return fmt.Errorf("rate and factor cannot both be zero")
	}
	if rate != 0 && factor != 0 {
		okRate := orm.EqualRound(rate, orm.Round(1.0/factor, uomConversionDecimals), uomConversionDecimals)
		okFactor := orm.EqualRound(factor, orm.Round(1.0/rate, uomConversionDecimals), uomConversionDecimals)
		if !okRate && !okFactor {
			return fmt.Errorf("rate and factor are incompatible")
		}
	}
	return nil
}

func SearchTemplate(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	return searchColumns(ctx, group, "product_template", []string{"name", "code"}, text)
}

func SearchProduct(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	return searchColumns(ctx, group, "product_product", []string{"name", "code", "description"}, text)
}

func SearchCategory(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	return searchColumns(ctx, group, "product_category", []string{"name"}, text)
}

func SearchUom(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	return searchColumns(ctx, group, "product_uom", []string{"name", "symbol"}, text)
}

func searchColumns(ctx context.Context, group *orm.Group, table string, columns []string, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	conditions := make([]string, len(columns))
	for i, column := range columns {
		conditions[i] = column + " ILIKE $1"
	}
	query := fmt.Sprintf("SELECT id FROM %s WHERE %s ORDER BY id",
		table, strings.Join(conditions, " OR "))
	rows, err := db.QueryContext(ctx, query, "%"+text+"%")
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
