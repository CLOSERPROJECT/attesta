package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleOrganizationHomeAffiliatedMember(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-org-home"
	account := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "member-1",
		Email:          "member@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"viewer", "approver"},
		Status:         "active",
		CreatedAt:      now,
	}
	sessions := map[string]AccountUser{sessionID: account}
	identity := testIdentityForSessions(now, sessions)
	identity.getOrganizationBySlugFunc = func(_ context.Context, slug string) (*IdentityOrg, error) {
		if slug != "acme" {
			return nil, ErrIdentityNotFound
		}
		return &IdentityOrg{
			ID:         "team-1",
			Slug:       "acme",
			Name:       "Acme Org",
			LogoFileID: "logo-1",
			Roles: []IdentityRole{
				{Slug: "viewer", Name: "Viewer", Palette: "blue"},
				{Slug: "approver", Name: "Approver", Palette: "green"},
			},
		}, nil
	}

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

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Acme Org",
		`src="/organization/logo/acme"`,
		"Viewer",
		"Approver",
		`data-role-palette="blue"`,
		`id="leave-organization-dialog"`,
		`action="/my/leave-organization"`,
		`class="dialog"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in organization home, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, "only organization admin") || strings.Contains(body, "only Org admin") {
		t.Fatalf("non-admin member leave dialog must not mention sole-admin gate, got:\n%s", body)
	}
	if strings.Contains(body, "Manage organization") {
		t.Fatal("non-admin member must not see Manage organization")
	}
}

func TestHandleOrganizationHomeUnaffiliatedRedirectsOnboarding(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-org-home-unaffiliated"
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
}

func TestOrganizationHomeLeaveDialogMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := OrganizationHomeView{
		PageBase:         PageBase{Body: "organization_home_body"},
		OrganizationName: "Acme Org",
		LeavePath:        leaveOrganizationPath(),
		CanLeave:         true,
		Roles: []OrgAdminRoleOption{
			{Slug: "viewer", Name: "Viewer", Palette: "blue"},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "organization_home_body", view); err != nil {
		t.Fatalf("ExecuteTemplate: %v", err)
	}
	body := out.String()
	dialogStart := strings.Index(body, `id="leave-organization-dialog"`)
	if dialogStart == -1 {
		t.Fatalf("expected leave dialog, got:\n%s", body)
	}
	dialogEnd := strings.Index(body[dialogStart:], `</dialog>`)
	if dialogEnd == -1 {
		t.Fatal("expected closing dialog tag")
	}
	dialog := body[dialogStart : dialogStart+dialogEnd]
	for _, want := range []string{
		`class="dialog"`,
		`class="dialog-card"`,
		`class="dialog-head"`,
		`class="dialog-title u-text-danger"`,
		`class="dialog-subtitle"`,
		`class="dialog-actions"`,
		`action="/my/leave-organization"`,
		`class="btn btn-danger"`,
	} {
		if !strings.Contains(dialog, want) {
			t.Fatalf("expected %q in leave dialog markup, got:\n%s", want, dialog)
		}
	}
	if strings.Contains(dialog, "only Org admin") || strings.Contains(dialog, "only organization admin") {
		t.Fatalf("leave dialog must not repeat sole-admin gate copy, got:\n%s", dialog)
	}
}

func TestOrganizationHomeLeaveGateBlocksSoleOrgAdmin(t *testing.T) {
	identity := &fakeIdentityStore{
		listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
			return []IdentityUser{
				{ID: "admin-1", Email: "owner@example.com", IsOrgAdmin: true},
			}, nil
		},
	}
	canLeave, reason := organizationHomeLeaveGate(context.Background(), identity, &AccountUser{
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
	canLeave, reason = organizationHomeLeaveGate(context.Background(), identity, &AccountUser{
		IdentityUserID: "admin-1",
		Email:          "owner@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"org-admin"},
	})
	if !canLeave || reason != "" {
		t.Fatalf("shared admins can leave: canLeave=%v reason=%q", canLeave, reason)
	}
}

func TestOrganizationHomeLeaveBlockedMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := OrganizationHomeView{
		OrganizationName: "Acme Org",
		LeavePath:        leaveOrganizationPath(),
		CanLeave:         false,
		LeaveReason:      "You're the only Org admin. Add another before leaving.",
	}
	if err := tmpl.ExecuteTemplate(&out, "organization_home_body", view); err != nil {
		t.Fatalf("ExecuteTemplate: %v", err)
	}
	body := out.String()
	if strings.Contains(body, `id="leave-organization-dialog"`) {
		t.Fatal("blocked leave must not render confirm dialog")
	}
	if !strings.Contains(body, `disabled`) || !strings.Contains(body, "only Org admin") {
		t.Fatalf("expected disabled leave + reason, got:\n%s", body)
	}
}

func TestPageBaseForUserShowMyOrgLinkForAffiliatedNonAdmin(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	server := &Server{
		identity:    &fakeIdentityStore{},
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
	if !base.ShowMyOrgLink {
		t.Fatal("expected ShowMyOrgLink for affiliated non-admin")
	}
	if userIsOrgAdmin(user) {
		t.Fatal("fixture must be non-admin")
	}
}

func TestLayoutMyOrgLinkPointsToOrganizationHome(t *testing.T) {
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
	if !strings.Contains(body, `href="/my/organization"`) {
		t.Fatalf("expected My organization link to /my/organization, got:\n%s", body)
	}
	if strings.Contains(body, `href="/my/organization/profile"`) {
		t.Fatalf("My organization must not link to profile, got:\n%s", body)
	}
	if strings.Contains(body, "Leave organization") {
		t.Fatalf("expected leave organization removed from account menu, got:\n%s", body)
	}
	if strings.Contains(body, `action="/my/leave-organization"`) {
		t.Fatalf("expected no leave form in account menu, got:\n%s", body)
	}
}

func TestMemberRolesForOrganizationHome(t *testing.T) {
	org := IdentityOrg{
		Slug: "acme",
		Roles: []IdentityRole{
			{Slug: "viewer", Name: "Viewer", Palette: "blue"},
		},
	}
	user := &AccountUser{RoleSlugs: []string{"viewer", "org-admin"}}
	roles := memberRolesForOrganizationHome(user, org)
	if len(roles) != 2 {
		t.Fatalf("roles = %#v, want 2", roles)
	}
	if roles[0].Name != "Viewer" || roles[0].Palette != "blue" {
		t.Fatalf("first role = %#v", roles[0])
	}
	if roles[1].Name != "Org Admin" {
		t.Fatalf("second role = %#v", roles[1])
	}

	if memberRolesForOrganizationHome(nil, org) != nil {
		t.Fatal("nil user should return nil")
	}
	// Whitespace-only role slugs canonify to the sentinel "item".
	dupes := memberRolesForOrganizationHome(&AccountUser{RoleSlugs: []string{"viewer", "viewer", "  ", "custom"}}, org)
	if len(dupes) != 3 {
		t.Fatalf("dupes = %#v, want viewer+item+custom", dupes)
	}
	if dupes[1].Slug != "item" || dupes[2].Name != "custom" {
		t.Fatalf("unexpected roles = %#v", dupes)
	}
}

func TestHandleOrganizationHomeOrgAdminManageLink(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-org-home-admin"
	account := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "admin-1",
		Email:          "admin@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      nil,
		Status:         "active",
		CreatedAt:      now,
	}
	sessions := map[string]AccountUser{sessionID: account}
	identity := testIdentityForSessions(now, sessions)
	identity.getOrganizationBySlugFunc = func(_ context.Context, slug string) (*IdentityOrg, error) {
		if slug != "acme" {
			return nil, ErrIdentityNotFound
		}
		return &IdentityOrg{ID: "team-1", Slug: "acme", Name: "Acme Org", Roles: nil}, nil
	}

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

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Manage organization",
		`href="/my/organization/profile"`,
		"No roles assigned yet.",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q, got:\n%s", want, body)
		}
	}
}

func TestHandleOrganizationHomeNotFoundAndMethod(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-org-home-missing"
	account := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "member-2",
		Email:          "member2@example.com",
		OrgSlug:        "gone",
		RoleSlugs:      []string{"viewer"},
		Status:         "active",
		CreatedAt:      now,
	}
	sessions := map[string]AccountUser{sessionID: account}
	identity := testIdentityForSessions(now, sessions)
	identity.getOrganizationBySlugFunc = func(_ context.Context, _ string) (*IdentityOrg, error) {
		return nil, ErrIdentityNotFound
	}

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
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}

	methodReq := httptest.NewRequest(http.MethodPost, organizationPath(""), nil)
	methodReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	methodRec := httptest.NewRecorder()
	server.handleMyRoutes(methodRec, methodReq)
	if methodRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method status=%d", methodRec.Code)
	}
}

func TestHandleOrganizationHomeIdentityError(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-org-home-err"
	account := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "member-3",
		Email:          "member3@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"viewer"},
		Status:         "active",
		CreatedAt:      now,
	}
	sessions := map[string]AccountUser{sessionID: account}
	identity := testIdentityForSessions(now, sessions)
	identity.getOrganizationBySlugFunc = func(_ context.Context, _ string) (*IdentityOrg, error) {
		return nil, errors.New("identity unavailable")
	}

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
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestHandleOrganizationHomeManageLinkAuthErrorStillRenders(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-org-home-auth-err"
	account := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "member-4",
		Email:          "member4@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"viewer"},
		Status:         "active",
		CreatedAt:      now,
	}
	sessions := map[string]AccountUser{sessionID: account}
	identity := testIdentityForSessions(now, sessions)
	identity.getOrganizationBySlugFunc = func(_ context.Context, slug string) (*IdentityOrg, error) {
		return &IdentityOrg{ID: "t", Slug: "acme", Name: "Acme", Roles: []IdentityRole{{Slug: "viewer", Name: "Viewer"}}}, nil
	}

	server := &Server{
		identity: identity,
		store:    NewMemoryStore(),
		tmpl:     parseTestTemplates(t),
		authorizer: fakeAuthorizer{
			accessDecide: func(_ *AccountUser, _, _ string, _ map[string]interface{}, _ string) (bool, error) {
				return false, errors.New("cerbos down")
			},
		},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, organizationPath(""), nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "Manage organization") {
		t.Fatal("auth error must not show manage link")
	}
}
