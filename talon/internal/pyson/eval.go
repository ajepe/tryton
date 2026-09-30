package pyson

import (
	"fmt"
	"strconv"
	"strings"
)

func EvalValue(value interface{}, context map[string]interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		class, ok := v["__class__"].(string)
		if !ok {
			return v
		}
		switch class {
		case "Eval":
			expr, _ := v["value"].(string)
			if result, ok := context[expr]; ok {
				return result
			}
			if args, ok := v["args"].([]interface{}); ok && len(args) > 1 {
				return EvalValue(args[1], context)
			}
			return EvalExpression(expr, context)
		case "Context":
			key, _ := v["key"].(string)
			return context[key]
		case "ActiveID":
			return context["active_id"]
		case "ActiveIds":
			return context["active_ids"]
		case "CurrentCompany":
			return context["company"]
		case "CurrentLanguage":
			return context["language"]
		case "CurrentUser":
			return context["user"]
		case "Not":
			return !Truthy(EvalValue(v["value"], context))
		case "And":
			for _, item := range asList(v["value"]) {
				if !Truthy(EvalValue(item, context)) {
					return false
				}
			}
			return true
		case "Or":
			for _, item := range asList(v["value"]) {
				if Truthy(EvalValue(item, context)) {
					return true
				}
			}
			return false
		case "If":
			if Truthy(EvalValue(v["condition"], context)) {
				return EvalValue(v["then"], context)
			}
			return EvalValue(v["else"], context)
		default:
			return v
		}
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = EvalValue(item, context)
		}
		return result
	default:
		return value
	}
}

func EvalExpression(expression string, context map[string]interface{}) interface{} {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return nil
	}
	if result, ok := context[expression]; ok {
		return result
	}
	value, err := Parse(expression)
	if err != nil {
		return expression
	}
	if text, ok := value.(string); ok {
		if result, exists := context[text]; exists {
			return result
		}
		return text
	}
	return EvalValue(value, context)
}

func Truthy(value interface{}) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		return true
	}
}

func MatchDomain(domain []interface{}, record map[string]interface{}, context map[string]interface{}) bool {
	return matchDomain(domain, record, context)
}

func isDomainOperator(value string) bool {
	switch strings.ToUpper(value) {
	case "AND", "OR", "NOT":
		return true
	}
	return false
}

func matchDomain(domain []interface{}, record map[string]interface{}, context map[string]interface{}) bool {
	if len(domain) == 0 {
		return true
	}
	if operator, ok := domain[0].(string); ok && isDomainOperator(operator) {
		switch strings.ToUpper(operator) {
		case "NOT":
			if len(domain) < 2 {
				return true
			}
			return !matchExpression(domain[1], record, context)
		case "AND":
			for _, item := range domain[1:] {
				if !matchExpression(item, record, context) {
					return false
				}
			}
			return true
		case "OR":
			for _, item := range domain[1:] {
				if matchExpression(item, record, context) {
					return true
				}
			}
			return false
		}
	}

	result := true
	pendingOr := false
	for _, item := range domain {
		if operator, ok := item.(string); ok && isDomainOperator(operator) {
			pendingOr = strings.ToUpper(operator) == "OR"
			continue
		}
		value := matchExpression(item, record, context)
		if pendingOr {
			result = result || value
			pendingOr = false
		} else {
			result = result && value
		}
	}
	return result
}

func matchExpression(item interface{}, record map[string]interface{}, context map[string]interface{}) bool {
	clause, ok := item.([]interface{})
	if !ok {
		return true
	}
	if len(clause) >= 3 {
		field, fieldOK := clause[0].(string)
		operator, operatorOK := clause[1].(string)
		if fieldOK && operatorOK && !isDomainOperator(field) {
			expected := EvalValue(clause[2], context)
			return compareValues(record[field], operator, expected)
		}
	}
	return matchDomain(clause, record, context)
}

func compareValues(actual interface{}, op string, expected interface{}) bool {
	switch op {
	case "=", "==":
		return equalValues(actual, expected)
	case "!=", "<>":
		return !equalValues(actual, expected)
	case "<":
		return lessValues(actual, expected)
	case "<=":
		return lessValues(actual, expected) || equalValues(actual, expected)
	case ">":
		return lessValues(expected, actual)
	case ">=":
		return lessValues(expected, actual) || equalValues(actual, expected)
	case "in":
		for _, item := range asList(expected) {
			if equalValues(actual, item) {
				return true
			}
		}
		return false
	case "not in":
		for _, item := range asList(expected) {
			if equalValues(actual, item) {
				return false
			}
		}
		return true
	case "like":
		return likeValues(actual, expected, false)
	case "not like":
		return !likeValues(actual, expected, false)
	case "ilike":
		return likeValues(actual, expected, true)
	case "not ilike":
		return !likeValues(actual, expected, true)
	case "is":
		return equalValues(actual, expected)
	case "is not":
		return !equalValues(actual, expected)
	default:
		return false
	}
}

func equalValues(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	af, aok := number(a)
	bf, bok := number(b)
	if aok && bok {
		return af == bf
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func lessValues(a, b interface{}) bool {
	af, aok := number(a)
	bf, bok := number(b)
	if aok && bok {
		return af < bf
	}
	as, aok2 := a.(string)
	bs, bok2 := b.(string)
	if aok2 && bok2 {
		return as < bs
	}
	return false
}

func number(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func likeValues(actual, expected interface{}, caseInsensitive bool) bool {
	as, ok1 := actual.(string)
	es, ok2 := expected.(string)
	if !ok1 || !ok2 {
		return false
	}
	if caseInsensitive {
		as = strings.ToLower(as)
		es = strings.ToLower(es)
	}
	return matchPattern(as, es)
}

func matchPattern(value, pattern string) bool {
	parts := strings.Split(pattern, "%")
	if len(parts) == 1 {
		return value == pattern
	}
	position := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		index := strings.Index(value[position:], part)
		if index < 0 {
			return false
		}
		if i == 0 && !strings.HasPrefix(pattern, "%") && index != 0 {
			return false
		}
		position += index + len(part)
	}
	if !strings.HasSuffix(pattern, "%") {
		return strings.HasSuffix(value, parts[len(parts)-1])
	}
	return true
}

func asList(value interface{}) []interface{} {
	switch v := value.(type) {
	case []interface{}:
		return v
	case nil:
		return nil
	default:
		return []interface{}{value}
	}
}

type parser struct {
	source string
	pos    int
}

func Parse(source string) (interface{}, error) {
	p := &parser{source: source}
	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	if p.pos != len(p.source) {
		return nil, fmt.Errorf("unexpected input at %d in %q", p.pos, source)
	}
	return value, nil
}

func (p *parser) skipSpace() {
	for p.pos < len(p.source) && strings.ContainsRune(" \t\n\r", rune(p.source[p.pos])) {
		p.pos++
	}
}

func (p *parser) peek() byte {
	if p.pos >= len(p.source) {
		return 0
	}
	return p.source[p.pos]
}

func (p *parser) parseValue() (interface{}, error) {
	p.skipSpace()
	if p.pos >= len(p.source) {
		return nil, fmt.Errorf("unexpected end of input")
	}
	c := p.peek()
	switch {
	case c == '[':
		return p.parseSequence('[', ']')
	case c == '(':
		return p.parseSequence('(', ')')
	case c == '{':
		return p.parseDict()
	case c == '\'' || c == '"':
		return p.parseString(c)
	}
	if isDigit(c) || c == '-' || c == '+' {
		return p.parseNumber()
	}
	identifier := p.parseIdentifier()
	switch identifier {
	case "True":
		return true, nil
	case "False":
		return false, nil
	case "None":
		return nil, nil
	case "":
		return nil, fmt.Errorf("invalid value at %d", p.pos)
	}
	p.skipSpace()
	if p.peek() == '(' {
		return p.parseCall(identifier)
	}
	return identifier, nil
}

func (p *parser) parseSequence(open, close byte) ([]interface{}, error) {
	p.pos++
	values := []interface{}{}
	for {
		p.skipSpace()
		if p.pos >= len(p.source) {
			return nil, fmt.Errorf("unterminated sequence")
		}
		if p.peek() == close {
			p.pos++
			return values, nil
		}
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		values = append(values, value)
		p.skipSpace()
		if p.peek() == ',' {
			p.pos++
		}
	}
}

func (p *parser) parseDict() (map[string]interface{}, error) {
	p.pos++
	values := map[string]interface{}{}
	for {
		p.skipSpace()
		if p.pos >= len(p.source) {
			return nil, fmt.Errorf("unterminated dict")
		}
		if p.peek() == '}' {
			p.pos++
			return values, nil
		}
		key, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if p.peek() != ':' {
			return nil, fmt.Errorf("expected ':' in dict")
		}
		p.pos++
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		values[fmt.Sprintf("%v", key)] = value
		p.skipSpace()
		if p.peek() == ',' {
			p.pos++
		}
	}
}

func (p *parser) parseString(quote byte) (string, error) {
	p.pos++
	var b strings.Builder
	for p.pos < len(p.source) {
		c := p.source[p.pos]
		if c == '\\' && p.pos+1 < len(p.source) {
			p.pos++
			b.WriteByte(p.source[p.pos])
			p.pos++
			continue
		}
		if c == quote {
			p.pos++
			return b.String(), nil
		}
		b.WriteByte(c)
		p.pos++
	}
	return "", fmt.Errorf("unterminated string")
}

func (p *parser) parseNumber() (interface{}, error) {
	start := p.pos
	for p.pos < len(p.source) && (isDigit(p.source[p.pos]) || strings.ContainsRune(".eE+-", rune(p.source[p.pos]))) {
		p.pos++
	}
	token := p.source[start:p.pos]
	if strings.ContainsAny(token, ".eE") {
		value, err := strconv.ParseFloat(token, 64)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
	value, err := strconv.ParseInt(token, 10, 64)
	if err != nil {
		return nil, err
	}
	return int(value), nil
}

func (p *parser) parseIdentifier() string {
	start := p.pos
	for p.pos < len(p.source) {
		c := p.source[p.pos]
		if isDigit(c) || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '.' {
			p.pos++
			continue
		}
		break
	}
	return p.source[start:p.pos]
}

func (p *parser) parseCall(name string) (interface{}, error) {
	p.pos++
	args := []interface{}{}
	for {
		p.skipSpace()
		if p.pos >= len(p.source) {
			return nil, fmt.Errorf("unterminated call %s", name)
		}
		if p.peek() == ')' {
			p.pos++
			break
		}
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		args = append(args, value)
		p.skipSpace()
		if p.peek() == ',' {
			p.pos++
		}
	}
	switch name {
	case "Eval":
		return map[string]interface{}{"__class__": "Eval", "value": args[0], "args": args}, nil
	case "Context":
		key, _ := args[0].(string)
		return map[string]interface{}{"__class__": "Context", "key": key}, nil
	case "ActiveID":
		return map[string]interface{}{"__class__": "ActiveID"}, nil
	case "ActiveIds":
		return map[string]interface{}{"__class__": "ActiveIds"}, nil
	case "CurrentCompany":
		return map[string]interface{}{"__class__": "CurrentCompany"}, nil
	case "CurrentLanguage":
		return map[string]interface{}{"__class__": "CurrentLanguage"}, nil
	case "CurrentUser":
		return map[string]interface{}{"__class__": "CurrentUser"}, nil
	case "Decimal":
		if len(args) > 0 {
			if f, ok := number(args[0]); ok {
				return f, nil
			}
		}
		return 0.0, nil
	default:
		return map[string]interface{}{"__class__": name, "args": args}, nil
	}
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
