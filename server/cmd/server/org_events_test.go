package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrgEventsRouteRequiresAuthentication(t *testing.T) {
	server := &Server{
		store:       NewMemoryStore(),
		tmpl:        testTemplates(),
		enforceAuth: true,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/org/events?stream=pv-recovery", nil)
	rec := httptest.NewRecorder()
	server.newMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body=%s, want 401", rec.Code, rec.Body.String())
	}
}
