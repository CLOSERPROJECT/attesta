package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleOrganizationRootRedirects(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	t.Run("member to app home", func(t *testing.T) {
		sessionID := "session-org-root-member"
		account := AccountUser{
			ID:             primitive.NewObjectID(),
			IdentityUserID: "member-1",
			Email:          "member@example.com",
			OrgSlug:        "acme",
			RoleSlugs:      []string{"viewer"},
			Status:         "active",
			CreatedAt:      now,
		}
		sessions := map[string]AccountUser{sessionID: account}
		identity := testIdentityForSessions(now, sessions)
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, organizationPath(""), nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusSeeOther, rec.Body.String())
		}
		if loc := rec.Header().Get("Location"); loc != appHomePath {
			t.Fatalf("location = %q, want %q", loc, appHomePath)
		}
	})

	t.Run("org admin to profile", func(t *testing.T) {
		sessionID := "session-org-root-admin"
		account := AccountUser{
			ID:             primitive.NewObjectID(),
			IdentityUserID: "admin-1",
			Email:          "admin@example.com",
			OrgSlug:        "acme",
			RoleSlugs:      []string{"org-admin"},
			Status:         "active",
			CreatedAt:      now,
		}
		sessions := map[string]AccountUser{sessionID: account}
		identity := testIdentityForSessions(now, sessions)
		server := &Server{
			identity: identity,
			store:    NewMemoryStore(),
			tmpl:     parseTestTemplates(t),
			authorizer: fakeAuthorizer{
				accessDecide: func(user *AccountUser, resourceKind, _ string, _ map[string]interface{}, action string) (bool, error) {
					if resourceKind == cerbosResourceOrgAdminConsole && action == cerbosActionAccess {
						return true, nil
					}
					return fakeCanAccessDecision(user, resourceKind, nil, action), nil
				},
			},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, organizationPath(""), nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusSeeOther, rec.Body.String())
		}
		if loc := rec.Header().Get("Location"); loc != organizationPath("profile") {
			t.Fatalf("location = %q, want %q", loc, organizationPath("profile"))
		}
	})

	t.Run("unaffiliated to onboarding", func(t *testing.T) {
		sessionID := "session-org-root-unaffiliated"
		account := AccountUser{
			ID:             primitive.NewObjectID(),
			IdentityUserID: "user-1",
			Email:          "solo@example.com",
			Status:         "active",
			CreatedAt:      now,
		}
		sessions := map[string]AccountUser{sessionID: account}
		identity := testIdentityForSessions(now, sessions)
		server := &Server{
			identity:    identity,
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, organizationPath(""), nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusSeeOther, rec.Body.String())
		}
		if loc := rec.Header().Get("Location"); loc != onboardingPath() {
			t.Fatalf("location = %q, want %q", loc, onboardingPath())
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		sessionID := "session-org-root-post"
		account := AccountUser{
			ID:             primitive.NewObjectID(),
			IdentityUserID: "member-1",
			Email:          "member@example.com",
			OrgSlug:        "acme",
			Status:         "active",
			CreatedAt:      now,
		}
		sessions := map[string]AccountUser{sessionID: account}
		server := &Server{
			identity:    testIdentityForSessions(now, sessions),
			store:       NewMemoryStore(),
			tmpl:        parseTestTemplates(t),
			authorizer:  fakeAuthorizer{},
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, organizationPath(""), nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleMyRoutes(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestAffiliationLeaveGateBlocksSoleOrgAdmin(t *testing.T) {
	identity := &fakeIdentityStore{
		listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
			return []IdentityUser{
				{ID: "admin-1", Email: "owner@example.com", IsOrgAdmin: true},
			}, nil
		},
	}
	canLeave, reason := affiliationLeaveGate(context.Background(), identity, &AccountUser{
		IdentityUserID: "admin-1",
		Email:          "owner@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"org-admin"},
	})
	if canLeave {
		t.Fatal("sole Org admin must not leave")
	}
	if !strings.Contains(reason, "only Org admin") {
		t.Fatalf("LeaveReason = %q", reason)
	}

	identity.listOrganizationUsersFunc = func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
		return []IdentityUser{
			{ID: "admin-1", Email: "owner@example.com", IsOrgAdmin: true},
			{ID: "admin-2", Email: "co@example.com", IsOrgAdmin: true},
		}, nil
	}
	canLeave, reason = affiliationLeaveGate(context.Background(), identity, &AccountUser{
		IdentityUserID: "admin-1",
		Email:          "owner@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"org-admin"},
	})
	if !canLeave || reason != "" {
		t.Fatalf("shared admins can leave: canLeave=%v reason=%q", canLeave, reason)
	}
}

func TestPageBaseForUserAccountSettingsAffiliatedMember(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	identity := &fakeIdentityStore{
		getOrganizationBySlugFunc: func(_ context.Context, slug string) (*IdentityOrg, error) {
			if slug != "acme" {
				return nil, ErrIdentityNotFound
			}
			return &IdentityOrg{
				ID:   "team-1",
				Slug: "acme",
				Name: "Acme Org",
			}, nil
		},
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	user := &AccountUser{
		IdentityUserID: "member-1",
		Email:          "member@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"viewer"},
		Status:         "active",
	}
	base := server.pageBaseForUser(user, "home_picker_body", "", "")
	if base.ShowMyOrgLink {
		t.Fatal("non-admin must not get My organization console link")
	}
	if !base.ShowAccountSettings {
		t.Fatal("expected ShowAccountSettings for affiliated member")
	}
	if !base.CanLeave {
		t.Fatal("member must be able to leave")
	}
	if base.AccountOrgName != "Acme Org" {
		t.Fatalf("AccountOrgName = %q", base.AccountOrgName)
	}
	if base.LeavePath != leaveOrganizationPath() {
		t.Fatalf("LeavePath = %q", base.LeavePath)
	}
}

func TestPageBaseForUserHidesAccountSettingsWhenUnaffiliated(t *testing.T) {
	server := &Server{
		identity:    &fakeIdentityStore{},
		store:       NewMemoryStore(),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC) },
	}
	base := server.pageBaseForUser(&AccountUser{
		IdentityUserID: "user-1",
		Email:          "solo@example.com",
		Status:         "active",
	}, "home_picker_body", "", "")
	if base.ShowAccountSettings {
		t.Fatal("unaffiliated must not see Account settings")
	}
}

func TestLayoutAccountSettingsMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var rendered bytes.Buffer
	view := PageBase{
		ShowLogout:          true,
		ShowAccountSettings: true,
		AccountOrgName:      "Acme Org",
		LeavePath:           leaveOrganizationPath(),
		CanLeave:            true,
	}
	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", view); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := rendered.String()
	for _, want := range []string{
		"Account settings",
		`id="account-settings-dialog"`,
		`class="dialog-head dialog-head-ruled"`,
		`class="u-flex-center u-justify-between u-gap-4"`,
		`id="leave-organization-dialog"`,
		`action="/my/leave-organization"`,
		`class="dialog"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in layout, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, "list-row") || strings.Contains(body, "role-pill") || strings.Contains(body, "data-role-palette") {
		t.Fatalf("account settings must not use list-row or role badges, got:\n%s", body)
	}
	if strings.Contains(body, `href="/my/organization"`) && !strings.Contains(body, `href="/my/organization/profile"`) {
		t.Fatalf("bare /my/organization link must not appear, got:\n%s", body)
	}
}

func TestLayoutAccountSettingsLeaveBlocked(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var rendered bytes.Buffer
	view := PageBase{
		ShowLogout:          true,
		ShowAccountSettings: true,
		AccountOrgName:      "Acme Org",
		CanLeave:            false,
		LeaveReason:         "You're the only Org admin. Add another before leaving.",
	}
	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", view); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := rendered.String()
	if strings.Contains(body, `id="leave-organization-dialog"`) {
		t.Fatal("blocked leave must not render confirm dialog")
	}
	if !strings.Contains(body, `disabled`) || !strings.Contains(body, "only Org admin") {
		t.Fatalf("expected disabled leave + reason, got:\n%s", body)
	}
}

func TestLayoutMyOrgLinkPointsToProfile(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var rendered bytes.Buffer
	view := PageBase{
		ShowLogout:    true,
		ShowMyOrgLink: true,
	}
	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", view); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := rendered.String()
	if !strings.Contains(body, `href="/my/organization/profile"`) {
		t.Fatalf("expected My organization link to profile, got:\n%s", body)
	}
	if strings.Contains(body, "Account settings") {
		t.Fatalf("ShowMyOrgLink alone must not imply Account settings, got:\n%s", body)
	}
}
