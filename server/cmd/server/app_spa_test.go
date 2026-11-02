package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppBuildDirEnvPrecedence(t *testing.T) {
	t.Setenv("ATTESTA_APP_DIR", "")
	t.Setenv("ATTESTA_ROOT_DIR", "/tmp/attesta-root")
	if got := appBuildDir(); got != filepath.Join("/tmp/attesta-root", "app", "build") {
		t.Fatalf("root dir = %q", got)
	}
	t.Setenv("ATTESTA_APP_DIR", "/explicit/app")
	if got := appBuildDir(); got != "/explicit/app" {
		t.Fatalf("app dir wins = %q", got)
	}
	t.Setenv("ATTESTA_APP_DIR", "")
	t.Setenv("ATTESTA_ROOT_DIR", "")
	if got := appBuildDir(); got != filepath.Join("..", "app", "build") {
		t.Fatalf("default = %q", got)
	}
}

func TestHandleAppSPAMethodAuthStaticAndDevProxy(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("<html>app-shell</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ATTESTA_APP_DIR", root)

	open := &Server{enforceAuth: false}

	rec := httptest.NewRecorder()
	open.handleAppSPA(rec, httptest.NewRequest(http.MethodPost, "/app", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("post status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	open.handleAppSPA(rec, httptest.NewRequest(http.MethodGet, "/app", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "app-shell") {
		t.Fatalf("static get = %d %q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	open.handleAppSPA(rec, httptest.NewRequest(http.MethodHead, "/app", nil))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("static head = %d body=%q", rec.Code, rec.Body.String())
	}

	gated := &Server{enforceAuth: true}
	rec = httptest.NewRecorder()
	gated.handleAppSPA(rec, httptest.NewRequest(http.MethodGet, "/app", nil))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("unauth redirect = %d", rec.Code)
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app/dev" {
			t.Fatalf("proxied path = %q", r.URL.Path)
		}
		w.Header().Set("X-Dev", "1")
		_, _ = w.Write([]byte("from-dev"))
	}))
	t.Cleanup(backend.Close)

	dev := &Server{enforceAuth: false, appDevServer: backend.URL}
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/app/dev", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	dev.handleAppSPA(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "from-dev" || rec.Header().Get("X-Dev") != "1" {
		t.Fatalf("dev proxy = %d %q hdr=%v", rec.Code, rec.Body.String(), rec.Header())
	}

	bad := &Server{enforceAuth: false, appDevServer: "://bad"}
	rec = httptest.NewRecorder()
	bad.proxyAppDev(rec, httptest.NewRequest(http.MethodGet, "/app", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("bad dev url = %d", rec.Code)
	}

	down := &Server{enforceAuth: false, appDevServer: "http://127.0.0.1:1"}
	rec = httptest.NewRecorder()
	down.proxyAppDev(rec, httptest.NewRequest(http.MethodGet, "/app", nil))
	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "unavailable") {
		t.Fatalf("down proxy = %d %q", rec.Code, rec.Body.String())
	}
}

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

	t.Run("missing asset 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/app/_app/missing.js", nil)
		server.serveAppStatic(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("path escape rejected", func(t *testing.T) {
		_, _, err := readAppBuildAsset("../outside")
		if err == nil {
			t.Fatal("expected escape error")
		}
	})
}
