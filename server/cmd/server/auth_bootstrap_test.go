package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerMuxMountsPlatformAdminRoutes(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")
	server := &Server{
		store:       NewMemoryStore(),
		tmpl:        testTemplates(),
		enforceAuth: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/orgs", nil)
	rec := httptest.NewRecorder()
	server.newMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/login?next=%2Fadmin%2Forgs" {
		t.Fatalf("location = %q", rec.Header().Get("Location"))
	}
}
