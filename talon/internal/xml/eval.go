package xml

import (
	"fmt"
	"strconv"
	"strings"
)

func EvalLiteral(expr string) (interface{}, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, nil
	}
	switch expr {
	case "True":
		return true, nil
	case "False":
		return false, nil
	case "None":
		return nil, nil
	}

	if strings.HasPrefix(expr, "[") && !strings.HasSuffix(expr, "]") {
		return nil, fmt.Errorf("unbalanced brackets in %q", expr)
	}
	if strings.HasPrefix(expr, "{") && !strings.HasSuffix(expr, "}") {
		return nil, fmt.Errorf("unbalanced brackets in %q", expr)
	}
	if (strings.HasPrefix(expr, "'") && !strings.HasSuffix(expr, "'")) ||
		(strings.HasPrefix(expr, `"`) && !strings.HasSuffix(expr, `"`)) {
		return nil, fmt.Errorf("unterminated string in %q", expr)
	}

	if strings.HasPrefix(expr, "[") && strings.HasSuffix(expr, "]") {
		items, err := splitTopLevel(expr[1:len(expr)-1], ',')
		if err != nil {
			return nil, err
		}
		values := make([]interface{}, 0, len(items))
		for _, item := range items {
			if strings.TrimSpace(item) == "" {
				continue
			}
			value, err := EvalLiteral(item)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, nil
	}

	if strings.HasPrefix(expr, "{") && strings.HasSuffix(expr, "}") {
		items, err := splitTopLevel(expr[1:len(expr)-1], ',')
		if err != nil {
			return nil, err
		}
		values := map[string]interface{}{}
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			parts := strings.SplitN(item, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid dict entry %q", item)
			}
			key, err := EvalLiteral(parts[0])
			if err != nil {
				return nil, err
			}
			value, err := EvalLiteral(parts[1])
			if err != nil {
				return nil, err
			}
			values[fmt.Sprintf("%v", key)] = value
		}
		return values, nil
	}

	if strings.HasPrefix(expr, "Decimal(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSpace(expr[len("Decimal(") : len(expr)-1])
		value, err := EvalLiteral(inner)
		if err != nil {
			return nil, err
		}
		switch v := value.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid Decimal %q", expr)
			}
			return f, nil
		default:
			return nil, fmt.Errorf("invalid Decimal %q", expr)
		}
	}

	if (strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'") && len(expr) >= 2) ||
		(strings.HasPrefix(expr, `"`) && strings.HasSuffix(expr, `"`) && len(expr) >= 2) {
		quote := expr[:1]
		inner := expr[1 : len(expr)-1]
		inner = strings.ReplaceAll(inner, `\`+quote, quote)
		inner = strings.ReplaceAll(inner, `\\`, `\`)
		return inner, nil
	}

	if value, err := strconv.ParseInt(expr, 10, 64); err == nil {
		return int(value), nil
	}
	if value, err := strconv.ParseFloat(expr, 64); err == nil {
		return value, nil
	}
	return expr, nil
}

func splitTopLevel(value string, separator rune) ([]string, error) {
	parts := []string{}
	depth := 0
	quote := rune(0)
	current := strings.Builder{}
	for _, r := range value {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
			current.WriteRune(r)
		case r == '\'' || r == '"':
			quote = r
			current.WriteRune(r)
		case r == '[' || r == '{' || r == '(':
			depth++
			current.WriteRune(r)
		case r == ']' || r == '}' || r == ')':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unbalanced brackets in %q", value)
			}
			current.WriteRune(r)
		case r == separator && depth == 0:
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated string in %q", value)
	}
	if depth != 0 {
		return nil, fmt.Errorf("unbalanced brackets in %q", value)
	}
	parts = append(parts, current.String())
	return parts, nil
}
