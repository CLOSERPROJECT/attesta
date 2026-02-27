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
	failSetUserRoles       bool
	failDisableUser        bool
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

func (s *adminFailingStore) SetUserRoles(ctx context.Context, userID string, roleSlugs []string) error {
	if s.failSetUserRoles {
		return errors.New("set user roles failed")
	}
	return s.MemoryStore.SetUserRoles(ctx, userID, roleSlugs)
}

func (s *adminFailingStore) DisableUser(ctx context.Context, userID string) error {
	if s.failDisableUser {
		return errors.New("disable user failed")
	}
	return s.MemoryStore.DisableUser(ctx, userID)
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

func TestHandleAdminOrgsDuplicateSlugRendersExplicitError(t *testing.T) {
	store := NewMemoryStore()
	admin, err := store.CreateUser(t.Context(), AccountUser{
		UserID:          "platform-admin-dup-org",
		Email:           "platform-dup-org@example.com",
		IsPlatformAdmin: true,
		Status:          "active",
		CreatedAt:       time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	sessionID := createSessionForTestUser(t, store, admin)
	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}

	firstReq := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader("name=Acme+Org"))
	firstReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	firstReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	firstRec := httptest.NewRecorder()
	server.handleAdminOrgs(firstRec, firstReq)
	if firstRec.Code != http.StatusSeeOther {
		t.Fatalf("first create status = %d, want %d", firstRec.Code, http.StatusSeeOther)
	}

	dupReq := httptest.NewRequest(http.MethodPost, "/admin/orgs", strings.NewReader("name=Acme_Org"))
	dupReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	dupReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	dupRec := httptest.NewRecorder()
	server.handleAdminOrgs(dupRec, dupReq)
	if dupRec.Code != http.StatusOK {
		t.Fatalf("duplicate create status = %d, want %d", dupRec.Code, http.StatusOK)
	}
	if !strings.Contains(dupRec.Body.String(), "already exists") {
		t.Fatalf("expected duplicate slug message, got %q", dupRec.Body.String())
	}

	orgs, err := store.ListOrganizations(t.Context())
	if err != nil {
		t.Fatalf("ListOrganizations error: %v", err)
	}
	if len(orgs) != 1 {
		t.Fatalf("org count = %d, want %d", len(orgs), 1)
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

	userReq := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=user%40acme.org&roles=qa-reviewer"))
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

func TestHandleOrgAdminRolesDuplicateSlugRendersExplicitError(t *testing.T) {
	store := NewMemoryStore()
	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme Org"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "Org Admin", Slug: "org-admin", CreatedAt: time.Now().UTC()})
	orgID := org.ID
	adminUser, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "org-admin-dup-role",
		OrgID:     &orgID,
		OrgSlug:   org.Slug,
		Email:     "org-admin-dup-role@acme.org",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	sessionID := createSessionForTestUser(t, store, adminUser)
	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}

	firstReq := httptest.NewRequest(http.MethodPost, "/org-admin/roles", strings.NewReader("name=QA+Reviewer"))
	firstReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	firstReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	firstRec := httptest.NewRecorder()
	server.handleOrgAdminRoles(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first create status = %d, want %d", firstRec.Code, http.StatusOK)
	}

	dupReq := httptest.NewRequest(http.MethodPost, "/org-admin/roles", strings.NewReader("name=QA_reviewer"))
	dupReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	dupReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	dupRec := httptest.NewRecorder()
	server.handleOrgAdminRoles(dupRec, dupReq)
	if dupRec.Code != http.StatusOK {
		t.Fatalf("duplicate create status = %d, want %d", dupRec.Code, http.StatusOK)
	}
	if !strings.Contains(dupRec.Body.String(), "already exists") {
		t.Fatalf("expected duplicate role message, got %q", dupRec.Body.String())
	}

	roles, err := store.ListRolesByOrg(t.Context(), org.Slug)
	if err != nil {
		t.Fatalf("ListRolesByOrg error: %v", err)
	}
	count := 0
	for _, role := range roles {
		if role.Slug == "qa-reviewer" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("qa-reviewer count = %d, want %d", count, 1)
	}
}

func TestHandleOrgAdminUsersGetRendersInviteAndUserCollections(t *testing.T) {
	store := NewMemoryStore()
	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme Org"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "Org Admin", Slug: "org-admin", CreatedAt: time.Now().UTC()})
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "QA Reviewer", Slug: "qa-reviewer", CreatedAt: time.Now().UTC()})
	orgID := org.ID
	adminUser, err := store.CreateUser(t.Context(), AccountUser{
		UserID:       "org-admin-counts",
		OrgID:        &orgID,
		OrgSlug:      org.Slug,
		Email:        "org-admin-counts@acme.org",
		PasswordHash: "hash-admin",
		RoleSlugs:    []string{"org-admin"},
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser admin error: %v", err)
	}
	if _, err := store.CreateUser(t.Context(), AccountUser{
		UserID:       "active-user",
		OrgID:        &orgID,
		OrgSlug:      org.Slug,
		Email:        "active-user@acme.org",
		PasswordHash: "hash-active",
		RoleSlugs:    []string{"qa-reviewer"},
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateUser active error: %v", err)
	}
	if _, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "deleted-user",
		OrgID:     &orgID,
		OrgSlug:   org.Slug,
		Email:     "deleted-user@acme.org",
		RoleSlugs: []string{"qa-reviewer"},
		Status:    "deleted",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateUser deleted error: %v", err)
	}
	now := time.Now().UTC()
	if _, err := store.CreateInvite(t.Context(), Invite{
		OrgID:           org.ID,
		Email:           "invitee@acme.org",
		UserID:          "invited-1",
		RoleSlugs:       []string{"qa-reviewer"},
		TokenHash:       "token-1",
		ExpiresAt:       now.Add(24 * time.Hour),
		CreatedAt:       now,
		CreatedByUserID: adminUser.UserID,
	}); err != nil {
		t.Fatalf("CreateInvite primary error: %v", err)
	}
	if _, err := store.CreateInvite(t.Context(), Invite{
		OrgID:           org.ID,
		Email:           "other@acme.org",
		UserID:          "invited-2",
		RoleSlugs:       []string{"qa-reviewer"},
		TokenHash:       "token-2",
		ExpiresAt:       now.Add(24 * time.Hour),
		CreatedAt:       now,
		CreatedByUserID: "someone-else",
	}); err != nil {
		t.Fatalf("CreateInvite secondary error: %v", err)
	}

	sessionID := createSessionForTestUser(t, store, adminUser)
	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}
	req := httptest.NewRequest(http.MethodGet, "/org-admin/users", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "INVITES 1 USERS 2") {
		t.Fatalf("expected invite/user counts in response, got %q", rec.Body.String())
	}
}

func TestHandleOrgAdminUsersInviteAllowsNoRoles(t *testing.T) {
	store := NewMemoryStore()
	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme Org"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "Org Admin", Slug: "org-admin", CreatedAt: time.Now().UTC()})
	orgID := org.ID
	adminUser, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "org-admin-no-roles",
		OrgID:     &orgID,
		OrgSlug:   org.Slug,
		Email:     "org-admin-no-roles@acme.org",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	sessionID := createSessionForTestUser(t, store, adminUser)
	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=user%40acme.org"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "/invite/") {
		t.Fatalf("expected invite link in response, got %q", rec.Body.String())
	}
	user, err := store.GetUserByEmail(t.Context(), "user@acme.org")
	if err != nil {
		t.Fatalf("GetUserByEmail error: %v", err)
	}
	if len(user.RoleSlugs) != 0 {
		t.Fatalf("expected invited user to have no roles, got %#v", user.RoleSlugs)
	}
	invites, err := store.ListInvitesByCreator(t.Context(), adminUser.UserID, org.ID)
	if err != nil {
		t.Fatalf("ListInvitesByCreator error: %v", err)
	}
	if len(invites) != 1 {
		t.Fatalf("invite count = %d, want %d", len(invites), 1)
	}
	if len(invites[0].RoleSlugs) != 0 {
		t.Fatalf("expected invite with no roles, got %#v", invites[0].RoleSlugs)
	}
}

func TestHandleOrgAdminUsersInviteRejectsExistingEmailFromAnotherOrg(t *testing.T) {
	store := NewMemoryStore()
	orgA, err := store.CreateOrganization(t.Context(), Organization{Name: "Org A"})
	if err != nil {
		t.Fatalf("CreateOrganization orgA error: %v", err)
	}
	orgB, err := store.CreateOrganization(t.Context(), Organization{Name: "Org B"})
	if err != nil {
		t.Fatalf("CreateOrganization orgB error: %v", err)
	}
	_, _ = store.CreateRole(t.Context(), Role{OrgID: orgA.ID, OrgSlug: orgA.Slug, Name: "Org Admin", Slug: "org-admin", CreatedAt: time.Now().UTC()})
	_, _ = store.CreateRole(t.Context(), Role{OrgID: orgA.ID, OrgSlug: orgA.Slug, Name: "QA Reviewer", Slug: "qa-reviewer", CreatedAt: time.Now().UTC()})
	_, _ = store.CreateRole(t.Context(), Role{OrgID: orgB.ID, OrgSlug: orgB.Slug, Name: "QA Reviewer", Slug: "qa-reviewer", CreatedAt: time.Now().UTC()})
	orgAID := orgA.ID
	orgBID := orgB.ID
	adminUser, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "org-admin-cross-org",
		OrgID:     &orgAID,
		OrgSlug:   orgA.Slug,
		Email:     "org-admin-cross-org@orga.org",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser admin error: %v", err)
	}
	if _, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "existing-user",
		OrgID:     &orgBID,
		OrgSlug:   orgB.Slug,
		Email:     "existing@shared.org",
		RoleSlugs: []string{"qa-reviewer"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateUser existing error: %v", err)
	}

	sessionID := createSessionForTestUser(t, store, adminUser)
	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}
	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=existing%40shared.org&roles=qa-reviewer"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "email already belongs to another organization") {
		t.Fatalf("expected cross-org email error, got %q", rec.Body.String())
	}
	invites, err := store.ListInvitesByCreator(t.Context(), adminUser.UserID, orgA.ID)
	if err != nil {
		t.Fatalf("ListInvitesByCreator error: %v", err)
	}
	if len(invites) != 0 {
		t.Fatalf("unexpected invites for cross-org email: %#v", invites)
	}
}

func TestHandleOrgAdminUsersInviteExistingUserMergesRoles(t *testing.T) {
	store := NewMemoryStore()
	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Merge Org"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "Org Admin", Slug: "org-admin", CreatedAt: time.Now().UTC()})
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "QA Reviewer", Slug: "qa-reviewer", CreatedAt: time.Now().UTC()})
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "Approver", Slug: "approver", CreatedAt: time.Now().UTC()})
	orgID := org.ID
	adminUser, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "org-admin-merge",
		OrgID:     &orgID,
		OrgSlug:   org.Slug,
		Email:     "org-admin-merge@acme.org",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser admin error: %v", err)
	}
	if _, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "existing-user",
		OrgID:     &orgID,
		OrgSlug:   org.Slug,
		Email:     "existing-user@acme.org",
		RoleSlugs: []string{"qa-reviewer"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateUser existing error: %v", err)
	}

	sessionID := createSessionForTestUser(t, store, adminUser)
	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}
	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=invite&email=existing-user%40acme.org&roles=approver"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleOrgAdminUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	user, err := store.GetUserByEmail(t.Context(), "existing-user@acme.org")
	if err != nil {
		t.Fatalf("GetUserByEmail error: %v", err)
	}
	if !containsRole(user.RoleSlugs, "qa-reviewer") || !containsRole(user.RoleSlugs, "approver") {
		t.Fatalf("expected merged role slugs, got %#v", user.RoleSlugs)
	}
	invites, err := store.ListInvitesByCreator(t.Context(), adminUser.UserID, org.ID)
	if err != nil {
		t.Fatalf("ListInvitesByCreator error: %v", err)
	}
	if len(invites) != 1 {
		t.Fatalf("invite count = %d, want %d", len(invites), 1)
	}
	if len(invites[0].RoleSlugs) != 1 || invites[0].RoleSlugs[0] != "approver" {
		t.Fatalf("invite roles = %#v, want [approver]", invites[0].RoleSlugs)
	}
}

func TestHandleOrgAdminUsersSetRolesIntentUpdatesUserAndProtectsSelf(t *testing.T) {
	store := NewMemoryStore()
	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Roles Org"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "Org Admin", Slug: "org-admin", CreatedAt: time.Now().UTC()})
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "QA Reviewer", Slug: "qa-reviewer", CreatedAt: time.Now().UTC()})
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "Approver", Slug: "approver", CreatedAt: time.Now().UTC()})
	orgID := org.ID
	adminUser, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "org-admin-set-roles",
		OrgID:     &orgID,
		OrgSlug:   org.Slug,
		Email:     "org-admin-set-roles@acme.org",
		RoleSlugs: []string{"org-admin", "qa-reviewer"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser admin error: %v", err)
	}
	if _, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "target-user",
		OrgID:     &orgID,
		OrgSlug:   org.Slug,
		Email:     "target-user@acme.org",
		RoleSlugs: []string{"qa-reviewer"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateUser target error: %v", err)
	}

	sessionID := createSessionForTestUser(t, store, adminUser)
	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}

	updateReq := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=set_roles&userId=target-user&roles=approver"))
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	updateRec := httptest.NewRecorder()
	server.handleOrgAdminUsers(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("set_roles status = %d, want %d", updateRec.Code, http.StatusOK)
	}
	target, err := store.GetUserByUserID(t.Context(), "target-user")
	if err != nil {
		t.Fatalf("GetUserByUserID target error: %v", err)
	}
	if len(target.RoleSlugs) != 1 || target.RoleSlugs[0] != "approver" {
		t.Fatalf("target roles = %#v, want [approver]", target.RoleSlugs)
	}

	selfReq := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=set_roles&userId=org-admin-set-roles&roles=qa-reviewer"))
	selfReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	selfReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	selfRec := httptest.NewRecorder()
	server.handleOrgAdminUsers(selfRec, selfReq)
	if selfRec.Code != http.StatusOK {
		t.Fatalf("self set_roles status = %d, want %d", selfRec.Code, http.StatusOK)
	}
	if !strings.Contains(selfRec.Body.String(), "cannot remove org-admin from your own account") {
		t.Fatalf("expected self org-admin protection message, got %q", selfRec.Body.String())
	}
}

func TestHandleOrgAdminUsersDeleteUserIntentDisablesUserAndRejectsSelf(t *testing.T) {
	store := NewMemoryStore()
	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Delete Org"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "Org Admin", Slug: "org-admin", CreatedAt: time.Now().UTC()})
	_, _ = store.CreateRole(t.Context(), Role{OrgID: org.ID, OrgSlug: org.Slug, Name: "QA Reviewer", Slug: "qa-reviewer", CreatedAt: time.Now().UTC()})
	orgID := org.ID
	adminUser, err := store.CreateUser(t.Context(), AccountUser{
		UserID:       "org-admin-delete",
		OrgID:        &orgID,
		OrgSlug:      org.Slug,
		Email:        "org-admin-delete@acme.org",
		PasswordHash: "hash-admin",
		RoleSlugs:    []string{"org-admin"},
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser admin error: %v", err)
	}
	if _, err := store.CreateUser(t.Context(), AccountUser{
		UserID:       "delete-target",
		OrgID:        &orgID,
		OrgSlug:      org.Slug,
		Email:        "delete-target@acme.org",
		PasswordHash: "hash-target",
		RoleSlugs:    []string{"qa-reviewer"},
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateUser target error: %v", err)
	}

	sessionID := createSessionForTestUser(t, store, adminUser)
	server := &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}

	deleteReq := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=delete_user&userId=delete-target"))
	deleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deleteReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	deleteRec := httptest.NewRecorder()
	server.handleOrgAdminUsers(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete_user status = %d, want %d", deleteRec.Code, http.StatusOK)
	}
	deleted, err := store.GetUserByUserID(t.Context(), "delete-target")
	if err != nil {
		t.Fatalf("GetUserByUserID deleted error: %v", err)
	}
	if deleted.Status != "deleted" || deleted.PasswordHash != "" || len(deleted.RoleSlugs) != 0 {
		t.Fatalf("deleted account not disabled as expected: %#v", deleted)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/org-admin/users", nil)
	getReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	getRec := httptest.NewRecorder()
	server.handleOrgAdminUsers(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}
	if !strings.Contains(getRec.Body.String(), "USERS 1") {
		t.Fatalf("expected deleted user hidden from org admin view, got %q", getRec.Body.String())
	}

	selfReq := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("intent=delete_user&userId=org-admin-delete"))
	selfReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	selfReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	selfRec := httptest.NewRecorder()
	server.handleOrgAdminUsers(selfRec, selfReq)
	if selfRec.Code != http.StatusOK {
		t.Fatalf("self delete status = %d, want %d", selfRec.Code, http.StatusOK)
	}
	if !strings.Contains(selfRec.Body.String(), "cannot delete yourself") {
		t.Fatalf("expected self-delete protection message, got %q", selfRec.Body.String())
	}
}

func TestHandleOrgAdminUsersIntentErrorPaths(t *testing.T) {
	base := NewMemoryStore()
	orgA, err := base.CreateOrganization(t.Context(), Organization{Name: "Org A"})
	if err != nil {
		t.Fatalf("CreateOrganization orgA error: %v", err)
	}
	orgB, err := base.CreateOrganization(t.Context(), Organization{Name: "Org B"})
	if err != nil {
		t.Fatalf("CreateOrganization orgB error: %v", err)
	}
	_, _ = base.CreateRole(t.Context(), Role{OrgID: orgA.ID, OrgSlug: orgA.Slug, Name: "Org Admin", Slug: "org-admin", CreatedAt: time.Now().UTC()})
	_, _ = base.CreateRole(t.Context(), Role{OrgID: orgA.ID, OrgSlug: orgA.Slug, Name: "QA Reviewer", Slug: "qa-reviewer", CreatedAt: time.Now().UTC()})
	_, _ = base.CreateRole(t.Context(), Role{OrgID: orgB.ID, OrgSlug: orgB.Slug, Name: "QA Reviewer", Slug: "qa-reviewer", CreatedAt: time.Now().UTC()})

	orgAID := orgA.ID
	orgBID := orgB.ID
	adminUser, err := base.CreateUser(t.Context(), AccountUser{
		UserID:    "org-admin-errors",
		OrgID:     &orgAID,
		OrgSlug:   orgA.Slug,
		Email:     "org-admin-errors@orga.org",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser admin error: %v", err)
	}
	if _, err := base.CreateUser(t.Context(), AccountUser{
		UserID:    "same-org-user",
		OrgID:     &orgAID,
		OrgSlug:   orgA.Slug,
		Email:     "same-org-user@orga.org",
		RoleSlugs: []string{"qa-reviewer"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateUser same-org error: %v", err)
	}
	if _, err := base.CreateUser(t.Context(), AccountUser{
		UserID:    "other-org-user",
		OrgID:     &orgBID,
		OrgSlug:   orgB.Slug,
		Email:     "other-org-user@orgb.org",
		RoleSlugs: []string{"qa-reviewer"},
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateUser other-org error: %v", err)
	}
	sessionID := createSessionForTestUser(t, base, adminUser)

	newServer := func(store Store) *Server {
		return &Server{store: store, tmpl: testTemplates(), enforceAuth: true, now: time.Now}
	}
	newPost := func(payload string) (*http.Request, *httptest.ResponseRecorder) {
		req := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		return req, httptest.NewRecorder()
	}

	t.Run("unsupported action", func(t *testing.T) {
		server := newServer(base)
		req, rec := newPost("intent=nope")
		server.handleOrgAdminUsers(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if !strings.Contains(rec.Body.String(), "unsupported action") {
			t.Fatalf("expected unsupported action error, got %q", rec.Body.String())
		}
	})

	t.Run("set roles missing user id", func(t *testing.T) {
		server := newServer(base)
		req, rec := newPost("intent=set_roles")
		server.handleOrgAdminUsers(rec, req)
		if !strings.Contains(rec.Body.String(), "user is required") {
			t.Fatalf("expected user required error, got %q", rec.Body.String())
		}
	})

	t.Run("set roles user not found", func(t *testing.T) {
		server := newServer(base)
		req, rec := newPost("intent=set_roles&userId=missing&roles=qa-reviewer")
		server.handleOrgAdminUsers(rec, req)
		if !strings.Contains(rec.Body.String(), "user not found") {
			t.Fatalf("expected user not found error, got %q", rec.Body.String())
		}
	})

	t.Run("set roles cross org user", func(t *testing.T) {
		server := newServer(base)
		req, rec := newPost("intent=set_roles&userId=other-org-user&roles=qa-reviewer")
		server.handleOrgAdminUsers(rec, req)
		if !strings.Contains(rec.Body.String(), "user does not belong to your organization") {
			t.Fatalf("expected cross-org user error, got %q", rec.Body.String())
		}
	})

	t.Run("set roles role not found", func(t *testing.T) {
		server := newServer(base)
		req, rec := newPost("intent=set_roles&userId=same-org-user&roles=missing-role")
		server.handleOrgAdminUsers(rec, req)
		if !strings.Contains(rec.Body.String(), "role not found") {
			t.Fatalf("expected role not found error, got %q", rec.Body.String())
		}
	})

	t.Run("set roles update failure", func(t *testing.T) {
		server := newServer(&adminFailingStore{MemoryStore: base, failSetUserRoles: true})
		req, rec := newPost("intent=set_roles&userId=same-org-user&roles=qa-reviewer")
		server.handleOrgAdminUsers(rec, req)
		if !strings.Contains(rec.Body.String(), "failed to update user roles") {
			t.Fatalf("expected set roles failure message, got %q", rec.Body.String())
		}
	})

	t.Run("delete user missing id", func(t *testing.T) {
		server := newServer(base)
		req, rec := newPost("intent=delete_user")
		server.handleOrgAdminUsers(rec, req)
		if !strings.Contains(rec.Body.String(), "user is required") {
			t.Fatalf("expected user required error, got %q", rec.Body.String())
		}
	})

	t.Run("delete user not found", func(t *testing.T) {
		server := newServer(base)
		req, rec := newPost("intent=delete_user&userId=missing")
		server.handleOrgAdminUsers(rec, req)
		if !strings.Contains(rec.Body.String(), "user not found") {
			t.Fatalf("expected user not found error, got %q", rec.Body.String())
		}
	})

	t.Run("delete user cross org", func(t *testing.T) {
		server := newServer(base)
		req, rec := newPost("intent=delete_user&userId=other-org-user")
		server.handleOrgAdminUsers(rec, req)
		if !strings.Contains(rec.Body.String(), "user does not belong to your organization") {
			t.Fatalf("expected cross-org delete error, got %q", rec.Body.String())
		}
	})

	t.Run("delete user failure", func(t *testing.T) {
		server := newServer(&adminFailingStore{MemoryStore: base, failDisableUser: true})
		req, rec := newPost("intent=delete_user&userId=same-org-user")
		server.handleOrgAdminUsers(rec, req)
		if !strings.Contains(rec.Body.String(), "failed to delete user") {
			t.Fatalf("expected delete failure error, got %q", rec.Body.String())
		}
	})

	t.Run("invite update roles failure", func(t *testing.T) {
		server := newServer(&adminFailingStore{MemoryStore: base, failSetUserRoles: true})
		req, rec := newPost("intent=invite&email=same-org-user%40orga.org&roles=qa-reviewer")
		server.handleOrgAdminUsers(rec, req)
		if !strings.Contains(rec.Body.String(), "failed to update user roles") {
			t.Fatalf("expected invite update roles failure message, got %q", rec.Body.String())
		}
	})
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

	userReq := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("email="))
	userReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	userReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	userRec := httptest.NewRecorder()
	server.handleOrgAdminUsers(userRec, userReq)
	if userRec.Code != http.StatusOK {
		t.Fatalf("user status = %d, want %d", userRec.Code, http.StatusOK)
	}
	if !strings.Contains(userRec.Body.String(), "email is required") {
		t.Fatalf("expected email validation error, got %q", userRec.Body.String())
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

		reqRoleMissing := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("email=u%40x.io&roles=missing-role"))
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
		reqCreateUserFail := httptest.NewRequest(http.MethodPost, "/org-admin/users", strings.NewReader("email=new%40x.io&roles=qa-reviewer"))
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
