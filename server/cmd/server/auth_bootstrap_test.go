package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerMuxDoesNotMountPlatformAdminRoutes(t *testing.T) {
	server := &Server{
		store: NewMemoryStore(),
		tmpl:  testTemplates(),
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/orgs", nil)
	rec := httptest.NewRecorder()
	server.newMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
