package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type publicCatalogFailingStore struct {
	*MemoryStore
	failListOrganizations bool
	failListRolesByOrg    bool
}

func (s *publicCatalogFailingStore) ListOrganizations(ctx context.Context) ([]Organization, error) {
	if s.failListOrganizations {
		return nil, errors.New("list organizations failed")
	}
	return s.MemoryStore.ListOrganizations(ctx)
}

func (s *publicCatalogFailingStore) ListRolesByOrg(ctx context.Context, orgSlug string) ([]Role, error) {
	if s.failListRolesByOrg {
		return nil, errors.New("list roles failed")
	}
	return s.MemoryStore.ListRolesByOrg(ctx, orgSlug)
}

func TestHandlePublicCatalog(t *testing.T) {
	store := NewMemoryStore()

	acmeOrg, err := store.CreateOrganization(t.Context(), Organization{
		Name:      "Acme Org",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateOrganization(acme): %v", err)
	}
	betaOrg, err := store.CreateOrganization(t.Context(), Organization{
		Name:      "Beta Org",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateOrganization(beta): %v", err)
	}

	if _, err := store.CreateRole(t.Context(), Role{
		OrgID:     acmeOrg.ID,
		OrgSlug:   acmeOrg.Slug,
		Name:      "Inspector",
		Color:     "#f0f3ea",
		Border:    "#d9e0d0",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateRole(acme/inspector): %v", err)
	}
	if _, err := store.CreateRole(t.Context(), Role{
		OrgID:     betaOrg.ID,
		OrgSlug:   betaOrg.Slug,
		Name:      "Assembler",
		Color:     "#f8efe0",
		Border:    "#e9d4a8",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateRole(beta/assembler): %v", err)
	}
	if _, err := store.CreateRole(t.Context(), Role{
		OrgID:     acmeOrg.ID,
		OrgSlug:   acmeOrg.Slug,
		Name:      "Org Admin",
		Color:     "#000000",
		Border:    "#ffffff",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateRole(acme/org-admin): %v", err)
	}

	server := &Server{store: store}
	req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	rec := httptest.NewRecorder()
	server.handlePublicCatalog(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want %q", got, "application/json")
	}

	var got PublicCatalogResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	want := PublicCatalogResponse{
		Organizations: []PublicCatalogOrganization{
			{Name: "Acme Org", Slug: "acme-org"},
			{Name: "Beta Org", Slug: "beta-org"},
		},
		Roles: []PublicCatalogRole{
			{OrgSlug: "acme-org", Name: "Inspector", Slug: "inspector", Color: "#f0f3ea", Border: "#d9e0d0"},
			{OrgSlug: "beta-org", Name: "Assembler", Slug: "assembler", Color: "#f8efe0", Border: "#e9d4a8"},
		},
	}
	if len(got.Organizations) != len(want.Organizations) {
		t.Fatalf("organizations len = %d, want %d", len(got.Organizations), len(want.Organizations))
	}
	for i, item := range want.Organizations {
		if got.Organizations[i] != item {
			t.Fatalf("organizations[%d] = %#v, want %#v", i, got.Organizations[i], item)
		}
	}
	if len(got.Roles) != len(want.Roles) {
		t.Fatalf("roles len = %d, want %d", len(got.Roles), len(want.Roles))
	}
	for i, item := range want.Roles {
		if got.Roles[i] != item {
			t.Fatalf("roles[%d] = %#v, want %#v", i, got.Roles[i], item)
		}
	}
	for _, role := range got.Roles {
		if role.Slug == "org-admin" {
			t.Fatalf("unexpected org-admin role in response: %#v", role)
		}
	}
}

func TestHandlePublicCatalogMethodNotAllowed(t *testing.T) {
	server := &Server{store: NewMemoryStore()}
	req := httptest.NewRequest(http.MethodPost, "/api/catalog", nil)
	rec := httptest.NewRecorder()

	server.handlePublicCatalog(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandlePublicCatalogStoreErrors(t *testing.T) {
	t.Run("list organizations", func(t *testing.T) {
		store := &publicCatalogFailingStore{
			MemoryStore:           NewMemoryStore(),
			failListOrganizations: true,
			failListRolesByOrg:    false,
		}
		server := &Server{store: store}
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		rec := httptest.NewRecorder()

		server.handlePublicCatalog(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("list roles by org", func(t *testing.T) {
		store := &publicCatalogFailingStore{
			MemoryStore:        NewMemoryStore(),
			failListRolesByOrg: true,
		}
		if _, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme"}); err != nil {
			t.Fatalf("CreateOrganization: %v", err)
		}
		server := &Server{store: store}
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		rec := httptest.NewRecorder()

		server.handlePublicCatalog(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}
