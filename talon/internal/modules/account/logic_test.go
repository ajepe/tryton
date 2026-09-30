package account

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

func TestMoveLineBalance(t *testing.T) {
	debit, credit := MoveLineBalance([]map[string]interface{}{
		{"debit": 100.0, "credit": 0.0},
		{"debit": 0.0, "credit": 100.0},
	})
	if debit != 100 || credit != 100 {
		t.Errorf("MoveLineBalance() = (%v, %v), want (100, 100)", debit, credit)
	}
	if !IsMoveBalanced(debit, credit) {
		t.Error("IsMoveBalanced() = false, want true")
	}
	if IsMoveBalanced(100, 99.99) {
		t.Error("IsMoveBalanced(100, 99.99) = true, want false")
	}
}

func TestValidateMoveLine(t *testing.T) {
	if err := ValidateMoveLine(nil, nil, testRecord(map[string]interface{}{
		"debit": 10.0,
	})); err != nil {
		t.Errorf("ValidateMoveLine() error = %v", err)
	}
	if err := ValidateMoveLine(nil, nil, testRecord(map[string]interface{}{
		"debit":  10.0,
		"credit": 10.0,
	})); err == nil {
		t.Error("ValidateMoveLine() expected error for debit and credit both set")
	}
	if err := ValidateMoveLine(nil, nil, testRecord(map[string]interface{}{
		"debit": -1.0,
	})); err == nil {
		t.Error("ValidateMoveLine() expected error for negative debit")
	}
}

func TestOnChangeMoveLine(t *testing.T) {
	got, err := OnChangeMoveLine(nil, nil, map[string]interface{}{
		"debit": 10.0,
	}, []string{"debit"})
	if err != nil {
		t.Fatalf("OnChangeMoveLine() error = %v", err)
	}
	if got["credit"] != 0.0 {
		t.Errorf("credit = %v, want 0", got["credit"])
	}

	got, err = OnChangeMoveLine(nil, nil, map[string]interface{}{
		"credit": 5.0,
	}, []string{"credit"})
	if err != nil {
		t.Fatalf("OnChangeMoveLine() error = %v", err)
	}
	if got["debit"] != 0.0 {
		t.Errorf("debit = %v, want 0", got["debit"])
	}
}

func TestValidateFiscalyear(t *testing.T) {
	if err := ValidateFiscalyear(nil, nil, testRecord(map[string]interface{}{
		"start_date": "2026-01-01",
		"end_date":   "2026-12-31",
		"state":      "open",
	})); err != nil {
		t.Errorf("ValidateFiscalyear() error = %v", err)
	}
	if err := ValidateFiscalyear(nil, nil, testRecord(map[string]interface{}{
		"start_date": "2026-12-31",
		"end_date":   "2026-01-01",
	})); err == nil {
		t.Error("ValidateFiscalyear() expected error for inverted dates")
	}
	if err := ValidateFiscalyear(nil, nil, testRecord(map[string]interface{}{
		"start_date": "2026-01-01",
		"end_date":   "2026-12-31",
		"state":      "bogus",
	})); err == nil {
		t.Error("ValidateFiscalyear() expected error for invalid state")
	}
}

func TestValidateJournal(t *testing.T) {
	if err := ValidateJournal(nil, nil, testRecord(map[string]interface{}{
		"type": "cash",
	})); err != nil {
		t.Errorf("ValidateJournal() error = %v", err)
	}
	if err := ValidateJournal(nil, nil, testRecord(map[string]interface{}{
		"type": "bogus",
	})); err == nil {
		t.Error("ValidateJournal() expected error for invalid type")
	}
}
