package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleOrganizationLogo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					if slug != "acme" {
						t.Fatalf("slug = %q, want acme", slug)
					}
					return &IdentityOrg{Slug: "acme", LogoFileID: "logo-1"}, nil
				},
				getOrganizationLogoFunc: func(ctx context.Context, fileID string) (IdentityFile, error) {
					if fileID != "logo-1" {
						t.Fatalf("fileID = %q, want logo-1", fileID)
					}
					return IdentityFile{
						ID:          "logo-1",
						Filename:    "logo.png",
						ContentType: "image/png",
						Data:        []byte{0x89, 'P', 'N', 'G'},
					}, nil
				},
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/organization/logo/acme", nil)
		rec := httptest.NewRecorder()
		server.handleOrganizationLogo(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if got := rec.Header().Get("Content-Type"); got != "image/png" {
			t.Fatalf("content-type = %q, want image/png", got)
		}
		if body := rec.Body.Bytes(); len(body) == 0 {
			t.Fatal("expected logo body")
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		server := &Server{}
		req := httptest.NewRequest(http.MethodPost, "/organization/logo/acme", nil)
		rec := httptest.NewRecorder()
		server.handleOrganizationLogo(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("not found when org has no logo", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					return &IdentityOrg{Slug: slug}, nil
				},
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/organization/logo/acme", nil)
		rec := httptest.NewRecorder()
		server.handleOrganizationLogo(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("not found when logo download fails", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					return &IdentityOrg{Slug: slug, LogoFileID: "logo-1"}, nil
				},
				getOrganizationLogoFunc: func(ctx context.Context, fileID string) (IdentityFile, error) {
					return IdentityFile{}, ErrIdentityNotFound
				},
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/organization/logo/acme", nil)
		rec := httptest.NewRecorder()
		server.handleOrganizationLogo(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
}

func TestOrganizationLogoURLMap(t *testing.T) {
	t.Run("maps only organizations with logos", func(t *testing.T) {
		logos := organizationLogoURLMap(context.Background(), &fakeIdentityStore{
			listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
				return []IdentityOrg{
					{Slug: "acme", LogoFileID: "logo-1"},
					{Slug: "beta"},
				}, nil
			},
		})
		if len(logos) != 1 {
			t.Fatalf("logo map len = %d, want 1", len(logos))
		}
		if logos["acme"] != "/organization/logo/acme" {
			t.Fatalf("logo map acme = %q", logos["acme"])
		}
	})

	t.Run("returns empty map on list error", func(t *testing.T) {
		logos := organizationLogoURLMap(context.Background(), &fakeIdentityStore{
			listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
				return nil, errors.New("boom")
			},
		})
		if len(logos) != 0 {
			t.Fatalf("logo map len = %d, want 0", len(logos))
		}
	})
}
