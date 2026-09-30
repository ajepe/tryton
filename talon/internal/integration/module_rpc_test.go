package integration

import (
	"context"
	"strconv"
	"strings"
	"testing"
)

func TestModuleOnChangeOverRPC(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	if _, err := h.sessions.CreateUser(ctx, "moduleadmin", "pass", "Module Admin", true); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	body := `{"id": 1, "method": "model.product.uom.on_change", "params": [{"factor": 2}, ["factor"], {}]}`
	response := authenticatedRPC(t, h, "moduleadmin", "pass", body)
	if !strings.Contains(response, `"rate":0.5`) {
		t.Errorf("on_change response = %s, want rate 0.5", response)
	}

	body = `{"id": 2, "method": "model.country.country.on_change", "params": [{"code": "us"}, ["code"], {}]}`
	response = authenticatedRPC(t, h, "moduleadmin", "pass", body)
	if !strings.Contains(response, `"code":"us"`) {
		t.Errorf("on_change response = %s, want unchanged code", response)
	}

	createBody := `{"id": 3, "method": "model.country.country.create", "params": [[{"name": "Uppercase Test", "code": "zz"}], {}]}`
	response = authenticatedRPC(t, h, "moduleadmin", "pass", createBody)
	if strings.Contains(response, "error") {
		t.Fatalf("country create response = %s", response)
	}
	var code string
	if err := h.db.QueryRowContext(ctx,
		`SELECT code FROM country_country WHERE name = 'Uppercase Test'`).Scan(&code); err != nil {
		t.Fatalf("country query error = %v", err)
	}
	if code != "ZZ" {
		t.Errorf("stored code = %q, want ZZ (preprocessed)", code)
	}
}

func TestModuleButtonOverRPC(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	if _, err := h.sessions.CreateUser(ctx, "buttonadmin", "pass", "Button Admin", true); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	var partyID int
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO party_party (name, code) VALUES ('Original', 'ORIG') RETURNING id`,
	).Scan(&partyID); err != nil {
		t.Fatalf("insert party error = %v", err)
	}

	body := `{"id": 1, "method": "model.party.party.duplicate", "params": [[` +
		strconv.Itoa(partyID) + `], {}]}`
	response := authenticatedRPC(t, h, "buttonadmin", "pass", body)
	if !strings.Contains(response, `"result":[`) {
		t.Fatalf("duplicate response = %s", response)
	}

	var count int
	if err := h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM party_party WHERE name = 'Original (copy)'`).Scan(&count); err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 1 {
		t.Errorf("duplicated parties = %d, want 1", count)
	}
}
