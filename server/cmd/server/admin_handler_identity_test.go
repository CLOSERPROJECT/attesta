package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPlatformAdminIdentitySession(t *testing.T) {
	t.Run("missing identity", func(t *testing.T) {
		t.Setenv("ADMIN_EMAIL", "admin@example.com")
		t.Setenv("ADMIN_PASSWORD", "change-me")
		server := &Server{}
		if _, err := server.platformAdminIdentitySession(context.Background()); !errors.Is(err, ErrIdentityUnauthorized) {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("login failure", func(t *testing.T) {
		t.Setenv("ADMIN_EMAIL", "admin@example.com")
		t.Setenv("ADMIN_PASSWORD", "change-me")
		server := &Server{
			identity: &fakeIdentityStore{
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return IdentitySession{}, ErrIdentityUnauthorized
				},
			},
		}
		if _, err := server.platformAdminIdentitySession(context.Background()); !errors.Is(err, ErrIdentityUnauthorized) {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestBootstrapPlatformAdminIdentity(t *testing.T) {
	t.Run("no identity or creds is a no-op", func(t *testing.T) {
		server := &Server{}
		if err := server.bootstrapPlatformAdminIdentity(context.Background()); err != nil {
			t.Fatalf("error = %v", err)
		}

		t.Setenv("ADMIN_EMAIL", "")
		t.Setenv("ADMIN_PASSWORD", "")
		server = &Server{identity: &fakeIdentityStore{
			ensurePlatformAdminAccountFunc: func(ctx context.Context, email, password string) error {
				t.Fatal("did not expect bootstrap call")
				return nil
			},
		}}
		if err := server.bootstrapPlatformAdminIdentity(context.Background()); err != nil {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("delegates to identity store", func(t *testing.T) {
		t.Setenv("ADMIN_EMAIL", "ADMIN@example.com")
		t.Setenv("ADMIN_PASSWORD", "change-me")
		var gotEmail string
		var gotPassword string
		server := &Server{
			identity: &fakeIdentityStore{
				ensurePlatformAdminAccountFunc: func(ctx context.Context, email, password string) error {
					gotEmail = email
					gotPassword = password
					return nil
				},
			},
		}
		if err := server.bootstrapPlatformAdminIdentity(context.Background()); err != nil {
			t.Fatalf("error = %v", err)
		}
		if gotEmail != "admin@example.com" || gotPassword != "change-me" {
			t.Fatalf("bootstrap args = %q %q", gotEmail, gotPassword)
		}
	})

	t.Run("returns identity bootstrap error", func(t *testing.T) {
		t.Setenv("ADMIN_EMAIL", "admin@example.com")
		t.Setenv("ADMIN_PASSWORD", "change-me")
		server := &Server{
			identity: &fakeIdentityStore{
				ensurePlatformAdminAccountFunc: func(ctx context.Context, email, password string) error {
					return errors.New("boom")
				},
			},
		}
		if err := server.bootstrapPlatformAdminIdentity(context.Background()); err == nil || err.Error() != "boom" {
			t.Fatalf("error = %v, want boom", err)
		}
	})
}

func TestPlatformOrganizationsAndRenderPlatformAdmin(t *testing.T) {
	now := time.Now().UTC()

	t.Run("platform organizations handles nil and errors", func(t *testing.T) {
		server := &Server{}
		if got := server.platformOrganizations(context.Background()); got != nil {
			t.Fatalf("got = %#v, want nil", got)
		}

		server.identity = &fakeIdentityStore{
			listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
				return nil, errors.New("boom")
			},
		}
		if got := server.platformOrganizations(context.Background()); got != nil {
			t.Fatalf("got = %#v, want nil", got)
		}
	})

	t.Run("platform organizations sorts values", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
					return []IdentityOrg{
						{ID: "team-2", Slug: "zeta", Name: "Zeta Org"},
						{ID: "team-1", Slug: "acme", Name: "Acme Org"},
					}, nil
				},
			},
		}
		orgs := server.platformOrganizations(context.Background())
		if len(orgs) != 2 || orgs[0].Slug != "acme" || orgs[1].Slug != "zeta" {
			t.Fatalf("orgs = %#v", orgs)
		}
	})

	t.Run("render platform admin", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
					return []IdentityOrg{{ID: "team-1", Slug: "acme", Name: "Acme Org"}}, nil
				},
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		rec := httptest.NewRecorder()
		server.renderPlatformAdmin(rec, &AccountUser{Email: "admin@example.com", IsPlatformAdmin: true}, "invite sent", PlatformAdminErrors{})
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "PLATFORM_ADMIN ORGS 1 invite sent") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})
}

func TestEnsurePlatformAdminOwnsOrganizationErrorBranches(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()

	t.Run("current user lookup failure closes session", func(t *testing.T) {
		var deletedSecret string
		server := &Server{
			identity: &fakeIdentityStore{
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{}, errors.New("boom")
				},
				deleteSessionFunc: func(ctx context.Context, sessionSecret string) error {
					deletedSecret = sessionSecret
					return nil
				},
			},
			now: func() time.Time { return now },
		}
		if _, err := server.ensurePlatformAdminOwnsOrganization(context.Background(), "acme", "http://attesta.local/invite/accept"); err == nil {
			t.Fatal("expected error")
		}
		if deletedSecret != "platform-session" {
			t.Fatalf("deletedSecret = %q", deletedSecret)
		}
	})

	t.Run("membership listing failure closes session", func(t *testing.T) {
		var deletedSecret string
		server := &Server{
			identity: &fakeIdentityStore{
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "platform-user", Email: "admin@example.com", Status: "active"}, nil
				},
				listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
					return nil, errors.New("boom")
				},
				deleteSessionFunc: func(ctx context.Context, sessionSecret string) error {
					deletedSecret = sessionSecret
					return nil
				},
			},
			now: func() time.Time { return now },
		}
		if _, err := server.ensurePlatformAdminOwnsOrganization(context.Background(), "acme", "http://attesta.local/invite/accept"); err == nil {
			t.Fatal("expected error")
		}
		if deletedSecret != "platform-session" {
			t.Fatalf("deletedSecret = %q", deletedSecret)
		}
	})

	t.Run("promotion failure closes session", func(t *testing.T) {
		var deletedSecret string
		server := &Server{
			identity: &fakeIdentityStore{
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "platform-user", Email: "admin@example.com", Status: "active"}, nil
				},
				listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
					return []IdentityMembership{{ID: "membership-admin", Email: "admin@example.com", Confirmed: true, IsOrgAdmin: false}}, nil
				},
				updateOrganizationMembershipAsAdminFunc: func(ctx context.Context, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
					return IdentityMembership{}, errors.New("boom")
				},
				deleteSessionFunc: func(ctx context.Context, sessionSecret string) error {
					deletedSecret = sessionSecret
					return nil
				},
			},
			now: func() time.Time { return now },
		}
		if _, err := server.ensurePlatformAdminOwnsOrganization(context.Background(), "acme", "http://attesta.local/invite/accept"); err == nil {
			t.Fatal("expected error")
		}
		if deletedSecret != "platform-session" {
			t.Fatalf("deletedSecret = %q", deletedSecret)
		}
	})

	t.Run("owner match by email returns session", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "platform-user", Email: "admin@example.com", Status: "active"}, nil
				},
				listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
					return []IdentityMembership{{ID: "membership-admin", Email: "admin@example.com", Confirmed: true, IsOrgAdmin: true}}, nil
				},
			},
			now: func() time.Time { return now },
		}
		session, err := server.ensurePlatformAdminOwnsOrganization(context.Background(), "acme", "http://attesta.local/invite/accept")
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if session == nil || session.Secret != "platform-session" {
			t.Fatalf("session = %#v", session)
		}
	})

	t.Run("add membership by user id failure closes session", func(t *testing.T) {
		var deletedSecret string
		server := &Server{
			identity: &fakeIdentityStore{
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "platform-user", Email: "admin@example.com", Status: "active"}, nil
				},
				listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
					return nil, nil
				},
				addOrganizationUserByIDAsAdminFunc: func(ctx context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
					return IdentityMembership{}, errors.New("boom")
				},
				deleteSessionFunc: func(ctx context.Context, sessionSecret string) error {
					deletedSecret = sessionSecret
					return nil
				},
			},
			now: func() time.Time { return now },
		}
		if _, err := server.ensurePlatformAdminOwnsOrganization(context.Background(), "acme", "http://attesta.local/invite/accept"); err == nil {
			t.Fatal("expected error")
		}
		if deletedSecret != "platform-session" {
			t.Fatalf("deletedSecret = %q", deletedSecret)
		}
	})

	t.Run("missing appwrite identity on current user closes session", func(t *testing.T) {
		var deletedSecret string
		server := &Server{
			identity: &fakeIdentityStore{
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{Status: "active"}, nil
				},
				listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
					return nil, nil
				},
				deleteSessionFunc: func(ctx context.Context, sessionSecret string) error {
					deletedSecret = sessionSecret
					return nil
				},
			},
			now: func() time.Time { return now },
		}
		if _, err := server.ensurePlatformAdminOwnsOrganization(context.Background(), "acme", "http://attesta.local/invite/accept"); !errors.Is(err, ErrIdentityUnauthorized) {
			t.Fatalf("error = %v", err)
		}
		if deletedSecret != "platform-session" {
			t.Fatalf("deletedSecret = %q", deletedSecret)
		}
	})
}

func TestHandleAdminOrgsCreateOrganizationWithPlatformAdmin(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	var createName string
	var createSessionSecret string
	var loginEmail string
	var deletedSecret string

	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				loginEmail = email
				return fakeIdentitySession("platform-session", "user-1", now.Add(time.Hour)), nil
			},
			createOrganizationFunc: func(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
				createSessionSecret = sessionSecret
				createName = name
				return IdentityOrg{ID: "team-1", Slug: "fresh-org", Name: "Fresh Org"}, nil
			},
			deleteSessionFunc: func(ctx context.Context, sessionSecret string) error {
				deletedSecret = sessionSecret
				return nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				return nil, ErrIdentityNotFound
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader("name=Fresh+Org"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleAdminOrgs(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/admin/orgs" {
		t.Fatalf("location = %q", rec.Header().Get("Location"))
	}
	if createName != "Fresh Org" {
		t.Fatalf("createName = %q, want Fresh Org", createName)
	}
	if createSessionSecret != "platform-session" || loginEmail != "admin@example.com" || deletedSecret != "platform-session" {
		t.Fatalf("session wiring create=%q login=%q deleted=%q", createSessionSecret, loginEmail, deletedSecret)
	}
}

func TestHandleAdminOrgsGetAndValidationErrors(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()

	t.Run("get renders organizations", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
					return []IdentityOrg{{ID: "team-1", Slug: "acme", Name: "Acme Org"}}, nil
				},
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, "/admin/orgs", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if !strings.Contains(rec.Body.String(), "PLATFORM_ADMIN ORGS 1") {
			t.Fatalf("body = %q", rec.Body.String())
		}
	})

	t.Run("identity unavailable", func(t *testing.T) {
		server := &Server{tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodGet, "/admin/orgs", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("invalid subpath", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodGet, "/admin/orgs/unknown", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("missing organization name", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "organization name is required") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("duplicate organization slug", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					org := IdentityOrg{ID: "team-1", Slug: "fresh-org", Name: "Fresh Org"}
					return &org, nil
				},
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader("name=Fresh+Org"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "organization slug already exists") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPut, "/admin/orgs", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("platform admin appwrite login fails on create org", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					return nil, ErrIdentityNotFound
				},
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return IdentitySession{}, ErrIdentityUnauthorized
				},
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader("name=Fresh+Org"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to create organization") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})
}

func TestHandleAdminOrgsInviteOrgAdminWithPlatformAdmin(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	var loginEmail string
	var inviteOrgSlug string
	var inviteEmail string
	var inviteRedirect string
	var inviteSessionSecret string
	var deletedSecret string

	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				loginEmail = email
				return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "platform-user", Email: "admin@example.com", Status: "active"}, nil
			},
			listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
				return []IdentityOrg{{ID: "team-1", Slug: "acme", Name: "Acme Org"}}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				if strings.TrimSpace(slug) != "acme" {
					return nil, ErrIdentityNotFound
				}
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}
				return &org, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return []IdentityMembership{{ID: "membership-admin", UserID: "platform-user", Email: "admin@example.com", Confirmed: true, IsOrgAdmin: true}}, nil
			},
			getUserByEmailFunc: func(ctx context.Context, email string) (IdentityUser, error) {
				return IdentityUser{ID: "user-2", Email: email, Status: "active"}, nil
			},
			inviteOrganizationUserFunc: func(ctx context.Context, sessionSecret, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
				inviteSessionSecret = sessionSecret
				inviteOrgSlug = orgSlug
				inviteEmail = email
				inviteRedirect = redirectURL
				if !isOrgAdmin {
					t.Fatal("expected org-admin invite to request owner access")
				}
				if len(roleSlugs) != 0 {
					t.Fatalf("roleSlugs = %#v, want none", roleSlugs)
				}
				return IdentityMembership{ID: "membership-1", Email: email, IsOrgAdmin: true}, nil
			},
			deleteSessionFunc: func(ctx context.Context, sessionSecret string) error {
				deletedSecret = sessionSecret
				return nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	form := url.Values{}
	form.Set("intent", "invite_org_admin")
	form.Set("org_slug", "acme")
	form.Set("email", "owner@example.com")
	req := httptest.NewRequest(http.MethodPost, "http://attesta.local/admin/orgs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleAdminOrgs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if inviteOrgSlug != "acme" || inviteEmail != "owner@example.com" {
		t.Fatalf("invite args = %q %q", inviteOrgSlug, inviteEmail)
	}
	if inviteSessionSecret != "platform-session" || loginEmail != "admin@example.com" || deletedSecret != "platform-session" {
		t.Fatalf("session wiring invite=%q login=%q deleted=%q", inviteSessionSecret, loginEmail, deletedSecret)
	}
	if inviteRedirect != "http://attesta.local/invite/accept" {
		t.Fatalf("invite redirect = %q", inviteRedirect)
	}
	if !strings.Contains(rec.Body.String(), "PLATFORM_ADMIN ORGS 1 invite sent") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHandleAdminOrgsInviteOrgAdminSendsInviteForUnknownEmail(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	var loginEmail string
	var inviteSessionSecret string
	var inviteEmail string
	var inviteRedirect string
	var deletedSecret string

	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				loginEmail = email
				return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "platform-user", Email: "admin@example.com", Status: "active"}, nil
			},
			listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
				return []IdentityOrg{{ID: "team-1", Slug: "acme", Name: "Acme Org"}}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				if strings.TrimSpace(slug) != "acme" {
					return nil, ErrIdentityNotFound
				}
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}
				return &org, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return []IdentityMembership{{ID: "membership-admin", UserID: "platform-user", Email: "admin@example.com", Confirmed: true, IsOrgAdmin: true}}, nil
			},
			getUserByEmailFunc: func(ctx context.Context, email string) (IdentityUser, error) {
				return IdentityUser{}, ErrIdentityNotFound
			},
			inviteOrganizationUserFunc: func(ctx context.Context, sessionSecret, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
				inviteSessionSecret = sessionSecret
				inviteEmail = email
				inviteRedirect = redirectURL
				return IdentityMembership{ID: "membership-2", Email: email, IsOrgAdmin: true}, nil
			},
			deleteSessionFunc: func(ctx context.Context, sessionSecret string) error {
				deletedSecret = sessionSecret
				return nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	form := url.Values{}
	form.Set("intent", "invite_org_admin")
	form.Set("org_slug", "acme")
	form.Set("email", "new-owner@example.com")
	req := httptest.NewRequest(http.MethodPost, "http://attesta.local/admin/orgs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleAdminOrgs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if loginEmail != "admin@example.com" || inviteSessionSecret != "platform-session" || inviteEmail != "new-owner@example.com" {
		t.Fatalf("invite wiring login=%q session=%q email=%q", loginEmail, inviteSessionSecret, inviteEmail)
	}
	if inviteRedirect != "http://attesta.local/invite/accept" || deletedSecret != "platform-session" {
		t.Fatalf("redirect=%q deleted=%q", inviteRedirect, deletedSecret)
	}
	if !strings.Contains(rec.Body.String(), "PLATFORM_ADMIN ORGS 1 invite sent") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHandleAdminOrgsInviteOrgAdminUpdatesExistingMembershipOnly(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	var loginEmail string
	var updateMembershipID string
	var deletedSecret string

	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				loginEmail = email
				return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "platform-user", Email: "admin@example.com", Status: "active"}, nil
			},
			listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
				return []IdentityOrg{{ID: "team-1", Slug: "acme", Name: "Acme Org"}}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}
				return &org, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return []IdentityMembership{
					{ID: "membership-admin", UserID: "platform-user", Email: "admin@example.com", Confirmed: true, IsOrgAdmin: true},
					{ID: "membership-1", Email: "member@example.com", Confirmed: true, IsOrgAdmin: false},
				}, nil
			},
			updateOrganizationMembershipAsAdminFunc: func(ctx context.Context, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
				updateMembershipID = membershipID
				return IdentityMembership{ID: membershipID, Email: "member@example.com", Confirmed: true, IsOrgAdmin: true}, nil
			},
			deleteSessionFunc: func(ctx context.Context, sessionSecret string) error {
				deletedSecret = sessionSecret
				return nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	form := url.Values{}
	form.Set("intent", "invite_org_admin")
	form.Set("org_slug", "acme")
	form.Set("email", "member@example.com")
	req := httptest.NewRequest(http.MethodPost, "http://attesta.local/admin/orgs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleAdminOrgs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if updateMembershipID != "membership-1" {
		t.Fatalf("update membership id = %q", updateMembershipID)
	}
	if loginEmail != "admin@example.com" || deletedSecret != "platform-session" {
		t.Fatalf("login=%q deleted=%q", loginEmail, deletedSecret)
	}
	if !strings.Contains(rec.Body.String(), "PLATFORM_ADMIN ORGS 1 org admin access updated") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHandleAdminOrgsInviteOrgAdminErrors(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()

	t.Run("already org admin", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "platform-user", Email: "admin@example.com", Status: "active"}, nil
				},
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}
					return &org, nil
				},
				listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
					return []IdentityMembership{
						{ID: "membership-admin", UserID: "platform-user", Email: "admin@example.com", Confirmed: true, IsOrgAdmin: true},
						{ID: "membership-1", Email: "member@example.com", Confirmed: true, IsOrgAdmin: true},
					}, nil
				},
				deleteSessionFunc: func(ctx context.Context, sessionSecret string) error { return nil },
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		form := url.Values{}
		form.Set("intent", "invite_org_admin")
		form.Set("org_slug", "acme")
		form.Set("email", "member@example.com")
		req := httptest.NewRequest(http.MethodPost, "http://attesta.local/admin/orgs", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "org admin access already assigned") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("reject cross-org email", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "platform-user", Email: "admin@example.com", Status: "active"}, nil
				},
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}
					return &org, nil
				},
				listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
					return []IdentityMembership{{ID: "membership-admin", UserID: "platform-user", Email: "admin@example.com", Confirmed: true, IsOrgAdmin: true}}, nil
				},
				getUserByEmailFunc: func(ctx context.Context, email string) (IdentityUser, error) {
					return IdentityUser{ID: "user-2", Email: email, OrgSlug: "other-org", Status: "active"}, nil
				},
				deleteSessionFunc: func(ctx context.Context, sessionSecret string) error { return nil },
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		form := url.Values{}
		form.Set("intent", "invite_org_admin")
		form.Set("org_slug", "acme")
		form.Set("email", "owner@example.com")
		req := httptest.NewRequest(http.MethodPost, "http://attesta.local/admin/orgs", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "email already belongs to another organization") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing email", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		form := url.Values{}
		form.Set("intent", "invite_org_admin")
		form.Set("org_slug", "acme")
		req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "email is required") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing organization", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		form := url.Values{}
		form.Set("intent", "invite_org_admin")
		form.Set("email", "owner@example.com")
		req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "organization is required") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("organization not found", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					return nil, ErrIdentityNotFound
				},
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		form := url.Values{}
		form.Set("intent", "invite_org_admin")
		form.Set("org_slug", "acme")
		form.Set("email", "owner@example.com")
		req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "organization not found") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("platform appwrite login fails", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return IdentitySession{}, ErrIdentityUnauthorized
				},
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}
					return &org, nil
				},
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		form := url.Values{}
		form.Set("intent", "invite_org_admin")
		form.Set("org_slug", "acme")
		form.Set("email", "owner@example.com")
		req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to create invite") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("existing user lookup failure", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "platform-user", Email: "admin@example.com", Status: "active"}, nil
				},
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}
					return &org, nil
				},
				listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
					return []IdentityMembership{{ID: "membership-admin", UserID: "platform-user", Email: "admin@example.com", Confirmed: true, IsOrgAdmin: true}}, nil
				},
				getUserByEmailFunc: func(ctx context.Context, email string) (IdentityUser, error) {
					return IdentityUser{}, errors.New("boom")
				},
				deleteSessionFunc: func(ctx context.Context, sessionSecret string) error { return nil },
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		form := url.Values{}
		form.Set("intent", "invite_org_admin")
		form.Set("org_slug", "acme")
		form.Set("email", "owner@example.com")
		req := httptest.NewRequest(http.MethodPost, "http://attesta.local/admin/orgs", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to create invite") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("invite failure", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "platform-user", Email: "admin@example.com", Status: "active"}, nil
				},
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}
					return &org, nil
				},
				listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
					return []IdentityMembership{{ID: "membership-admin", UserID: "platform-user", Email: "admin@example.com", Confirmed: true, IsOrgAdmin: true}}, nil
				},
				getUserByEmailFunc: func(ctx context.Context, email string) (IdentityUser, error) {
					return IdentityUser{}, ErrIdentityNotFound
				},
				inviteOrganizationUserFunc: func(ctx context.Context, sessionSecret, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
					return IdentityMembership{}, errors.New("boom")
				},
				deleteSessionFunc: func(ctx context.Context, sessionSecret string) error { return nil },
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		form := url.Values{}
		form.Set("intent", "invite_org_admin")
		form.Set("org_slug", "acme")
		form.Set("email", "owner@example.com")
		req := httptest.NewRequest(http.MethodPost, "http://attesta.local/admin/orgs", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to create invite") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})
}

func TestHandleAdminOrgsCreateOrganizationWithLogo(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	var updateSlug string
	var updateLogoID string
	var uploadedFilename string

	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
			},
			createOrganizationFunc: func(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
				return IdentityOrg{ID: "team-1", Slug: "fresh-org", Name: "Fresh Org"}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				return nil, ErrIdentityNotFound
			},
			uploadOrganizationLogoFunc: func(ctx context.Context, orgSlug string, upload IdentityFile) (IdentityFile, error) {
				uploadedFilename = upload.Filename
				return IdentityFile{ID: "logo-1", Filename: upload.Filename, ContentType: upload.ContentType}, nil
			},
			updateOrganizationAsAdminFunc: func(ctx context.Context, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
				updateSlug = currentSlug
				updateLogoID = logoFileID
				return IdentityOrg{ID: "team-1", Slug: currentSlug, Name: name, LogoFileID: logoFileID}, nil
			},
			deleteSessionFunc: func(ctx context.Context, sessionSecret string) error { return nil },
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	body := &strings.Builder{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("name", "Fresh Org"); err != nil {
		t.Fatalf("WriteField name error: %v", err)
	}
	part, err := writer.CreateFormFile("logo", "logo.png")
	if err != nil {
		t.Fatalf("CreateFormFile error: %v", err)
	}
	if _, err := io.WriteString(part, "PNG"); err != nil {
		t.Fatalf("WriteString logo error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader(body.String()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleAdminOrgs(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if uploadedFilename != "logo.png" || updateSlug != "fresh-org" || updateLogoID != "logo-1" {
		t.Fatalf("uploaded=%q updateSlug=%q updateLogoID=%q", uploadedFilename, updateSlug, updateLogoID)
	}
}

func TestHandleAdminOrgsCreateOrganizationErrors(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()

	t.Run("invalid logo type", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					return nil, ErrIdentityNotFound
				},
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		body := &strings.Builder{}
		writer := multipart.NewWriter(body)
		if err := writer.WriteField("name", "Fresh Org"); err != nil {
			t.Fatalf("WriteField name error: %v", err)
		}
		part, err := writer.CreateFormFile("logo", "logo.txt")
		if err != nil {
			t.Fatalf("CreateFormFile error: %v", err)
		}
		if _, err := io.WriteString(part, "TEXT"); err != nil {
			t.Fatalf("WriteString logo error: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("writer.Close error: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader(body.String()))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "logo must be a PNG, JPG, WEBP, or SVG image") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("create organization failure", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					return nil, ErrIdentityNotFound
				},
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
				},
				createOrganizationFunc: func(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
					return IdentityOrg{}, errors.New("boom")
				},
				deleteSessionFunc: func(ctx context.Context, sessionSecret string) error { return nil },
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader("name=Fresh+Org"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to create organization") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("logo upload failure", func(t *testing.T) {
		server := &Server{
			store: NewMemoryStore(),
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					return nil, ErrIdentityNotFound
				},
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
				},
				createOrganizationFunc: func(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
					return IdentityOrg{ID: "team-1", Slug: "fresh-org", Name: "Fresh Org"}, nil
				},
				uploadOrganizationLogoFunc: func(ctx context.Context, orgSlug string, upload IdentityFile) (IdentityFile, error) {
					return IdentityFile{}, errors.New("boom")
				},
				deleteSessionFunc: func(ctx context.Context, sessionSecret string) error { return nil },
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		body := &strings.Builder{}
		writer := multipart.NewWriter(body)
		if err := writer.WriteField("name", "Fresh Org"); err != nil {
			t.Fatalf("WriteField name error: %v", err)
		}
		part, err := writer.CreateFormFile("logo", "logo.png")
		if err != nil {
			t.Fatalf("CreateFormFile error: %v", err)
		}
		if _, err := io.WriteString(part, "PNG"); err != nil {
			t.Fatalf("WriteString logo error: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("writer.Close error: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader(body.String()))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to upload logo") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("organization update failure after logo", func(t *testing.T) {
		server := &Server{
			store: NewMemoryStore(),
			identity: &fakeIdentityStore{
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					return nil, ErrIdentityNotFound
				},
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
				},
				createOrganizationFunc: func(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
					return IdentityOrg{ID: "team-1", Slug: "fresh-org", Name: "Fresh Org"}, nil
				},
				uploadOrganizationLogoFunc: func(ctx context.Context, orgSlug string, upload IdentityFile) (IdentityFile, error) {
					return IdentityFile{ID: "logo-1", Filename: upload.Filename, ContentType: upload.ContentType}, nil
				},
				updateOrganizationAsAdminFunc: func(ctx context.Context, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
					return IdentityOrg{}, errors.New("boom")
				},
				deleteSessionFunc: func(ctx context.Context, sessionSecret string) error { return nil },
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		body := &strings.Builder{}
		writer := multipart.NewWriter(body)
		if err := writer.WriteField("name", "Fresh Org"); err != nil {
			t.Fatalf("WriteField name error: %v", err)
		}
		part, err := writer.CreateFormFile("logo", "logo.png")
		if err != nil {
			t.Fatalf("CreateFormFile error: %v", err)
		}
		if _, err := io.WriteString(part, "PNG"); err != nil {
			t.Fatalf("WriteString logo error: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("writer.Close error: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader(body.String()))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()

		server.handleAdminOrgs(rec, req)

		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to update organization") {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})
}

func TestEnsurePlatformAdminOwnsOrganizationAddsMembershipWhenMissing(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	var addedOrgSlug string
	var addedUserID string
	server := &Server{
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "platform-user", Email: "admin@example.com", Status: "active"}, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return nil, nil
			},
			addOrganizationUserByIDAsAdminFunc: func(ctx context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
				addedOrgSlug = orgSlug
				addedUserID = userID
				return IdentityMembership{ID: "membership-admin", UserID: userID, Email: "admin@example.com", Confirmed: true, IsOrgAdmin: true}, nil
			},
		},
		now: func() time.Time { return now },
	}

	session, err := server.ensurePlatformAdminOwnsOrganization(context.Background(), "acme", "http://attesta.local/invite/accept")

	if err != nil {
		t.Fatalf("ensurePlatformAdminOwnsOrganization error: %v", err)
	}
	if session == nil || session.Secret != "platform-session" || addedOrgSlug != "acme" || addedUserID != "platform-user" {
		t.Fatalf("session=%#v addedOrgSlug=%q addedUserID=%q", session, addedOrgSlug, addedUserID)
	}
}

func TestEnsurePlatformAdminOwnsOrganizationFallsBackToEmailInviteWithoutUserID(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	var invitedEmail string
	server := &Server{
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{Email: "admin@example.com", Status: "active"}, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return nil, nil
			},
			inviteOrganizationUserAsAdminFunc: func(ctx context.Context, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
				invitedEmail = email
				return IdentityMembership{ID: "membership-admin", Email: email, Confirmed: true, IsOrgAdmin: true}, nil
			},
		},
		now: func() time.Time { return now },
	}

	session, err := server.ensurePlatformAdminOwnsOrganization(context.Background(), "acme", "http://attesta.local/invite/accept")

	if err != nil {
		t.Fatalf("ensurePlatformAdminOwnsOrganization error: %v", err)
	}
	if session == nil || session.Secret != "platform-session" || invitedEmail != "admin@example.com" {
		t.Fatalf("session=%#v invitedEmail=%q", session, invitedEmail)
	}
}

func TestEnsurePlatformAdminOwnsOrganizationPromotesMembershipWhenNeeded(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	var updatedMembershipID string
	server := &Server{
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				return fakeIdentitySession("platform-session", "platform-user", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "platform-user", Email: "admin@example.com", Status: "active"}, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return []IdentityMembership{{ID: "membership-admin", UserID: "platform-user", Email: "admin@example.com", Confirmed: true, IsOrgAdmin: false}}, nil
			},
			updateOrganizationMembershipAsAdminFunc: func(ctx context.Context, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
				updatedMembershipID = membershipID
				return IdentityMembership{ID: membershipID, UserID: "platform-user", Email: "admin@example.com", Confirmed: true, IsOrgAdmin: true}, nil
			},
		},
		now: func() time.Time { return now },
	}

	session, err := server.ensurePlatformAdminOwnsOrganization(context.Background(), "acme", "http://attesta.local/invite/accept")

	if err != nil {
		t.Fatalf("ensurePlatformAdminOwnsOrganization error: %v", err)
	}
	if session == nil || session.Secret != "platform-session" || updatedMembershipID != "membership-admin" {
		t.Fatalf("session=%#v updatedMembershipID=%q", session, updatedMembershipID)
	}
}

func TestHandleOrgAdminUsersCreateOrgWithIdentity(t *testing.T) {
	now := time.Now().UTC()
	currentUser := IdentityUser{
		ID:         "user-1",
		Email:      "owner@example.com",
		Labels:     []string{identityOrgAdminLabel},
		IsOrgAdmin: true,
		Status:     "active",
	}
	createdOrg := IdentityOrg{}
	var createSessionSecret string
	var createName string

	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return currentUser, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				if strings.EqualFold(strings.TrimSpace(slug), strings.TrimSpace(createdOrg.Slug)) && createdOrg.Slug != "" {
					org := createdOrg
					return &org, nil
				}
				return nil, ErrIdentityNotFound
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				if createdOrg.Slug == "" {
					return nil, nil
				}
				return []IdentityUser{currentUser}, nil
			},
			createOrganizationFunc: func(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
				createSessionSecret = sessionSecret
				createName = name
				createdOrg = IdentityOrg{ID: "team-1", Slug: "fresh-org", Name: "Fresh Org"}
				currentUser.OrgSlug = createdOrg.Slug
				currentUser.OrgName = createdOrg.Name
				return createdOrg, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	form := url.Values{}
	form.Set("intent", "create_org")
	form.Set("name", "Fresh Org")
	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if createSessionSecret != "session-1" || createName != "Fresh Org" {
		t.Fatalf("create args = %q %q", createSessionSecret, createName)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "ORG_ADMIN fresh-org") {
		t.Fatalf("expected org admin body, got %q", body)
	}
	if !strings.Contains(body, "USERS 1") {
		t.Fatalf("expected current user row after bootstrap, got %q", body)
	}
}

func TestHandleOrgAdminUsersCreateOrgIdentityValidation(t *testing.T) {
	now := time.Now().UTC()
	createCalls := 0
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{
					ID:         "user-1",
					Email:      "owner@example.com",
					Labels:     []string{identityOrgAdminLabel},
					IsOrgAdmin: true,
					Status:     "active",
				}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				if strings.EqualFold(strings.TrimSpace(slug), "fresh-org") {
					org := IdentityOrg{ID: "team-1", Slug: "fresh-org", Name: "Fresh Org"}
					return &org, nil
				}
				return nil, ErrIdentityNotFound
			},
			createOrganizationFunc: func(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
				createCalls++
				return IdentityOrg{}, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=create_org&name=Fresh+Org"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if createCalls != 0 {
		t.Fatalf("create calls = %d, want 0", createCalls)
	}
	if !strings.Contains(rec.Body.String(), "organization slug already exists") {
		t.Fatalf("expected duplicate slug message, got %q", rec.Body.String())
	}
}

func TestHandleOrgAdminUsersUpdateOrgWithIdentityLogo(t *testing.T) {
	now := time.Now().UTC()
	currentUser := IdentityUser{
		ID:         "user-1",
		Email:      "owner@example.com",
		OrgSlug:    "acme",
		OrgName:    "Acme Org",
		Labels:     []string{identityOrgAdminLabel},
		IsOrgAdmin: true,
		Status:     "active",
	}
	org := IdentityOrg{
		ID:         "team-1",
		Slug:       "acme",
		Name:       "Acme Org",
		LogoFileID: "logo-old",
		Roles:      []IdentityRole{{Slug: "qa-reviewer", Name: "QA Reviewer"}},
	}
	var uploaded IdentityFile
	var updateSessionSecret string
	var updateCurrentSlug string
	var updateName string
	var updateLogoFileID string
	var updateRoles []IdentityRole

	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return currentUser, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				if strings.EqualFold(strings.TrimSpace(slug), "updated-name-org") && org.Slug == "updated-name-org" {
					current := org
					return &current, nil
				}
				if strings.EqualFold(strings.TrimSpace(slug), "acme") {
					current := org
					return &current, nil
				}
				return nil, ErrIdentityNotFound
			},
			uploadOrganizationLogoFunc: func(ctx context.Context, orgSlug string, file IdentityFile) (IdentityFile, error) {
				uploaded = file
				return IdentityFile{ID: "logo-new", Filename: file.Filename, ContentType: file.ContentType}, nil
			},
			updateOrganizationFunc: func(ctx context.Context, sessionSecret, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
				updateSessionSecret = sessionSecret
				updateCurrentSlug = currentSlug
				updateName = name
				updateLogoFileID = logoFileID
				updateRoles = append([]IdentityRole(nil), roles...)
				org = IdentityOrg{
					ID:         "team-1",
					Slug:       "updated-name-org",
					Name:       "Updated Name Org",
					LogoFileID: logoFileID,
					Roles:      append([]IdentityRole(nil), roles...),
				}
				currentUser.OrgSlug = org.Slug
				currentUser.OrgName = org.Name
				return org, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	body := &strings.Builder{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("intent", "update_org"); err != nil {
		t.Fatalf("WriteField intent error: %v", err)
	}
	if err := writer.WriteField("name", "Updated Name Org"); err != nil {
		t.Fatalf("WriteField name error: %v", err)
	}
	part, err := writer.CreateFormFile("logo", "logo.png")
	if err != nil {
		t.Fatalf("CreateFormFile error: %v", err)
	}
	if _, err := io.WriteString(part, "PNG"); err != nil {
		t.Fatalf("WriteString logo error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader(body.String()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if updateSessionSecret != "session-1" || updateCurrentSlug != "acme" || updateName != "Updated Name Org" || updateLogoFileID != "logo-new" {
		t.Fatalf("update args = %q %q %q %q", updateSessionSecret, updateCurrentSlug, updateName, updateLogoFileID)
	}
	if uploaded.Filename != "logo.png" {
		t.Fatalf("uploaded file = %#v", uploaded)
	}
	if len(updateRoles) != 1 || updateRoles[0].Slug != "qa-reviewer" {
		t.Fatalf("update roles = %#v", updateRoles)
	}
	if !strings.Contains(rec.Body.String(), "ORG_ADMIN updated-name-org") {
		t.Fatalf("expected updated org slug in body, got %q", rec.Body.String())
	}
}

func TestHandleOrgAdminLogoWithIdentity(t *testing.T) {
	now := time.Now().UTC()
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{
					ID:         "user-1",
					Email:      "owner@example.com",
					OrgSlug:    "acme",
					OrgName:    "Acme Org",
					Labels:     []string{identityOrgAdminLabel},
					IsOrgAdmin: true,
					Status:     "active",
				}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org", LogoFileID: "logo-1"}
				return &org, nil
			},
			getOrganizationLogoFunc: func(ctx context.Context, fileID string) (IdentityFile, error) {
				return IdentityFile{
					ID:          fileID,
					Filename:    "logo.svg",
					ContentType: "image/svg+xml",
					Data:        []byte("<svg/>"),
				}, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/org-admin/logo/logo-1", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminLogo(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("content type = %q", rec.Header().Get("Content-Type"))
	}
	if body := rec.Body.String(); body != "<svg/>" {
		t.Fatalf("body = %q", body)
	}
}

func TestHandleOrgAdminRolesWithIdentity(t *testing.T) {
	now := time.Now().UTC()
	org := IdentityOrg{
		ID:    "team-1",
		Slug:  "acme",
		Name:  "Acme Org",
		Roles: []IdentityRole{{Slug: "qa-reviewer", Name: "QA Reviewer"}},
	}
	var updateSessionSecret string
	var updatedRoles []IdentityRole

	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{
					ID:         "user-1",
					Email:      "owner@example.com",
					OrgSlug:    "acme",
					OrgName:    "Acme Org",
					Labels:     []string{identityOrgAdminLabel},
					IsOrgAdmin: true,
					Status:     "active",
				}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				current := org
				return &current, nil
			},
			updateOrganizationFunc: func(ctx context.Context, sessionSecret, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
				updateSessionSecret = sessionSecret
				updatedRoles = append([]IdentityRole(nil), roles...)
				org.Roles = append([]IdentityRole(nil), roles...)
				return org, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/roles", strings.NewReader("name=Approver&palette=blue"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminRoles(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if updateSessionSecret != "session-1" {
		t.Fatalf("session secret = %q", updateSessionSecret)
	}
	if len(updatedRoles) != 2 || updatedRoles[1].Slug != "approver" {
		t.Fatalf("updated roles = %#v", updatedRoles)
	}
}

func TestHandleOrgAdminUsersSetRolesWithIdentity(t *testing.T) {
	now := time.Now().UTC()
	users := []IdentityUser{
		{
			ID:         "user-1",
			Email:      "owner@example.com",
			OrgSlug:    "acme",
			Labels:     []string{identityOrgAdminLabel, encodeIdentityRoleLabel("qa-reviewer")},
			IsOrgAdmin: true,
			Status:     "active",
		},
		{
			ID:      "user-2",
			Email:   "member@example.com",
			OrgSlug: "acme",
			Labels:  []string{encodeIdentityRoleLabel("qa-reviewer")},
			Status:  "active",
		},
	}
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return users[0], nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{
					ID:   "team-1",
					Slug: "acme",
					Name: "Acme Org",
					Roles: []IdentityRole{
						{Slug: "qa-reviewer", Name: "QA Reviewer"},
						{Slug: "approver", Name: "Approver"},
					},
				}
				return &org, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return append([]IdentityUser(nil), users...), nil
			},
			updateUserLabelsFunc: func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
				for idx := range users {
					if users[idx].ID != userID {
						continue
					}
					users[idx].Labels = append([]string(nil), labels...)
					users[idx].IsOrgAdmin = hasIdentityLabel(labels, identityOrgAdminLabel)
					return users[idx], nil
				}
				return IdentityUser{}, ErrIdentityNotFound
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	targetID := stableIdentityUserObjectID("user-2").Hex()
	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=set_roles&userMongoId="+targetID+"&roles=approver&roles=org-admin"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !hasIdentityLabel(users[1].Labels, identityOrgAdminLabel) || !containsRole(decodeIdentityRoleLabels(users[1].Labels), "approver") {
		t.Fatalf("updated user labels = %#v", users[1].Labels)
	}

	selfID := stableIdentityUserObjectID("user-1").Hex()
	selfReq := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=set_roles&userMongoId="+selfID+"&roles=approver"))
	selfReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	selfReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	selfRec := httptest.NewRecorder()

	server.handleOrgAdminUsers(selfRec, selfReq)

	if selfRec.Code != http.StatusOK {
		t.Fatalf("self status = %d, want %d", selfRec.Code, http.StatusOK)
	}
	if !strings.Contains(selfRec.Body.String(), "cannot remove org-admin from your own account") {
		t.Fatalf("expected self-protection message, got %q", selfRec.Body.String())
	}
}

func TestHandleOrgAdminUsersInviteWithIdentity(t *testing.T) {
	now := time.Now().UTC()
	inviteCalls := 0
	var invitedRoles []string
	var invitedAdmin bool
	var invitedRedirect string
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{
					ID:         "user-1",
					Email:      "owner@example.com",
					OrgSlug:    "acme",
					OrgName:    "Acme Org",
					Labels:     []string{identityOrgAdminLabel},
					IsOrgAdmin: true,
					Status:     "active",
				}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{
					ID:   "team-1",
					Slug: "acme",
					Name: "Acme Org",
					Roles: []IdentityRole{
						{Slug: "qa-reviewer", Name: "QA Reviewer"},
						{Slug: "approver", Name: "Approver"},
					},
				}
				return &org, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return nil, nil
			},
			inviteOrganizationUserFunc: func(ctx context.Context, sessionSecret, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
				inviteCalls++
				invitedRoles = append([]string(nil), roleSlugs...)
				invitedAdmin = isOrgAdmin
				invitedRedirect = redirectURL
				return IdentityMembership{ID: "membership-1", Email: email}, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return []IdentityUser{{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}}, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=new%40example.com&roles=approver&roles=org-admin"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if inviteCalls != 1 || len(invitedRoles) != 1 || invitedRoles[0] != "approver" || !invitedAdmin {
		t.Fatalf("invite call = %d roles=%#v admin=%v", inviteCalls, invitedRoles, invitedAdmin)
	}
	if !strings.Contains(invitedRedirect, "/invite/accept") {
		t.Fatalf("redirect = %q", invitedRedirect)
	}
}

func TestHandleOrgAdminUsersInviteIdentityDuplicatePending(t *testing.T) {
	now := time.Now().UTC()
	inviteCalls := 0
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org", Roles: []IdentityRole{{Slug: "approver", Name: "Approver"}}}
				return &org, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return []IdentityMembership{{ID: "membership-1", Email: "pending@example.com", RoleSlugs: []string{"approver"}, Confirmed: false}}, nil
			},
			inviteOrganizationUserFunc: func(ctx context.Context, sessionSecret, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
				inviteCalls++
				return IdentityMembership{}, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return nil, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=pending%40example.com&roles=approver"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()

	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if inviteCalls != 0 {
		t.Fatalf("invite calls = %d, want 0", inviteCalls)
	}
}

func TestHandleOrgAdminUsersInviteIdentityExistingAndCrossOrg(t *testing.T) {
	now := time.Now().UTC()
	updatedUsers := make(map[string][]string)
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org", Roles: []IdentityRole{{Slug: "approver", Name: "Approver"}}}
				return &org, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return nil, nil
			},
			getUserByEmailFunc: func(ctx context.Context, email string) (IdentityUser, error) {
				switch email {
				case "member@example.com":
					return IdentityUser{ID: "user-2", Email: email, OrgSlug: "acme", Labels: []string{encodeIdentityRoleLabel("approver")}, Status: "active"}, nil
				case "other@example.com":
					return IdentityUser{ID: "user-3", Email: email, OrgSlug: "other-org", Labels: []string{encodeIdentityRoleLabel("approver")}, Status: "active"}, nil
				default:
					return IdentityUser{}, ErrIdentityNotFound
				}
			},
			updateUserLabelsFunc: func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
				updatedUsers[userID] = append([]string(nil), labels...)
				return IdentityUser{ID: userID, Labels: labels}, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return []IdentityUser{{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}}, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=member%40example.com&roles=approver"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()
	server.handleOrgAdminUsers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("existing member status = %d, want %d", rec.Code, http.StatusOK)
	}
	if len(updatedUsers["user-2"]) != 1 || updatedUsers["user-2"][0] != encodeIdentityRoleLabel("approver") {
		t.Fatalf("updated user labels = %#v", updatedUsers)
	}

	reqOther := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=other%40example.com&roles=approver"))
	reqOther.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqOther.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	recOther := httptest.NewRecorder()
	server.handleOrgAdminUsers(recOther, reqOther)
	if recOther.Code != http.StatusOK {
		t.Fatalf("cross-org status = %d, want %d", recOther.Code, http.StatusOK)
	}
	if !strings.Contains(recOther.Body.String(), "email already belongs to another organization") {
		t.Fatalf("expected cross-org error, got %q", recOther.Body.String())
	}
}

func TestHandleOrgAdminUsersDeleteUserWithIdentity(t *testing.T) {
	now := time.Now().UTC()
	deletedMemberships := []string{}
	updatedUsers := map[string][]string{}
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return []IdentityMembership{
					{ID: "membership-1", UserID: "user-1", Email: "owner@example.com", Confirmed: true},
					{ID: "membership-2", UserID: "user-2", Email: "member@example.com", Confirmed: true},
				}, nil
			},
			deleteOrganizationMembershipFunc: func(ctx context.Context, sessionSecret, orgSlug, membershipID string) error {
				deletedMemberships = append(deletedMemberships, membershipID)
				return nil
			},
			getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Email: "member@example.com", Labels: []string{"custom:keep", encodeIdentityRoleLabel("approver"), identityOrgAdminLabel}, Status: "active"}, nil
			},
			updateUserLabelsFunc: func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
				updatedUsers[userID] = append([]string(nil), labels...)
				return IdentityUser{ID: userID, Labels: labels}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}
				return &org, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return []IdentityUser{{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}}, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	targetID := stableIdentityUserObjectID("user-2").Hex()
	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=delete_user&userMongoId="+targetID))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()
	server.handleOrgAdminUsers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if len(deletedMemberships) != 1 || deletedMemberships[0] != "membership-2" {
		t.Fatalf("deleted memberships = %#v", deletedMemberships)
	}
	if labels := updatedUsers["user-2"]; len(labels) != 1 || labels[0] != "custom:keep" {
		t.Fatalf("updated user labels = %#v", updatedUsers)
	}
}

func TestIdentityOrgAdminHelpers(t *testing.T) {
	if got := inviteRedirectURL(httptest.NewRequest(http.MethodGet, "/org-admin/users", nil)); !strings.Contains(got, "/invite/accept") {
		t.Fatalf("invite redirect = %q", got)
	}
	t.Setenv("APPWRITE_INVITE_REDIRECT_URL", "https://app.example/invite/accept")
	if got := inviteRedirectURL(httptest.NewRequest(http.MethodGet, "/org-admin/users", nil)); got != "https://app.example/invite/accept" {
		t.Fatalf("configured invite redirect = %q", got)
	}
	if _, err := sessionSecretFromRequest(httptest.NewRequest(http.MethodGet, "/", nil)); err == nil {
		t.Fatal("expected missing cookie error")
	}
	reqEmpty := httptest.NewRequest(http.MethodGet, "/", nil)
	reqEmpty.AddCookie(&http.Cookie{Name: "attesta_session", Value: "   "})
	if _, err := sessionSecretFromRequest(reqEmpty); !errors.Is(err, ErrIdentityUnauthorized) {
		t.Fatalf("empty cookie error = %v", err)
	}

	withAdmin := ensureOrgAdminRoleOption([]Role{{Slug: "qa-reviewer", Name: "QA Reviewer"}})
	if len(withAdmin) != 2 || withAdmin[0].Slug != "org-admin" {
		t.Fatalf("with admin = %#v", withAdmin)
	}

	rows := buildOrgAdminInviteRowsFromMemberships([]IdentityMembership{
		{Email: "pending@example.com", RoleSlugs: []string{"approver"}, Confirmed: false, InvitedAt: time.Now().Add(-8 * 24 * time.Hour)},
		{Email: "accepted@example.com", RoleSlugs: []string{"approver"}, Confirmed: true, JoinedAt: time.Now()},
	}, time.Now())
	if len(rows) != 2 || rows[0].Status != "expired" || rows[1].Status != "accepted" {
		t.Fatalf("invite rows = %#v", rows)
	}
}

func TestHandleOrgAdminRolesIdentityErrors(t *testing.T) {
	now := time.Now().UTC()
	newServer := func(org IdentityOrg, updateErr error) *Server {
		return &Server{
			store: NewMemoryStore(),
			identity: &fakeIdentityStore{
				getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
					return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}, nil
				},
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					current := org
					return &current, nil
				},
				updateOrganizationFunc: func(ctx context.Context, sessionSecret, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
					if updateErr != nil {
						return IdentityOrg{}, updateErr
					}
					current := org
					current.Roles = roles
					return current, nil
				},
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
	}

	reqDup := httptest.NewRequest(http.MethodPost, "/org-admin/roles", strings.NewReader("name=QA+Reviewer&palette=blue"))
	reqDup.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqDup.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	recDup := httptest.NewRecorder()
	newServer(IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org", Roles: []IdentityRole{{Slug: "qa-reviewer", Name: "QA Reviewer"}}}, nil).handleOrgAdminRoles(recDup, reqDup)
	if recDup.Code != http.StatusOK || !strings.Contains(recDup.Body.String(), "role slug already exists") {
		t.Fatalf("duplicate role response = %d %q", recDup.Code, recDup.Body.String())
	}

	reqFail := httptest.NewRequest(http.MethodPost, "/org-admin/roles", strings.NewReader("name=Approver&palette=blue"))
	reqFail.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqFail.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	recFail := httptest.NewRecorder()
	newServer(IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}, errors.New("boom")).handleOrgAdminRoles(recFail, reqFail)
	if recFail.Code != http.StatusOK || !strings.Contains(recFail.Body.String(), "failed to create role") {
		t.Fatalf("failed role response = %d %q", recFail.Code, recFail.Body.String())
	}
}

func TestHandleOrgAdminUsersIdentityBranchErrors(t *testing.T) {
	now := time.Now().UTC()
	baseIdentity := func() *fakeIdentityStore {
		return &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org", Roles: []IdentityRole{{Slug: "approver", Name: "Approver"}}}
				return &org, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) { return nil, nil },
		}
	}

	t.Run("invite updates pending membership", func(t *testing.T) {
		fake := baseIdentity()
		updated := 0
		fake.listOrganizationMembershipsFunc = func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			return []IdentityMembership{{ID: "membership-1", Email: "pending@example.com", Confirmed: false}}, nil
		}
		fake.updateOrganizationMembershipFunc = func(ctx context.Context, sessionSecret, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
			updated++
			return IdentityMembership{ID: membershipID, Email: "pending@example.com"}, nil
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=pending%40example.com&roles=approver"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || updated != 1 {
			t.Fatalf("pending update response = %d updated=%d body=%q", rec.Code, updated, rec.Body.String())
		}
	})

	t.Run("set roles rejects unknown user", func(t *testing.T) {
		fake := baseIdentity()
		fake.listOrganizationUsersFunc = func(ctx context.Context, orgSlug string) ([]IdentityUser, error) { return nil, nil }
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=set_roles&userMongoId="+stableIdentityUserObjectID("missing").Hex()+"&roles=approver"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "user not found") {
			t.Fatalf("set roles unknown user response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("delete user rejects self", func(t *testing.T) {
		fake := baseIdentity()
		fake.listOrganizationMembershipsFunc = func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			return []IdentityMembership{{ID: "membership-1", UserID: "user-1", Email: "owner@example.com", Confirmed: true}}, nil
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=delete_user&userMongoId="+stableIdentityUserObjectID("user-1").Hex()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "cannot delete yourself") {
			t.Fatalf("delete self response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("invite rejects unknown role", func(t *testing.T) {
		fake := baseIdentity()
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=user%40example.com&roles=missing"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "role not found") {
			t.Fatalf("invite unknown role response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("invite handles existing user lookup failure", func(t *testing.T) {
		fake := baseIdentity()
		fake.listOrganizationMembershipsFunc = func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) { return nil, nil }
		fake.getUserByEmailFunc = func(ctx context.Context, email string) (IdentityUser, error) {
			return IdentityUser{}, errors.New("boom")
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=user%40example.com&roles=approver"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to load existing user") {
			t.Fatalf("invite lookup failure response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("set roles handles update failure", func(t *testing.T) {
		fake := baseIdentity()
		fake.listOrganizationUsersFunc = func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
			return []IdentityUser{{ID: "user-2", Email: "member@example.com", OrgSlug: "acme", Labels: []string{encodeIdentityRoleLabel("approver")}, Status: "active"}}, nil
		}
		fake.updateUserLabelsFunc = func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
			return IdentityUser{}, errors.New("boom")
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=set_roles&userMongoId="+stableIdentityUserObjectID("user-2").Hex()+"&roles=approver&roles=org-admin"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to update user roles") {
			t.Fatalf("set roles update failure response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("delete user handles delete failure", func(t *testing.T) {
		fake := baseIdentity()
		fake.listOrganizationMembershipsFunc = func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			return []IdentityMembership{{ID: "membership-2", UserID: "user-2", Email: "member@example.com", Confirmed: true}}, nil
		}
		fake.deleteOrganizationMembershipFunc = func(ctx context.Context, sessionSecret, orgSlug, membershipID string) error {
			return errors.New("boom")
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=delete_user&userMongoId="+stableIdentityUserObjectID("user-2").Hex()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to delete user") {
			t.Fatalf("delete failure response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("create org handles create failure", func(t *testing.T) {
		fake := &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "owner@example.com", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				return nil, ErrIdentityNotFound
			},
			createOrganizationFunc: func(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
				return IdentityOrg{}, errors.New("boom")
			},
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=create_org&name=Fresh+Org"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to create organization") {
			t.Fatalf("create org failure response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("create org handles logo upload failure", func(t *testing.T) {
		fake := &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "owner@example.com", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				return nil, ErrIdentityNotFound
			},
			createOrganizationFunc: func(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
				return IdentityOrg{ID: "team-1", Slug: "fresh-org", Name: "Fresh Org"}, nil
			},
			uploadOrganizationLogoFunc: func(ctx context.Context, orgSlug string, upload IdentityFile) (IdentityFile, error) {
				return IdentityFile{}, errors.New("boom")
			},
		}
		body := &strings.Builder{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("intent", "create_org")
		_ = writer.WriteField("name", "Fresh Org")
		part, _ := writer.CreateFormFile("logo", "logo.png")
		_, _ = io.WriteString(part, "PNG")
		_ = writer.Close()
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader(body.String()))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to upload logo") {
			t.Fatalf("create org upload failure response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("update org handles duplicate slug", func(t *testing.T) {
		fake := baseIdentity()
		fake.getOrganizationBySlugFunc = func(ctx context.Context, slug string) (*IdentityOrg, error) {
			switch slug {
			case "acme":
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}
				return &org, nil
			case "other-org":
				org := IdentityOrg{ID: "team-2", Slug: "other-org", Name: "Other Org"}
				return &org, nil
			default:
				return nil, ErrIdentityNotFound
			}
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=update_org&name=Other+Org"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "organization slug already exists") {
			t.Fatalf("update org duplicate response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("update org handles update failure", func(t *testing.T) {
		fake := baseIdentity()
		fake.getOrganizationBySlugFunc = func(ctx context.Context, slug string) (*IdentityOrg, error) {
			org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}
			return &org, nil
		}
		fake.updateOrganizationFunc = func(ctx context.Context, sessionSecret, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
			return IdentityOrg{}, errors.New("boom")
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=update_org&name=Updated+Org"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to update organization") {
			t.Fatalf("update org failure response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("delete user handles label cleanup failure", func(t *testing.T) {
		fake := baseIdentity()
		fake.listOrganizationMembershipsFunc = func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			return []IdentityMembership{{ID: "membership-2", UserID: "user-2", Email: "member@example.com", Confirmed: true}}, nil
		}
		fake.deleteOrganizationMembershipFunc = func(ctx context.Context, sessionSecret, orgSlug, membershipID string) error { return nil }
		fake.getUserByIDFunc = func(ctx context.Context, userID string) (IdentityUser, error) {
			return IdentityUser{ID: userID, Labels: []string{encodeIdentityRoleLabel("approver"), identityOrgAdminLabel}}, nil
		}
		fake.updateUserLabelsFunc = func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
			return IdentityUser{}, errors.New("boom")
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=delete_user&userMongoId="+stableIdentityUserObjectID("user-2").Hex()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to delete user") {
			t.Fatalf("delete cleanup failure response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("invite handles membership list failure", func(t *testing.T) {
		fake := baseIdentity()
		fake.listOrganizationMembershipsFunc = func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			return nil, errors.New("boom")
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=user%40example.com&roles=approver"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to create invite") {
			t.Fatalf("invite membership failure response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("invite handles update membership failure", func(t *testing.T) {
		fake := baseIdentity()
		fake.listOrganizationMembershipsFunc = func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			return []IdentityMembership{{ID: "membership-1", Email: "pending@example.com", Confirmed: false}}, nil
		}
		fake.updateOrganizationMembershipFunc = func(ctx context.Context, sessionSecret, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
			return IdentityMembership{}, errors.New("boom")
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=pending%40example.com&roles=approver"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to create invite") {
			t.Fatalf("invite update membership failure response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("invite handles confirmed user label update failure", func(t *testing.T) {
		fake := baseIdentity()
		fake.listOrganizationMembershipsFunc = func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			return []IdentityMembership{{ID: "membership-1", UserID: "user-2", Email: "member@example.com", Confirmed: true}}, nil
		}
		fake.updateUserLabelsFunc = func(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
			return IdentityUser{}, errors.New("boom")
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=member%40example.com&roles=approver"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to update user roles") {
			t.Fatalf("invite confirmed label failure response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("invite handles create invite failure", func(t *testing.T) {
		fake := baseIdentity()
		fake.listOrganizationMembershipsFunc = func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) { return nil, nil }
		fake.getUserByEmailFunc = func(ctx context.Context, email string) (IdentityUser, error) {
			return IdentityUser{}, ErrIdentityNotFound
		}
		fake.inviteOrganizationUserFunc = func(ctx context.Context, sessionSecret, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
			return IdentityMembership{}, errors.New("boom")
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=new%40example.com&roles=approver"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to create invite") {
			t.Fatalf("invite create failure response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("set roles rejects unknown role", func(t *testing.T) {
		fake := baseIdentity()
		fake.listOrganizationUsersFunc = func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
			return []IdentityUser{{ID: "user-2", Email: "member@example.com", OrgSlug: "acme", Labels: []string{encodeIdentityRoleLabel("approver")}, Status: "active"}}, nil
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=set_roles&userMongoId="+stableIdentityUserObjectID("user-2").Hex()+"&roles=missing"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "role not found") {
			t.Fatalf("set roles unknown role response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("delete user handles load user failure", func(t *testing.T) {
		fake := baseIdentity()
		fake.listOrganizationMembershipsFunc = func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			return []IdentityMembership{{ID: "membership-2", UserID: "user-2", Email: "member@example.com", Confirmed: true}}, nil
		}
		fake.deleteOrganizationMembershipFunc = func(ctx context.Context, sessionSecret, orgSlug, membershipID string) error { return nil }
		fake.getUserByIDFunc = func(ctx context.Context, userID string) (IdentityUser, error) {
			return IdentityUser{}, errors.New("boom")
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=delete_user&userMongoId="+stableIdentityUserObjectID("user-2").Hex()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "failed to delete user") {
			t.Fatalf("delete load user failure response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("update org handles org not found", func(t *testing.T) {
		fake := baseIdentity()
		fake.getOrganizationBySlugFunc = func(ctx context.Context, slug string) (*IdentityOrg, error) {
			return nil, ErrIdentityNotFound
		}
		server := &Server{store: NewMemoryStore(), identity: fake, tmpl: testTemplates(), enforceAuth: true, now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=update_org&name=Updated+Org"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("update org not found response = %d", rec.Code)
		}
	})
}

func TestLoadOrgAdminStateIdentityFallbacks(t *testing.T) {
	now := time.Now().UTC()
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				return &IdentityOrg{
					ID:   "team-1",
					Slug: "acme-org",
					Name: "Acme Org",
					Roles: []IdentityRole{
						{Slug: "qa-reviewer", Name: "QA Reviewer", Color: "#123", Border: "#456"},
					},
				}, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return []IdentityUser{{ID: "user-1", Email: "member@example.com", OrgSlug: "acme-org", Labels: []string{encodeIdentityRoleLabel("qa-reviewer")}, Status: "active"}}, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return nil, errors.New("boom")
			},
		},
		now: func() time.Time { return now },
	}
	adminOrgID := stableOrgObjectID("acme-org")
	admin := &AccountUser{ID: stableIdentityUserObjectID("owner"), Email: "owner@example.com", OrgID: &adminOrgID, OrgSlug: "acme-org", RoleSlugs: []string{"org-admin"}, Status: "active"}
	gotOrg, roles, users, invites, err := server.loadOrgAdminState(t.Context(), admin, "acme-org")
	if err != nil {
		t.Fatalf("loadOrgAdminState error: %v", err)
	}
	if gotOrg.Slug != "acme-org" || len(roles) != 2 || len(users) != 1 || invites != nil {
		t.Fatalf("state = %#v %#v %#v %#v", gotOrg, roles, users, invites)
	}
}

func TestHandleOrgAdminLogoIdentityBranches(t *testing.T) {
	now := time.Now().UTC()
	baseServer := func(identity *fakeIdentityStore) *Server {
		if identity.getSessionFunc == nil {
			identity.getSessionFunc = func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			}
		}
		if identity.getCurrentUserFunc == nil {
			identity.getCurrentUserFunc = func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}, nil
			}
		}
		if identity.listOrganizationUsersFunc == nil {
			identity.listOrganizationUsersFunc = func(ctx context.Context, orgSlug string) ([]IdentityUser, error) { return nil, nil }
		}
		return &Server{
			store:       NewMemoryStore(),
			identity:    identity,
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
	}

	t.Run("method not allowed", func(t *testing.T) {
		server := baseServer(&fakeIdentityStore{})
		req := httptest.NewRequest(http.MethodPost, "/org-admin/logo/logo-1", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminLogo(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("logo not found paths", func(t *testing.T) {
		tests := []struct {
			name     string
			path     string
			identity *fakeIdentityStore
		}{
			{
				name:     "blank id",
				path:     "/org-admin/logo/",
				identity: &fakeIdentityStore{},
			},
			{
				name: "org missing",
				path: "/org-admin/logo/logo-1",
				identity: &fakeIdentityStore{
					getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) { return nil, ErrIdentityNotFound },
				},
			},
			{
				name: "id mismatch",
				path: "/org-admin/logo/logo-1",
				identity: &fakeIdentityStore{
					getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
						org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org", LogoFileID: "other-logo"}
						return &org, nil
					},
				},
			},
			{
				name: "load error",
				path: "/org-admin/logo/logo-1",
				identity: &fakeIdentityStore{
					getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
						org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org", LogoFileID: "logo-1"}
						return &org, nil
					},
					getOrganizationLogoFunc: func(ctx context.Context, fileID string) (IdentityFile, error) {
						return IdentityFile{}, ErrIdentityNotFound
					},
				},
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				server := baseServer(tc.identity)
				req := httptest.NewRequest(http.MethodGet, tc.path, nil)
				req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
				rec := httptest.NewRecorder()
				server.handleOrgAdminLogo(rec, req)
				if rec.Code != http.StatusNotFound {
					t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
				}
			})
		}
	})

	t.Run("blank content type falls back", func(t *testing.T) {
		server := baseServer(&fakeIdentityStore{
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org", LogoFileID: "logo-1"}
				return &org, nil
			},
			getOrganizationLogoFunc: func(ctx context.Context, fileID string) (IdentityFile, error) {
				return IdentityFile{ID: fileID, Filename: "logo.bin", Data: []byte("BIN")}, nil
			},
		})
		req := httptest.NewRequest(http.MethodGet, "/org-admin/logo/logo-1", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminLogo(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Header().Get("Content-Type") != "application/octet-stream" {
			t.Fatalf("content type = %q", rec.Header().Get("Content-Type"))
		}
	})
}

func TestHandleOrgAdminRolesIdentityAdditionalBranches(t *testing.T) {
	now := time.Now().UTC()
	baseServer := func(user IdentityUser, org *IdentityOrg, updateErr error) *Server {
		return &Server{
			store: NewMemoryStore(),
			identity: &fakeIdentityStore{
				getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
					return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return user, nil
				},
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					if org == nil {
						return nil, ErrIdentityNotFound
					}
					current := *org
					return &current, nil
				},
				updateOrganizationFunc: func(ctx context.Context, sessionSecret, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
					if updateErr != nil {
						return IdentityOrg{}, updateErr
					}
					current := *org
					current.Roles = roles
					return current, nil
				},
				listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) { return nil, nil },
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
	}

	t.Run("get renders org admin", func(t *testing.T) {
		server := baseServer(
			IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"},
			&IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"},
			nil,
		)
		req := httptest.NewRequest(http.MethodGet, "/org-admin/roles", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminRoles(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ORG_ADMIN acme") {
			t.Fatalf("response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("no organization context", func(t *testing.T) {
		server := baseServer(
			IdentityUser{ID: "user-1", Email: "owner@example.com", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"},
			nil,
			nil,
		)
		req := httptest.NewRequest(http.MethodPost, "/org-admin/roles", strings.NewReader("name=Approver"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminRoles(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "create organization first") {
			t.Fatalf("response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing name and missing org", func(t *testing.T) {
		server := baseServer(
			IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"},
			&IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"},
			nil,
		)
		req := httptest.NewRequest(http.MethodPost, "/org-admin/roles", strings.NewReader("palette=blue"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminRoles(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "role name is required") {
			t.Fatalf("response = %d %q", rec.Code, rec.Body.String())
		}

		notFound := baseServer(
			IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"},
			nil,
			nil,
		)
		req2 := httptest.NewRequest(http.MethodPost, "/org-admin/roles", strings.NewReader("name=Approver"))
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req2.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec2 := httptest.NewRecorder()
		notFound.handleOrgAdminRoles(rec2, req2)
		if rec2.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec2.Code, http.StatusNotFound)
		}
	})

	t.Run("missing palette derives default style", func(t *testing.T) {
		org := &IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"}
		server := baseServer(
			IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"},
			org,
			nil,
		)
		req := httptest.NewRequest(http.MethodPost, "/org-admin/roles", strings.NewReader("name=Escalations"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminRoles(rec, req)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		server := baseServer(
			IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"},
			&IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org"},
			nil,
		)
		req := httptest.NewRequest(http.MethodPut, "/org-admin/roles", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminRoles(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestHandleOrgAdminUsersIdentityAdditionalBranches(t *testing.T) {
	now := time.Now().UTC()
	baseServer := func(user IdentityUser) *Server {
		return &Server{
			store: NewMemoryStore(),
			identity: &fakeIdentityStore{
				getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
					return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return user, nil
				},
				getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
					org := IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org", Roles: []IdentityRole{{Slug: "approver", Name: "Approver"}}}
					return &org, nil
				},
				listOrganizationUsersFunc:       func(ctx context.Context, orgSlug string) ([]IdentityUser, error) { return nil, nil },
				listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) { return nil, nil },
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
	}

	t.Run("get and unsupported action", func(t *testing.T) {
		server := baseServer(IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"})
		getReq := httptest.NewRequest(http.MethodGet, "/org-admin/users", nil)
		getReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		getRec := httptest.NewRecorder()
		server.handleOrgAdminUsers(getRec, getReq)
		if getRec.Code != http.StatusOK || !strings.Contains(getRec.Body.String(), "ORG_ADMIN acme") {
			t.Fatalf("get response = %d %q", getRec.Code, getRec.Body.String())
		}

		postReq := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=unsupported"))
		postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		postReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		postRec := httptest.NewRecorder()
		server.handleOrgAdminUsers(postRec, postReq)
		if postRec.Code != http.StatusOK || !strings.Contains(postRec.Body.String(), "unsupported action") {
			t.Fatalf("post response = %d %q", postRec.Code, postRec.Body.String())
		}
	})

	t.Run("default invite validation and missing user ids", func(t *testing.T) {
		server := baseServer(IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"})
		tests := []struct {
			body string
			want string
		}{
			{body: "", want: "email is required"},
			{body: "intent=set_roles", want: "user is required"},
			{body: "intent=delete_user", want: "user is required"},
		}
		for _, tc := range tests {
			req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
			rec := httptest.NewRecorder()
			server.handleOrgAdminUsers(rec, req)
			if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), tc.want) {
				t.Fatalf("body=%q response=%d %q", tc.body, rec.Code, rec.Body.String())
			}
		}
	})

	t.Run("create org validation without org context", func(t *testing.T) {
		server := baseServer(IdentityUser{ID: "user-1", Email: "owner@example.com", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"})

		reqInvite := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=user%40example.com"))
		reqInvite.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqInvite.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		recInvite := httptest.NewRecorder()
		server.handleOrgAdminUsers(recInvite, reqInvite)
		if recInvite.Code != http.StatusOK || !strings.Contains(recInvite.Body.String(), "create organization first") {
			t.Fatalf("invite response = %d %q", recInvite.Code, recInvite.Body.String())
		}

		reqCreate := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=create_org"))
		reqCreate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqCreate.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		recCreate := httptest.NewRecorder()
		server.handleOrgAdminUsers(recCreate, reqCreate)
		if recCreate.Code != http.StatusOK || !strings.Contains(recCreate.Body.String(), "organization name is required") {
			t.Fatalf("create response = %d %q", recCreate.Code, recCreate.Body.String())
		}
	})

	t.Run("create org rejected when org already exists", func(t *testing.T) {
		server := baseServer(IdentityUser{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"})
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=create_org&name=Fresh+Org"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "organization already exists for your account") {
			t.Fatalf("response = %d %q", rec.Code, rec.Body.String())
		}
	})
}

func TestHandleOrgAdminUsersIdentityMultipartLogoValidation(t *testing.T) {
	now := time.Now().UTC()
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "owner@example.com", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				return nil, ErrIdentityNotFound
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	buildRequest := func(t *testing.T, files [][]byte, names []string) *http.Request {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err := writer.WriteField("intent", "create_org"); err != nil {
			t.Fatalf("WriteField intent error: %v", err)
		}
		if err := writer.WriteField("name", "Fresh Org"); err != nil {
			t.Fatalf("WriteField name error: %v", err)
		}
		for idx := range files {
			part, err := writer.CreateFormFile("logo", names[idx])
			if err != nil {
				t.Fatalf("CreateFormFile error: %v", err)
			}
			if _, err := part.Write(files[idx]); err != nil {
				t.Fatalf("part.Write error: %v", err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("writer.Close error: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		return req
	}

	t.Run("logo too large", func(t *testing.T) {
		t.Setenv("ORG_LOGO_MAX_BYTES", "64")
		req := buildRequest(t, [][]byte{[]byte(strings.Repeat("a", 4096))}, []string{"logo.png"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "logo file too large") {
			t.Fatalf("response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("multiple logo files", func(t *testing.T) {
		req := buildRequest(
			t,
			[][]byte{
				{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'},
				{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'},
			},
			[]string{"logo-one.png", "logo-two.png"},
		)
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "upload a single logo file") {
			t.Fatalf("response = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid multipart form", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("--broken"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}
