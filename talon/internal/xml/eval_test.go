package xml

import (
	"reflect"
	"testing"
)

func TestEvalLiteral(t *testing.T) {
	tests := []struct {
		expr string
		want interface{}
	}{
		{"", nil},
		{"True", true},
		{"False", false},
		{"None", nil},
		{"42", 42},
		{"1.5", 1.5},
		{"'hello'", "hello"},
		{`"hello"`, "hello"},
		{"[]", []interface{}{}},
		{"[1, 2, 3]", []interface{}{1, 2, 3}},
		{"['a', 'b,c']", []interface{}{"a", "b,c"}},
		{"Decimal('1.23')", 1.23},
		{"{'a': 1}", map[string]interface{}{"a": 1}},
		{"'it\\'s'", "it's"},
	}
	for _, tt := range tests {
		got, err := EvalLiteral(tt.expr)
		if err != nil {
			t.Errorf("EvalLiteral(%q) error = %v", tt.expr, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("EvalLiteral(%q) = %#v, want %#v", tt.expr, got, tt.want)
		}
	}
}

func TestEvalLiteralErrors(t *testing.T) {
	if _, err := EvalLiteral("'unterminated"); err == nil {
		t.Error("EvalLiteral() expected error for unterminated string")
	}
	if _, err := EvalLiteral("[1, 2"); err == nil {
		t.Error("EvalLiteral() expected error for unbalanced brackets")
	}
}
