package pyson

import (
	"fmt"
	"strings"
)

type Decoder struct {
	context map[string]interface{}
}

func NewDecoder(ctx map[string]interface{}) *Decoder {
	return &Decoder{context: ctx}
}

func (d *Decoder) Decode(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		class, ok := v["__class__"].(string)
		if !ok {
			return v
		}
		switch class {
		case "Eval":
			return d.decodeEval(v)
		case "Context":
			return d.decodeContext(v)
		case "ActiveID":
			return d.context["active_id"]
		case "ActiveIds":
			return d.context["active_ids"]
		case "CurrentCompany":
			return d.context["company"]
		case "CurrentLanguage":
			return d.context["language"]
		default:
			return v
		}
	default:
		return value
	}
}

func (d *Decoder) decodeEval(v map[string]interface{}) interface{} {
	expr, ok := v["value"].(string)
	if !ok {
		return nil
	}
	return d.eval(expr)
}

func (d *Decoder) decodeContext(v map[string]interface{}) interface{} {
	key, ok := v["key"].(string)
	if !ok {
		return nil
	}
	if val, ok := d.context[key]; ok {
		return val
	}
	return nil
}

func (d *Decoder) eval(expr string) interface{} {
	expr = strings.TrimSpace(expr)
	if val, ok := d.context[expr]; ok {
		return val
	}
	if strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'") {
		return expr[1 : len(expr)-1]
	}
	if strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"") {
		return expr[1 : len(expr)-1]
	}
	if n, err := fmt.Sscanf(expr, "%d", new(int)); err == nil && n == 1 {
		var i int
		fmt.Sscanf(expr, "%d", &i)
		return i
	}
	if n, err := fmt.Sscanf(expr, "%f", new(float64)); err == nil && n == 1 {
		var f float64
		fmt.Sscanf(expr, "%f", &f)
		return f
	}
	return expr
}

type Encoder struct{}

func NewEncoder() *Encoder {
	return &Encoder{}
}

func (e *Encoder) Encode(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return map[string]interface{}{
			"__class__": "Eval",
			"value":     v,
		}
	default:
		return value
	}
}

func (e *Encoder) EncodeDomain(domain []interface{}) interface{} {
	return map[string]interface{}{
		"__class__": "Domain",
		"value":     domain,
	}
}

func (e *Encoder) EncodeContext(key string) interface{} {
	return map[string]interface{}{
		"__class__": "Context",
		"key":       key,
	}
}

func (e *Encoder) EncodeEval(value interface{}) interface{} {
	return map[string]interface{}{
		"__class__": "Eval",
		"value":     value,
	}
}

func (e *Encoder) EncodeActiveID() interface{} {
	return map[string]interface{}{
		"__class__": "ActiveID",
	}
}

func (e *Encoder) EncodeActiveIds() interface{} {
	return map[string]interface{}{
		"__class__": "ActiveIds",
	}
}

func (e *Encoder) EncodeCurrentCompany() interface{} {
	return map[string]interface{}{
		"__class__": "CurrentCompany",
	}
}

func (e *Encoder) EncodeCurrentLanguage() interface{} {
	return map[string]interface{}{
		"__class__": "CurrentLanguage",
	}
}

func (e *Encoder) EncodeNot(value interface{}) interface{} {
	return map[string]interface{}{
		"__class__": "Not",
		"value":     value,
	}
}

func (e *Encoder) EncodeAnd(values []interface{}) interface{} {
	return map[string]interface{}{
		"__class__": "And",
		"value":     values,
	}
}

func (e *Encoder) EncodeOr(values []interface{}) interface{} {
	return map[string]interface{}{
		"__class__": "Or",
		"value":     values,
	}
}

func (e *Encoder) EncodeCompare(field, op string, value interface{}) interface{} {
	return []interface{}{field, op, value}
}

func (e *Encoder) EncodeIn(field string, values []interface{}) interface{} {
	return []interface{}{field, "in", values}
}

func (e *Encoder) EncodeNotIn(field string, values []interface{}) interface{} {
	return []interface{}{field, "not in", values}
}

func (e *Encoder) EncodeLike(field, pattern string) interface{} {
	return []interface{}{field, "like", pattern}
}

func (e *Encoder) EncodeILike(field, pattern string) interface{} {
	return []interface{}{field, "ilike", pattern}
}

func (e *Encoder) EncodeChildOf(field string, value interface{}) interface{} {
	return []interface{}{field, "child_of", value}
}

func (e *Encoder) EncodeParentOf(field string, value interface{}) interface{} {
	return []interface{}{field, "parent_of", value}
}
