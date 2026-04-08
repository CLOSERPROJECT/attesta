package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleAboutRendersMovedFooterContent(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	server := &Server{tmpl: tmpl}

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	rec := httptest.NewRecorder()
	server.handleAbout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Project information, licensing, and funding acknowledgements for Attesta.") {
		t.Fatalf("expected about intro, got %q", body)
	}
	if !strings.Contains(body, "Forkbomb bv") {
		t.Fatalf("expected moved footer content, got %q", body)
	}
	if !strings.Contains(body, `href="/about"`) {
		t.Fatalf("expected navbar about link, got %q", body)
	}
	if strings.Contains(body, "site-footer") {
		t.Fatalf("expected footer markup to be removed from layout, got %q", body)
	}
}

func TestServerMuxMountsAboutRoute(t *testing.T) {
	server := &Server{
		store: NewMemoryStore(),
		tmpl:  testTemplates(),
	}

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	rec := httptest.NewRecorder()
	server.newMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "ABOUT") {
		t.Fatalf("body = %q, want ABOUT marker", rec.Body.String())
	}
}
