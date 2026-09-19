// Package web serves the built frontend out of the binary itself, so the whole
// application ships as a single static executable with no files beside it.
package web

import (
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Handler serves the embedded single-page app.
//
// Unknown paths fall through to index.html rather than 404ing, because the
// frontend routes on the client: a hard refresh on /accounts has to return the
// app shell, not an error.
func Handler() (http.Handler, error) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, fmt.Errorf("open embedded frontend: %w", err)
	}
	return &spaHandler{fsys: sub, files: http.FileServer(http.FS(sub))}, nil
}

type spaHandler struct {
	fsys  fs.FS
	files http.Handler
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// fs.FS paths are slash-separated with no leading slash. Cleaning first
	// collapses any ".." segments; anything that escapes the root simply fails
	// to open below and falls through to the shell, since fs.Sub is rooted.
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")

	if name == "" || name == "." {
		h.serveIndex(w)
		return
	}

	f, err := h.fsys.Open(name)
	if err != nil {
		h.serveIndex(w)
		return
	}
	info, statErr := f.Stat()
	_ = f.Close()
	if statErr != nil || info.IsDir() {
		h.serveIndex(w)
		return
	}

	// Vite fingerprints everything under assets/, so those URLs are immutable
	// and can be cached indefinitely. Anything else might change in place.
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	h.files.ServeHTTP(w, r)
}

// serveIndex writes the app shell. It is never cached: a stale shell points the
// browser at fingerprinted asset URLs that no longer exist after a deploy.
func (h *spaHandler) serveIndex(w http.ResponseWriter) {
	b, err := fs.ReadFile(h.fsys, "index.html")
	if err != nil {
		http.Error(w, "no frontend was built into this binary", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}
