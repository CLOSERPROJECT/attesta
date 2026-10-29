package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeAppStaticSPAFallback(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("<html>app-shell</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	assets := filepath.Join(root, "_app")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, "entry.js"), []byte("console.log(1)"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ATTESTA_APP_DIR", root)

	server := &Server{}

	t.Run("root serves index", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/app", nil)
		server.serveAppStatic(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "app-shell") {
			t.Fatalf("body = %q", rec.Body.String())
		}
	})

	t.Run("client route falls back to index", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/app/about", nil)
		server.serveAppStatic(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "app-shell") {
			t.Fatalf("body = %q", rec.Body.String())
		}
	})

	t.Run("asset is served", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/app/_app/entry.js", nil)
		server.serveAppStatic(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%q", rec.Code, rec.Body.String())
		}
		if got := rec.Body.String(); got != "console.log(1)" {
			t.Fatalf("body = %q", got)
		}
	})
}
