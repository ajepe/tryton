package integration

import (
	"context"
	"strconv"
	"strings"
	"testing"
)

func TestRecordRulesFilterAndProtect(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	userID, err := h.sessions.CreateUser(ctx, "ruleuser", "rulepass", "Rule User", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	var modelID int
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO ir_model (name, model) VALUES ('Party', 'party.party') RETURNING id`,
	).Scan(&modelID); err != nil {
		t.Fatalf("insert ir_model error = %v", err)
	}
	var groupID int
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO res_group (name) VALUES ('Rule Testers') RETURNING id`,
	).Scan(&groupID); err != nil {
		t.Fatalf("insert res_group error = %v", err)
	}
	if _, err := h.db.ExecContext(ctx,
		`INSERT INTO "res_user-res_group" ("user", "group") VALUES ($1, $2)`,
		userID, groupID); err != nil {
		t.Fatalf("membership error = %v", err)
	}
	if _, err := h.db.ExecContext(ctx,
		`INSERT INTO ir_model_access (model, "group", perm_read, perm_write, perm_create, perm_delete)
		 VALUES ($1, $2, true, true, true, true)`,
		modelID, groupID); err != nil {
		t.Fatalf("access error = %v", err)
	}

	var ruleGroupID int
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO ir_rule_group (name, model, global_p, default_p, perm_read, perm_write, perm_create, perm_delete)
		 VALUES ('Own party only', 'party.party', false, false, true, true, true, true)
		 RETURNING id`).Scan(&ruleGroupID); err != nil {
		t.Fatalf("rule group error = %v", err)
	}
	if _, err := h.db.ExecContext(ctx,
		`INSERT INTO "ir_rule_group-res_group" (rule_group, "group") VALUES ($1, $2)`,
		ruleGroupID, groupID); err != nil {
		t.Fatalf("rule group membership error = %v", err)
	}
	if _, err := h.db.ExecContext(ctx,
		`INSERT INTO ir_rule (rule_group, domain) VALUES ($1, $2)`,
		ruleGroupID, `[('code', '=', 'RULE-A')]`); err != nil {
		t.Fatalf("rule error = %v", err)
	}

	var partyA, partyB int
	h.db.QueryRowContext(ctx,
		`INSERT INTO party_party (name, code) VALUES ('Allowed', 'RULE-A') RETURNING id`).Scan(&partyA)
	h.db.QueryRowContext(ctx,
		`INSERT INTO party_party (name, code) VALUES ('Hidden', 'RULE-B') RETURNING id`).Scan(&partyB)

	domain, err := h.security.RecordDomain(ctx, userID, "party.party", "read")
	if err != nil {
		t.Fatalf("RecordDomain() error = %v", err)
	}
	if len(domain) == 0 {
		t.Fatal("RecordDomain() is empty, want a restriction")
	}

	body := `{"id": 1, "method": "model.party.party.search", "params": [[], 0, 10, "code ASC", {}]}`
	response := authenticatedRPC(t, h, "ruleuser", "rulepass", body)
	if !strings.Contains(response, itoaInt(partyA)) || strings.Contains(response, itoaInt(partyB)) {
		t.Errorf("search response = %s, want only party %d", response, partyA)
	}

	readBody := `{"id": 2, "method": "model.party.party.read", "params": [[` + itoaInt(partyB) + `], ["name"], {}]}`
	response = authenticatedRPC(t, h, "ruleuser", "rulepass", readBody)
	if strings.Contains(response, "Hidden") {
		t.Errorf("read response = %s, want hidden record filtered", response)
	}

	writeBody := `{"id": 3, "method": "model.party.party.write", "params": [[` + itoaInt(partyB) + `], {"name": "Nope"}, {}]}`
	response = authenticatedRPC(t, h, "ruleuser", "rulepass", writeBody)
	if !strings.Contains(response, "access denied") {
		t.Errorf("write response = %s, want access denied", response)
	}

	writeAllowed := `{"id": 4, "method": "model.party.party.write", "params": [[` + itoaInt(partyA) + `], {"name": "Allowed Updated"}, {}]}`
	response = authenticatedRPC(t, h, "ruleuser", "rulepass", writeAllowed)
	if strings.Contains(response, "error") {
		t.Errorf("write allowed response = %s, want success", response)
	}
}

func TestRecordRulesAdminBypass(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	adminID, err := h.sessions.CreateUser(ctx, "ruleadmin", "rulepass", "Rule Admin", true)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	domain, err := h.security.RecordDomain(ctx, adminID, "party.party", "read")
	if err != nil {
		t.Fatalf("RecordDomain() error = %v", err)
	}
	if len(domain) != 0 {
		t.Errorf("RecordDomain(admin) = %v, want empty", domain)
	}
}

func itoaInt(value int) string {
	return strconv.Itoa(value)
}
