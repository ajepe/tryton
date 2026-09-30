package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/ajepe/talon/internal/i18n"
)

func TestPOImportAndTranslate(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	entries, err := i18n.ParsePO(strings.NewReader(`
msgctxt "field:party.party,name:"
msgid "Name"
msgstr "Nom"

msgctxt "field:party.party,code:"
msgid "Code"
msgstr "Code FR"
`))
	if err != nil {
		t.Fatalf("ParsePO() error = %v", err)
	}

	loader := i18n.NewPOLoader(h.db, "party")
	imported, err := loader.ImportPO(ctx, "fr", entries)
	if err != nil {
		t.Fatalf("ImportPO() error = %v", err)
	}
	if imported != 2 {
		t.Errorf("ImportPO() imported = %d, want 2", imported)
	}

	manager := i18n.NewManager(h.db)
	if got := manager.Translate(ctx, "party.party,name", "fr", "Name"); got != "Nom" {
		t.Errorf("Translate() = %q, want Nom", got)
	}

	entries, _ = i18n.ParsePO(strings.NewReader(`
msgctxt "field:party.party,name:"
msgid "Name"
msgstr "Nom complet"
`))
	if _, err := loader.ImportPO(ctx, "fr", entries); err != nil {
		t.Fatalf("ImportPO(update) error = %v", err)
	}
	manager.Invalidate()
	if got := manager.Translate(ctx, "party.party,name", "fr", "Name"); got != "Nom complet" {
		t.Errorf("Translate() after update = %q, want Nom complet", got)
	}

	var count int
	if err := h.db.QueryRowContext(ctx,
		`SELECT count(*) FROM ir_translation WHERE lang = 'fr' AND module = 'party'`).Scan(&count); err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 2 {
		t.Errorf("translations = %d, want 2", count)
	}

	entries, _ = i18n.ParsePO(strings.NewReader(`
msgctxt "field:party.party,code:"
msgid "Code"
msgstr ""
`))
	if _, err := loader.ImportPO(ctx, "fr", entries); err != nil {
		t.Fatalf("ImportPO(delete) error = %v", err)
	}
	if err := h.db.QueryRowContext(ctx,
		`SELECT count(*) FROM ir_translation WHERE lang = 'fr' AND module = 'party'`).Scan(&count); err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 1 {
		t.Errorf("translations after delete = %d, want 1", count)
	}
}
