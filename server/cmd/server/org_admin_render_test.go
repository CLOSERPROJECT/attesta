package main

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoadOrgAdminStateErrorBranches(t *testing.T) {
	adminOrgID := stableOrgObjectID("acme")
	admin := &AccountUser{
		ID:        stableIdentityUserObjectID("owner"),
		Email:     "owner@example.com",
		OrgID:     &adminOrgID,
		OrgSlug:   "acme",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
	}

	t.Run("identity missing", func(t *testing.T) {
		server := &Server{}
		if _, _, _, _, err := server.loadOrgAdminState(t.Context(), admin, "acme"); !errors.Is(err, ErrIdentityNotFound) {
			t.Fatalf("loadOrgAdminState error = %v, want %v", err, ErrIdentityNotFound)
		}
	})

	t.Run("organization missing", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					return nil, ErrIdentityNotFound
				},
			},
		}
		if _, _, _, _, err := server.loadOrgAdminState(t.Context(), admin, "acme"); !errors.Is(err, ErrIdentityNotFound) {
			t.Fatalf("loadOrgAdminState error = %v, want %v", err, ErrIdentityNotFound)
		}
	})

	t.Run("list users error", func(t *testing.T) {
		boom := errors.New("boom")
		server := &Server{
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					return &IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme"}, nil
				},
				listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
					return nil, boom
				},
			},
		}
		if _, _, _, _, err := server.loadOrgAdminState(t.Context(), admin, "acme"); !errors.Is(err, boom) {
			t.Fatalf("loadOrgAdminState error = %v, want %v", err, boom)
		}
	})
}

func TestRenderOrgAdminWithErrorsBranches(t *testing.T) {
	now := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)
	adminOrgID := stableOrgObjectID("acme")
	admin := &AccountUser{
		ID:        stableIdentityUserObjectID("owner"),
		Email:     "owner@example.com",
		OrgID:     &adminOrgID,
		OrgSlug:   "acme",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
	}

	t.Run("setup branch renders without org context", func(t *testing.T) {
		server := &Server{tmpl: testTemplates(), now: func() time.Time { return now }}
		rec := httptest.NewRecorder()

		server.renderOrgAdminWithErrors(rec, &AccountUser{Email: "owner@example.com", RoleSlugs: []string{"org-admin"}, Status: "active"}, "", "/invite/token", OrgAdminErrors{
			Invite: " invite failed ",
		})

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "/invite/token") || !strings.Contains(body, "invite failed") {
			t.Fatalf("body = %q", body)
		}
	})

	t.Run("organization not found returns 404", func(t *testing.T) {
		server := &Server{
			tmpl: testTemplates(),
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					return nil, ErrIdentityNotFound
				},
			},
			now: func() time.Time { return now },
		}
		rec := httptest.NewRecorder()

		server.renderOrgAdminWithErrors(rec, admin, "acme", "", OrgAdminErrors{})

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("loaded state renders org admin page", func(t *testing.T) {
		server := &Server{
			tmpl: testTemplates(),
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					return &IdentityOrg{
						ID:   "team-1",
						Slug: "acme",
						Name: "Acme",
						Roles: []IdentityRole{
							{Slug: "qa-reviewer", Name: "QA Reviewer", Color: "#123", Border: "#456"},
						},
					}, nil
				},
				listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
					return []IdentityUser{{ID: "user-1", Email: "member@example.com", OrgSlug: "acme", Labels: []string{encodeIdentityRoleLabel("qa-reviewer")}, Status: "active"}}, nil
				},
				listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
					return []IdentityMembership{{Email: "invitee@example.com", Confirmed: false, InvitedAt: now}}, nil
				},
			},
			now: func() time.Time { return now },
		}
		rec := httptest.NewRecorder()

		server.renderOrgAdminWithErrors(rec, admin, "acme", "/invite/token", OrgAdminErrors{Users: " user error "})

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "ORG_ADMIN acme") || !strings.Contains(body, "/invite/token") || !strings.Contains(body, "user error") {
			t.Fatalf("body = %q", body)
		}
	})

	t.Run("template error returns 500", func(t *testing.T) {
		tmpl := template.Must(template.New("broken").Parse(`{{define "org_admin.html"}}{{template "missing" .}}{{end}}`))
		server := &Server{tmpl: tmpl, now: func() time.Time { return now }}
		rec := httptest.NewRecorder()

		server.renderOrgAdminWithErrors(rec, &AccountUser{Email: "owner@example.com"}, "", "", OrgAdminErrors{})

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}
