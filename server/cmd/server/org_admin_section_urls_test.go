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

	t.Run("GET legacy users redirects to members", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/org-admin/users", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
		rec := httptest.NewRecorder()

		server.handleOrgAdminUsers(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
		if loc := rec.Header().Get("Location"); loc != "/org-admin/members" {
			t.Fatalf("location = %q, want /org-admin/members", loc)
		}
	})

	t.Run("GET profile renders active profile panel", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/org-admin/profile", nil)
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
		req := httptest.NewRequest(http.MethodGet, "/org-admin/members", nil)
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
		req := httptest.NewRequest(http.MethodGet, "/org-admin/roles", nil)
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
		{path: "/org-admin/profile", want: "profile"},
		{path: "/org-admin/members", want: "members"},
		{path: "/org-admin/roles", want: "roles"},
		{path: "/org-admin/users", want: "profile"},
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

	if !strings.Contains(body, `class="sidebar-nav-link is-active"`) {
		t.Fatalf("expected active sidebar nav link for %q", panel)
	}
	if !strings.Contains(body, `href="/org-admin/`+panel+`"`) {
		t.Fatalf("expected nav href for %q panel", panel)
	}
	if !strings.Contains(body, `data-org-admin-default-panel="`+panel+`"`) {
		preview := body
		if len(preview) > 400 {
			preview = preview[:400]
		}
		t.Fatalf("expected default panel %q, got body prefix:\n%s", panel, preview)
	}

	panelID := `id="org-admin-panel-` + panel + `"` 
	panelStart := strings.Index(body, panelID)
	if panelStart == -1 {
		t.Fatalf("expected %s section in body", panelID)
	}
	panelEnd := strings.Index(body[panelStart:], ">")
	if panelEnd == -1 {
		t.Fatalf("expected opening tag for %s", panelID)
	}
	panelTag := body[panelStart : panelStart+panelEnd+1]
	if strings.Contains(panelTag, "hidden") {
		t.Fatalf("active panel %q must not be hidden, got tag %q", panel, panelTag)
	}

	for _, other := range []string{"profile", "roles", "members"} {
		if other == panel {
			continue
		}
		otherID := `id="org-admin-panel-` + other + `"` 
		otherStart := strings.Index(body, otherID)
		if otherStart == -1 {
			t.Fatalf("expected %s section in body", otherID)
		}
		otherEnd := strings.Index(body[otherStart:], ">")
		if otherEnd == -1 {
			t.Fatalf("expected opening tag for %s", otherID)
		}
		otherTag := body[otherStart : otherStart+otherEnd+1]
		if !strings.Contains(otherTag, "hidden") {
			t.Fatalf("inactive panel %q should be hidden, got tag %q", other, otherTag)
		}
	}
}

func TestOrgAdminActivePanelTemplateRendering(t *testing.T) {
	tmpl := parseTestTemplates(t)
	base := OrgAdminView{
		Header: PageHeaderView{
			Title:    "Organization admin dashboard",
			BackHref: "/",
		},
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
