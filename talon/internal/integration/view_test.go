package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/testsupport"
	xmlpkg "github.com/ajepe/talon/internal/xml"
)

func TestFieldsViewGetWithInheritance(t *testing.T) {
	db := testsupport.OpenDB(t)
	loader := xmlpkg.NewLoader(db, orm.DefaultPool(), "testmod")
	ctx := context.Background()

	path := writeModuleXML(t, `<?xml version="1.0"?>
<tryton>
  <data>
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
    <record model="ir.ui.view" id="party_view_form_inherit">
      <field name="name">party.party.form.inherit</field>
      <field name="model">party.party</field>
      <field name="type">form</field>
      <field name="priority" eval="20"/>
      <field name="inherit" ref="party_view_form"/>
      <field name="arch" type="xml">
        <data>
          <xpath expr="//field[@name='name']" position="after">
            <field name="code"/>
          </xpath>
        </data>
      </field>
    </record>
  </data>
</tryton>`)

	if err := loader.LoadFile(path); err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	var viewID int
	if err := db.QueryRowContext(ctx,
		`SELECT db_id FROM ir_model_data WHERE module = 'testmod' AND fs_id = 'party_view_form'`,
	).Scan(&viewID); err != nil {
		t.Fatalf("view id error = %v", err)
	}

	h := newHarnessWithDB(t, db)
	if _, err := h.sessions.CreateUser(ctx, "viewadmin", "pass", "View Admin", true); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	body := `{"id": 1, "method": "model.party.party.fields_view_get", "params": [` +
		itoaInt(viewID) + `, "form", {}]}`
	response := authenticatedRPC(t, h, "viewadmin", "pass", body)

	if !strings.Contains(response, `name=\"code\"`) {
		t.Fatalf("response = %s, want inherited field code", response)
	}
	indexName := strings.Index(response, `name=\"name\"`)
	indexCode := strings.Index(response, `name=\"code\"`)
	if indexName >= 0 && indexCode >= 0 && indexCode < indexName {
		t.Errorf("code should be after name: %s", response)
	}
}
