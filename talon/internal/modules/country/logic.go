package country

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/ajepe/talon/internal/orm"
)

func CountryFlag(code string) string {
	if code == "" {
		return ""
	}
	flag := make([]rune, 0, len(code))
	for _, c := range code {
		flag = append(flag, rune(127397+int(c)))
	}
	return string(flag)
}

func SubdivisionFlag(code string) string {
	if code == "" {
		return ""
	}
	code = strings.ReplaceAll(strings.ToLower(code), "-", "")
	var b strings.Builder
	b.WriteString("🏴")
	for _, c := range code {
		b.WriteRune(rune(917504 + int(c)))
	}
	b.WriteRune('\U000e007f')
	return b.String()
}

func PostalCodeRecName(id int, postalCode, city string) string {
	switch {
	case city != "" && postalCode != "":
		return fmt.Sprintf("%s (%s)", city, postalCode)
	case postalCode != "":
		return postalCode
	case city != "":
		return city
	default:
		return fmt.Sprintf("%d", id)
	}
}

func OnChangeCountry(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if value, ok := values["code"]; ok {
		result["code"] = value
	}
	return result, nil
}

func OnChangeSubdivision(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if value, ok := values["code"]; ok {
		result["code"] = value
	}
	return result, nil
}

func PreprocessCountry(ctx context.Context, group *orm.Group, values map[string]interface{}, mode string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	for key, value := range values {
		result[key] = value
	}
	for _, name := range []string{"code", "iso_code"} {
		if value, ok := result[name]; ok {
			result[name] = strings.ToUpper(strings.TrimSpace(orm.AsString(value)))
		}
	}
	return result, nil
}

func PreprocessSubdivision(ctx context.Context, group *orm.Group, values map[string]interface{}, mode string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	for key, value := range values {
		result[key] = value
	}
	if value, ok := result["code"]; ok {
		result["code"] = strings.ToUpper(strings.TrimSpace(orm.AsString(value)))
	}
	return result, nil
}

func ValidateCountry(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if strings.TrimSpace(orm.AsString(valueAt(record, "name"))) == "" {
		return fmt.Errorf("name is required")
	}
	if code := strings.TrimSpace(orm.AsString(valueAt(record, "code"))); code != "" && len([]rune(code)) != 2 {
		return fmt.Errorf("code must be 2 characters long")
	}
	if code := strings.TrimSpace(orm.AsString(valueAt(record, "iso_code"))); code != "" && len([]rune(code)) != 3 {
		return fmt.Errorf("iso_code must be 3 characters long")
	}
	for _, c := range orm.AsString(valueAt(record, "code")) {
		if !unicode.IsLetter(c) {
			return fmt.Errorf("code must only contain letters")
		}
	}
	return nil
}

func ValidateSubdivision(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if strings.TrimSpace(orm.AsString(valueAt(record, "name"))) == "" {
		return fmt.Errorf("name is required")
	}
	country, ok := record.Get("country")
	id, isInt := orm.AsInt(country)
	if !ok || !isInt || id == 0 {
		return fmt.Errorf("country is required")
	}
	return nil
}

func ValidatePostalCode(ctx context.Context, group *orm.Group, record *orm.Record) error {
	country, ok := record.Get("country")
	id, isInt := orm.AsInt(country)
	if !ok || !isInt || id == 0 {
		return fmt.Errorf("country is required")
	}
	if strings.TrimSpace(orm.AsString(valueAt(record, "postal_code"))) == "" {
		return fmt.Errorf("postal_code is required")
	}
	return nil
}

func SearchCountry(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	pattern := "%" + text + "%"
	rows, err := db.QueryContext(ctx,
		`SELECT id FROM country_country
		 WHERE name ILIKE $1 OR code ILIKE $1 OR iso_code ILIKE $1
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

func SearchPostalCode(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	pattern := "%" + text + "%"
	rows, err := db.QueryContext(ctx,
		`SELECT id FROM country_postal_code
		 WHERE postal_code ILIKE $1 OR city ILIKE $1
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

func valueAt(r *orm.Record, name string) interface{} {
	v, _ := r.Get(name)
	return v
}
