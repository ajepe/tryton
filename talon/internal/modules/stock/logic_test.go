package stock

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

func TestNextStateMove(t *testing.T) {
	tests := []struct {
		action  string
		current string
		want    string
		wantErr bool
	}{
		{"draft", "assigned", "draft", false},
		{"assign", "draft", "assigned", false},
		{"do", "draft", "done", false},
		{"do", "assigned", "done", false},
		{"do", "done", "", true},
		{"cancel", "done", "cancelled", false},
		{"cancel", "cancelled", "", true},
		{"bogus", "draft", "", true},
	}
	for _, tt := range tests {
		got, err := NextState(tt.action, moveTransitions, tt.current)
		if (err != nil) != tt.wantErr {
			t.Errorf("NextState(%q, move, %q) error = %v, wantErr %v", tt.action, tt.current, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("NextState(%q, move, %q) = %q, want %q", tt.action, tt.current, got, tt.want)
		}
	}
}

func TestNextStateShipmentOut(t *testing.T) {
	got, err := NextState("wait", shipmentOutTransitions, "draft")
	if err != nil {
		t.Fatalf("NextState(wait) error = %v", err)
	}
	if got != "waiting" {
		t.Errorf("NextState(wait, draft) = %q, want waiting", got)
	}
	if _, err := NextState("pack", shipmentOutTransitions, "waiting"); err == nil {
		t.Error("NextState(pack, waiting) expected error")
	}
}

func TestNextStateShipmentIn(t *testing.T) {
	got, err := NextState("receive", shipmentInTransitions, "draft")
	if err != nil {
		t.Fatalf("NextState(receive) error = %v", err)
	}
	if got != "received" {
		t.Errorf("NextState(receive, draft) = %q, want received", got)
	}
}

func TestValidateMove(t *testing.T) {
	if err := ValidateMove(nil, nil, testRecord(map[string]interface{}{
		"quantity":      1.0,
		"from_location": 1,
		"to_location":   2,
	})); err != nil {
		t.Errorf("ValidateMove() error = %v", err)
	}
	if err := ValidateMove(nil, nil, testRecord(map[string]interface{}{
		"quantity": -1.0,
	})); err == nil {
		t.Error("ValidateMove() expected error for negative quantity")
	}
	if err := ValidateMove(nil, nil, testRecord(map[string]interface{}{
		"from_location": 1,
		"to_location":   1,
	})); err == nil {
		t.Error("ValidateMove() expected error for equal locations")
	}
}

func TestValidateShipmentOut(t *testing.T) {
	if err := ValidateShipmentOut(nil, nil, testRecord(map[string]interface{}{
		"customer": 1,
		"state":    "draft",
	})); err != nil {
		t.Errorf("ValidateShipmentOut() error = %v", err)
	}
	if err := ValidateShipmentOut(nil, nil, testRecord(map[string]interface{}{
		"state": "draft",
	})); err == nil {
		t.Error("ValidateShipmentOut() expected error for missing customer")
	}
}

func TestValidateInventoryLine(t *testing.T) {
	if err := ValidateInventoryLine(nil, nil, testRecord(map[string]interface{}{
		"product":  1,
		"quantity": 5.0,
	})); err != nil {
		t.Errorf("ValidateInventoryLine() error = %v", err)
	}
	if err := ValidateInventoryLine(nil, nil, testRecord(map[string]interface{}{
		"product":  1,
		"quantity": -5.0,
	})); err == nil {
		t.Error("ValidateInventoryLine() expected error for negative quantity")
	}
}
