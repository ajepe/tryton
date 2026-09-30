package rpc

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"time"
)

type Codec struct{}

func NewCodec() *Codec {
	return &Codec{}
}

func (c *Codec) Encode(value interface{}) interface{} {
	switch v := value.(type) {
	case time.Time:
		return map[string]interface{}{
			"__class__": "datetime",
			"year":      v.Year(),
			"month":     int(v.Month()),
			"day":       v.Day(),
			"hour":      v.Hour(),
			"minute":    v.Minute(),
			"second":    v.Second(),
		}

	case time.Duration:
		return map[string]interface{}{
			"__class__": "timedelta",
			"seconds":   v.Seconds(),
		}
	case *big.Rat:
		return map[string]interface{}{
			"__class__": "Decimal",
			"decimal":   v.FloatString(10),
		}
	case []byte:
		return map[string]interface{}{
			"__class__": "bytes",
			"base64":    base64.StdEncoding.EncodeToString(v),
		}
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for k, val := range v {
			result[k] = c.Encode(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = c.Encode(val)
		}
		return result
	default:
		return value
	}
}

func (c *Codec) Decode(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		class, ok := v["__class__"].(string)
		if !ok {
			result := make(map[string]interface{}, len(v))
			for k, val := range v {
				result[k] = c.Decode(val)
			}
			return result
		}
		switch class {
		case "datetime":
			return time.Date(
				int(v["year"].(float64)),
				time.Month(int(v["month"].(float64))),
				int(v["day"].(float64)),
				int(v["hour"].(float64)),
				int(v["minute"].(float64)),
				int(v["second"].(float64)),
				0, time.UTC,
			)
		case "date":
			return time.Date(
				int(v["year"].(float64)),
				time.Month(int(v["month"].(float64))),
				int(v["day"].(float64)),
				0, 0, 0, 0, time.UTC,
			)
		case "timedelta":
			return time.Duration(v["seconds"].(float64)) * time.Second
		case "Decimal":
			r := new(big.Rat)
			r.SetString(v["decimal"].(string))
			return r
		case "bytes":
			data, err := base64.StdEncoding.DecodeString(v["base64"].(string))
			if err != nil {
				return v["base64"]
			}
			return data
		default:
			result := make(map[string]interface{}, len(v))
			for k, val := range v {
				result[k] = c.Decode(val)
			}
			return result
		}
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = c.Decode(val)
		}
		return result
	default:
		return value
	}
}

func (c *Codec) PrepareObject(value interface{}) interface{} {
	return c.Encode(value)
}

func (c *Codec) ConvertJSONObject(value interface{}) interface{} {
	return c.Decode(value)
}

func (c *Codec) EncodeParams(params []interface{}) []interface{} {
	result := make([]interface{}, len(params))
	for i, p := range params {
		result[i] = c.Encode(p)
	}
	return result
}

func (c *Codec) DecodeResult(result interface{}) interface{} {
	return c.Decode(result)
}

func (c *Codec) DecodeError(err interface{}) error {
	if err == nil {
		return nil
	}
	arr, ok := err.([]interface{})
	if !ok || len(arr) < 2 {
		return fmt.Errorf("unknown error: %v", err)
	}
	errType, _ := arr[0].(string)
	errData, _ := arr[1].([]interface{})
	switch errType {
	case "UserError":
		msg := ""
		if len(errData) > 0 {
			msg, _ = errData[0].(string)
		}
		return &UserError{Message: msg}
	case "UserWarning":
		name := ""
		msg := ""
		if len(errData) > 0 {
			name, _ = errData[0].(string)
		}
		if len(errData) > 1 {
			msg, _ = errData[1].(string)
		}
		return &UserWarning{Name: name, Message: msg}
	case "ConcurrencyException":
		msg := ""
		if len(errData) > 0 {
			msg, _ = errData[0].(string)
		}
		return &ConcurrencyException{Message: msg}
	case "LoginException":
		name := ""
		message := ""
		errType := ""
		if len(errData) > 0 {
			name, _ = errData[0].(string)
		}
		if len(errData) > 1 {
			message, _ = errData[1].(string)
		}
		if len(errData) > 2 {
			errType, _ = errData[2].(string)
		}
		return &LoginException{Name: name, Message: message, Type: errType}
	default:
		return fmt.Errorf("%s: %v", errType, errData)
	}
}

type UserError struct {
	Message     string
	Description string
	Domain      string
}

func (e *UserError) Error() string { return e.Message }

type UserWarning struct {
	Name        string
	Message     string
	Description string
}

func (e *UserWarning) Error() string { return e.Message }

type ConcurrencyException struct {
	Message string
}

func (e *ConcurrencyException) Error() string { return e.Message }

type LoginException struct {
	Name    string
	Message string
	Type    string
}

func (e *LoginException) Error() string { return e.Message }
