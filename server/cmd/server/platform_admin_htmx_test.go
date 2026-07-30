package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleAdminCategoriesHTMXReturnsAdminConsole(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := newCategoriesAdminServer(t, store)

	req := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "admin-console")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleAdminCategories(rec, req)

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
	for _, want := range []string{
		`hx-get="/admin/orgs"`,
		`hx-target="#admin-console"`,
		"Manage stream discovery taxonomy",
		"Supply Chain",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in categories console, got: %s", want, body)
		}
	}
}

func TestHandleAdminOrgsHTMXSearchStillReturnsResults(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       NewMemoryStore(),
		identity:    &fakeIdentityStore{},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/orgs?q=acme", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "platform-admin-results")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleAdminOrgs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "PLATFORM_ADMIN_RESULTS") {
		t.Fatalf("expected results partial, got: %s", body)
	}
	if strings.Contains(body, "ADMIN_CONSOLE") || strings.Contains(body, `id="admin-console"`) {
		t.Fatalf("search HTMX must not return admin_console, got: %s", body)
	}
}
