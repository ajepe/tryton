package orm

import (
	"testing"
	"time"

	"github.com/ajepe/talon/internal/metadata"
)

func TestNewRecord(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	if r.IsNew() != true {
		t.Error("NewRecord().IsNew() = false, want true")
	}
	if r.ID() >= 0 {
		t.Errorf("NewRecord().ID() = %d, want negative", r.ID())
	}
}

func TestNewExistingRecord(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewExistingRecord(m, 42)

	if r.IsNew() != false {
		t.Error("NewExistingRecord().IsNew() = true, want false")
	}
	if r.ID() != 42 {
		t.Errorf("NewExistingRecord().ID() = %d, want 42", r.ID())
	}
}

func TestRecord_SetGet(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	r.Set("name", "John")
	v, ok := r.Get("name")
	if !ok {
		t.Fatal("Get() not found")
	}
	if v != "John" {
		t.Errorf("Get() = %v, want John", v)
	}
}

func TestRecord_IsModified(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	if r.IsModified("name") {
		t.Error("IsModified() = true for unset field")
	}

	r.Set("name", "John")
	if !r.IsModified("name") {
		t.Error("IsModified() = false for modified field")
	}
}

func TestRecord_ModifiedFields(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	r.Set("name", "John")
	r.Set("age", 30)

	fields := r.ModifiedFields()
	if len(fields) != 2 {
		t.Errorf("ModifiedFields() returned %d fields, want 2", len(fields))
	}
}

func TestRecord_IsLoaded(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	if r.IsLoaded("name") {
		t.Error("IsLoaded() = true for unset field")
	}

	r.Set("name", "John")
	if !r.IsLoaded("name") {
		t.Error("IsLoaded() = false for loaded field")
	}
}

func TestRecord_GetContext(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	ctx := r.GetContext()
	if ctx == nil {
		t.Error("GetContext() returned nil")
	}
}

func TestRecord_SetContext(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	ctx := map[string]interface{}{
		"language": "en",
		"company":  1,
	}
	r.SetContext(ctx)

	v, ok := r.Get("language")
	if !ok {
		t.Fatal("Get() not found after SetContext")
	}
	if v != "en" {
		t.Errorf("Get(language) = %v, want en", v)
	}
}

func TestRecord_Validate(t *testing.T) {
	m := &metadata.Model{
		Name: "test.model",
		Fields: map[string]*metadata.Field{
			"name": {Name: "name", Type: metadata.FieldChar, Required: true},
		},
	}
	r := NewRecord(m)

	err := r.Validate()
	if err == nil {
		t.Error("Validate() expected error for missing required field")
	}

	r.Set("name", "John")
	err = r.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestRecord_GetOnChangeValue(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	r.Set("name", "John")
	r.Set("age", 30)

	values := r.GetOnChangeValue()
	if len(values) != 2 {
		t.Errorf("GetOnChangeValue() returned %d values, want 2", len(values))
	}
}

func TestRecord_SetOnChange(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	values := map[string]interface{}{
		"name": "Jane",
	}
	r.SetOnChange(values)

	v, ok := r.Get("name")
	if !ok {
		t.Fatal("Get() not found after SetOnChange")
	}
	if v != "Jane" {
		t.Errorf("Get(name) = %v, want Jane", v)
	}
}

func TestRecord_Copy(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)
	r.Set("name", "John")

	r2 := r.Copy(nil)
	if !r2.IsNew() {
		t.Error("Copy().IsNew() = false, want true")
	}

	v, ok := r2.Get("name")
	if !ok {
		t.Fatal("Copy().Get() not found")
	}
	if v != "John" {
		t.Errorf("Copy().Get(name) = %v, want John", v)
	}
}

func TestRecord_ExportData(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)
	r.Set("name", "John")
	r.Set("age", 30)

	data := r.ExportData([]string{"name", "age"})
	if len(data) != 2 {
		t.Errorf("ExportData() returned %d values, want 2", len(data))
	}
}

func TestRecord_ImportData(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	data := map[string]interface{}{
		"name": "John",
		"age":  30,
	}
	r.ImportData(data)

	v, ok := r.Get("name")
	if !ok {
		t.Fatal("Get() not found after ImportData")
	}
	if v != "John" {
		t.Errorf("Get(name) = %v, want John", v)
	}
}

func TestRecord_RecName(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	name := r.RecName()
	if name == "" {
		t.Error("RecName() returned empty string")
	}

	r.Set("name", "John")
	name = r.RecName()
	if name != "John" {
		t.Errorf("RecName() = %s, want John", name)
	}
}

func TestRecord_Destroy(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	if r.IsDestroyed() {
		t.Error("IsDestroyed() = true before Destroy()")
	}

	r.Destroy()
	if !r.IsDestroyed() {
		t.Error("IsDestroyed() = false after Destroy()")
	}
}

func TestRecord_GetTimestamp(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	ts := r.GetTimestamp()
	if ts == nil {
		t.Error("GetTimestamp() returned nil")
	}
}

func TestRecord_SetTimestamp(t *testing.T) {
	m := &metadata.Model{Name: "test.model"}
	r := NewRecord(m)

	r.SetTimestamp("name", time.Now())
	ts := r.GetTimestamp()
	if ts == nil {
		t.Error("GetTimestamp() returned nil after SetTimestamp")
	}
}
