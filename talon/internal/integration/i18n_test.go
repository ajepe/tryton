package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/ajepe/talon/internal/i18n"
)

func TestTranslationLookup(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	if _, err := h.db.ExecContext(ctx,
		`INSERT INTO ir_lang (code, name, direction, date_format, decimal_point, thousands_sep)
		 VALUES ('fr', 'Français', 'ltr', '%d/%m/%Y', ',', ' ')`); err != nil {
		t.Fatalf("insert ir_lang error = %v", err)
	}
	if _, err := h.db.ExecContext(ctx,
		`INSERT INTO ir_translation (name, lang, src, value, module, fuzzy)
		 VALUES ('party.party,name', 'fr', 'Name', 'Nom', 'party', false)`); err != nil {
		t.Fatalf("insert ir_translation error = %v", err)
	}

	mgr := i18n.NewManager(h.db)
	if got := mgr.Translate(ctx, "party.party,name", "fr", "Name"); got != "Nom" {
		t.Errorf("Translate(fr) = %q, want Nom", got)
	}
	if got := mgr.Translate(ctx, "party.party,name", "en", "Name"); got != "Name" {
		t.Errorf("Translate(en) = %q, want Name", got)
	}

	lang, err := mgr.Language(ctx, "fr")
	if err != nil {
		t.Fatalf("Language() error = %v", err)
	}
	if lang.DecimalPoint != "," {
		t.Errorf("decimal point = %q, want ,", lang.DecimalPoint)
	}

	languages, err := mgr.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(languages) != 1 || languages[0].Code != "fr" {
		t.Errorf("List() = %v, want [fr]", languages)
	}
}

func TestFieldsGetTranslationOverRPC(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	if _, err := h.sessions.CreateUser(ctx, "transadmin", "pass", "Trans", true); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := h.db.ExecContext(ctx,
		`INSERT INTO ir_lang (code, name, direction, date_format, decimal_point, thousands_sep)
		 VALUES ('fr', 'Français', 'ltr', '%d/%m/%Y', ',', ' ')`); err != nil {
		t.Fatalf("insert ir_lang error = %v", err)
	}
	if _, err := h.db.ExecContext(ctx,
		`INSERT INTO ir_translation (name, lang, src, value, module, fuzzy)
		 VALUES ('party.party,name', 'fr', 'Name', 'Nom', 'party', false)`); err != nil {
		t.Fatalf("insert ir_translation error = %v", err)
	}

	body := `{"id": 1, "method": "model.party.party.fields_get", "params": [[], {"language": "fr"}]}`
	response := authenticatedRPC(t, h, "transadmin", "pass", body)
	if !strings.Contains(response, `"Nom"`) {
		t.Errorf("fields_get response = %s, want Nom label", response)
	}
}
