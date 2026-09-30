package ir

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajepe/talon/internal/orm"
)

var moduleTransitions = map[string]map[string]string{
	"activate": {
		"not activated": "to activate",
	},
	"upgrade": {
		"activated": "to upgrade",
	},
	"deactivate": {
		"activated": "to remove",
	},
	"cancel": {
		"to activate": "not activated",
		"to remove":   "not activated",
		"to upgrade":  "activated",
	},
}

func SequencePreview(prefix string, numberNext, padding int, suffix string) string {
	return prefix + formatIncremental(numberNext, padding) + suffix
}

func formatIncremental(number, padding int) string {
	s := fmt.Sprintf("%d", number)
	if padding > len(s) {
		s = strings.Repeat("0", padding-len(s)) + s
	}
	return s
}

func OnChangeSequence(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	prefix := orm.AsString(values["prefix"])
	suffix := orm.AsString(values["suffix"])
	numberNext, _ := orm.AsInt(values["number_next"])
	if numberNext == 0 {
		numberNext = 1
	}
	padding, _ := orm.AsInt(values["padding"])
	result["preview"] = SequencePreview(prefix, numberNext, padding, suffix)
	return result, nil
}

func ValidateSequence(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if v, ok := record.Get("number_increment"); ok {
		if n, ok := orm.AsInt(v); ok && n < 1 {
			return fmt.Errorf("number_increment must be greater than zero")
		}
	}
	if v, ok := record.Get("number_next"); ok {
		if n, ok := orm.AsInt(v); ok && n < 1 {
			return fmt.Errorf("number_next must be greater than zero")
		}
	}
	if v, ok := record.Get("padding"); ok {
		if n, ok := orm.AsInt(v); ok && n < 0 {
			return fmt.Errorf("padding must not be negative")
		}
	}
	for _, name := range []string{"prefix", "suffix"} {
		if v, ok := record.Get(name); ok {
			if err := validateAffix(orm.AsString(v)); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	return nil
}

func ButtonGetSequence(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	results := make([]interface{}, 0, len(records))
	for _, r := range records {
		increment := 1
		if v, ok := r.Get("number_increment"); ok {
			if n, ok := orm.AsInt(v); ok && n > 0 {
				increment = n
			}
		}
		var next int
		query := `UPDATE ir_sequence
			SET number_next = number_next + $1
			WHERE id = $2
			RETURNING number_next - $1`
		if err := db.QueryRowContext(ctx, query, increment, r.ID()).Scan(&next); err != nil {
			return nil, err
		}
		if v, ok := r.Get("number_next"); ok {
			if n, ok := orm.AsInt(v); ok {
				r.Set("number_next", n+increment)
			}
		}
		padding, _ := orm.AsInt(valueAt(r, "padding"))
		results = append(results, SequencePreview(
			orm.AsString(valueAt(r, "prefix")), next, padding, orm.AsString(valueAt(r, "suffix"))))
	}
	return results, nil
}

func validateAffix(affix string) error {
	for i := 0; i < len(affix); i++ {
		if affix[i] != '$' {
			continue
		}
		if i+1 < len(affix) && affix[i+1] == '$' {
			i++
			continue
		}
		if i+1 >= len(affix) || affix[i+1] != '{' {
			return fmt.Errorf("invalid placeholder at position %d", i)
		}
		end := strings.IndexByte(affix[i+2:], '}')
		if end < 0 {
			return fmt.Errorf("unclosed placeholder at position %d", i)
		}
		key := affix[i+2 : i+2+end]
		if !validAffixKey(key) {
			return fmt.Errorf("unknown placeholder %q", key)
		}
		i += 2 + end
	}
	return nil
}

func validAffixKey(key string) bool {
	switch key {
	case "date", "year", "month", "day":
		return true
	}
	return strings.HasPrefix(key, "date_") && len(key) > len("date_")
}

func ValidateCron(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if v, ok := record.Get("model"); ok && strings.TrimSpace(orm.AsString(v)) == "" {
		return fmt.Errorf("model is required")
	}
	if v, ok := record.Get("function"); ok && strings.TrimSpace(orm.AsString(v)) == "" {
		return fmt.Errorf("function is required")
	}
	if v, ok := record.Get("interval_number"); ok {
		if n, ok := orm.AsInt(v); ok && n < 1 {
			return fmt.Errorf("interval_number must be greater than zero")
		}
	}
	if v, ok := record.Get("interval_type"); ok {
		switch orm.AsString(v) {
		case "", "minutes", "hours", "days", "weeks", "months":
		default:
			return fmt.Errorf("invalid interval_type")
		}
	}
	return nil
}

func OnChangeAttachment(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if _, ok := values["description"]; ok {
		description := orm.AsString(values["description"])
		if idx := strings.IndexByte(description, '\n'); idx >= 0 {
			description = description[:idx]
		}
		result["summary"] = description
	}
	return result, nil
}

func ValidateLang(ctx context.Context, group *orm.Group, record *orm.Record) error {
	decimalPoint := orm.AsString(valueAt(record, "decimal_point"))
	thousandsSep := orm.AsString(valueAt(record, "thousands_sep"))
	if decimalPoint != "" && decimalPoint == thousandsSep {
		return fmt.Errorf("decimal point and thousands separator must differ")
	}
	if v, ok := record.Get("direction"); ok {
		switch orm.AsString(v) {
		case "", "ltr", "rtl":
		default:
			return fmt.Errorf("invalid direction")
		}
	}
	return nil
}

func ValidateView(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if v, ok := record.Get("model"); ok && strings.TrimSpace(orm.AsString(v)) == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

func NextModuleState(action, current string) (string, error) {
	if current == "" {
		current = "not activated"
	}
	transitions, ok := moduleTransitions[action]
	if !ok {
		return "", fmt.Errorf("unknown action %s", action)
	}
	next, ok := transitions[current]
	if !ok {
		return "", fmt.Errorf("cannot %s module in state %q", action, current)
	}
	return next, nil
}

func ButtonModuleTransition(action string) func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
	return func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
		ids := []interface{}{}
		for _, r := range records {
			next, err := NextModuleState(action, orm.AsString(valueAt(r, "state")))
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

func SearchAttachment(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	return searchColumn(ctx, group, "ir_attachment", "description", text)
}

func SearchTranslation(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	pattern := "%" + text + "%"
	rows, err := db.QueryContext(ctx,
		`SELECT id FROM ir_translation
		 WHERE src ILIKE $1 OR value ILIKE $1 OR name ILIKE $1
		 ORDER BY id`, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIDs(rows)
}

func SearchMenu(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	return searchColumn(ctx, group, "ir_ui_menu", "name", text)
}

func searchColumn(ctx context.Context, group *orm.Group, table, column, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	query := fmt.Sprintf("SELECT id FROM %s WHERE %s ILIKE $1 ORDER BY id", table, column)
	rows, err := db.QueryContext(ctx, query, "%"+text+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIDs(rows)
}

func scanIDs(rows interface {
	Next() bool
	Scan(...interface{}) error
}) ([]int, error) {
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
