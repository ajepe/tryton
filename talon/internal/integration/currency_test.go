package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/ajepe/talon/internal/testsupport"
)

func TestCurrencyComputeOverRPC(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	if _, err := h.sessions.CreateUser(ctx, "curradmin", "pass", "Currency Admin", true); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	var usd, eur int
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO currency_currency (name, code, rounding, digits, active)
		 VALUES ('US Dollar', 'USD', 0.01, 2, true) RETURNING id`).Scan(&usd); err != nil {
		t.Fatalf("insert USD error = %v", err)
	}
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO currency_currency (name, code, rounding, digits, active)
		 VALUES ('Euro', 'EUR', 0.01, 2, true) RETURNING id`).Scan(&eur); err != nil {
		t.Fatalf("insert EUR error = %v", err)
	}
	if _, err := h.db.ExecContext(ctx,
		`INSERT INTO currency_currency_rate (currency, date, rate)
		 VALUES ($1, '2026-01-01', 1), ($2, '2026-01-01', 1.1)`, usd, eur); err != nil {
		t.Fatalf("insert rates error = %v", err)
	}

	body := `{"id": 1, "method": "model.currency.currency.compute", "params": [` +
		itoaInt(usd) + `, 100, ` + itoaInt(eur) + `, {}]}`
	response := authenticatedRPC(t, h, "curradmin", "pass", body)
	if !strings.Contains(response, "110") {
		t.Errorf("compute response = %s, want 110", response)
	}

	same := `{"id": 2, "method": "model.currency.currency.compute", "params": [` +
		itoaInt(usd) + `, 42.5, ` + itoaInt(usd) + `, {}]}`
	response = authenticatedRPC(t, h, "curradmin", "pass", same)
	if !strings.Contains(response, "42.5") {
		t.Errorf("compute same-currency response = %s, want 42.5", response)
	}

	missing := `{"id": 3, "method": "model.currency.currency.compute", "params": [` +
		itoaInt(usd) + `, 10, ` + itoaInt(eur) + `, {"date": "2020-01-01"}]}`
	response = authenticatedRPC(t, h, "curradmin", "pass", missing)
	if !strings.Contains(response, "no rate") {
		t.Errorf("compute missing-rate response = %s, want no rate error", response)
	}
}

func TestSchemaHasCurrencyAndCompanyTables(t *testing.T) {
	db := testsupport.OpenDB(t)
	for _, table := range []string{
		"currency_currency",
		"currency_currency_rate",
		"company_company",
		"company_employee",
	} {
		if !testsupport.TableExists(t, db, table) {
			t.Errorf("table %s missing", table)
		}
	}
	if testsupport.TableExists(t, db, "country_currency") {
		t.Error("legacy country_currency table should not exist")
	}
}
