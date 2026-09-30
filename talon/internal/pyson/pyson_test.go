package pyson

import (
	"testing"
)

func TestDecoder_Decode_Eval(t *testing.T) {
	ctx := map[string]interface{}{
		"name": "John",
	}
	d := NewDecoder(ctx)

	input := map[string]interface{}{
		"__class__": "Eval",
		"value":     "name",
	}

	result := d.Decode(input)
	if result != "John" {
		t.Errorf("Decode() = %v, want John", result)
	}
}

func TestDecoder_Decode_Context(t *testing.T) {
	ctx := map[string]interface{}{
		"company": 1,
	}
	d := NewDecoder(ctx)

	input := map[string]interface{}{
		"__class__": "Context",
		"key":       "company",
	}

	result := d.Decode(input)
	if result == nil {
		t.Error("Decode() = nil, want non-nil")
	}
}

func TestDecoder_Decode_ActiveID(t *testing.T) {
	ctx := map[string]interface{}{
		"active_id": 42,
	}
	d := NewDecoder(ctx)

	input := map[string]interface{}{
		"__class__": "ActiveID",
	}

	result := d.Decode(input)
	if result == nil {
		t.Error("Decode() = nil, want non-nil")
	}
}

func TestDecoder_Decode_ActiveIds(t *testing.T) {
	ctx := map[string]interface{}{
		"active_ids": []int{1, 2, 3},
	}
	d := NewDecoder(ctx)

	input := map[string]interface{}{
		"__class__": "ActiveIds",
	}

	result := d.Decode(input)
	if result == nil {
		t.Error("Decode() = nil, want non-nil")
	}
}

func TestDecoder_Decode_CurrentCompany(t *testing.T) {
	ctx := map[string]interface{}{
		"company": 5,
	}
	d := NewDecoder(ctx)

	input := map[string]interface{}{
		"__class__": "CurrentCompany",
	}

	result := d.Decode(input)
	if result == nil {
		t.Error("Decode() = nil, want non-nil")
	}
}

func TestDecoder_Decode_CurrentLanguage(t *testing.T) {
	ctx := map[string]interface{}{
		"language": "en",
	}
	d := NewDecoder(ctx)

	input := map[string]interface{}{
		"__class__": "CurrentLanguage",
	}

	result := d.Decode(input)
	if result != "en" {
		t.Errorf("Decode() = %v, want en", result)
	}
}

func TestDecoder_Decode_PlainMap(t *testing.T) {
	d := NewDecoder(map[string]interface{}{})

	input := map[string]interface{}{
		"key": "value",
	}

	result := d.Decode(input)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Decode() did not return map")
	}
	if m["key"] != "value" {
		t.Errorf("Decode()[key] = %v, want value", m["key"])
	}
}

func TestDecoder_Decode_PlainString(t *testing.T) {
	d := NewDecoder(map[string]interface{}{})

	result := d.Decode("hello")
	if result != "hello" {
		t.Errorf("Decode() = %v, want hello", result)
	}
}

func TestEncoder_Encode_Eval(t *testing.T) {
	e := NewEncoder()

	result := e.Encode("test")
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Encode() did not return map")
	}
	if m["__class__"] != "Eval" {
		t.Errorf("__class__ = %v, want Eval", m["__class__"])
	}
}

func TestEncoder_EncodeContext(t *testing.T) {
	e := NewEncoder()

	result := e.EncodeContext("company")
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("EncodeContext() did not return map")
	}
	if m["__class__"] != "Context" {
		t.Errorf("__class__ = %v, want Context", m["__class__"])
	}
	if m["key"] != "company" {
		t.Errorf("key = %v, want company", m["key"])
	}
}

func TestEncoder_EncodeActiveID(t *testing.T) {
	e := NewEncoder()

	result := e.EncodeActiveID()
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("EncodeActiveID() did not return map")
	}
	if m["__class__"] != "ActiveID" {
		t.Errorf("__class__ = %v, want ActiveID", m["__class__"])
	}
}

func TestEncoder_EncodeNot(t *testing.T) {
	e := NewEncoder()

	result := e.EncodeNot("value")
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("EncodeNot() did not return map")
	}
	if m["__class__"] != "Not" {
		t.Errorf("__class__ = %v, want Not", m["__class__"])
	}
}

func TestEncoder_EncodeAnd(t *testing.T) {
	e := NewEncoder()

	result := e.EncodeAnd([]interface{}{"a", "b"})
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("EncodeAnd() did not return map")
	}
	if m["__class__"] != "And" {
		t.Errorf("__class__ = %v, want And", m["__class__"])
	}
}

func TestEncoder_EncodeOr(t *testing.T) {
	e := NewEncoder()

	result := e.EncodeOr([]interface{}{"a", "b"})
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("EncodeOr() did not return map")
	}
	if m["__class__"] != "Or" {
		t.Errorf("__class__ = %v, want Or", m["__class__"])
	}
}

func TestEncoder_EncodeCompare(t *testing.T) {
	e := NewEncoder()

	result := e.EncodeCompare("age", ">", 18)
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatal("EncodeCompare() did not return slice")
	}
	if len(arr) != 3 {
		t.Errorf("EncodeCompare() returned %d items, want 3", len(arr))
	}
}

func TestEncoder_EncodeIn(t *testing.T) {
	e := NewEncoder()

	result := e.EncodeIn("status", []interface{}{"active", "pending"})
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatal("EncodeIn() did not return slice")
	}
	if arr[1] != "in" {
		t.Errorf("operator = %v, want in", arr[1])
	}
}

func TestEncoder_EncodeLike(t *testing.T) {
	e := NewEncoder()

	result := e.EncodeLike("name", "%john%")
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatal("EncodeLike() did not return slice")
	}
	if arr[1] != "like" {
		t.Errorf("operator = %v, want like", arr[1])
	}
}

func TestEncoder_EncodeILike(t *testing.T) {
	e := NewEncoder()

	result := e.EncodeILike("name", "%john%")
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatal("EncodeILike() did not return slice")
	}
	if arr[1] != "ilike" {
		t.Errorf("operator = %v, want ilike", arr[1])
	}
}

func TestEncoder_EncodeChildOf(t *testing.T) {
	e := NewEncoder()

	result := e.EncodeChildOf("parent", 5)
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatal("EncodeChildOf() did not return slice")
	}
	if arr[1] != "child_of" {
		t.Errorf("operator = %v, want child_of", arr[1])
	}
}

func TestEncoder_EncodeParentOf(t *testing.T) {
	e := NewEncoder()

	result := e.EncodeParentOf("child", 5)
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatal("EncodeParentOf() did not return slice")
	}
	if arr[1] != "parent_of" {
		t.Errorf("operator = %v, want parent_of", arr[1])
	}
}

func TestDomainParser_Parse(t *testing.T) {
	fields := map[string]interface{}{
		"name": "Name",
		"age":  "Age",
	}
	p := NewDomainParser(fields)

	domain, err := p.Parse("name = John")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if domain == nil {
		t.Fatal("Parse() returned nil")
	}
}

func TestDomainParser_String(t *testing.T) {
	fields := map[string]interface{}{
		"name": "Name",
	}
	p := NewDomainParser(fields)

	domain := []interface{}{"name", "=", "John"}
	result := p.String(domain)
	if result == "" {
		t.Error("String() returned empty")
	}
}

func TestDomainParser_Completion(t *testing.T) {
	fields := map[string]interface{}{
		"name":     "Name",
		"age":      "Age",
		"address":  "Address",
	}
	p := NewDomainParser(fields)

	completions := p.Completion("a")
	if len(completions) == 0 {
		t.Error("Completion() returned empty")
	}
}

func TestDomainInversion_Evaluate(t *testing.T) {
	fields := map[string]interface{}{
		"age": "Age",
	}
	d := NewDomainInversion(fields)

	domain := []interface{}{"age", ">", 18.0}
	values := map[string]interface{}{
		"age": 25.0,
	}

	result := d.Evaluate(domain, values)
	if !result {
		t.Error("Evaluate() = false, want true")
	}
}

func TestDomainInversion_Simplify(t *testing.T) {
	fields := map[string]interface{}{}
	d := NewDomainInversion(fields)

	domain := []interface{}{"a", "=", 1.0}
	result := d.Simplify(domain)
	if len(result) == 0 {
		t.Error("Simplify() returned empty")
	}
}

func TestDomainInversion_Canonicalize(t *testing.T) {
	fields := map[string]interface{}{}
	d := NewDomainInversion(fields)

	domain := []interface{}{"a", "=", 1.0}
	result := d.Canonicalize(domain)
	if len(result) == 0 {
		t.Error("Canonicalize() returned empty")
	}
}

func TestDomainInversion_Localize(t *testing.T) {
	fields := map[string]interface{}{}
	d := NewDomainInversion(fields)

	domain := []interface{}{"a", "=", 1.0}
	result := d.Localize(domain, "en")
	if len(result) == 0 {
		t.Error("Localize() returned empty")
	}
}

func TestDomainInversion_SolveUnique(t *testing.T) {
	fields := map[string]interface{}{}
	d := NewDomainInversion(fields)

	domain := []interface{}{
		[]interface{}{"a", "=", 1.0},
		[]interface{}{"b", "=", 2.0},
	}
	result := d.SolveUnique(domain)
	if len(result) == 0 {
		t.Error("SolveUnique() returned empty")
	}
}

func TestDomainInversion_Invert(t *testing.T) {
	fields := map[string]interface{}{}
	d := NewDomainInversion(fields)

	domain := []interface{}{
		"a", "=", 1.0,
		"b", "=", 2.0,
	}
	result := d.Invert(domain, "a")
	if len(result) == 0 {
		t.Error("Invert() returned empty")
	}
}

func TestDomainInversion_Like(t *testing.T) {
	fields := map[string]interface{}{}
	d := NewDomainInversion(fields)

	result := d.like("john doe", "%doe%")
	if !result {
		t.Error("like() = false, want true")
	}
}

func TestDomainInversion_ILike(t *testing.T) {
	fields := map[string]interface{}{}
	d := NewDomainInversion(fields)

	result := d.ilike("John Doe", "%doe%")
	if !result {
		t.Error("ilike() = false, want true")
	}
}
