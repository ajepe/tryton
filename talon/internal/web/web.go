package web

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var hashPattern = regexp.MustCompile(`[.-][A-Za-z0-9_-]{8,}\.`)

type Handler struct {
	root string
}

func NewHandler(root string) (*Handler, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "stat", Path: root, Err: os.ErrInvalid}
	}
	return &Handler{root: root}, nil
}

func (h *Handler) Root() string {
	return h.root
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	upath := path.Clean("/" + r.URL.Path)
	filePath := filepath.Join(h.root, filepath.FromSlash(upath))
	if !strings.HasPrefix(filePath, filepath.Clean(h.root)+string(os.PathSeparator)) &&
		filePath != filepath.Clean(h.root) {
		http.NotFound(w, r)
		return
	}

	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		if strings.Contains(path.Base(upath), ".") {
			http.NotFound(w, r)
			return
		}
		h.serveFile(w, r, filepath.Join(h.root, "index.html"), false)
		return
	}
	h.serveFile(w, r, filePath, true)
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, filePath string, cacheable bool) {
	file, err := os.Open(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	if cacheable && isImmutable(filepath.Base(filePath)) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}

	if acceptsGzip(r) && compressible(contentType) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")
		gzipWriter := gzip.NewWriter(w)
		defer gzipWriter.Close()
		if r.Method == http.MethodHead {
			return
		}
		_, _ = io.Copy(gzipWriter, file)
		return
	}

	http.ServeContent(w, r, filepath.Base(filePath), info.ModTime(), file)
}

func isImmutable(name string) bool {
	if strings.HasPrefix(name, "index.") {
		return false
	}
	return hashPattern.MatchString(name)
}

func acceptsGzip(r *http.Request) bool {
	for _, encoding := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		if strings.TrimSpace(strings.SplitN(encoding, ";", 2)[0]) == "gzip" {
			return true
		}
	}
	return false
}

func compressible(contentType string) bool {
	if contentType == "" {
		return false
	}
	base := strings.SplitN(contentType, ";", 2)[0]
	base = strings.TrimSpace(base)
	if strings.HasPrefix(base, "text/") {
		return true
	}
	switch base {
	case "application/json", "application/javascript", "application/xml",
		"application/manifest+json", "image/svg+xml":
		return true
	}
	return false
}
