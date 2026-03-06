package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type publicCatalogFailingStore struct {
	*MemoryStore
	failListOrganizations bool
	failListRolesByOrg    bool
	failLoadStream        bool
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

func (s *publicCatalogFailingStore) LoadFormataBuilderStream(ctx context.Context) (*FormataBuilderStream, error) {
	if s.failLoadStream {
		return nil, errors.New("load stream failed")
	}
	return s.MemoryStore.LoadFormataBuilderStream(ctx)
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
	orgID := acmeOrg.ID
	adminUser, err := store.CreateUser(t.Context(), AccountUser{
		OrgID:     &orgID,
		OrgSlug:   acmeOrg.Slug,
		Email:     "org-admin-catalog@example.com",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser(org-admin): %v", err)
	}
	if _, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Stream:               `{"stream":"from-db"}`,
		UpdatedAt:            time.Now().UTC(),
		UpdatedByUserMongoID: adminUser.ID,
	}); err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}
	sessionID := createSessionForTestUser(t, store, adminUser)

	server := &Server{store: store, enforceAuth: true, now: time.Now}
	req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
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
		Stream: `{"stream":"from-db"}`,
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

func TestHandlePublicCatalogAuthz(t *testing.T) {
	store := NewMemoryStore()
	server := &Server{store: store, enforceAuth: true, now: time.Now}

	t.Run("unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		rec := httptest.NewRecorder()
		server.handlePublicCatalog(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("forbidden for non admin", func(t *testing.T) {
		user, err := store.CreateUser(t.Context(), AccountUser{
			Email:     "plain-user-catalog@example.com",
			RoleSlugs: []string{"inspector"},
			Status:    "active",
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}
		sessionID := createSessionForTestUser(t, store, user)
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handlePublicCatalog(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("platform admin allowed", func(t *testing.T) {
		admin, err := store.CreateUser(t.Context(), AccountUser{
			Email:           "platform-catalog@example.com",
			IsPlatformAdmin: true,
			Status:          "active",
			CreatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}
		sessionID := createSessionForTestUser(t, store, admin)
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handlePublicCatalog(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestHandlePublicCatalogStoreErrors(t *testing.T) {
	t.Run("list organizations", func(t *testing.T) {
		store := &publicCatalogFailingStore{
			MemoryStore:           NewMemoryStore(),
			failListOrganizations: true,
			failListRolesByOrg:    false,
		}
		admin, err := store.CreateUser(t.Context(), AccountUser{
			Email:           "platform-list-orgs-catalog@example.com",
			IsPlatformAdmin: true,
			Status:          "active",
			CreatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		sessionID := createSessionForTestUser(t, store.MemoryStore, admin)
		server := &Server{store: store, enforceAuth: true, now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
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
		admin, err := store.CreateUser(t.Context(), AccountUser{
			Email:           "platform-list-roles-catalog@example.com",
			IsPlatformAdmin: true,
			Status:          "active",
			CreatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		sessionID := createSessionForTestUser(t, store.MemoryStore, admin)
		server := &Server{store: store, enforceAuth: true, now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()

		server.handlePublicCatalog(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("load stream", func(t *testing.T) {
		base := NewMemoryStore()
		store := &publicCatalogFailingStore{MemoryStore: base}
		admin, err := store.CreateUser(t.Context(), AccountUser{
			Email:           "platform-stream-catalog@example.com",
			IsPlatformAdmin: true,
			Status:          "active",
			CreatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		if _, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme"}); err != nil {
			t.Fatalf("CreateOrganization: %v", err)
		}
		if _, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Stream:               "stream-v1",
			UpdatedAt:            time.Now().UTC(),
			UpdatedByUserMongoID: admin.ID,
		}); err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}
		sessionID := createSessionForTestUser(t, store.MemoryStore, admin)
		server := &Server{store: store, enforceAuth: true, now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()

		server.handlePublicCatalog(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var got PublicCatalogResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Stream != "stream-v1" {
			t.Fatalf("stream = %q, want %q", got.Stream, "stream-v1")
		}
	})

	t.Run("invalid user role cookie id still unauthorized", func(t *testing.T) {
		server := &Server{store: NewMemoryStore(), enforceAuth: true, now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: primitive.NewObjectID().Hex()})
		rec := httptest.NewRecorder()
		server.handlePublicCatalog(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("load stream failure", func(t *testing.T) {
		base := NewMemoryStore()
		store := &publicCatalogFailingStore{MemoryStore: base, failLoadStream: true}
		admin, err := store.CreateUser(t.Context(), AccountUser{
			Email:           "platform-stream-error-catalog@example.com",
			IsPlatformAdmin: true,
			Status:          "active",
			CreatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		if _, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme"}); err != nil {
			t.Fatalf("CreateOrganization: %v", err)
		}
		sessionID := createSessionForTestUser(t, store.MemoryStore, admin)
		server := &Server{store: store, enforceAuth: true, now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()

		server.handlePublicCatalog(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}
