package filestore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestName(t *testing.T) {
	first := Name([]byte("hello"))
	second := Name([]byte("hello"))
	other := Name([]byte("world"))

	if first != second {
		t.Errorf("Name() not deterministic: %q != %q", first, second)
	}
	if first == other {
		t.Error("Name() should differ for different content")
	}
	if len(first) != 64 {
		t.Errorf("Name() length = %d, want 64", len(first))
	}
}

func TestSaveGetDelete(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	name, err := store.Save([]byte("content"))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if !store.Exists(name) {
		t.Error("Exists() = false after Save")
	}

	data, err := store.Get(name)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(data) != "content" {
		t.Errorf("Get() = %q, want content", data)
	}

	if err := store.Delete(name); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if store.Exists(name) {
		t.Error("Exists() = true after Delete")
	}
	if err := store.Delete(name); err != nil {
		t.Errorf("Delete() on missing file error = %v", err)
	}
}

func TestSaveDeduplicates(t *testing.T) {
	store, _ := New(t.TempDir())
	firstName, _ := store.Save([]byte("same"))
	secondName, _ := store.Save([]byte("same"))
	if firstName != secondName {
		t.Errorf("Save() names differ: %q != %q", firstName, secondName)
	}
}

func TestPathLayout(t *testing.T) {
	store, _ := New(t.TempDir())
	name, _ := store.Save([]byte("content"))
	path := store.Path(name)
	dir := filepath.Dir(path)
	if filepath.Base(dir) != name[2:4] {
		t.Errorf("base dir = %q, want %q", filepath.Base(dir), name[2:4])
	}
	parent := filepath.Base(filepath.Dir(dir))
	if parent != name[:2] {
		t.Errorf("parent dir = %q, want %q", parent, name[:2])
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("stored file missing: %v", err)
	}
}

func TestNewRequiresRoot(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Error("New() expected error for empty root")
	}
}
