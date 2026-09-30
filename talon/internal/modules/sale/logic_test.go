package sale

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
	if got := LineAmount(3, 4.5); got != 13.5 {
		t.Errorf("LineAmount(3, 4.5) = %v, want 13.5", got)
	}
	if got := LineAmount(1, 0.005); got != 0.01 {
		t.Errorf("LineAmount(1, 0.005) = %v, want 0.01", got)
	}
}

func TestSumLineAmounts(t *testing.T) {
	total := SumLineAmounts([]map[string]interface{}{
		{"amount": 10.0},
		{"quantity": 2.0, "unit_price": 5.0},
	})
	if total != 20 {
		t.Errorf("SumLineAmounts() = %v, want 20", total)
	}
}

func TestOnChangeLine(t *testing.T) {
	got, err := OnChangeLine(nil, nil, map[string]interface{}{
		"quantity":   2.0,
		"unit_price": 5.0,
	}, []string{"quantity"})
	if err != nil {
		t.Fatalf("OnChangeLine() error = %v", err)
	}
	if got["amount"] != 10.0 {
		t.Errorf("amount = %v, want 10", got["amount"])
	}
}

func TestNextStateSale(t *testing.T) {
	tests := []struct {
		action  string
		current string
		want    string
		wantErr bool
	}{
		{"quote", "draft", "quotation", false},
		{"confirm", "quotation", "confirmed", false},
		{"draft", "confirmed", "draft", false},
		{"draft", "cancelled", "draft", false},
		{"cancel", "quotation", "cancelled", false},
		{"cancel", "done", "", true},
		{"confirm", "draft", "", true},
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

func TestValidateSaleLine(t *testing.T) {
	if err := ValidateSaleLine(nil, nil, testRecord(map[string]interface{}{
		"quantity":   2.0,
		"unit_price": 5.0,
		"amount":     10.0,
	})); err != nil {
		t.Errorf("ValidateSaleLine() error = %v", err)
	}
	if err := ValidateSaleLine(nil, nil, testRecord(map[string]interface{}{
		"quantity":   2.0,
		"unit_price": 5.0,
		"amount":     7.0,
	})); err == nil {
		t.Error("ValidateSaleLine() expected error for mismatched amount")
	}
}

func TestValidateSale(t *testing.T) {
	if err := ValidateSale(nil, nil, testRecord(map[string]interface{}{
		"customer": 1,
		"state":    "quotation",
	})); err != nil {
		t.Errorf("ValidateSale() error = %v", err)
	}
	if err := ValidateSale(nil, nil, testRecord(map[string]interface{}{
		"state": "draft",
	})); err == nil {
		t.Error("ValidateSale() expected error for missing customer")
	}
}
