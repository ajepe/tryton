package orm

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"
)

func AsString(v interface{}) string {
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	case json.Number:
		return s.String()
	default:
		return ""
	}
}

func AsFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, err == nil
	case []byte:
		f, err := strconv.ParseFloat(strings.TrimSpace(string(n)), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func AsInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(n))
		return i, err == nil
	default:
		return 0, false
	}
}

func AsBool(v interface{}) bool {
	switch b := v.(type) {
	case bool:
		return b
	case int:
		return b != 0
	case int64:
		return b != 0
	case float64:
		return b != 0
	case string:
		s := strings.TrimSpace(strings.ToLower(b))
		return s == "true" || s == "1" || s == "yes"
	default:
		return false
	}
}

func Round(value float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(value*pow) / pow
}

func EqualRound(a, b float64, decimals int) bool {
	return Round(a-b, decimals) == 0
}

func AsDate(v interface{}) (time.Time, bool) {
	switch d := v.(type) {
	case time.Time:
		return d, true
	case string:
		for _, layout := range []string{
			"2006-01-02",
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
		} {
			if t, err := time.Parse(layout, strings.TrimSpace(d)); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

func AsDateString(v interface{}) string {
	if t, ok := AsDate(v); ok {
		return t.Format("2006-01-02")
	}
	return ""
}
