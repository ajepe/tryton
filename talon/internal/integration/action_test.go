package integration

import (
	"context"
	"testing"

	"github.com/ajepe/talon/internal/action"
)

func TestActionValueAndKeyword(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	var actionID int
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO ir_action (name, type, usage) VALUES ('Parties', 'ir.action.act_window', 'menu') RETURNING id`,
	).Scan(&actionID); err != nil {
		t.Fatalf("insert action error = %v", err)
	}
	if _, err := h.db.ExecContext(ctx,
		`INSERT INTO ir_action_act_window (action, res_model, view_mode, domain, "limit", "order")
		 VALUES ($1, 'party.party', 'tree,form', '[]', 50, 'name ASC')`, actionID); err != nil {
		t.Fatalf("insert act_window error = %v", err)
	}
	var viewID int
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO ir_ui_view (name, model, type, arch, priority)
		 VALUES ('party.party.tree', 'party.party', 'tree', '<tree/>', 16) RETURNING id`,
	).Scan(&viewID); err != nil {
		t.Fatalf("insert view error = %v", err)
	}
	if _, err := h.db.ExecContext(ctx,
		`INSERT INTO ir_action_keyword (action, keyword, model)
		 VALUES ($1, 'tree_open', 'party.party,1')`, actionID); err != nil {
		t.Fatalf("insert keyword error = %v", err)
	}

	mgr := action.NewManager(h.db)

	value, err := mgr.GetActionValue(ctx, actionID, nil)
	if err != nil {
		t.Fatalf("GetActionValue() error = %v", err)
	}
	if value["type"] != "ir.action.act_window" {
		t.Errorf("type = %v", value["type"])
	}
	if value["res_model"] != "party.party" {
		t.Errorf("res_model = %v", value["res_model"])
	}
	if value["name"] != "Parties" {
		t.Errorf("name = %v", value["name"])
	}
	views, ok := value["views"].([][]interface{})
	if !ok {
		t.Fatalf("views = %#v", value["views"])
	}
	if len(views) != 2 {
		t.Fatalf("views length = %d, want 2", len(views))
	}
	if views[0][0] != viewID || views[0][1] != "tree" {
		t.Errorf("view[0] = %v, want [%d tree]", views[0], viewID)
	}
	if views[1][0] != 0 || views[1][1] != "form" {
		t.Errorf("view[1] = %v, want [0 form]", views[1])
	}

	keywords, err := mgr.GetKeyword(ctx, "tree_open", "party.party", 1)
	if err != nil {
		t.Fatalf("GetKeyword() error = %v", err)
	}
	if len(keywords) != 1 || keywords[0]["name"] != "Parties" {
		t.Errorf("keywords = %v", keywords)
	}

	if _, err := mgr.GetActionValue(ctx, 999999, nil); err == nil {
		t.Error("GetActionValue() expected error for missing action")
	}
}
