package company

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ajepe/talon/internal/orm"
)

func ValidateCompany(ctx context.Context, group *orm.Group, record *orm.Record) error {
	party, ok := record.Get("party")
	if id, isInt := orm.AsInt(party); !ok || !isInt || id == 0 {
		return fmt.Errorf("party is required")
	}
	currency, ok := record.Get("currency")
	if id, isInt := orm.AsInt(currency); !ok || !isInt || id == 0 {
		return fmt.Errorf("currency is required")
	}
	return nil
}

func ValidateEmployee(ctx context.Context, group *orm.Group, record *orm.Record) error {
	party, ok := record.Get("party")
	if id, isInt := orm.AsInt(party); !ok || !isInt || id == 0 {
		return fmt.Errorf("party is required")
	}
	company, ok := record.Get("company")
	if id, isInt := orm.AsInt(company); !ok || !isInt || id == 0 {
		return fmt.Errorf("company is required")
	}
	start, hasStart := orm.AsDate(valueAt(record, "start_date"))
	end, hasEnd := orm.AsDate(valueAt(record, "end_date"))
	if hasStart && hasEnd && end.Before(start) {
		return fmt.Errorf("end_date must be after start_date")
	}
	return nil
}

func IsEmployeeActive(startDate, endDate string, date time.Time) bool {
	start, hasStart := asDateOrZero(startDate)
	end, hasEnd := asDateOrZero(endDate)
	if hasStart && date.Before(start) {
		return false
	}
	if hasEnd && date.After(end) {
		return false
	}
	return true
}

func asDateOrZero(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	parsed, ok := orm.AsDate(value)
	return parsed, ok
}

func OnChangeEmployee(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	triggered := map[string]bool{}
	for _, field := range fields {
		triggered[field] = true
	}
	if triggered["start_date"] || triggered["end_date"] {
		date := time.Now()
		if contextMap, ok := values["_date"].(string); ok && contextMap != "" {
			if parsed, ok := orm.AsDate(contextMap); ok {
				date = parsed
			}
		}
		start, _ := values["start_date"].(string)
		end, _ := values["end_date"].(string)
		result["active"] = IsEmployeeActive(start, end, date)
	}
	return result, nil
}

func SearchCompany(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	rows, err := db.QueryContext(ctx,
		`SELECT c.id
		 FROM company_company c
		 JOIN party_party p ON p.id = c.party
		 WHERE p.name ILIKE $1 OR p.code ILIKE $1
		 ORDER BY c.id`, "%"+strings.TrimSpace(text)+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIDs(rows)
}

func SearchEmployee(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	rows, err := db.QueryContext(ctx,
		`SELECT e.id
		 FROM company_employee e
		 JOIN party_party p ON p.id = e.party
		 WHERE p.name ILIKE $1 OR p.code ILIKE $1
		 ORDER BY e.id`, "%"+strings.TrimSpace(text)+"%")
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
