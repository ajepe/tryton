package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/testsupport"
	xmlpkg "github.com/ajepe/talon/internal/xml"
)

func writeModuleXML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.xml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func TestXMLLoaderCreatesAndUpdates(t *testing.T) {
	db := testsupport.OpenDB(t)
	loader := xmlpkg.NewLoader(db, orm.DefaultPool(), "testmod")
	ctx := context.Background()

	path := writeModuleXML(t, `<?xml version="1.0"?>
<tryton>
  <data>
    <record model="ir.lang" id="lang_en">
      <field name="code">en</field>
      <field name="name">English</field>
      <field name="direction">ltr</field>
      <field name="date_format">%m/%d/%Y</field>
      <field name="decimal_point">.</field>
      <field name="thousands_sep">,</field>
    </record>
    <record model="ir.lang" id="lang_fr">
      <field name="code">fr</field>
      <field name="name">Français</field>
      <field name="direction">ltr</field>
    </record>
    <record model="party.party" id="party_acme">
      <field name="name">Acme</field>
      <field name="code">ACME</field>
      <field name="lang" ref="lang_en"/>
      <field name="active" eval="True"/>
    </record>
    <record model="ir.ui.view" id="party_view_form">
      <field name="name">party.party.form</field>
      <field name="model">party.party</field>
      <field name="type">form</field>
      <field name="priority" eval="16"/>
      <field name="arch" type="xml">
        <form>
          <field name="name"/>
        </form>
      </field>
    </record>
  </data>
</tryton>`)

	if err := loader.LoadFile(path); err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	var langID int
	if err := db.QueryRowContext(ctx,
		`SELECT db_id FROM ir_model_data WHERE module = 'testmod' AND fs_id = 'lang_en'`,
	).Scan(&langID); err != nil {
		t.Fatalf("lang_en model data error = %v", err)
	}
	if langID == 0 {
		t.Fatal("lang_en db_id = 0")
	}

	var partyLang int
	var partyName string
	if err := db.QueryRowContext(ctx,
		`SELECT lang, name FROM party_party WHERE code = 'ACME'`,
	).Scan(&partyLang, &partyName); err != nil {
		t.Fatalf("party query error = %v", err)
	}
	if partyName != "Acme" {
		t.Errorf("party name = %q, want Acme", partyName)
	}
	if partyLang != langID {
		t.Errorf("party lang = %d, want %d", partyLang, langID)
	}

	var priority int
	var arch string
	if err := db.QueryRowContext(ctx,
		`SELECT priority, arch FROM ir_ui_view WHERE model = 'party.party'`,
	).Scan(&priority, &arch); err != nil {
		t.Fatalf("view query error = %v", err)
	}
	if priority != 16 {
		t.Errorf("view priority = %d, want 16", priority)
	}
	if arch == "" {
		t.Error("view arch is empty")
	}

	updatePath := writeModuleXML(t, `<?xml version="1.0"?>
<tryton>
  <data>
    <record model="party.party" id="party_acme">
      <field name="name">Acme Corporation</field>
    </record>
  </data>
</tryton>`)
	if err := loader.LoadFile(updatePath); err != nil {
		t.Fatalf("LoadFile(update) error = %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM party_party WHERE code = 'ACME'`).Scan(&count); err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 1 {
		t.Errorf("party count = %d, want 1", count)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT name FROM party_party WHERE code = 'ACME'`).Scan(&partyName); err != nil {
		t.Fatalf("party query error = %v", err)
	}
	if partyName != "Acme Corporation" {
		t.Errorf("party name = %q, want Acme Corporation", partyName)
	}
}

func TestXMLLoaderUnknownModel(t *testing.T) {
	db := testsupport.OpenDB(t)
	loader := xmlpkg.NewLoader(db, orm.DefaultPool(), "testmod")
	path := writeModuleXML(t, `<tryton><data><record model="unknown.model" id="x"/></data></tryton>`)
	if err := loader.LoadFile(path); err == nil {
		t.Error("LoadFile() expected error for unknown model")
	}
}
