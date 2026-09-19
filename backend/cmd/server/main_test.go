package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// addrOf returns the host:port an httptest server listens on, in the same form
// the ADDR environment variable takes.
func addrOf(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	return strings.TrimPrefix(srv.URL, "http://")
}

// The container HEALTHCHECK depends on this, and Compose will not start the
// frontend until it passes, so a false negative here stalls the whole stack.
func TestProbeHealthAcceptsHealthyServer(t *testing.T) {
	var probedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := probeHealth(addrOf(t, srv)); err != nil {
		t.Fatalf("probeHealth on a healthy server: %v", err)
	}
	if probedPath != "/api/v1/health" {
		t.Errorf("probed %q, want /api/v1/health", probedPath)
	}
}

func TestProbeHealthRejectsUnhealthyServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	if err := probeHealth(addrOf(t, srv)); err == nil {
		t.Error("expected an error for a 503, got nil")
	}
}

func TestProbeHealthRejectsMalformedAddr(t *testing.T) {
	for _, addr := range []string{"", "not-an-address", "localhost"} {
		t.Run(addr, func(t *testing.T) {
			if err := probeHealth(addr); err == nil {
				t.Errorf("expected an error for ADDR %q, got nil", addr)
			}
		})
	}
}

// A wildcard listen address cannot be dialled, so the probe has to rewrite it to
// the loopback or the health check fails against a correctly running server.
func TestProbeHealthRewritesWildcardHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, port, ok := strings.Cut(addrOf(t, srv), ":")
	if !ok {
		t.Fatalf("could not split a port out of %q", srv.URL)
	}

	for _, host := range []string{"", "0.0.0.0"} {
		t.Run("host="+host, func(t *testing.T) {
			if err := probeHealth(host + ":" + port); err != nil {
				t.Errorf("probeHealth(%q): %v", host+":"+port, err)
			}
		})
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", []string{}},
		{"   ", []string{}},
		{"http://a", []string{"http://a"}},
		{"http://a,http://b", []string{"http://a", "http://b"}},
		{" http://a , http://b ", []string{"http://a", "http://b"}},
		{"http://a,,http://b,", []string{"http://a", "http://b"}},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got := splitAndTrim(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitAndTrim(%q) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

// An unparseable LOG_LEVEL must not stop the server from starting.
func TestLogLevel(t *testing.T) {
	tests := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"info":    slog.LevelInfo,
		"WARN":    slog.LevelWarn,
		"error":   slog.LevelError,
		"":        slog.LevelInfo,
		"verbose": slog.LevelInfo,
	}
	for in, want := range tests {
		t.Run(in, func(t *testing.T) {
			if got := logLevel(in); got != want {
				t.Errorf("logLevel(%q) = %v, want %v", in, got, want)
			}
		})
	}
}
