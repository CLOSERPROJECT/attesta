package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
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
	compactBody := strings.Join(strings.Fields(body), " ")
	if !strings.Contains(compactBody, "Project information, licensing, and funding acknowledgements for Attesta.") {
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

func TestHandleAboutAdditionalBranches(t *testing.T) {
	now := time.Now().UTC()

	t.Run("not found on nested path", func(t *testing.T) {
		server := &Server{tmpl: testTemplates()}
		req := httptest.NewRequest(http.MethodGet, "/about/team", nil)
		rec := httptest.NewRecorder()

		server.handleAbout(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		server := &Server{tmpl: testTemplates()}
		req := httptest.NewRequest(http.MethodPost, "/about", nil)
		rec := httptest.NewRecorder()

		server.handleAbout(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("authenticated request uses user page base", func(t *testing.T) {
		user := AccountUser{
			ID:           primitive.NewObjectID(),
			Email:        "org-admin@example.com",
			OrgSlug:      "acme",
			RoleSlugs:    []string{"org-admin"},
			Status:       "active",
			PasswordHash: "unused",
		}
		server := &Server{
			authorizer:  fakeAuthorizer{},
			tmpl:        testTemplates(),
			identity:    testIdentityForSessions(now, map[string]AccountUser{"session-1": user}),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, "/about", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()

		server.handleAbout(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "ABOUT") || !strings.Contains(body, "MyOrg") {
			t.Fatalf("body = %q", body)
		}
	})
}
