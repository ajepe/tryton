package ir

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

func TestSequencePreview(t *testing.T) {
	tests := []struct {
		prefix     string
		numberNext int
		padding    int
		suffix     string
		want       string
	}{
		{"INV/", 42, 5, "/2026", "INV/00042/2026"},
		{"", 1, 0, "", "1"},
		{"A", 7, 3, "B", "A007B"},
	}
	for _, tt := range tests {
		if got := SequencePreview(tt.prefix, tt.numberNext, tt.padding, tt.suffix); got != tt.want {
			t.Errorf("SequencePreview(%q, %d, %d, %q) = %q, want %q",
				tt.prefix, tt.numberNext, tt.padding, tt.suffix, got, tt.want)
		}
	}
}

func TestValidateSequence(t *testing.T) {
	if err := ValidateSequence(nil, nil, testRecord(map[string]interface{}{
		"number_increment": 0,
	})); err == nil {
		t.Error("ValidateSequence() expected error for zero increment")
	}

	if err := ValidateSequence(nil, nil, testRecord(map[string]interface{}{
		"number_increment": 1,
		"number_next":      1,
		"padding":          0,
		"prefix":           "INV/${date_Y}/",
		"suffix":           "$$",
	})); err != nil {
		t.Errorf("ValidateSequence() error = %v", err)
	}

	if err := ValidateSequence(nil, nil, testRecord(map[string]interface{}{
		"prefix": "INV/${bogus}",
	})); err == nil {
		t.Error("ValidateSequence() expected error for unknown placeholder")
	}

	if err := ValidateSequence(nil, nil, testRecord(map[string]interface{}{
		"prefix": "INV/$",
	})); err == nil {
		t.Error("ValidateSequence() expected error for dangling $")
	}
}

func TestNextModuleState(t *testing.T) {
	tests := []struct {
		action  string
		current string
		want    string
		wantErr bool
	}{
		{"activate", "not activated", "to activate", false},
		{"upgrade", "activated", "to upgrade", false},
		{"deactivate", "activated", "to remove", false},
		{"cancel", "to activate", "not activated", false},
		{"cancel", "to remove", "not activated", false},
		{"cancel", "to upgrade", "activated", false},
		{"activate", "activated", "", true},
		{"bogus", "activated", "", true},
	}
	for _, tt := range tests {
		got, err := NextModuleState(tt.action, tt.current)
		if (err != nil) != tt.wantErr {
			t.Errorf("NextModuleState(%q, %q) error = %v, wantErr %v", tt.action, tt.current, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("NextModuleState(%q, %q) = %q, want %q", tt.action, tt.current, got, tt.want)
		}
	}
}

func TestOnChangeAttachment(t *testing.T) {
	got, err := OnChangeAttachment(nil, nil, map[string]interface{}{
		"description": "first line\nsecond line",
	}, nil)
	if err != nil {
		t.Fatalf("OnChangeAttachment() error = %v", err)
	}
	if got["summary"] != "first line" {
		t.Errorf("summary = %v, want first line", got["summary"])
	}
}

func TestValidateLang(t *testing.T) {
	if err := ValidateLang(nil, nil, testRecord(map[string]interface{}{
		"decimal_point": ".",
		"thousands_sep": ",",
		"direction":     "ltr",
	})); err != nil {
		t.Errorf("ValidateLang() error = %v", err)
	}
	if err := ValidateLang(nil, nil, testRecord(map[string]interface{}{
		"decimal_point": ".",
		"thousands_sep": ".",
	})); err == nil {
		t.Error("ValidateLang() expected error for equal separators")
	}
}

func TestValidateCron(t *testing.T) {
	if err := ValidateCron(nil, nil, testRecord(map[string]interface{}{
		"model":           "party.party",
		"function":        "do_something",
		"interval_number": 1,
		"interval_type":   "days",
	})); err != nil {
		t.Errorf("ValidateCron() error = %v", err)
	}
	if err := ValidateCron(nil, nil, testRecord(map[string]interface{}{
		"model": "", "function": "",
	})); err == nil {
		t.Error("ValidateCron() expected error for empty model and function")
	}
}
