package country

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

func TestCountryFlag(t *testing.T) {
	if got := CountryFlag("US"); got != "🇺🇸" {
		t.Errorf("CountryFlag(US) = %q, want 🇺🇸", got)
	}
	if got := CountryFlag(""); got != "" {
		t.Errorf("CountryFlag(\"\") = %q, want empty", got)
	}
}

func TestSubdivisionFlag(t *testing.T) {
	want := "🏴" +
		string(rune(917621)) +
		string(rune(917619)) +
		string(rune(917603)) +
		string(rune(917601)) +
		"\U000e007f"
	if got := SubdivisionFlag("US-CA"); got != want {
		t.Errorf("SubdivisionFlag(US-CA) = %q, want %q", got, want)
	}
	if got := SubdivisionFlag(""); got != "" {
		t.Errorf("SubdivisionFlag(\"\") = %q, want empty", got)
	}
}

func TestPostalCodeRecName(t *testing.T) {
	tests := []struct {
		id         int
		postalCode string
		city       string
		want       string
	}{
		{1, "75001", "Paris", "Paris (75001)"},
		{2, "75001", "", "75001"},
		{3, "", "Paris", "Paris"},
		{4, "", "", "4"},
	}
	for _, tt := range tests {
		if got := PostalCodeRecName(tt.id, tt.postalCode, tt.city); got != tt.want {
			t.Errorf("PostalCodeRecName(%d, %q, %q) = %q, want %q",
				tt.id, tt.postalCode, tt.city, got, tt.want)
		}
	}
}

func TestOnChangeCountry(t *testing.T) {
	got, err := OnChangeCountry(nil, nil, map[string]interface{}{
		"code":     " us ",
		"iso_code": "usa",
	}, nil)
	if err != nil {
		t.Fatalf("OnChangeCountry() error = %v", err)
	}
	if got["code"] != " us " {
		t.Errorf("code = %v, want unchanged input", got["code"])
	}
	if _, ok := got["flag"]; ok {
		t.Error("on_change must not compute function fields")
	}
}

func TestPreprocessCountry(t *testing.T) {
	got, err := PreprocessCountry(nil, nil, map[string]interface{}{
		"code":     " us ",
		"iso_code": "usa",
		"name":     "United States",
	}, "create")
	if err != nil {
		t.Fatalf("PreprocessCountry() error = %v", err)
	}
	if got["code"] != "US" {
		t.Errorf("code = %v, want US", got["code"])
	}
	if got["iso_code"] != "USA" {
		t.Errorf("iso_code = %v, want USA", got["iso_code"])
	}
	if got["name"] != "United States" {
		t.Errorf("name = %v, want unchanged", got["name"])
	}
}

func TestValidateCountry(t *testing.T) {
	if err := ValidateCountry(nil, nil, testRecord(map[string]interface{}{
		"name": "United States",
		"code": "US",
	})); err != nil {
		t.Errorf("ValidateCountry() error = %v", err)
	}
	if err := ValidateCountry(nil, nil, testRecord(map[string]interface{}{
		"name": "United States",
		"code": "USA",
	})); err == nil {
		t.Error("ValidateCountry() expected error for 3-letter code")
	}
	if err := ValidateCountry(nil, nil, testRecord(map[string]interface{}{
		"name": "United States",
		"code": "U1",
	})); err == nil {
		t.Error("ValidateCountry() expected error for non-letter code")
	}
}

func TestValidateSubdivision(t *testing.T) {
	if err := ValidateSubdivision(nil, nil, testRecord(map[string]interface{}{
		"name":    "California",
		"country": 1,
	})); err != nil {
		t.Errorf("ValidateSubdivision() error = %v", err)
	}
	if err := ValidateSubdivision(nil, nil, testRecord(map[string]interface{}{
		"name": "California",
	})); err == nil {
		t.Error("ValidateSubdivision() expected error for missing country")
	}
}
