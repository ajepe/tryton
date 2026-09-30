package integration

import (
	"context"
	"strings"
	"testing"
)

func TestCompanySearchAndEmployeeValidation(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	if _, err := h.sessions.CreateUser(ctx, "coadmin", "pass", "Company Admin", true); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	var partyID, currencyID int
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO party_party (name, code) VALUES ('Acme Holding', 'HOLD') RETURNING id`,
	).Scan(&partyID); err != nil {
		t.Fatalf("insert party error = %v", err)
	}
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO currency_currency (name, code, rounding, digits, active)
		 VALUES ('US Dollar', 'USD', 0.01, 2, true) RETURNING id`).Scan(&currencyID); err != nil {
		t.Fatalf("insert currency error = %v", err)
	}
	var companyID int
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO company_company (party, currency) VALUES ($1, $2) RETURNING id`,
		partyID, currencyID).Scan(&companyID); err != nil {
		t.Fatalf("insert company error = %v", err)
	}

	body := `{"id": 1, "method": "model.company.company.search", "params": [["name", "ilike", "Acme"], 0, 10, "id ASC", {}]}`
	response := authenticatedRPC(t, h, "coadmin", "pass", body)
	if !strings.Contains(response, itoaInt(companyID)) {
		t.Errorf("company search response = %s, want company %d", response, companyID)
	}

	var employeeParty int
	h.db.QueryRowContext(ctx,
		`INSERT INTO party_party (name, code) VALUES ('Jane Doe', 'JDOE') RETURNING id`).Scan(&employeeParty)

	createBody := `{"id": 2, "method": "model.company.employee.create", "params": [[{` +
		`"party": ` + itoaInt(employeeParty) + `, "company": ` + itoaInt(companyID) + `,` +
		`"start_date": "2026-01-01", "end_date": "2025-01-01"}], {}]}`
	response = authenticatedRPC(t, h, "coadmin", "pass", createBody)
	if !strings.Contains(response, "end_date must be after start_date") {
		t.Errorf("employee create response = %s, want validation error", response)
	}

	createBody = `{"id": 3, "method": "model.company.employee.create", "params": [[{` +
		`"party": ` + itoaInt(employeeParty) + `, "company": ` + itoaInt(companyID) + `,` +
		`"start_date": "2026-01-01"}], {}]}`
	response = authenticatedRPC(t, h, "coadmin", "pass", createBody)
	if strings.Contains(response, "error") {
		t.Errorf("employee create response = %s, want success", response)
	}
}
