package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	h, err := Handler()
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}
	return h
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestServesIndexAtRoot(t *testing.T) {
	rec := get(t, newHandler(t), "/")

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rec.Body.String(), "<html") {
		t.Error("body does not look like an HTML document")
	}
}

// A client-side route must survive a hard refresh: the server has never heard
// of /accounts, but returning 404 would break the app.
func TestUnknownPathFallsBackToShell(t *testing.T) {
	h := newHandler(t)

	for _, path := range []string{"/accounts", "/accounts/123", "/deeply/nested/route"} {
		t.Run(path, func(t *testing.T) {
			rec := get(t, h, path)
			if rec.Code != http.StatusOK {
				t.Errorf("got %d, want 200", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "<html") {
				t.Error("expected the app shell")
			}
		})
	}
}

// The shell must never be cached: a stale one asks for fingerprinted asset URLs
// that no longer exist after a deploy.
func TestShellIsNotCached(t *testing.T) {
	rec := get(t, newHandler(t), "/")

	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}
}

// Path traversal must not escape the embedded filesystem. fs.Sub is rooted, so
// these should fall through to the shell rather than reading anything.
func TestTraversalIsContained(t *testing.T) {
	h := newHandler(t)

	for _, path := range []string{"/../etc/passwd", "/../../../../etc/passwd", "/assets/../../etc/passwd"} {
		t.Run(path, func(t *testing.T) {
			rec := get(t, h, path)
			if rec.Code != http.StatusOK {
				t.Errorf("got %d, want 200 (the shell)", rec.Code)
			}
			if strings.Contains(rec.Body.String(), "root:") {
				t.Fatal("served content from outside the embedded filesystem")
			}
		})
	}
}
