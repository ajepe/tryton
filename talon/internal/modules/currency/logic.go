package currency

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ajepe/talon/internal/orm"
)

type RateError struct {
	Currency string
	Date     string
}

func (e *RateError) Error() string {
	return fmt.Sprintf("no rate for currency %s at date %s", e.Currency, e.Date)
}

func Round(amount, rounding float64) float64 {
	if rounding == 0 {
		return amount
	}
	return math.RoundToEven(amount/rounding) * rounding
}

func IsZero(amount, rounding float64) bool {
	if rounding == 0 {
		return amount == 0
	}
	return math.Abs(Round(amount, rounding)) < math.Abs(rounding)
}

func today(context map[string]interface{}) string {
	if context != nil {
		if date, ok := context["date"].(string); ok && date != "" {
			return date
		}
	}
	return time.Now().Format("2006-01-02")
}

func LatestRate(ctx context.Context, group *orm.Group, currencyID int, date string) (float64, string, error) {
	db := group.DB()
	if db == nil {
		return 0, "", fmt.Errorf("no database connection")
	}
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	var rate float64
	var currencyName string
	err := db.QueryRowContext(ctx,
		`SELECT r.rate, COALESCE(c.name, '')
		 FROM currency_currency_rate r
		 JOIN currency_currency c ON c.id = r.currency
		 WHERE r.currency = $1 AND r.date <= $2
		 ORDER BY r.date DESC, r.id DESC LIMIT 1`,
		currencyID, date).Scan(&rate, &currencyName)
	if err != nil {
		err = db.QueryRowContext(ctx,
			`SELECT COALESCE(name, '') FROM currency_currency WHERE id = $1`,
			currencyID).Scan(&currencyName)
		if err != nil {
			return 0, "", err
		}
		return 0, currencyName, nil
	}
	return rate, currencyName, nil
}

func Compute(ctx context.Context, group *orm.Group, fromID int, amount float64, toID int, rpcContext map[string]interface{}, round bool) (float64, error) {
	date := today(rpcContext)
	if fromID == toID {
		return amount, nil
	}

	fromRate, fromName, err := LatestRate(ctx, group, fromID, date)
	if err != nil {
		return 0, err
	}
	toRate, toName, err := LatestRate(ctx, group, toID, date)
	if err != nil {
		return 0, err
	}
	if fromRate == 0 {
		return 0, &RateError{Currency: fromName, Date: date}
	}
	if toRate == 0 {
		return 0, &RateError{Currency: toName, Date: date}
	}

	converted := amount * toRate / fromRate
	if round {
		rounding, err := roundingOf(ctx, group, toID)
		if err != nil {
			return 0, err
		}
		converted = Round(converted, rounding)
	}
	return converted, nil
}

func roundingOf(ctx context.Context, group *orm.Group, currencyID int) (float64, error) {
	db := group.DB()
	if db == nil {
		return 0, fmt.Errorf("no database connection")
	}
	var rounding float64
	err := db.QueryRowContext(ctx,
		`SELECT COALESCE(rounding, 0) FROM currency_currency WHERE id = $1`,
		currencyID).Scan(&rounding)
	return rounding, err
}

func ValidateCurrency(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if strings.TrimSpace(orm.AsString(valueAt(record, "name"))) == "" {
		return fmt.Errorf("name is required")
	}
	if code := strings.TrimSpace(orm.AsString(valueAt(record, "code"))); code == "" {
		return fmt.Errorf("code is required")
	} else if len([]rune(code)) > 3 {
		return fmt.Errorf("code must be at most 3 characters long")
	}
	if v, ok := record.Get("digits"); ok {
		if digits, ok := orm.AsInt(v); ok && digits < 0 {
			return fmt.Errorf("digits must not be negative")
		}
	}
	if v, ok := record.Get("rounding"); ok {
		if rounding, ok := orm.AsFloat(v); ok && rounding <= 0 {
			return fmt.Errorf("rounding must be greater than zero")
		}
	}
	return nil
}

func ValidateRate(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if v, ok := record.Get("rate"); ok {
		if rate, ok := orm.AsFloat(v); ok && rate <= 0 {
			return fmt.Errorf("rate must be greater than zero")
		}
	}
	currency, ok := record.Get("currency")
	if id, isInt := orm.AsInt(currency); !ok || !isInt || id == 0 {
		return fmt.Errorf("currency is required")
	}
	return nil
}

func SearchCurrency(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	pattern := "%" + text + "%"
	rows, err := db.QueryContext(ctx,
		`SELECT id FROM currency_currency
		 WHERE code ILIKE $1 OR numeric_code ILIKE $1 OR name ILIKE $1
		 ORDER BY code, id`, pattern)
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
