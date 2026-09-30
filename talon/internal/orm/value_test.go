package orm

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAsFloat(t *testing.T) {
	tests := []struct {
		input interface{}
		want  float64
		ok    bool
	}{
		{1.5, 1.5, true},
		{2, 2, true},
		{int64(3), 3, true},
		{json.Number("4.25"), 4.25, true},
		{"5.5", 5.5, true},
		{"abc", 0, false},
		{nil, 0, false},
	}
	for _, tt := range tests {
		got, ok := AsFloat(tt.input)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("AsFloat(%v) = (%v, %v), want (%v, %v)", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestAsInt(t *testing.T) {
	tests := []struct {
		input interface{}
		want  int
		ok    bool
	}{
		{3, 3, true},
		{3.9, 3, true},
		{int64(4), 4, true},
		{json.Number("5"), 5, true},
		{"6", 6, true},
		{"abc", 0, false},
	}
	for _, tt := range tests {
		got, ok := AsInt(tt.input)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("AsInt(%v) = (%v, %v), want (%v, %v)", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestRound(t *testing.T) {
	if got := Round(1.005, 2); got != 1.0 && got != 1.01 {
		t.Errorf("Round(1.005, 2) = %v, want ~1.0", got)
	}
	if got := Round(1.23456, 2); got != 1.23 {
		t.Errorf("Round(1.23456, 2) = %v, want 1.23", got)
	}
	if !EqualRound(0.1+0.2, 0.3, 9) {
		t.Error("EqualRound(0.1+0.2, 0.3, 9) = false, want true")
	}
}

func TestAsDate(t *testing.T) {
	got, ok := AsDate("2026-09-29")
	if !ok {
		t.Fatal("AsDate(2026-09-29) not ok")
	}
	if got.Year() != 2026 || got.Month() != time.September || got.Day() != 29 {
		t.Errorf("AsDate() = %v, want 2026-09-29", got)
	}
	if _, ok := AsDate("not-a-date"); ok {
		t.Error("AsDate(not-a-date) ok = true, want false")
	}
	if got := AsDateString("2026-09-29"); got != "2026-09-29" {
		t.Errorf("AsDateString() = %q, want 2026-09-29", got)
	}
}
