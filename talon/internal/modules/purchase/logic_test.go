package purchase

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

func TestLineAmount(t *testing.T) {
	if got := LineAmount(4, 2.5); got != 10 {
		t.Errorf("LineAmount(4, 2.5) = %v, want 10", got)
	}
}

func TestSumLineAmounts(t *testing.T) {
	total := SumLineAmounts([]map[string]interface{}{
		{"amount": 5.0},
		{"quantity": 3.0, "unit_price": 2.0},
	})
	if total != 11 {
		t.Errorf("SumLineAmounts() = %v, want 11", total)
	}
}

func TestOnChangeLine(t *testing.T) {
	got, err := OnChangeLine(nil, nil, map[string]interface{}{
		"quantity":   3.0,
		"unit_price": 4.0,
	}, []string{"unit_price"})
	if err != nil {
		t.Fatalf("OnChangeLine() error = %v", err)
	}
	if got["amount"] != 12.0 {
		t.Errorf("amount = %v, want 12", got["amount"])
	}
}

func TestNextStatePurchase(t *testing.T) {
	tests := []struct {
		action  string
		current string
		want    string
		wantErr bool
	}{
		{"quote", "draft", "quotation", false},
		{"confirm", "quotation", "confirmed", false},
		{"cancel", "quotation", "cancelled", false},
		{"draft", "confirmed", "draft", false},
		{"quote", "confirmed", "", true},
	}
	for _, tt := range tests {
		got, err := NextState(tt.action, tt.current)
		if (err != nil) != tt.wantErr {
			t.Errorf("NextState(%q, %q) error = %v, wantErr %v", tt.action, tt.current, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("NextState(%q, %q) = %q, want %q", tt.action, tt.current, got, tt.want)
		}
	}
}

func TestValidatePurchase(t *testing.T) {
	if err := ValidatePurchase(nil, nil, testRecord(map[string]interface{}{
		"supplier": 1,
		"state":    "quotation",
	})); err != nil {
		t.Errorf("ValidatePurchase() error = %v", err)
	}
	if err := ValidatePurchase(nil, nil, testRecord(map[string]interface{}{
		"state": "draft",
	})); err == nil {
		t.Error("ValidatePurchase() expected error for missing supplier")
	}
}
