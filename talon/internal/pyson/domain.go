package pyson

import (
	"fmt"
	"strings"
)

type DomainParser struct {
	fields map[string]interface{}
}

func NewDomainParser(fields map[string]interface{}) *DomainParser {
	return &DomainParser{fields: fields}
}

func (p *DomainParser) Parse(domain string) ([]interface{}, error) {
	tokens := p.tokenize(domain)
	return p.parseExpression(tokens)
}

func (p *DomainParser) tokenize(domain string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)
	for _, c := range domain {
		if inQuote {
			if c == quoteChar {
				inQuote = false
			} else {
				current.WriteRune(c)
			}
			continue
		}
		switch c {
		case '\'', '"':
			inQuote = true
			quoteChar = c
		case ' ', '\t', '\n', '\r':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(c)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

func (p *DomainParser) parseExpression(tokens []string) ([]interface{}, error) {
	if len(tokens) == 0 {
		return []interface{}{}, nil
	}
	if len(tokens) == 1 {
		return []interface{}{tokens[0]}, nil
	}
	if len(tokens) >= 3 {
		field := tokens[0]
		op := tokens[1]
		value := strings.Join(tokens[2:], " ")
		return []interface{}{field, op, value}, nil
	}
	return nil, fmt.Errorf("invalid domain expression: %v", tokens)
}

func (p *DomainParser) String(domain []interface{}) string {
	var parts []string
	for _, d := range domain {
		if arr, ok := d.([]interface{}); ok && len(arr) == 3 {
			parts = append(parts, fmt.Sprintf("%s %s %s", arr[0], arr[1], arr[2]))
		} else {
			parts = append(parts, fmt.Sprintf("%v", d))
		}
	}
	return strings.Join(parts, " AND ")
}

func (p *DomainParser) Completion(text string) []string {
	var completions []string
	for name := range p.fields {
		if strings.HasPrefix(name, text) {
			completions = append(completions, name)
		}
	}
	return completions
}

type DomainInversion struct {
	fields map[string]interface{}
}

func NewDomainInversion(fields map[string]interface{}) *DomainInversion {
	return &DomainInversion{fields: fields}
}

func (d *DomainInversion) Simplify(domain []interface{}) []interface{} {
	return domain
}

func (d *DomainInversion) Canonicalize(domain []interface{}) []interface{} {
	return domain
}

func (d *DomainInversion) Localize(domain []interface{}, language string) []interface{} {
	return domain
}

func (d *DomainInversion) Evaluate(domain []interface{}, values map[string]interface{}) bool {
	for _, item := range domain {
		if arr, ok := item.([]interface{}); ok && len(arr) == 3 {
			field := arr[0].(string)
			op := arr[1].(string)
			value := arr[2]
			if !d.compare(values[field], op, value) {
				return false
			}
		}
	}
	return true
}

func (d *DomainInversion) compare(a interface{}, op string, b interface{}) bool {
	switch op {
	case "=":
		return a == b
	case "!=":
		return a != b
	case "<":
		return d.lessThan(a, b)
	case "<=":
		return d.lessThan(a, b) || a == b
	case ">":
		return d.lessThan(b, a)
	case ">=":
		return d.lessThan(b, a) || a == b
	case "in":
		if arr, ok := b.([]interface{}); ok {
			for _, v := range arr {
				if a == v {
					return true
				}
			}
		}
		return false
	case "not in":
		if arr, ok := b.([]interface{}); ok {
			for _, v := range arr {
				if a == v {
					return false
				}
			}
		}
		return true
	case "like":
		return d.like(a, b)
	case "ilike":
		return d.ilike(a, b)
	default:
		return false
	}
}

func (d *DomainInversion) lessThan(a, b interface{}) bool {
	switch av := a.(type) {
	case string:
		if bv, ok := b.(string); ok {
			return av < bv
		}
	case int:
		if bv, ok := b.(int); ok {
			return av < bv
		}
	case float64:
		if bv, ok := b.(float64); ok {
			return av < bv
		}
	}
	return false
}

func (d *DomainInversion) like(a, b interface{}) bool {
	as, ok1 := a.(string)
	bs, ok2 := b.(string)
	if !ok1 || !ok2 {
		return false
	}
	return strings.Contains(as, strings.ReplaceAll(bs, "%", ""))
}

func (d *DomainInversion) ilike(a, b interface{}) bool {
	as, ok1 := a.(string)
	bs, ok2 := b.(string)
	if !ok1 || !ok2 {
		return false
	}
	return strings.Contains(strings.ToLower(as), strings.ToLower(strings.ReplaceAll(bs, "%", "")))
}

func (d *DomainInversion) SolveUnique(domain []interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for _, d := range domain {
		if arr, ok := d.([]interface{}); ok && len(arr) == 3 {
			field := arr[0].(string)
			op := arr[1].(string)
			value := arr[2]
			if op == "=" {
				result[field] = value
			}
		}
	}
	return result
}

func (d *DomainInversion) Invert(domain []interface{}, field string) []interface{} {
	var result []interface{}
	for _, d := range domain {
		if arr, ok := d.([]interface{}); ok && len(arr) == 3 {
			if arr[0] == field {
				continue
			}
		}
		result = append(result, d)
	}
	return result
}
