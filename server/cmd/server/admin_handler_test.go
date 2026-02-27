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

type adminFailingStore struct {
	*MemoryStore
	failCreateOrganization bool
	failCreateUser         bool
	failCreateInvite       bool
	failCreateRole         bool
}

func (s *adminFailingStore) CreateOrganization(ctx context.Context, org Organization) (Organization, error) {
	if s.failCreateOrganization {
		return Organization{}, errors.New("create organization failed")
	}
	return s.MemoryStore.CreateOrganization(ctx, org)
}

func (s *adminFailingStore) CreateRole(ctx context.Context, role Role) (Role, error) {
	if s.failCreateRole {
		return Role{}, errors.New("create role failed")
	}
	return s.MemoryStore.CreateRole(ctx, role)
}

func (s *adminFailingStore) CreateUser(ctx context.Context, user AccountUser) (AccountUser, error) {
	if s.failCreateUser {
		return AccountUser{}, errors.New("create user failed")
	}
	return s.MemoryStore.CreateUser(ctx, user)
}

func (s *adminFailingStore) CreateInvite(ctx context.Context, invite Invite) (Invite, error) {
	if s.failCreateInvite {
		return Invite{}, errors.New("create invite failed")
	}
	return s.MemoryStore.CreateInvite(ctx, invite)
}

func createSessionForTestUser(t *testing.T, store *MemoryStore, user AccountUser) string {
	t.Helper()
	now := time.Now().UTC()
	session, err := store.CreateSession(t.Context(), Session{
		SessionID:   "s-" + user.UserID,
		UserID:      user.UserID,
		UserMongoID: user.ID,
		OrgID:       user.OrgID,
		CreatedAt:   now,
		LastLoginAt: now,
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	return session.SessionID
}

func TestHandleAdminOrgsAccessControl(t *testing.T) {
	store := NewMemoryStore()
	user, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "user-a",
		Email:     "user-a@example.com",
		RoleSlugs: []string{"dep1"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	sessionID := createSessionForTestUser(t, store, user)

	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}
	req := httptest.NewRequest(http.MethodGet, "/admin/orgs", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleAdminOrgs(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestHandleAdminOrgsCreateOrgAndInvite(t *testing.T) {
	store := NewMemoryStore()
	admin, err := store.CreateUser(t.Context(), AccountUser{
		UserID:          "platform-admin",
		Email:           "platform@example.com",
		IsPlatformAdmin: true,
		Status:          "active",
		CreatedAt:       time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	sessionID := createSessionForTestUser(t, store, admin)

	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}

	createReq := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader("name=Acme+Org"))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	createRec := httptest.NewRecorder()
	server.handleAdminOrgs(createRec, createReq)
	if createRec.Code != http.StatusSeeOther {
		t.Fatalf("create org status = %d, want %d", createRec.Code, http.StatusSeeOther)
	}

	org, err := store.GetOrganizationBySlug(t.Context(), "acme-org")
	if err != nil {
		t.Fatalf("GetOrganizationBySlug error: %v", err)
	}
	inviteReq := httptest.NewRequest(http.MethodPost, "/admin/orgs/acme-org", strings.NewReader("email=org-admin%40acme.org"))
	inviteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	inviteReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	inviteRec := httptest.NewRecorder()
	server.handleAdminOrgs(inviteRec, inviteReq)

	if inviteRec.Code != http.StatusOK {
		t.Fatalf("invite status = %d, want %d", inviteRec.Code, http.StatusOK)
	}
	if !strings.Contains(inviteRec.Body.String(), "/invite/") {
		t.Fatalf("expected invite link in response, got %q", inviteRec.Body.String())
	}
	roles, err := store.ListRolesByOrg(t.Context(), org.Slug)
	if err != nil {
		t.Fatalf("ListRolesByOrg error: %v", err)
	}
	foundOrgAdmin := false
	for _, role := range roles {
		if role.Slug == "org-admin" {
			foundOrgAdmin = true
			break
		}
	}
	if !foundOrgAdmin {
		t.Fatal("expected org-admin role to exist")
	}
}

func TestHandleOrgAdminCreateRoleAndUserInvite(t *testing.T) {
	store := NewMemoryStore()
	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme Org"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "Org Admin", Slug: "org-admin", CreatedAt: time.Now().UTC()})
	orgID := org.ID
	adminUser, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "org-admin-user",
		OrgID:     &orgID,
		OrgSlug:   org.Slug,
		Email:     "org-admin@acme.org",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	sessionID := createSessionForTestUser(t, store, adminUser)
	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}

	roleReq := httptest.NewRequest(http.MethodPost, "/org-admin/roles", strings.NewReader("name=qa_reviewer"))
	roleReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	roleReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	roleRec := httptest.NewRecorder()
	server.handleOrgAdminRoles(roleRec, roleReq)
	if roleRec.Code != http.StatusOK {
		t.Fatalf("create role status = %d, want %d", roleRec.Code, http.StatusOK)
	}
	if _, err := store.GetRoleBySlug(t.Context(), org.Slug, "qa-reviewer"); err != nil {
		t.Fatalf("GetRoleBySlug error: %v", err)
	}

	userReq := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("email=user%40acme.org&role=qa-reviewer"))
	userReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	userReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	userRec := httptest.NewRecorder()
	server.handleOrgAdminUsers(userRec, userReq)
	if userRec.Code != http.StatusOK {
		t.Fatalf("invite user status = %d, want %d", userRec.Code, http.StatusOK)
	}
	if !strings.Contains(userRec.Body.String(), "/invite/") {
		t.Fatalf("expected invite link in response, got %q", userRec.Body.String())
	}
}

func TestHandleAdminOrgsGetShowsOrgsNav(t *testing.T) {
	store := NewMemoryStore()
	admin, err := store.CreateUser(t.Context(), AccountUser{
		UserID:          "platform-admin-nav",
		Email:           "platform-nav@example.com",
		IsPlatformAdmin: true,
		Status:          "active",
		CreatedAt:       time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	sessionID := createSessionForTestUser(t, store, admin)

	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}
	req := httptest.NewRequest(http.MethodGet, "/admin/orgs", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleAdminOrgs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "NAV Home Backoffice Orgs |") {
		t.Fatalf("expected Orgs nav marker, got %q", rec.Body.String())
	}
}

func TestHandleAdminOrgsInviteValidationError(t *testing.T) {
	store := NewMemoryStore()
	admin, err := store.CreateUser(t.Context(), AccountUser{
		UserID:          "platform-admin-invite",
		Email:           "platform-invite@example.com",
		IsPlatformAdmin: true,
		Status:          "active",
		CreatedAt:       time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if _, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme Org"}); err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	sessionID := createSessionForTestUser(t, store, admin)

	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}
	req := httptest.NewRequest(http.MethodPost, "/admin/orgs/acme-org", strings.NewReader("email="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleAdminOrgs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "email is required") {
		t.Fatalf("expected validation message, got %q", rec.Body.String())
	}
}

func TestHandleOrgAdminRolesAndUsersValidationPaths(t *testing.T) {
	store := NewMemoryStore()
	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Validation Org"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "Org Admin", Slug: "org-admin", CreatedAt: time.Now().UTC()})
	orgID := org.ID
	adminUser, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "org-admin-validation",
		OrgID:     &orgID,
		OrgSlug:   org.Slug,
		Email:     "org-admin-validation@acme.org",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	sessionID := createSessionForTestUser(t, store, adminUser)
	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}

	roleReq := httptest.NewRequest(http.MethodPost, "/org-admin/roles", strings.NewReader("name="))
	roleReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	roleReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	roleRec := httptest.NewRecorder()
	server.handleOrgAdminRoles(roleRec, roleReq)
	if roleRec.Code != http.StatusOK {
		t.Fatalf("role status = %d, want %d", roleRec.Code, http.StatusOK)
	}
	if !strings.Contains(roleRec.Body.String(), "role name is required") {
		t.Fatalf("expected role name error, got %q", roleRec.Body.String())
	}

	userReq := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("email=user%40acme.org&role="))
	userReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	userReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	userRec := httptest.NewRecorder()
	server.handleOrgAdminUsers(userRec, userReq)
	if userRec.Code != http.StatusOK {
		t.Fatalf("user status = %d, want %d", userRec.Code, http.StatusOK)
	}
	if !strings.Contains(userRec.Body.String(), "email and role are required") {
		t.Fatalf("expected user role validation error, got %q", userRec.Body.String())
	}
}

func TestAdminHandlersFailurePaths(t *testing.T) {
	t.Run("platform admin create organization failure", func(t *testing.T) {
		base := NewMemoryStore()
		admin, err := base.CreateUser(t.Context(), AccountUser{
			UserID:          "platform-admin-failure",
			Email:           "platform-failure@example.com",
			IsPlatformAdmin: true,
			Status:          "active",
			CreatedAt:       time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}
		store := &adminFailingStore{MemoryStore: base, failCreateOrganization: true}
		sessionID := createSessionForTestUser(t, base, admin)
		server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}
		req := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader("name=Acme+Org"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleAdminOrgs(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if !strings.Contains(rec.Body.String(), "failed to create organization") {
			t.Fatalf("expected failure message, got %q", rec.Body.String())
		}
	})

	t.Run("org admin missing role and create user failure", func(t *testing.T) {
		base := NewMemoryStore()
		org, _ := base.CreateOrganization(t.Context(), Organization{Name: "Fail Org"})
		_, _ = base.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "Org Admin", Slug: "org-admin", CreatedAt: time.Now().UTC()})
		orgID := org.ID
		admin, err := base.CreateUser(t.Context(), AccountUser{
			UserID:    "org-admin-failure",
			OrgID:     &orgID,
			OrgSlug:   org.Slug,
			Email:     "org-admin-failure@example.com",
			RoleSlugs: []string{"org-admin"},
			Status:    "active",
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}
		sessionID := createSessionForTestUser(t, base, admin)
		server := &Server{store: base, tmpl: testTemplates(), enforceAuth: true, now: time.Now}

		reqRoleMissing := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("email=u%40x.io&role=missing-role"))
		reqRoleMissing.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqRoleMissing.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		recRoleMissing := httptest.NewRecorder()
		server.handleOrgAdminUsers(recRoleMissing, reqRoleMissing)
		if recRoleMissing.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recRoleMissing.Code, http.StatusOK)
		}
		if !strings.Contains(recRoleMissing.Body.String(), "role not found") {
			t.Fatalf("expected role not found message, got %q", recRoleMissing.Body.String())
		}

		_, _ = base.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "QA Reviewer", Slug: "qa-reviewer", CreatedAt: time.Now().UTC()})
		failStore := &adminFailingStore{MemoryStore: base, failCreateUser: true}
		serverFail := &Server{store: failStore, tmpl: testTemplates(), enforceAuth: true, now: time.Now}
		reqCreateUserFail := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("email=new%40x.io&role=qa-reviewer"))
		reqCreateUserFail.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqCreateUserFail.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		recCreateUserFail := httptest.NewRecorder()
		serverFail.handleOrgAdminUsers(recCreateUserFail, reqCreateUserFail)
		if recCreateUserFail.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recCreateUserFail.Code, http.StatusOK)
		}
		if !strings.Contains(recCreateUserFail.Body.String(), "failed to create user") {
			t.Fatalf("expected create user failure message, got %q", recCreateUserFail.Body.String())
		}
	})
}

func TestAdminHandlersMethodAndParseErrors(t *testing.T) {
	store := NewMemoryStore()
	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme Org"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	if _, err := store.CreateRole(t.Context(), Role{
		OrgID:     org.ID,
		OrgSlug:   org.Slug,
		Name:      "Org Admin",
		Slug:      "org-admin",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateRole org-admin error: %v", err)
	}
	orgID := org.ID
	platformAdmin, err := store.CreateUser(t.Context(), AccountUser{
		UserID:          "platform-admin-methods",
		Email:           "platform-methods@example.com",
		IsPlatformAdmin: true,
		Status:          "active",
		CreatedAt:       time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser platform admin error: %v", err)
	}
	orgAdmin, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "org-admin-methods",
		OrgID:     &orgID,
		OrgSlug:   org.Slug,
		Email:     "org-admin-methods@example.com",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser org admin error: %v", err)
	}
	platformSession := createSessionForTestUser(t, store, platformAdmin)
	orgAdminSession := createSessionForTestUser(t, store, orgAdmin)
	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}

	t.Run("admin orgs method not allowed and parse errors", func(t *testing.T) {
		reqMethod := httptest.NewRequest(http.MethodPut, "/admin/orgs", nil)
		reqMethod.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformSession})
		recMethod := httptest.NewRecorder()
		server.handleAdminOrgs(recMethod, reqMethod)
		if recMethod.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", recMethod.Code, http.StatusMethodNotAllowed)
		}

		reqParse := httptest.NewRequest(http.MethodPost, "/admin/orgs?bad=%zz", strings.NewReader("name=Acme"))
		reqParse.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqParse.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformSession})
		recParse := httptest.NewRecorder()
		server.handleAdminOrgs(recParse, reqParse)
		if recParse.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recParse.Code, http.StatusBadRequest)
		}

		reqMissingOrg := httptest.NewRequest(http.MethodPost, "/admin/orgs/missing", strings.NewReader("email=a%40b.c"))
		reqMissingOrg.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqMissingOrg.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformSession})
		recMissingOrg := httptest.NewRecorder()
		server.handleAdminOrgs(recMissingOrg, reqMissingOrg)
		if recMissingOrg.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", recMissingOrg.Code, http.StatusNotFound)
		}

		reqExistingOrgGet := httptest.NewRequest(http.MethodGet, "/admin/orgs/"+org.Slug, nil)
		reqExistingOrgGet.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformSession})
		recExistingOrgGet := httptest.NewRecorder()
		server.handleAdminOrgs(recExistingOrgGet, reqExistingOrgGet)
		if recExistingOrgGet.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recExistingOrgGet.Code, http.StatusOK)
		}
	})

	t.Run("org admin roles method not allowed and parse error", func(t *testing.T) {
		reqMethod := httptest.NewRequest(http.MethodDelete, "/org-admin/roles", nil)
		reqMethod.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		recMethod := httptest.NewRecorder()
		server.handleOrgAdminRoles(recMethod, reqMethod)
		if recMethod.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", recMethod.Code, http.StatusMethodNotAllowed)
		}

		reqParse := httptest.NewRequest(http.MethodPost, "/org-admin/roles?bad=%zz", strings.NewReader("name=qa"))
		reqParse.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqParse.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		recParse := httptest.NewRecorder()
		server.handleOrgAdminRoles(recParse, reqParse)
		if recParse.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recParse.Code, http.StatusBadRequest)
		}
	})

	t.Run("org admin users get and parse error", func(t *testing.T) {
		reqGet := httptest.NewRequest(http.MethodGet, "/org-admin/users", nil)
		reqGet.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		recGet := httptest.NewRecorder()
		server.handleOrgAdminUsers(recGet, reqGet)
		if recGet.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recGet.Code, http.StatusOK)
		}

		reqParse := httptest.NewRequest(http.MethodPost, "/org-admin/users?bad=%zz", strings.NewReader("email=a%40b.c&role=org-admin"))
		reqParse.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqParse.AddCookie(&http.Cookie{Name: "attesta_session", Value: orgAdminSession})
		recParse := httptest.NewRecorder()
		server.handleOrgAdminUsers(recParse, reqParse)
		if recParse.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recParse.Code, http.StatusBadRequest)
		}
	})
}

func TestOrgAdminRoleCreationFailureRendersError(t *testing.T) {
	base := NewMemoryStore()
	org, err := base.CreateOrganization(t.Context(), Organization{Name: "Role Fail Org"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	if _, err := base.CreateRole(t.Context(), Role{
		OrgID:     org.ID,
		OrgSlug:   org.Slug,
		Name:      "Org Admin",
		Slug:      "org-admin",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateRole error: %v", err)
	}
	orgID := org.ID
	admin, err := base.CreateUser(t.Context(), AccountUser{
		UserID:    "org-admin-role-fail",
		OrgID:     &orgID,
		OrgSlug:   org.Slug,
		Email:     "org-admin-role-fail@example.com",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	sessionID := createSessionForTestUser(t, base, admin)
	store := &adminFailingStore{MemoryStore: base, failCreateRole: true}
	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/roles", strings.NewReader("name=qa_reviewer"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleOrgAdminRoles(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "failed to create role") {
		t.Fatalf("expected failure message, got %q", rec.Body.String())
	}
}

func TestRenderAdminTemplatesErrorPaths(t *testing.T) {
	store := NewMemoryStore()
	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Render Org"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	orgID := org.ID
	user := &AccountUser{
		UserID:          "render-user",
		OrgID:           &orgID,
		OrgSlug:         org.Slug,
		RoleSlugs:       []string{"org-admin"},
		IsPlatformAdmin: true,
	}
	server := &Server{store: store, tmpl: template.Must(template.New("broken").Parse("broken"))}

	recOrg := httptest.NewRecorder()
	server.renderOrgAdmin(recOrg, user, org.Slug, "", "")
	if recOrg.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recOrg.Code, http.StatusInternalServerError)
	}

	recMissing := httptest.NewRecorder()
	server.renderOrgAdmin(recMissing, user, "missing-org", "", "")
	if recMissing.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recMissing.Code, http.StatusNotFound)
	}

	recPlatform := httptest.NewRecorder()
	server.renderPlatformAdmin(recPlatform, user, "", "")
	if recPlatform.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recPlatform.Code, http.StatusInternalServerError)
	}
}
