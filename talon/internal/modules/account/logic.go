package account

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajepe/talon/internal/orm"
)

func MoveLineBalance(lines []map[string]interface{}) (debit, credit float64) {
	for _, line := range lines {
		d, _ := orm.AsFloat(line["debit"])
		c, _ := orm.AsFloat(line["credit"])
		debit += d
		credit += c
	}
	return debit, credit
}

func IsMoveBalanced(debit, credit float64) bool {
	return orm.EqualRound(debit, credit, 2)
}

func ValidateMove(ctx context.Context, group *orm.Group, record *orm.Record) error {
	switch orm.AsString(valueAt(record, "state")) {
	case "", "draft", "posted":
	default:
		return fmt.Errorf("invalid state")
	}
	if date := orm.AsDateString(valueAt(record, "date")); date == "" {
		if _, ok := record.Get("date"); ok {
			return fmt.Errorf("invalid date")
		}
	}
	return nil
}

func ValidateMoveLine(ctx context.Context, group *orm.Group, record *orm.Record) error {
	debit, _ := orm.AsFloat(valueAt(record, "debit"))
	credit, _ := orm.AsFloat(valueAt(record, "credit"))
	if debit < 0 || credit < 0 {
		return fmt.Errorf("debit and credit must not be negative")
	}
	if debit != 0 && credit != 0 {
		return fmt.Errorf("a move line cannot have both debit and credit")
	}
	return nil
}

func OnChangeMoveLine(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	triggered := map[string]bool{}
	for _, f := range fields {
		triggered[f] = true
	}
	if !triggered["credit"] {
		if debit, ok := orm.AsFloat(values["debit"]); ok && debit != 0 {
			result["credit"] = 0.0
		}
	}
	if !triggered["debit"] {
		if credit, ok := orm.AsFloat(values["credit"]); ok && credit != 0 {
			result["debit"] = 0.0
		}
	}
	return result, nil
}

func ButtonMovePost(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
	ids := []interface{}{}
	for _, r := range records {
		debit, credit, err := moveBalance(ctx, group, r)
		if err != nil {
			return nil, err
		}
		if !IsMoveBalanced(debit, credit) {
			return nil, fmt.Errorf("move %s is not balanced", r.RecName())
		}
		r.Set("state", "posted")
		if err := r.Save(); err != nil {
			return nil, err
		}
		ids = append(ids, r.ID())
	}
	return ids, nil
}

func ButtonMoveDraft(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
	ids := []interface{}{}
	for _, r := range records {
		if state := orm.AsString(valueAt(r, "state")); state != "" && state != "posted" {
			return nil, fmt.Errorf("cannot reset move in state %q to draft", state)
		}
		r.Set("state", "draft")
		if err := r.Save(); err != nil {
			return nil, err
		}
		ids = append(ids, r.ID())
	}
	return ids, nil
}

func moveBalance(ctx context.Context, group *orm.Group, r *orm.Record) (float64, float64, error) {
	if lines := linesFromRecord(r); lines != nil {
		debit, credit := MoveLineBalance(lines)
		return debit, credit, nil
	}
	db := group.DB()
	if db == nil {
		return 0, 0, fmt.Errorf("no database connection")
	}
	var debit, credit float64
	err := db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(debit), 0), COALESCE(SUM(credit), 0)
		 FROM account_move_line WHERE move = $1`, r.ID()).Scan(&debit, &credit)
	if err != nil {
		return 0, 0, err
	}
	return debit, credit, nil
}

func linesFromRecord(r *orm.Record) []map[string]interface{} {
	v, ok := r.Get("lines")
	if !ok {
		return nil
	}
	raw, ok := v.([]interface{})
	if !ok {
		if lines, ok := v.([]map[string]interface{}); ok {
			return lines
		}
		return nil
	}
	lines := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		if line, ok := item.(map[string]interface{}); ok {
			lines = append(lines, line)
		}
	}
	return lines
}

func ValidateFiscalyear(ctx context.Context, group *orm.Group, record *orm.Record) error {
	start, okStart := orm.AsDate(valueAt(record, "start_date"))
	end, okEnd := orm.AsDate(valueAt(record, "end_date"))
	if !okStart || !okEnd {
		return fmt.Errorf("start_date and end_date are required")
	}
	if end.Before(start) {
		return fmt.Errorf("end_date must be after start_date")
	}
	switch orm.AsString(valueAt(record, "state")) {
	case "", "open", "closed", "locked":
	default:
		return fmt.Errorf("invalid state")
	}
	return nil
}

func ValidatePeriod(ctx context.Context, group *orm.Group, record *orm.Record) error {
	start, okStart := orm.AsDate(valueAt(record, "start_date"))
	end, okEnd := orm.AsDate(valueAt(record, "end_date"))
	if !okStart || !okEnd {
		return fmt.Errorf("start_date and end_date are required")
	}
	if end.Before(start) {
		return fmt.Errorf("end_date must be after start_date")
	}
	switch orm.AsString(valueAt(record, "state")) {
	case "", "open", "closed", "locked":
	default:
		return fmt.Errorf("invalid state")
	}
	return nil
}

func ValidateAccount(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if strings.TrimSpace(orm.AsString(valueAt(record, "name"))) == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func ValidateJournal(ctx context.Context, group *orm.Group, record *orm.Record) error {
	switch orm.AsString(valueAt(record, "type")) {
	case "", "general", "revenue", "expense", "cash", "situation", "write-off":
	default:
		return fmt.Errorf("invalid journal type")
	}
	return nil
}

func SearchAccount(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	pattern := "%" + text + "%"
	rows, err := db.QueryContext(ctx,
		`SELECT id FROM account_account
		 WHERE code ILIKE $1 OR name ILIKE $1
		 ORDER BY id`, pattern)
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

func SearchMove(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	rows, err := db.QueryContext(ctx,
		`SELECT id FROM account_move WHERE number ILIKE $1 ORDER BY id`, "%"+text+"%")
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
