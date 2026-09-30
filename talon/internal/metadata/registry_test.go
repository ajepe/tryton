package metadata

import (
	"testing"
)

func TestRegisterModel(t *testing.T) {
	m := &Model{
		Name: "test.model",
		Fields: map[string]*Field{
			"name": {Name: "name", Type: FieldChar},
		},
	}

	err := RegisterModel("test.model", m)
	if err != nil {
		t.Fatalf("RegisterModel() error = %v", err)
	}

	got, ok := GetModel("test.model")
	if !ok {
		t.Fatal("GetModel() not found")
	}
	if got.Name != "test.model" {
		t.Errorf("GetModel().Name = %s, want test.model", got.Name)
	}
}

func TestRegisterModel_Duplicate(t *testing.T) {
	m := &Model{Name: "test.duplicate"}
	RegisterModel("test.duplicate", m)

	err := RegisterModel("test.duplicate", m)
	if err == nil {
		t.Error("RegisterModel() expected error for duplicate")
	}
}

func TestGetModel_NotFound(t *testing.T) {
	_, ok := GetModel("nonexistent.model")
	if ok {
		t.Error("GetModel() should return false for nonexistent model")
	}
}

func TestListModels(t *testing.T) {
	models := ListModels()
	if len(models) == 0 {
		t.Error("ListModels() returned empty")
	}
}

func TestModelToTable(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"party.party", "party_party"},
		{"account.move.line", "account_move_line"},
		{"ir.ui.view", "ir_ui_view"},
	}

	for _, tt := range tests {
		got := modelToTable(tt.input)
		if got != tt.want {
			t.Errorf("modelToTable(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestField_SQLType(t *testing.T) {
	tests := []struct {
		ftype FieldType
		want  string
	}{
		{FieldChar, "varchar"},
		{FieldText, "text"},
		{FieldInteger, "integer"},
		{FieldFloat, "double precision"},
		{FieldNumeric, "numeric"},
		{FieldBoolean, "boolean"},
		{FieldDate, "date"},
		{FieldDateTime, "timestamp with time zone"},
		{FieldTime, "time"},
		{FieldTimeDelta, "interval"},
		{FieldBinary, "bytea"},
		{FieldDict, "jsonb"},
		{FieldMany2One, "integer"},
	}

	for _, tt := range tests {
		f := &Field{Type: tt.ftype}
		got := f.SQLType()
		if got != tt.want {
			t.Errorf("SQLType(%s) = %q, want %q", tt.ftype, got, tt.want)
		}
	}
}

func TestField_IsRelation(t *testing.T) {
	tests := []struct {
		ftype FieldType
		want  bool
	}{
		{FieldMany2One, true},
		{FieldOne2One, true},
		{FieldOne2Many, true},
		{FieldMany2Many, true},
		{FieldReference, true},
		{FieldChar, false},
		{FieldText, false},
		{FieldInteger, false},
	}

	for _, tt := range tests {
		f := &Field{Type: tt.ftype}
		got := f.IsRelation()
		if got != tt.want {
			t.Errorf("IsRelation(%s) = %v, want %v", tt.ftype, got, tt.want)
		}
	}
}

func TestField_IsX2M(t *testing.T) {
	tests := []struct {
		ftype FieldType
		want  bool
	}{
		{FieldOne2Many, true},
		{FieldMany2Many, true},
		{FieldMany2One, false},
		{FieldChar, false},
	}

	for _, tt := range tests {
		f := &Field{Type: tt.ftype}
		got := f.IsX2M()
		if got != tt.want {
			t.Errorf("IsX2M(%s) = %v, want %v", tt.ftype, got, tt.want)
		}
	}
}

func TestField_Validate(t *testing.T) {
	f := &Field{Name: "test", Required: true}
	err := f.Validate(nil)
	if err == nil {
		t.Error("Validate() expected error for required field with nil value")
	}

	err = f.Validate("value")
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestField_Description(t *testing.T) {
	f := &Field{
		Name:     "test",
		Type:     FieldChar,
		Label:    "Test Field",
		Required: true,
	}

	desc := f.Description()
	if desc["name"] != "test" {
		t.Errorf("Description()[name] = %v, want test", desc["name"])
	}
	if desc["type"] != "char" {
		t.Errorf("Description()[type] = %v, want char", desc["type"])
	}
	if desc["string"] != "Test Field" {
		t.Errorf("Description()[string] = %v, want Test Field", desc["string"])
	}
	if desc["required"] != true {
		t.Errorf("Description()[required] = %v, want true", desc["required"])
	}
}

func TestNewField(t *testing.T) {
	f := NewField("test", FieldChar)
	if f.Name != "test" {
		t.Errorf("NewField().Name = %s, want test", f.Name)
	}
	if f.Type != FieldChar {
		t.Errorf("NewField().Type = %s, want char", f.Type)
	}
	if f.Loading != "eager" {
		t.Errorf("NewField().Loading = %s, want eager", f.Loading)
	}
}

func TestField_Setters(t *testing.T) {
	f := NewField("test", FieldChar)
	f.SetLabel("Label")
	f.SetHelp("Help")
	f.SetRequired(true)
	f.SetReadOnly(true)
	f.SetDomain("domain")
	f.SetLoading("lazy")

	if f.Label != "Label" {
		t.Errorf("Label = %s, want Label", f.Label)
	}
	if f.Help != "Help" {
		t.Errorf("Help = %s, want Help", f.Help)
	}
	if !f.Required {
		t.Error("Required = false, want true")
	}
	if !f.ReadOnly {
		t.Error("ReadOnly = false, want true")
	}
	if f.Domain != "domain" {
		t.Errorf("Domain = %s, want domain", f.Domain)
	}
	if f.Loading != "lazy" {
		t.Errorf("Loading = %s, want lazy", f.Loading)
	}
}
