package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

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
