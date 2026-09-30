package i18n

import (
	"testing"
	"time"
)

func TestStrftimeToGo(t *testing.T) {
	tests := []struct {
		format string
		want   string
	}{
		{"%Y-%m-%d", "2006-01-02"},
		{"%d/%m/%Y", "02/01/2006"},
		{"%m/%d/%Y %H:%M:%S", "01/02/2006 15:04:05"},
		{"%B %d, %Y", "January 02, 2006"},
		{"", "2006-01-02"},
		{"100%%", "100%"},
	}
	for _, tt := range tests {
		if got := StrftimeToGo(tt.format); got != tt.want {
			t.Errorf("StrftimeToGo(%q) = %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	lang := &Language{DateFormat: "%Y-%m-%d"}
	got := FormatDate(time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC), lang)
	if got != "2026-09-29" {
		t.Errorf("FormatDate() = %q, want 2026-09-29", got)
	}
}

func TestFormatNumber(t *testing.T) {
	lang := &Language{DecimalPoint: ".", ThousandsSep: ","}
	if got := FormatNumber(1234567.89, lang, 2); got != "1,234,567.89" {
		t.Errorf("FormatNumber() = %q, want 1,234,567.89", got)
	}
	if got := FormatNumber(-1000, lang, 0); got != "-1,000" {
		t.Errorf("FormatNumber() = %q, want -1,000", got)
	}
	eu := &Language{DecimalPoint: ",", ThousandsSep: "."}
	if got := FormatNumber(1234.5, eu, 1); got != "1.234,5" {
		t.Errorf("FormatNumber() = %q, want 1.234,5", got)
	}
}

func TestParentLanguage(t *testing.T) {
	if got := parentLanguage("pt_BR"); got != "pt" {
		t.Errorf("parentLanguage(pt_BR) = %q, want pt", got)
	}
	if got := parentLanguage("en@bold"); got != "en" {
		t.Errorf("parentLanguage(en@bold) = %q, want en", got)
	}
	if got := parentLanguage("en"); got != "" {
		t.Errorf("parentLanguage(en) = %q, want empty", got)
	}
}

func TestTranslateWithoutDB(t *testing.T) {
	mgr := NewManager(nil)
	if got := mgr.Translate(nil, "party.party,name", "fr", "Name"); got != "Name" {
		t.Errorf("Translate() = %q, want Name", got)
	}
	if got := mgr.Translate(nil, "party.party,name", "", "Name"); got != "Name" {
		t.Errorf("Translate() = %q, want Name", got)
	}
}
