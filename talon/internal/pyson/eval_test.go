package pyson

import "testing"

func TestParseLiterals(t *testing.T) {
	value, err := Parse("[('active', '=', True), ('amount', '>', 100)]")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	domain, ok := value.([]interface{})
	if !ok || len(domain) != 2 {
		t.Fatalf("Parse() = %#v, want 2 clauses", value)
	}
	first := domain[0].([]interface{})
	if first[0] != "active" || first[1] != "=" || first[2] != true {
		t.Errorf("first clause = %#v", first)
	}
}

func TestParseEvalAndContext(t *testing.T) {
	value, err := Parse("[('party', '=', Eval('user', 0)), ('company', '=', Context('company'))]")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	domain := value.([]interface{})
	context := map[string]interface{}{"user": 7, "company": 3}
	record := map[string]interface{}{"party": 7, "company": 3}
	if !MatchDomain(domain, record, context) {
		t.Error("MatchDomain() = false, want true")
	}
	if MatchDomain(domain, map[string]interface{}{"party": 8, "company": 3}, context) {
		t.Error("MatchDomain() = true for wrong party")
	}
}

func TestEvalExpression(t *testing.T) {
	context := map[string]interface{}{"user": 5}
	if got := EvalExpression("user", context); got != 5 {
		t.Errorf("EvalExpression(user) = %v, want 5", got)
	}
	if got := EvalExpression("missing", context); got != "missing" {
		t.Errorf("EvalExpression(missing) = %v, want missing", got)
	}
	if got := EvalExpression("Eval('user', 0)", context); got != 5 {
		t.Errorf("EvalExpression(Eval) = %v, want 5", got)
	}
	if got := EvalExpression("42", context); got != 42 {
		t.Errorf("EvalExpression(42) = %v, want 42", got)
	}
}

func TestMatchDomainOperators(t *testing.T) {
	tests := []struct {
		name     string
		domain   []interface{}
		record   map[string]interface{}
		expected bool
	}{
		{
			name:     "equality",
			domain:   []interface{}{[]interface{}{"code", "=", "ACME"}},
			record:   map[string]interface{}{"code": "ACME"},
			expected: true,
		},
		{
			name:     "not equal",
			domain:   []interface{}{[]interface{}{"code", "!=", "ACME"}},
			record:   map[string]interface{}{"code": "OTHER"},
			expected: true,
		},
		{
			name: "or",
			domain: []interface{}{
				"OR",
				[]interface{}{[]interface{}{"code", "=", "A"}},
				[]interface{}{[]interface{}{"code", "=", "B"}},
			},
			record:   map[string]interface{}{"code": "B"},
			expected: true,
		},
		{
			name: "and",
			domain: []interface{}{
				"AND",
				[]interface{}{[]interface{}{"code", "=", "B"}},
				[]interface{}{[]interface{}{"active", "=", true}},
			},
			record:   map[string]interface{}{"code": "B", "active": true},
			expected: true,
		},
		{
			name: "and fails",
			domain: []interface{}{
				"AND",
				[]interface{}{[]interface{}{"code", "=", "B"}},
				[]interface{}{[]interface{}{"active", "=", true}},
			},
			record:   map[string]interface{}{"code": "B", "active": false},
			expected: false,
		},
		{
			name:     "numeric greater",
			domain:   []interface{}{[]interface{}{"amount", ">", 100}},
			record:   map[string]interface{}{"amount": 150},
			expected: true,
		},
		{
			name:     "in list",
			domain:   []interface{}{[]interface{}{"state", "in", []interface{}{"draft", "done"}}},
			record:   map[string]interface{}{"state": "done"},
			expected: true,
		},
		{
			name:     "like",
			domain:   []interface{}{[]interface{}{"name", "like", "Acme%"}},
			record:   map[string]interface{}{"name": "Acme Corp"},
			expected: true,
		},
		{
			name:     "ilike",
			domain:   []interface{}{[]interface{}{"name", "ilike", "%corp%"}},
			record:   map[string]interface{}{"name": "ACME CORP"},
			expected: true,
		},
		{
			name:     "implicit and",
			domain:   []interface{}{[]interface{}{"code", "=", "A"}, []interface{}{"active", "=", true}},
			record:   map[string]interface{}{"code": "A", "active": true},
			expected: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchDomain(tt.domain, tt.record, nil); got != tt.expected {
				t.Errorf("MatchDomain() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTruthy(t *testing.T) {
	if Truthy(nil) || Truthy(false) || Truthy(0) || Truthy("") {
		t.Error("Truthy() true for falsy values")
	}
	if !Truthy(1) || !Truthy("x") || !Truthy(true) {
		t.Error("Truthy() false for truthy values")
	}
}
