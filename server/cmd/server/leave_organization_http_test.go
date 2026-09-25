package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleLeaveOrganizationSuccessRedirectsOnboarding(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-leave-ok"
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
	var deletedMembershipID string
	identity := testIdentityForSessions(now, sessions)
	identity.getCurrentUserFunc = func(_ context.Context, sessionSecret string) (IdentityUser, error) {
		user, ok := sessions[strings.TrimSpace(sessionSecret)]
		if !ok {
			return IdentityUser{}, ErrIdentityUnauthorized
		}
		out := identityUserFromAccountUser(user)
		out.MembershipID = "mem-member"
		return out, nil
	}
	identity.deleteOrganizationMembershipFunc = func(_ context.Context, _, _, membershipID string) error {
		deletedMembershipID = membershipID
		u := sessions[sessionID]
		u.OrgSlug = ""
		u.RoleSlugs = nil
		sessions[sessionID] = u
		return nil
	}
	identity.getUserByIDFunc = func(_ context.Context, userID string) (IdentityUser, error) {
		return IdentityUser{ID: userID, Email: "member@example.com", Labels: []string{encodeIdentityRoleLabel("viewer")}}, nil
	}
	identity.updateUserLabelsFunc = func(_ context.Context, userID string, labels []string) (IdentityUser, error) {
		return IdentityUser{ID: userID, Labels: labels}, nil
	}

	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, leaveOrganizationPath(), nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != onboardingPath() {
		t.Fatalf("location = %q, want %q", loc, onboardingPath())
	}
	if deletedMembershipID != "mem-member" {
		t.Fatalf("deleted membership = %q", deletedMembershipID)
	}
}

func TestHandleLeaveOrganizationSoleAdminRedirectsHomeWithError(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-leave-sole"
	account := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "admin-1",
		Email:          "owner@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"org-admin"},
		Status:         "active",
		CreatedAt:      now,
	}
	sessions := map[string]AccountUser{sessionID: account}
	var deleteCalled bool
	identity := testIdentityForSessions(now, sessions)
	identity.getCurrentUserFunc = func(_ context.Context, sessionSecret string) (IdentityUser, error) {
		user, ok := sessions[strings.TrimSpace(sessionSecret)]
		if !ok {
			return IdentityUser{}, ErrIdentityUnauthorized
		}
		out := identityUserFromAccountUser(user)
		out.MembershipID = "mem-admin"
		return out, nil
	}
	identity.listOrganizationUsersFunc = func(_ context.Context, _ string) ([]IdentityUser, error) {
		return []IdentityUser{{ID: "admin-1", OrgSlug: "acme", IsOrgAdmin: true}}, nil
	}
	identity.deleteOrganizationMembershipFunc = func(_ context.Context, _, _, _ string) error {
		deleteCalled = true
		return nil
	}

	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, leaveOrganizationPath(), nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, appHomePath+"?") {
		t.Fatalf("location = %q, want /my?error=…", loc)
	}
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	if got := parsed.Query().Get("error"); !strings.Contains(got, "only Org admin") {
		t.Fatalf("error query = %q", got)
	}
	if deleteCalled {
		t.Fatal("sole admin leave must not delete membership")
	}
}

func TestHandleLeaveOrganizationMethodAndNotAffiliated(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-leave-edges"
	account := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "solo-1",
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

	getReq := httptest.NewRequest(http.MethodGet, leaveOrganizationPath(), nil)
	getReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	getRec := httptest.NewRecorder()
	server.handleMyRoutes(getRec, getReq)
	if getRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("get status=%d", getRec.Code)
	}

	postReq := httptest.NewRequest(http.MethodPost, leaveOrganizationPath(), nil)
	postReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	postRec := httptest.NewRecorder()
	server.handleMyRoutes(postRec, postReq)
	if postRec.Code != http.StatusSeeOther || postRec.Header().Get("Location") != onboardingPath() {
		t.Fatalf("unaffiliated leave status=%d loc=%q", postRec.Code, postRec.Header().Get("Location"))
	}
}

func TestHandleLeaveOrganizationGenericError(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-leave-err"
	account := AccountUser{
		ID:             primitive.NewObjectID(),
		IdentityUserID: "member-err",
		Email:          "member-err@example.com",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"viewer"},
		Status:         "active",
		CreatedAt:      now,
	}
	sessions := map[string]AccountUser{sessionID: account}
	identity := testIdentityForSessions(now, sessions)
	identity.getCurrentUserFunc = func(_ context.Context, sessionSecret string) (IdentityUser, error) {
		user, ok := sessions[strings.TrimSpace(sessionSecret)]
		if !ok {
			return IdentityUser{}, ErrIdentityUnauthorized
		}
		out := identityUserFromAccountUser(user)
		out.MembershipID = "mem-1"
		return out, nil
	}
	identity.deleteOrganizationMembershipFunc = func(_ context.Context, _, _, _ string) error {
		return errors.New("appwrite down")
	}

	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, leaveOrganizationPath(), nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := parsed.Query().Get("error"); !strings.Contains(got, "failed to leave organization") {
		t.Fatalf("error=%q loc=%q", got, loc)
	}
}
