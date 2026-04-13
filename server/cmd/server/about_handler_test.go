package main

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestLayoutRendersFooterContent(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	var rendered bytes.Buffer
	view := PageBase{}

	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", view); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}

	body := rendered.String()
	compactBody := strings.Join(strings.Fields(body), " ")
	if !strings.Contains(body, "Forkbomb bv") {
		t.Fatalf("expected footer content, got %q", body)
	}
	if !strings.Contains(compactBody, "This repository/website is part of the CLOSER project") {
		t.Fatalf("expected footer legal text, got %q", body)
	}
	if !strings.Contains(body, "site-footer") {
		t.Fatalf("expected footer markup in layout, got %q", body)
	}
	if strings.Contains(body, `href="/about"`) {
		t.Fatalf("expected about link to be removed, got %q", body)
	}
}

func TestServerMuxAboutRouteReturnsNotFound(t *testing.T) {
	server := &Server{
		store: NewMemoryStore(),
		tmpl:  testTemplates(),
	}

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	rec := httptest.NewRecorder()
	server.newMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleAboutAdditionalBranches(t *testing.T) {
	t.Run("not found on nested path", func(t *testing.T) {
		server := &Server{tmpl: testTemplates()}
		req := httptest.NewRequest(http.MethodGet, "/about/team", nil)
		rec := httptest.NewRecorder()

		server.handleAbout(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("not found on exact path", func(t *testing.T) {
		server := &Server{tmpl: testTemplates()}
		req := httptest.NewRequest(http.MethodGet, "/about", nil)
		rec := httptest.NewRecorder()

		server.handleAbout(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("not found on other methods", func(t *testing.T) {
		server := &Server{tmpl: testTemplates()}
		req := httptest.NewRequest(http.MethodPost, "/about", nil)
		rec := httptest.NewRecorder()

		server.handleAbout(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
}
