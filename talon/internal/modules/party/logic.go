package party

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajepe/talon/internal/orm"
)

func OnChangeName(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if name, ok := values["name"].(string); ok {
		result["name"] = strings.TrimSpace(name)
	}
	return result, nil
}

func OnChangeCode(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if code, ok := values["code"].(string); ok {
		result["code"] = strings.ToUpper(strings.TrimSpace(code))
	}
	return result, nil
}

func OnChangeWithCategory(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	return result, nil
}

func ButtonDuplicate(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
	newIDs := []interface{}{}
	for _, r := range records {
		duplicate, err := group.Copy(r)
		if err != nil {
			return nil, err
		}
		duplicate.Set("name", r.RecName()+" (copy)")
		if err := duplicate.Save(); err != nil {
			return nil, err
		}
		newIDs = append(newIDs, duplicate.ID())
	}
	return newIDs, nil
}

func ButtonErase(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
	for _, r := range records {
		r.Set("active", false)
		if err := r.Save(); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func ButtonActivate(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
	for _, r := range records {
		r.Set("active", true)
		if err := r.Save(); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func ButtonDeactivate(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
	for _, r := range records {
		r.Set("active", false)
		if err := r.Save(); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func SearchByName(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}

	query := "SELECT id FROM party_party WHERE name ILIKE $1"
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

func SearchByCode(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}

	query := "SELECT id FROM party_party WHERE code ILIKE $1"
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

func GetRecName(ctx context.Context, group *orm.Group, id int) (string, error) {
	r := orm.NewExistingRecord(group.Model(), id)
	if err := r.Load([]string{"name", "code"}); err != nil {
		return "", err
	}
	return r.RecName(), nil
}

func ValidateParty(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if name, ok := record.Get("name"); ok {
		if s, ok := name.(string); ok && strings.TrimSpace(s) == "" {
			return fmt.Errorf("name is required")
		}
	}
	return nil
}

func OnChangeAddress(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if country, ok := values["country"].(float64); ok {
		result["country"] = int(country)
	}
	return result, nil
}

func OnChangeContactMechanism(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if value, ok := values["value"].(string); ok {
		result["value"] = strings.TrimSpace(value)
	}
	return result, nil
}
