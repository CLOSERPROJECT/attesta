package main

import (
	"errors"
	"html/template"
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
		`id="platform-admin-stream-catalog"`,
		`id="platform-admin-stream-filter"`,
		`id="platform-admin-stream-results"`,
		`name="category"`,
		`name="subCategory"`,
		`hx-push-url="false"`,
		"All categories",
		"All sub-categories",
		"Uncategorized",
		"Supply Chain",
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
	for _, gone := range []string{
		`id="my-home-category-sidebar"`,
		`class="category-sidebar"`,
		`class="my-home-catalog"`,
		`href="#cat-`,
		`nav-drawer-trigger`,
	} {
		if strings.Contains(body, gone) {
			t.Fatalf("platform catalog must not render %q, got: %s", gone, body)
		}
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

func TestHandleAdminStreamsHTMXTaxonomyPartials(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := newStreamsAdminServer(t, store)
	if err := os.WriteFile(
		filepath.Join(server.configDir, "uncategorized.yaml"),
		[]byte(strings.Replace(minimalCategorizedWorkflowYAML(""), `name: "Workflow"`, `name: "Uncategorized Workflow"`, 1)),
		0o644,
	); err != nil {
		t.Fatalf("write uncategorized config: %v", err)
	}

	request := func(target, rawQuery string) string {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/admin/streams?"+rawQuery, nil)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Target", target)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()
		server.handleAdminStreams(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("target %q status = %d; body = %s", target, rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}

	categoryBody := request(platformStreamsCatalogTargetID, "category=supply-chain")
	for _, want := range []string{
		`id="platform-admin-stream-catalog"`,
		`id="platform-admin-stream-filter"`,
		`id="platform-admin-stream-results"`,
		`value="supply-chain"`,
		"Procurement",
		"Order Fulfillment",
	} {
		if !strings.Contains(categoryBody, want) {
			t.Fatalf("category partial missing %q in %s", want, categoryBody)
		}
	}
	if strings.Contains(categoryBody, "<html") || strings.Contains(categoryBody, `id="admin-console"`) {
		t.Fatalf("category partial must only return filter and results, got: %s", categoryBody)
	}
	if strings.Contains(categoryBody, "Uncategorized Workflow") {
		t.Fatalf("category partial must exclude uncategorized stream, got: %s", categoryBody)
	}

	resultsBody := request(platformStreamsResultsTargetID, "category=uncategorized")
	if !strings.Contains(resultsBody, `id="platform-admin-stream-results"`) || !strings.Contains(resultsBody, "Uncategorized Workflow") {
		t.Fatalf("uncategorized results partial missing stream results: %s", resultsBody)
	}
	for _, gone := range []string{
		`id="platform-admin-stream-catalog"`,
		`id="platform-admin-stream-filter"`,
		"Accessible workflow",
		"<html",
	} {
		if strings.Contains(resultsBody, gone) {
			t.Fatalf("results partial must not contain %q, got: %s", gone, resultsBody)
		}
	}
}

func TestHandleAdminStreamsMethodNotAllowed(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := newStreamsAdminServer(t, store)

	req := httptest.NewRequest(http.MethodPost, "/admin/streams", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.handleAdminStreams(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleAdminStreamsCatalogLoadError(t *testing.T) {
	store := &failingListFormataStore{
		MemoryStore: NewMemoryStore(),
		err:         errors.New("list formata streams failed"),
	}
	seedPlatformAdminTaxonomy(t, store.MemoryStore)
	server := newStreamsAdminServer(t, store)

	req := httptest.NewRequest(http.MethodGet, "/admin/streams", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.handleAdminStreams(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

func TestHandleAdminStreamsFormataAuthErrorStillRenders(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := newStreamsAdminServer(t, store)
	server.authorizer = fakeAuthorizer{
		accessDecide: func(user *AccountUser, resourceKind, resourceID string, resourceAttr map[string]interface{}, action string) (bool, error) {
			if resourceKind == cerbosResourceFormataBuilder {
				return false, errors.New("cerbos unavailable")
			}
			return fakeCanAccessDecision(user, resourceKind, resourceAttr, action), nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/streams", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.handleAdminStreams(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="platform-admin-streams"`) {
		t.Fatalf("expected streams catalog despite formata auth error, got: %s", body)
	}
	if strings.Contains(body, "Create a stream") {
		t.Fatalf("Create a stream must be hidden when formata auth check errors, got: %s", body)
	}
}

func TestHandleAdminStreamsTemplateErrors(t *testing.T) {
	broken := template.Must(template.New("broken").Parse(`{{define "other"}}x{{end}}`))

	t.Run("full page", func(t *testing.T) {
		store := NewMemoryStore()
		seedPlatformAdminTaxonomy(t, store)
		server := newStreamsAdminServer(t, store)
		server.tmpl = broken

		req := httptest.NewRequest(http.MethodGet, "/admin/streams", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()
		server.handleAdminStreams(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("htmx console", func(t *testing.T) {
		store := NewMemoryStore()
		seedPlatformAdminTaxonomy(t, store)
		server := newStreamsAdminServer(t, store)
		server.tmpl = broken

		req := httptest.NewRequest(http.MethodGet, "/admin/streams", nil)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Target", "admin-console")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()
		server.handleAdminStreams(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}
