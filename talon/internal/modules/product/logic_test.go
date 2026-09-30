package product

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

func TestRecName(t *testing.T) {
	if got := RecName("ABC", "Widget"); got != "[ABC] Widget" {
		t.Errorf("RecName() = %q, want [ABC] Widget", got)
	}
	if got := RecName("", "Widget"); got != "Widget" {
		t.Errorf("RecName() = %q, want Widget", got)
	}
}

func TestUomConversion(t *testing.T) {
	if got := UomRateFromFactor(2); got != 0.5 {
		t.Errorf("UomRateFromFactor(2) = %v, want 0.5", got)
	}
	if got := UomRateFromFactor(0); got != 0 {
		t.Errorf("UomRateFromFactor(0) = %v, want 0", got)
	}
	if got := UomFactorFromRate(4); got != 0.25 {
		t.Errorf("UomFactorFromRate(4) = %v, want 0.25", got)
	}
	if got := UomFactorFromRate(0); got != 0 {
		t.Errorf("UomFactorFromRate(0) = %v, want 0", got)
	}
}

func TestOnChangeUom(t *testing.T) {
	got, err := OnChangeUom(nil, nil, map[string]interface{}{
		"factor": 2.0,
	}, []string{"factor"})
	if err != nil {
		t.Fatalf("OnChangeUom() error = %v", err)
	}
	if got["rate"] != 0.5 {
		t.Errorf("rate = %v, want 0.5", got["rate"])
	}

	got, err = OnChangeUom(nil, nil, map[string]interface{}{
		"rate": 4.0,
	}, []string{"rate"})
	if err != nil {
		t.Fatalf("OnChangeUom() error = %v", err)
	}
	if got["factor"] != 0.25 {
		t.Errorf("factor = %v, want 0.25", got["factor"])
	}
}

func TestValidateUom(t *testing.T) {
	if err := ValidateUom(nil, nil, testRecord(map[string]interface{}{
		"name":   "Unit",
		"rate":   2.0,
		"factor": 0.5,
	})); err != nil {
		t.Errorf("ValidateUom() error = %v", err)
	}
	if err := ValidateUom(nil, nil, testRecord(map[string]interface{}{
		"name":   "Unit",
		"rate":   2.0,
		"factor": 3.0,
	})); err == nil {
		t.Error("ValidateUom() expected error for incompatible rate/factor")
	}
	if err := ValidateUom(nil, nil, testRecord(map[string]interface{}{
		"name":   "Unit",
		"rate":   0.0,
		"factor": 0.0,
	})); err == nil {
		t.Error("ValidateUom() expected error for zero rate and factor")
	}
}

func TestValidateTemplate(t *testing.T) {
	if err := ValidateTemplate(nil, nil, testRecord(map[string]interface{}{
		"name": "Widget",
		"type": "goods",
	})); err != nil {
		t.Errorf("ValidateTemplate() error = %v", err)
	}
	if err := ValidateTemplate(nil, nil, testRecord(map[string]interface{}{
		"name": "Widget",
		"type": "bogus",
	})); err == nil {
		t.Error("ValidateTemplate() expected error for invalid type")
	}
	if err := ValidateTemplate(nil, nil, testRecord(map[string]interface{}{
		"name":       "Widget",
		"sale_price": -1.0,
	})); err == nil {
		t.Error("ValidateTemplate() expected error for negative sale price")
	}
}
