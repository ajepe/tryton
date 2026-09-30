package rpc

import (
	"math/big"
	"testing"
	"time"
)

func TestCodec_EncodeDatetime(t *testing.T) {
	c := NewCodec()
	dt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	result := c.Encode(dt)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Encode() did not return map")
	}

	if m["__class__"] != "datetime" {
		t.Errorf("__class__ = %v, want datetime", m["__class__"])
	}
	if m["year"] != 2024 {
		t.Errorf("year = %v, want 2024", m["year"])
	}
	if m["month"] != 1 {
		t.Errorf("month = %v, want 1", m["month"])
	}
	if m["day"] != 15 {
		t.Errorf("day = %v, want 15", m["day"])
	}
}

func TestCodec_EncodeTimedelta(t *testing.T) {
	c := NewCodec()
	td := 3600 * time.Second

	result := c.Encode(td)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Encode() did not return map")
	}

	if m["__class__"] != "timedelta" {
		t.Errorf("__class__ = %v, want timedelta", m["__class__"])
	}
	if m["seconds"] != 3600.0 {
		t.Errorf("seconds = %v, want 3600", m["seconds"])
	}
}

func TestCodec_EncodeDecimal(t *testing.T) {
	c := NewCodec()
	d := big.NewRat(12345, 100)

	result := c.Encode(d)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Encode() did not return map")
	}

	if m["__class__"] != "Decimal" {
		t.Errorf("__class__ = %v, want Decimal", m["__class__"])
	}
}

func TestCodec_EncodeBytes(t *testing.T) {
	c := NewCodec()
	b := []byte("hello")

	result := c.Encode(b)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Encode() did not return map")
	}

	if m["__class__"] != "bytes" {
		t.Errorf("__class__ = %v, want bytes", m["__class__"])
	}
}

func TestCodec_DecodeDatetime(t *testing.T) {
	c := NewCodec()
	input := map[string]interface{}{
		"__class__": "datetime",
		"year":      2024.0,
		"month":     1.0,
		"day":       15.0,
		"hour":      10.0,
		"minute":    30.0,
		"second":    0.0,
	}

	result := c.Decode(input)
	dt, ok := result.(time.Time)
	if !ok {
		t.Fatal("Decode() did not return time.Time")
	}

	if dt.Year() != 2024 {
		t.Errorf("year = %d, want 2024", dt.Year())
	}
}

func TestCodec_DecodeTimedelta(t *testing.T) {
	c := NewCodec()
	input := map[string]interface{}{
		"__class__": "timedelta",
		"seconds":   3600.0,
	}

	result := c.Decode(input)
	td, ok := result.(time.Duration)
	if !ok {
		t.Fatal("Decode() did not return time.Duration")
	}

	if td != 3600*time.Second {
		t.Errorf("timedelta = %v, want 1h", td)
	}
}

func TestCodec_DecodeError(t *testing.T) {
	c := NewCodec()

	tests := []struct {
		name   string
		input  interface{}
		errMsg string
	}{
		{
			name:   "UserError",
			input:  []interface{}{"UserError", []interface{}{"msg", "desc", "domain"}},
			errMsg: "msg",
		},
		{
			name:   "UserWarning",
			input:  []interface{}{"UserWarning", []interface{}{"name", "msg", "desc"}},
			errMsg: "msg",
		},
		{
			name:   "ConcurrencyException",
			input:  []interface{}{"ConcurrencyException", []interface{}{"conflict"}},
			errMsg: "conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.DecodeError(tt.input)
			if err == nil {
				t.Fatal("DecodeError() expected error")
			}
			if err.Error() != tt.errMsg {
				t.Errorf("DecodeError() = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestCodec_DecodeError_Nil(t *testing.T) {
	c := NewCodec()
	err := c.DecodeError(nil)
	if err != nil {
		t.Errorf("DecodeError(nil) = %v, want nil", err)
	}
}

func TestCodec_PrepareObject(t *testing.T) {
	c := NewCodec()
	result := c.PrepareObject("test")
	if result != "test" {
		t.Errorf("PrepareObject() = %v, want test", result)
	}
}

func TestCodec_ConvertJSONObject(t *testing.T) {
	c := NewCodec()
	input := map[string]interface{}{
		"key": "value",
	}

	result := c.ConvertJSONObject(input)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("ConvertJSONObject() did not return map")
	}

	if m["key"] != "value" {
		t.Errorf("ConvertJSONObject()[key] = %v, want value", m["key"])
	}
}

func TestCodec_EncodeParams(t *testing.T) {
	c := NewCodec()
	params := []interface{}{"test", 42}

	result := c.EncodeParams(params)
	if len(result) != 2 {
		t.Errorf("EncodeParams() returned %d items, want 2", len(result))
	}
}

func TestCodec_DecodeResult(t *testing.T) {
	c := NewCodec()
	result := c.DecodeResult("test")
	if result != "test" {
		t.Errorf("DecodeResult() = %v, want test", result)
	}
}
