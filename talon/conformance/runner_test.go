package conformance

import (
	"context"
	"testing"
)

func TestRunner_Compare(t *testing.T) {
	r := NewRunner("http://localhost:8000", "http://localhost:8001", "test")

	tests := []struct {
		name string
		a    interface{}
		b    interface{}
		want bool
	}{
		{
			name: "equal strings",
			a:    "hello",
			b:    "hello",
			want: true,
		},
		{
			name: "different strings",
			a:    "hello",
			b:    "world",
			want: false,
		},
		{
			name: "equal maps",
			a:    map[string]interface{}{"a": 1, "b": 2},
			b:    map[string]interface{}{"a": 1, "b": 2},
			want: true,
		},
		{
			name: "different maps",
			a:    map[string]interface{}{"a": 1},
			b:    map[string]interface{}{"a": 2},
			want: false,
		},
		{
			name: "equal slices",
			a:    []interface{}{1, 2, 3},
			b:    []interface{}{1, 2, 3},
			want: true,
		},
		{
			name: "different slices",
			a:    []interface{}{1, 2, 3},
			b:    []interface{}{1, 2, 4},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.compare(tt.a, tt.b, nil, false)
			if got != tt.want {
				t.Errorf("compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareIgnoresFields(t *testing.T) {
	r := NewRunner("", "", "test")
	a := map[string]interface{}{"name": "Acme", "create_date": "2026-01-01", "id": 1}
	b := map[string]interface{}{"name": "Acme", "create_date": "2026-02-02", "id": 2}
	if !r.compare(a, b, []string{"create_date", "id"}, false) {
		t.Error("compare() = false, want true with ignored fields")
	}
	if r.compare(a, b, []string{"create_date"}, false) {
		t.Error("compare() = true, want false when id differs")
	}
}

func TestCompareNormalizesNumbers(t *testing.T) {
	r := NewRunner("", "", "test")
	if !r.compare(1.0, 1, nil, false) {
		t.Error("compare(1.0, 1) = false, want true")
	}
	if !r.compare(2.5, 2.5, nil, false) {
		t.Error("compare(2.5, 2.5) = false, want true")
	}
}

func TestCompareSortLists(t *testing.T) {
	r := NewRunner("", "", "test")
	a := []interface{}{
		map[string]interface{}{"name": "B"},
		map[string]interface{}{"name": "A"},
	}
	b := []interface{}{
		map[string]interface{}{"name": "A"},
		map[string]interface{}{"name": "B"},
	}
	if !r.compare(a, b, nil, true) {
		t.Error("compare() with SortLists = false, want true")
	}
	if r.compare(a, b, nil, false) {
		t.Error("compare() without SortLists = true, want false")
	}
}

func TestParseCasesArray(t *testing.T) {
	cases, err := parseCases([]byte(`[{"method": "a"}, {"method": "b"}]`))
	if err != nil {
		t.Fatalf("parseCases() error = %v", err)
	}
	if len(cases) != 2 {
		t.Errorf("parseCases() = %d cases, want 2", len(cases))
	}

	cases, err = parseCases([]byte(`{"cases": [{"method": "a"}]}`))
	if err != nil {
		t.Fatalf("parseCases() error = %v", err)
	}
	if len(cases) != 1 {
		t.Errorf("parseCases() = %d cases, want 1", len(cases))
	}

	cases, err = parseCases([]byte(`{"method": "single"}`))
	if err != nil {
		t.Fatalf("parseCases() error = %v", err)
	}
	if len(cases) != 1 || cases[0].Method != "single" {
		t.Errorf("parseCases() = %v, want single case", cases)
	}
}

func TestFailures(t *testing.T) {
	r := NewRunner("", "", "test")
	r.results = []Result{{Match: true}, {Match: false}, {Match: false}}
	if r.Failures() != 2 {
		t.Errorf("Failures() = %d, want 2", r.Failures())
	}
}

func TestRunner_Report(t *testing.T) {
	r := NewRunner("http://localhost:8000", "http://localhost:8001", "test")
	r.results = []Result{
		{Method: "test1", Match: true},
		{Method: "test2", Match: false, Error: "mismatch"},
	}

	report := r.Report()
	if report == "" {
		t.Error("Report() returned empty string")
	}
}

func TestLoadCases(t *testing.T) {
	cases, err := LoadCases("./cases")
	if err != nil {
		t.Fatalf("LoadCases() error = %v", err)
	}
	_ = cases
}

func TestRunner_Run(t *testing.T) {
	r := NewRunner("http://localhost:8000", "http://localhost:8001", "test")
	ctx := context.Background()

	err := r.Run(ctx, []TestCase{
		{Method: "common.server.version", Params: []interface{}{}},
	})

	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(r.results) != 1 {
		t.Errorf("Run() produced %d results, want 1", len(r.results))
	}
}
