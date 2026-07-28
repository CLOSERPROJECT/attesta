package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func orgAdminSectionURLTestServer(t *testing.T, now time.Time) *Server {
	t.Helper()
	return &Server{
		authorizer: fakeAuthorizer{},
		store:      NewMemoryStore(),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{
					ID:         "user-1",
					Email:      "owner@example.com",
					OrgSlug:    "acme",
					Labels:     []string{identityOrgAdminLabel},
					IsOrgAdmin: true,
					Status:     "active",
				}, nil
			},
			getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
				org := IdentityOrg{
					ID:    "team-1",
					Slug:  "acme",
					Name:  "Acme Org",
					Roles: []IdentityRole{{Slug: "approver", Name: "Approver"}},
				}
				return &org, nil
			},
			listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
				return []IdentityUser{
					{ID: "user-1", Email: "owner@example.com", OrgSlug: "acme", Labels: []string{identityOrgAdminLabel}, IsOrgAdmin: true, Status: "active"},
				}, nil
			},
			listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
				return nil, nil
			},
		},
		tmpl:        parseTestTemplates(t),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
}

func TestOrgAdminSectionURLHandlers(t *testing.T) {
	now := time.Now().UTC()
	server := orgAdminSectionURLTestServer(t, now)

	t.Run("GET legacy users redirects to profile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/my/organization/users", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()

		server.handleOrgAdminUsers(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
		if loc := rec.Header().Get("Location"); loc != "/my/organization/profile" {
			t.Fatalf("location = %q, want /my/organization/profile", loc)
		}
	})

	t.Run("GET profile renders active profile panel", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/my/organization/profile", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()

		server.handleOrgAdminPage(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		assertOrgAdminActivePanel(t, body, "profile")
	})

	t.Run("GET members renders active members panel", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/my/organization/members", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()

		server.handleOrgAdminPage(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		assertOrgAdminActivePanel(t, body, "members")
	})

	t.Run("GET roles renders active roles panel", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/my/organization/roles", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()

		server.handleOrgAdminRoles(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := rec.Body.String()
		assertOrgAdminActivePanel(t, body, "roles")
	})
}

func TestResolveOrgAdminActivePanelFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/my/organization/profile", want: "profile"},
		{path: "/my/organization/members", want: "members"},
		{path: "/my/organization/roles", want: "roles"},
		{path: "/my/organization/users", want: "profile"},
		{path: "/organization/profile", want: "profile"},
		{path: "/organization/members", want: "members"},
		{path: "/organization/roles", want: "roles"},
		{path: "/organization/roles/", want: "roles"},
	}
	for _, tc := range tests {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		if got := resolveOrgAdminActivePanel(req, OrgAdminErrors{}, ""); got != tc.want {
			t.Fatalf("resolveOrgAdminActivePanel(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func assertOrgAdminActivePanel(t *testing.T, body, panel string) {
	t.Helper()

	if !strings.Contains(body, `id="admin-console"`) {
		t.Fatalf("expected admin-console soft-nav shell for %q", panel)
	}
	if !strings.Contains(body, `class="sidebar-nav-link is-active"`) {
		t.Fatalf("expected active sidebar nav link for %q", panel)
	}

	href := "/my/organization/" + panel
	activeIdx := strings.Index(body, `class="sidebar-nav-link is-active"`)
	if activeIdx == -1 {
		t.Fatalf("expected active sidebar nav link for %q", panel)
	}
	start := activeIdx - 200
	if start < 0 {
		start = 0
	}
	end := activeIdx + 350
	if end > len(body) {
		end = len(body)
	}
	window := body[start:end]
	if !strings.Contains(window, `href="`+href+`"`) {
		t.Fatalf("expected active soft-nav link for %q, got snippet:\n%s", panel, window)
	}
	if !strings.Contains(window, `aria-current="page"`) {
		t.Fatalf("expected aria-current on active %q link, got snippet:\n%s", panel, window)
	}
	if !strings.Contains(window, `hx-target="#admin-console"`) {
		t.Fatalf("expected hx-target on soft-nav for %q, got snippet:\n%s", panel, window)
	}

	panelID := `id="org-admin-panel-` + panel + `"`
	if !strings.Contains(body, panelID) {
		t.Fatalf("expected %s section in body", panelID)
	}

	for _, other := range []string{"profile", "roles", "members"} {
		if other == panel {
			continue
		}
		otherID := `id="org-admin-panel-` + other + `"`
		if strings.Contains(body, otherID) {
			t.Fatalf("inactive panel %q must not be rendered, found %s", other, otherID)
		}
	}
}

func TestOrgAdminActivePanelTemplateRendering(t *testing.T) {
	tmpl := parseTestTemplates(t)
	base := OrgAdminView{
		Organization: Organization{Name: "Acme Org", Slug: "acme"},
	}

	for _, panel := range []string{"profile", "roles", "members"} {
		t.Run(panel, func(t *testing.T) {
			view := base
			view.ActivePanel = panel
			var out bytes.Buffer
			if err := tmpl.ExecuteTemplate(&out, "org_admin_body", view); err != nil {
				t.Fatalf("render org admin template: %v", err)
			}
			assertOrgAdminActivePanel(t, out.String(), panel)
		})
	}
}

func TestLegacyOrgAdminRoutesReturnNotFound(t *testing.T) {
	server := orgAdminSectionURLTestServer(t, time.Now().UTC())
	mux := server.newMux()

	req := httptest.NewRequest(http.MethodGet, "/org-admin/profile", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestMyOrganizationProfileRouteViaMux(t *testing.T) {
	server := orgAdminSectionURLTestServer(t, time.Now().UTC())
	mux := server.newMux()

	req := httptest.NewRequest(http.MethodGet, "/my/organization/profile", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatalf("status = %d, want route registered (not 404)", rec.Code)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestUnknownMyRoutesReturnNotFound(t *testing.T) {
	server := orgAdminSectionURLTestServer(t, time.Now().UTC())
	mux := server.newMux()

	cases := []string{
		"/my/not-a-page",
		"/my/organization/not-a-section",
	}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}
