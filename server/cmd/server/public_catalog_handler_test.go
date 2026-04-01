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

type publicCatalogIdentity struct {
	fakeIdentityStore
	failListOrganizations bool
}

type failingCatalogStore struct {
	*MemoryStore
	err error
}

func (s *publicCatalogIdentity) ListOrganizations(ctx context.Context) ([]IdentityOrg, error) {
	if s.failListOrganizations {
		return nil, errors.New("list organizations failed")
	}
	return s.fakeIdentityStore.ListOrganizations(ctx)
}

func (s *failingCatalogStore) LoadFormataBuilderStream(ctx context.Context) (*FormataBuilderStream, error) {
	return nil, s.err
}

func catalogServer(now time.Time, identity *publicCatalogIdentity) *Server {
	return &Server{
		authorizer:  fakeAuthorizer{},
		store:       NewMemoryStore(),
		identity:    identity,
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
}

func catalogAuthIdentity(now time.Time, admin bool) *publicCatalogIdentity {
	labels := []string{encodeIdentityRoleLabel("inspector")}
	if admin {
		labels = []string{identityOrgAdminLabel}
	}
	return &publicCatalogIdentity{
		fakeIdentityStore: fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "org-admin@example.com", OrgSlug: "acme-org", Labels: labels, IsOrgAdmin: admin, Status: "active"}, nil
			},
		},
	}
}

func TestHandlePublicCatalog(t *testing.T) {
	now := time.Now().UTC()
	identity := catalogAuthIdentity(now, true)
	identity.listOrganizationsFunc = func(ctx context.Context) ([]IdentityOrg, error) {
		return []IdentityOrg{
			{
				ID:   "team-acme",
				Slug: "acme-org",
				Name: "Acme Org",
				Roles: []IdentityRole{
					{Slug: "inspector", Name: "Inspector", Color: "#f0f3ea", Border: "#d9e0d0"},
					{Slug: "org-admin", Name: "Org Admin", Color: "#000000", Border: "#ffffff"},
				},
			},
			{
				ID:   "team-beta",
				Slug: "beta-org",
				Name: "Beta Org",
				Roles: []IdentityRole{
					{Slug: "assembler", Name: "Assembler", Color: "#f8efe0", Border: "#e9d4a8"},
				},
			},
		}, nil
	}

	store := NewMemoryStore()
	if _, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Stream:    `{"stream":"from-db"}`,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}
	server := catalogServer(now, identity)
	server.store = store

	req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()
	server.handlePublicCatalog(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got PublicCatalogResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Organizations) != 2 || len(got.Roles) != 2 {
		t.Fatalf("response = %#v", got)
	}
	for _, role := range got.Roles {
		if role.Slug == "org-admin" {
			t.Fatalf("unexpected org-admin role in response: %#v", role)
		}
	}
	if got.Stream != `{"stream":"from-db"}` {
		t.Fatalf("stream = %q", got.Stream)
	}
}

func TestHandlePublicCatalogMethodNotAllowed(t *testing.T) {
	server := &Server{
		authorizer: fakeAuthorizer{}, store: NewMemoryStore(), identity: &fakeIdentityStore{}}
	req := httptest.NewRequest(http.MethodPost, "/api/catalog", nil)
	rec := httptest.NewRecorder()
	server.handlePublicCatalog(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandlePublicCatalogAuthz(t *testing.T) {
	now := time.Now().UTC()

	t.Run("unauthenticated", func(t *testing.T) {
		server := catalogServer(now, &publicCatalogIdentity{})
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		rec := httptest.NewRecorder()
		server.handlePublicCatalog(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("forbidden for non admin", func(t *testing.T) {
		server := catalogServer(now, catalogAuthIdentity(now, false))
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handlePublicCatalog(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("org admin allowed", func(t *testing.T) {
		identity := catalogAuthIdentity(now, true)
		identity.listOrganizationsFunc = func(ctx context.Context) ([]IdentityOrg, error) { return nil, nil }
		server := catalogServer(now, identity)
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handlePublicCatalog(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestHandlePublicCatalogStoreErrors(t *testing.T) {
	now := time.Now().UTC()

	t.Run("list organizations", func(t *testing.T) {
		identity := catalogAuthIdentity(now, true)
		identity.failListOrganizations = true
		server := catalogServer(now, identity)
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handlePublicCatalog(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("load stream", func(t *testing.T) {
		identity := catalogAuthIdentity(now, true)
		identity.listOrganizationsFunc = func(ctx context.Context) ([]IdentityOrg, error) { return nil, nil }
		server := catalogServer(now, identity)
		server.store = &failingCatalogStore{MemoryStore: NewMemoryStore(), err: errors.New("boom")}
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handlePublicCatalog(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("not configured", func(t *testing.T) {
		server := &Server{
			authorizer:  fakeAuthorizer{},
			identity:    catalogAuthIdentity(now, true),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handlePublicCatalog(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}
