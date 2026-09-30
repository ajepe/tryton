package party

import (
	"testing"

	"github.com/ajepe/talon/internal/metadata"
	"github.com/ajepe/talon/internal/orm"
)

func testRecord(values map[string]interface{}) *orm.Record {
	r := orm.NewRecord(&metadata.Model{Name: "test.model"})
	r.ImportData(values)
	return r
}

func TestOnChangeName(t *testing.T) {
	got, err := OnChangeName(nil, nil, map[string]interface{}{
		"name": "  Acme Corp  ",
	}, []string{"name"})
	if err != nil {
		t.Fatalf("OnChangeName() error = %v", err)
	}
	if got["name"] != "Acme Corp" {
		t.Errorf("name = %v, want Acme Corp", got["name"])
	}
}

func TestOnChangeCode(t *testing.T) {
	got, err := OnChangeCode(nil, nil, map[string]interface{}{
		"code": " acme ",
	}, []string{"code"})
	if err != nil {
		t.Fatalf("OnChangeCode() error = %v", err)
	}
	if got["code"] != "ACME" {
		t.Errorf("code = %v, want ACME", got["code"])
	}
}

func TestValidateParty(t *testing.T) {
	if err := ValidateParty(nil, nil, testRecord(map[string]interface{}{
		"name": "Acme",
	})); err != nil {
		t.Errorf("ValidateParty() error = %v", err)
	}
	if err := ValidateParty(nil, nil, testRecord(map[string]interface{}{
		"name": "   ",
	})); err == nil {
		t.Error("ValidateParty() expected error for blank name")
	}
}

func TestOnChangeAddress(t *testing.T) {
	got, err := OnChangeAddress(nil, nil, map[string]interface{}{
		"country": 42.0,
	}, []string{"country"})
	if err != nil {
		t.Fatalf("OnChangeAddress() error = %v", err)
	}
	if got["country"] != 42 {
		t.Errorf("country = %v, want 42", got["country"])
	}
}

func TestOnChangeContactMechanism(t *testing.T) {
	got, err := OnChangeContactMechanism(nil, nil, map[string]interface{}{
		"value": "  user@example.com ",
	}, []string{"value"})
	if err != nil {
		t.Fatalf("OnChangeContactMechanism() error = %v", err)
	}
	if got["value"] != "user@example.com" {
		t.Errorf("value = %v, want user@example.com", got["value"])
	}
}
