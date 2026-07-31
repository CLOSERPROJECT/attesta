package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleAdminRootRedirectsToOrganizations(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	server := &Server{
		authorizer:  fakeAuthorizer{},
		identity:    &fakeIdentityStore{},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.handleAdminRoot(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != adminPath("organizations") {
		t.Fatalf("Location = %q, want %q", loc, adminPath("organizations"))
	}
}

func TestHandleAdminRootRejectsUnauthenticated(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	server := &Server{
		authorizer:  fakeAuthorizer{},
		identity:    &fakeIdentityStore{},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return time.Now().UTC() },
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	server.handleAdminRoot(rec, req)

	if loc := rec.Header().Get("Location"); loc == adminPath("organizations") {
		t.Fatalf("must not redirect unauthenticated user to organizations, status=%d location=%q", rec.Code, loc)
	}
}

func TestMuxAdminRootRedirects(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	server := &Server{
		authorizer:  fakeAuthorizer{},
		identity:    &fakeIdentityStore{},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.newMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/admin/organizations" {
		t.Fatalf("Location = %q, want /admin/organizations", loc)
	}
}

func TestMuxAdminRootTrailingSlashRedirects(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	server := &Server{
		authorizer:  fakeAuthorizer{},
		identity:    &fakeIdentityStore{},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.newMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/admin/organizations" {
		t.Fatalf("Location = %q, want /admin/organizations", loc)
	}
}
