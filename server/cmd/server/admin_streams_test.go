package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newStreamsAdminServer(t *testing.T, store Store) *Server {
	t.Helper()
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "accessible.yaml")
	if err := os.WriteFile(path, []byte(minimalCategorizedWorkflowYAML(
		"  categorySlug: supply-chain\n  subCategorySlug: procurement\n",
	)), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	now := time.Now().UTC()
	return &Server{
		authorizer:  fakeAuthorizer{},
		store:       store,
		identity:    &fakeIdentityStore{},
		tmpl:        parseTestTemplates(t),
		configDir:   tempDir,
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
}

func TestHandleAdminStreamsRequiresPlatformAdmin(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	user := AccountUser{
		Email:     "member@example.com",
		RoleSlugs: []string{"org-admin"},
		OrgSlug:   "acme",
	}
	server := &Server{
		authorizer: fakeAuthorizer{},
		store:      NewMemoryStore(),
		identity: testIdentityForSessions(now, map[string]AccountUser{
			"member-session": user,
		}),
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/streams", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "member-session"})
	rec := httptest.NewRecorder()
	server.handleAdminStreams(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestHandleAdminStreamsRendersCatalog(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := newStreamsAdminServer(t, store)

	req := httptest.NewRequest(http.MethodGet, "/admin/streams", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.handleAdminStreams(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="admin-console"`,
		`href="/admin/streams"`,
		`aria-current="page"`,
		"Browse and manage platform streams",
		`id="platform-admin-streams"`,
		`id="my-home-category-sidebar"`,
		`class="my-home-catalog"`,
		"Supply Chain",
		"Procurement",
		`href="/my/organization/formata-builder?new=true"`,
		"Create a stream",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in streams catalog, got: %s", want, body)
		}
	}
	if strings.Contains(body, "Pending organization requests") {
		t.Fatalf("streams panel must not render organizations pending block, got: %s", body)
	}
}

func TestHandleAdminStreamsHTMXReturnsAdminConsole(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := newStreamsAdminServer(t, store)

	req := httptest.NewRequest(http.MethodGet, "/admin/streams", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "admin-console")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.handleAdminStreams(rec, req)

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
		`hx-get="/admin/organizations"`,
		`hx-get="/admin/streams"`,
		`hx-get="/admin/categories"`,
		`hx-target="#admin-console"`,
		"Browse and manage platform streams",
		`id="platform-admin-streams"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in streams console, got: %s", want, body)
		}
	}
}
