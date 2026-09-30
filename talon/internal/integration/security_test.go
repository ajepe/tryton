package integration

import (
	"context"
	"strings"
	"testing"
)

func TestSecurityModelAccess(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	adminID, err := h.sessions.CreateUser(ctx, "root", "rootpass", "Root", true)
	if err != nil {
		t.Fatalf("CreateUser(admin) error = %v", err)
	}
	userID, err := h.sessions.CreateUser(ctx, "jdoe", "userpass", "John Doe", false)
	if err != nil {
		t.Fatalf("CreateUser(user) error = %v", err)
	}

	var modelID int
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO ir_model (name, model) VALUES ('Party', 'party.party') RETURNING id`,
	).Scan(&modelID); err != nil {
		t.Fatalf("insert ir_model error = %v", err)
	}

	var groupID int
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO res_group (name) VALUES ('Party Readers') RETURNING id`,
	).Scan(&groupID); err != nil {
		t.Fatalf("insert res_group error = %v", err)
	}
	if _, err := h.db.ExecContext(ctx,
		`INSERT INTO "res_user-res_group" ("user", "group") VALUES ($1, $2)`,
		userID, groupID); err != nil {
		t.Fatalf("insert membership error = %v", err)
	}
	if _, err := h.db.ExecContext(ctx,
		`INSERT INTO ir_model_access (model, "group", perm_read, perm_write, perm_create, perm_delete)
		 VALUES ($1, $2, true, false, false, false)`,
		modelID, groupID); err != nil {
		t.Fatalf("insert ir_model_access error = %v", err)
	}

	if err := h.security.Check(ctx, userID, "party.party", "read"); err != nil {
		t.Errorf("Check(read) error = %v", err)
	}
	if err := h.security.Check(ctx, userID, "party.party", "write"); err == nil {
		t.Error("Check(write) expected error")
	}
	if err := h.security.Check(ctx, userID, "party.party", "create"); err == nil {
		t.Error("Check(create) expected error")
	}
	if err := h.security.Check(ctx, userID, "party.party", "delete"); err == nil {
		t.Error("Check(delete) expected error")
	}
	if err := h.security.Check(ctx, adminID, "party.party", "delete"); err != nil {
		t.Errorf("Check(admin delete) error = %v", err)
	}
	if err := h.security.Check(ctx, adminID, "account.move", "create"); err != nil {
		t.Errorf("Check(admin unknown model) error = %v", err)
	}

	if err := h.security.Check(ctx, userID, "sale.sale", "read"); err == nil {
		t.Error("Check(unrelated model) expected error")
	}
}

func TestSecurityFieldAccess(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	userID, err := h.sessions.CreateUser(ctx, "fielduser", "pass", "Field User", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	if err := h.security.CheckField(ctx, userID, "party.party", "name", "write"); err != nil {
		t.Errorf("CheckField() without rules error = %v", err)
	}

	var modelID int
	h.db.QueryRowContext(ctx,
		`INSERT INTO ir_model (name, model) VALUES ('Party', 'party.party') RETURNING id`).Scan(&modelID)
	var fieldID int
	h.db.QueryRowContext(ctx,
		`INSERT INTO ir_model_field (name, model, field_description, ttype)
		 VALUES ('code', $1, 'Code', 'char') RETURNING id`, modelID).Scan(&fieldID)

	var groupID int
	h.db.QueryRowContext(ctx,
		`INSERT INTO res_group (name) VALUES ('Field Readers') RETURNING id`).Scan(&groupID)
	h.db.ExecContext(ctx,
		`INSERT INTO "res_user-res_group" ("user", "group") VALUES ($1, $2)`, userID, groupID)
	h.db.ExecContext(ctx,
		`INSERT INTO ir_model_field_access (field, "group", perm_read, perm_write, perm_create, perm_delete)
		 VALUES ($1, $2, true, false, false, false)`, fieldID, groupID)

	if err := h.security.CheckField(ctx, userID, "party.party", "code", "read"); err != nil {
		t.Errorf("CheckField(read) error = %v", err)
	}
	if err := h.security.CheckField(ctx, userID, "party.party", "code", "write"); err == nil {
		t.Error("CheckField(write) expected error")
	}
}

func TestSecurityDeniesRPCWithoutAccess(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	if _, err := h.sessions.CreateUser(ctx, "limited", "limitedpass", "Limited", false); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	response := authenticatedRPC(t, h, "limited", "limitedpass",
		`{"id": 1, "method": "model.party.party.search", "params": [[], 0, 10, "", {}]}`)
	if !strings.Contains(response, "access denied") {
		t.Errorf("response = %s, want access denied", response)
	}
}
