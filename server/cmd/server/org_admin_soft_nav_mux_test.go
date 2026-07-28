package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOrgAdminSoftNavViaMuxReturnsRequestedPanel(t *testing.T) {
	now := time.Now().UTC()
	server := orgAdminSectionURLTestServer(t, now)
	mux := server.newMux()

	req := httptest.NewRequest(http.MethodGet, "/my/organization/roles", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "admin-console")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="admin-console"`) {
		t.Fatalf("expected admin-console fragment, got: %s", body)
	}
	if strings.Contains(body, `class="topbar"`) || strings.Contains(body, "<html") {
		t.Fatalf("HTMX soft-nav must not include layout, got: %s", body)
	}
	assertOrgAdminActivePanel(t, body, "roles")
}

func TestResolveOrgAdminActivePanelStrippedMyPrefix(t *testing.T) {
	// handleMyRoutes clones to /organization/... before handlers run.
	req := httptest.NewRequest(http.MethodGet, "/organization/roles", nil)
	if got := resolveOrgAdminActivePanel(req, OrgAdminErrors{}, ""); got != "roles" {
		t.Fatalf("resolveOrgAdminActivePanel(%q) = %q, want roles", req.URL.Path, got)
	}
}
