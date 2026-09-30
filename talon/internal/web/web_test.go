package web

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setup(t *testing.T) *Handler {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"index.html":                "<html>app</html>",
		"app.js":                    "console.log('app');",
		"assets/main.abcdef12.js":   "console.log('hashed');",
		"assets/style.12345678.css": "body { color: red; }",
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	handler, err := NewHandler(root)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return handler
}

func TestServeFile(t *testing.T) {
	handler := setup(t)
	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "console.log('app');" {
		t.Errorf("body = %q", rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("cache control = %q, want no-cache", rec.Header().Get("Cache-Control"))
	}
}

func TestImmutableCacheHeader(t *testing.T) {
	handler := setup(t)
	req := httptest.NewRequest(http.MethodGet, "/assets/main.abcdef12.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("cache control = %q, want immutable", rec.Header().Get("Cache-Control"))
	}
}

func TestGzipEncoding(t *testing.T) {
	handler := setup(t)
	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("content encoding = %q, want gzip", rec.Header().Get("Content-Encoding"))
	}
	reader, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gzip reader error = %v", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read error = %v", err)
	}
	if string(data) != "console.log('app');" {
		t.Errorf("decompressed body = %q", data)
	}
}

func TestSPAFallback(t *testing.T) {
	handler := setup(t)
	req := httptest.NewRequest(http.MethodGet, "/party/42", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<html>") {
		t.Errorf("body = %q, want index.html", rec.Body.String())
	}
}

func TestMissingAsset(t *testing.T) {
	handler := setup(t)
	req := httptest.NewRequest(http.MethodGet, "/missing.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestRejectsPost(t *testing.T) {
	handler := setup(t)
	req := httptest.NewRequest(http.MethodPost, "/app.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestDirectoryTraversal(t *testing.T) {
	handler := setup(t)
	req := httptest.NewRequest(http.MethodGet, "/../secret.txt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Errorf("traversal returned status 200")
	}
}

func TestNewHandlerRequiresDirectory(t *testing.T) {
	if _, err := NewHandler(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("NewHandler() expected error for missing directory")
	}
	file := filepath.Join(t.TempDir(), "file.txt")
	os.WriteFile(file, []byte("x"), 0o644)
	if _, err := NewHandler(file); err == nil {
		t.Error("NewHandler() expected error for file")
	}
}
