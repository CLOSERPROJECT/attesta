package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

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

func TestHandleAdminCategoriesRendersTaxonomyTree(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)

	server := &Server{
		authorizer:  fakeAuthorizer{},
		store:       store,
		identity:    &fakeIdentityStore{},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleAdminCategories(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"PLATFORM_ADMIN categories",
		"Supply Chain",
		"/static/taxonomy/batch-traceability.svg",
		"Procurement",
		"/static/taxonomy/procurement-workflow.svg",
		"Order Fulfillment",
		"/static/taxonomy/order-fulfillment.svg",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got: %s", want, body)
		}
	}
}
