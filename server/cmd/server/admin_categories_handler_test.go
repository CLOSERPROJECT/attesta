package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type failingListCategoriesStore struct {
	*MemoryStore
	err error
}

func (s *failingListCategoriesStore) ListCategories(_ context.Context) ([]Category, error) {
	return nil, s.err
}

type failingListSubCategoriesStore struct {
	*MemoryStore
	err error
}

func (s *failingListSubCategoriesStore) ListSubCategories(_ context.Context, _ string) ([]SubCategory, error) {
	return nil, s.err
}

func seedPlatformAdminTaxonomy(t *testing.T, store Store) {
	t.Helper()
	err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
	}, []SubCategory{
		{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1, Description: "PO management"},
		{CategorySlug: "supply-chain", Slug: "order-fulfillment", Name: "Order Fulfillment", Icon: "order-fulfillment", SortOrder: 2, Description: "Ship orders"},
	})
	if err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}
}

func newCategoriesAdminServer(t *testing.T, store Store) *Server {
	t.Helper()
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "workflow.yaml")
	if err := os.WriteFile(path, []byte(minimalCategorizedWorkflowYAML("")), 0o644); err != nil {
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

func postAdminCategories(t *testing.T, server *Server, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/admin/categories", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()
	server.handleAdminCategories(rec, req)
	return rec
}

func TestHandleAdminCategoriesCreateGroup(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := newCategoriesAdminServer(t, store)

	form := url.Values{}
	form.Set("intent", "create")
	form.Set("name", "New Group")
	form.Set("icon", "weee")
	rec := postAdminCategories(t, server, form)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "New Group") || !strings.Contains(body, "Category group created") {
		t.Fatalf("expected create confirmation and name in body, got: %s", body)
	}
	got, err := store.GetCategoryBySlug(t.Context(), "new-group")
	if err != nil {
		t.Fatalf("GetCategoryBySlug: %v", err)
	}
	if got.Name != "New Group" || got.Icon != "weee" {
		t.Fatalf("stored category = %#v", got)
	}
}

func TestHandleAdminCategoriesUpdateGroup(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := newCategoriesAdminServer(t, store)

	form := url.Values{}
	form.Set("intent", "update")
	form.Set("slug", "supply-chain")
	form.Set("name", "Updated Chain")
	form.Set("icon", "weee")
	rec := postAdminCategories(t, server, form)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Updated Chain") || !strings.Contains(body, "Category group updated") {
		t.Fatalf("expected update confirmation and name in body, got: %s", body)
	}
	got, err := store.GetCategoryBySlug(t.Context(), "supply-chain")
	if err != nil {
		t.Fatalf("GetCategoryBySlug: %v", err)
	}
	if got.Name != "Updated Chain" || got.Icon != "weee" {
		t.Fatalf("stored category = %#v", got)
	}
}

func TestHandleAdminCategoriesDeleteGroup(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	if _, err := store.CreateCategory(t.Context(), Category{
		Slug: "empty-group", Name: "Empty Group", Icon: "weee",
	}); err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}
	server := newCategoriesAdminServer(t, store)

	form := url.Values{}
	form.Set("intent", "delete")
	form.Set("slug", "empty-group")
	rec := postAdminCategories(t, server, form)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Category group deleted") {
		t.Fatalf("expected delete confirmation, got: %s", rec.Body.String())
	}
	if _, err := store.GetCategoryBySlug(t.Context(), "empty-group"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("expected deleted category, err = %v", err)
	}
}

func TestHandleAdminCategoriesReorderGroup(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	if _, err := store.CreateCategory(t.Context(), Category{
		Slug: "second-group", Name: "Second Group", Icon: "weee",
	}); err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}
	server := newCategoriesAdminServer(t, store)

	form := url.Values{}
	form.Set("intent", "reorder")
	form.Set("slug", "second-group")
	form.Set("direction", "up")
	rec := postAdminCategories(t, server, form)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Category group reordered") {
		t.Fatalf("expected reorder confirmation, got: %s", rec.Body.String())
	}
	list, err := store.ListCategories(t.Context())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(list) != 2 || list[0].Slug != "second-group" || list[1].Slug != "supply-chain" {
		t.Fatalf("order = %v, %v; want second-group then supply-chain", list[0].Slug, list[1].Slug)
	}
}

func TestHandleAdminCategoriesCreateGroupDuplicateSlug(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := newCategoriesAdminServer(t, store)

	form := url.Values{}
	form.Set("intent", "create")
	form.Set("name", "Supply Chain")
	form.Set("icon", "weee")
	rec := postAdminCategories(t, server, form)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "taxonomy slug already exists") {
		t.Fatalf("expected duplicate slug error, got: %s", body)
	}
	if !strings.Contains(body, `id="categories-editor-form"`) {
		t.Fatalf("expected create form to remain open, got: %s", body)
	}
}

func TestHandleAdminCategoriesPostNonAdminForbidden(t *testing.T) {
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

	form := url.Values{}
	form.Set("intent", "create")
	form.Set("name", "New Group")
	form.Set("icon", "weee")
	req := httptest.NewRequest(http.MethodPost, "/admin/categories", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "member-session"})
	rec := httptest.NewRecorder()

	server.handleAdminCategories(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestHandleAdminCategoriesCreateGroupHTMXPanelPartial(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := newCategoriesAdminServer(t, store)

	form := url.Values{}
	form.Set("intent", "create")
	form.Set("name", "Panel Group")
	form.Set("icon", "weee")
	req := httptest.NewRequest(http.MethodPost, "/admin/categories", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "platform-admin-categories")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleAdminCategories(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="platform-admin-categories"`) || !strings.Contains(body, "Panel Group") {
		t.Fatalf("expected HTMX panel partial with new group, got: %s", body)
	}
	if strings.Contains(body, "<html") || strings.Contains(body, `class="topbar"`) {
		t.Fatalf("HTMX panel must not include full layout, got: %s", body)
	}
}

func TestWantsCategoriesPanelPartial(t *testing.T) {
	t.Parallel()

	full := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	if wantsCategoriesPanelPartial(full) {
		t.Fatal("non-HTMX must be false")
	}

	console := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	console.Header.Set("HX-Request", "true")
	console.Header.Set("HX-Target", "admin-console")
	if wantsCategoriesPanelPartial(console) {
		t.Fatal("admin-console HTMX must not request categories panel partial")
	}

	panel := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	panel.Header.Set("HX-Request", "true")
	panel.Header.Set("HX-Target", "platform-admin-categories")
	if !wantsCategoriesPanelPartial(panel) {
		t.Fatal("categories panel HTMX must request panel partial")
	}
}

func TestHandleAdminCategoriesNonAdminForbidden(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "member-session"})
	rec := httptest.NewRecorder()

	server.handleAdminCategories(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestHandleAdminCategoriesStoreError(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	server := &Server{
		authorizer: fakeAuthorizer{},
		store: &failingListCategoriesStore{
			MemoryStore: NewMemoryStore(),
			err:         errors.New("list categories failed"),
		},
		identity:    &fakeIdentityStore{},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleAdminCategories(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandleAdminCategoriesRendersEditorPanel(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := newCategoriesAdminServer(t, store)

	req := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleAdminCategories(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`id="platform-admin-categories"`,
		"Manage stream discovery taxonomy",
		"1 groups · 2 categories",
		"Supply Chain",
		"/static/taxonomy/batch-traceability.svg",
		"Procurement",
		"/static/taxonomy/procurement-workflow.svg",
		"Order Fulfillment",
		"/static/taxonomy/order-fulfillment.svg",
		"PO management",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got: %s", want, body)
		}
	}
}

func TestHandleAdminCategoriesNewGroupForm(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := newCategoriesAdminServer(t, store)

	req := httptest.NewRequest(http.MethodGet, "/admin/categories?new=group", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleAdminCategories(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`name="intent"`,
		`value="create"`,
		`id="categories-editor-form"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got: %s", want, body)
		}
	}
	if strings.Contains(body, `name="description"`) || strings.Contains(body, "categories-editor-description") {
		t.Fatalf("group create form must not include description field, got: %s", body)
	}
}

func TestHandleAdminCategoriesHTMXPanelPartial(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := newCategoriesAdminServer(t, store)

	req := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "platform-admin-categories")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleAdminCategories(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="platform-admin-categories"`) {
		t.Fatalf("expected categories panel partial, got: %s", body)
	}
	if strings.Contains(body, `id="admin-console"`) || strings.Contains(body, "<html") || strings.Contains(body, `class="topbar"`) {
		t.Fatalf("panel HTMX must not include layout or console chrome, got: %s", body)
	}
}
