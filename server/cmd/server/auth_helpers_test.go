package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestEnvAndCookieHelpers(t *testing.T) {
	t.Setenv("INT_TEST_VALUE", "")
	if got := intEnvOr("INT_TEST_VALUE", 42); got != 42 {
		t.Fatalf("intEnvOr fallback = %d, want 42", got)
	}
	t.Setenv("INT_TEST_VALUE", "bad")
	if got := intEnvOr("INT_TEST_VALUE", 42); got != 42 {
		t.Fatalf("intEnvOr bad parse fallback = %d, want 42", got)
	}
	t.Setenv("INT_TEST_VALUE", "7")
	if got := intEnvOr("INT_TEST_VALUE", 42); got != 7 {
		t.Fatalf("intEnvOr parsed = %d, want 7", got)
	}

	t.Setenv("SESSION_TTL_DAYS", "0")
	if got := sessionTTLDays(); got != 30 {
		t.Fatalf("sessionTTLDays zero = %d, want 30", got)
	}
	t.Setenv("SESSION_TTL_DAYS", "14")
	if got := sessionTTLDays(); got != 14 {
		t.Fatalf("sessionTTLDays value = %d, want 14", got)
	}
	t.Setenv("RESET_TTL_HOURS", "0")
	if got := resetTTLHrs(); got != 24 {
		t.Fatalf("resetTTLHrs zero = %d, want 24", got)
	}
	t.Setenv("RESET_TTL_HOURS", "48")
	if got := resetTTLHrs(); got != 48 {
		t.Fatalf("resetTTLHrs value = %d, want 48", got)
	}

	t.Setenv("COOKIE_SECURE", "false")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if shouldSecureCookie(req) {
		t.Fatal("expected insecure cookie by default")
	}
	req.Header.Set("X-Forwarded-Proto", "https")
	if !shouldSecureCookie(req) {
		t.Fatal("expected secure cookie for forwarded https")
	}
	t.Setenv("COOKIE_SECURE", "true")
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	if !shouldSecureCookie(req2) {
		t.Fatal("expected secure cookie when COOKIE_SECURE=true")
	}

}

func TestReadSessionAndCurrentUser(t *testing.T) {
	now := time.Date(2026, 2, 26, 20, 0, 0, 0, time.UTC)
	var deletedSecrets []string
	server := &Server{
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				switch sessionSecret {
				case "session-valid":
					return fakeIdentitySession(sessionSecret, "user-1", now.Add(2*time.Hour)), nil
				case "session-expired":
					return fakeIdentitySession(sessionSecret, "user-1", now.Add(-1*time.Hour)), nil
				default:
					return IdentitySession{}, ErrIdentityUnauthorized
				}
			},
			deleteSessionFunc: func(ctx context.Context, sessionSecret string) error {
				deletedSecrets = append(deletedSecrets, sessionSecret)
				return nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				if sessionSecret != "session-valid" {
					return IdentityUser{}, ErrIdentityUnauthorized
				}
				return IdentityUser{
					ID:         "user-1",
					Email:      "u-session@example.com",
					OrgSlug:    "acme",
					Labels:     []string{identityOrgAdminLabel, encodeIdentityRoleLabel("qa-reviewer")},
					IsOrgAdmin: true,
					Status:     "active",
				}, nil
			},
		},
		now: func() time.Time { return now },
	}

	t.Run("missing cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if _, err := server.readSession(req); err == nil {
			t.Fatal("expected missing cookie error")
		}
	})

	t.Run("blank cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "  "})
		if _, err := server.readSession(req); err == nil {
			t.Fatal("expected blank cookie to fail")
		}
	})

	t.Run("expired cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-expired"})
		if _, err := server.readSession(req); err == nil {
			t.Fatal("expected expired session error")
		}
		if len(deletedSecrets) != 1 || deletedSecrets[0] != "session-expired" {
			t.Fatalf("deleted secrets = %#v, want [session-expired]", deletedSecrets)
		}
	})

	t.Run("current user success and failure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-valid"})
		gotUser, gotSession, err := server.currentUser(req)
		if err != nil {
			t.Fatalf("currentUser error: %v", err)
		}
		if gotUser.Email != "u-session@example.com" || gotUser.OrgSlug != "acme" || !containsRole(gotUser.RoleSlugs, "org-admin") || gotSession.Secret != "session-valid" {
			t.Fatalf("unexpected currentUser result user=%+v session=%+v", gotUser, gotSession)
		}

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-missing"})
		if _, _, err := server.currentUser(req2); err == nil {
			t.Fatal("expected currentUser error for invalid session")
		}
	})
}

func TestUserOrgAdminAndPageBaseFlags(t *testing.T) {
	orgID := primitive.NewObjectID()
	admin := &AccountUser{
		IsPlatformAdmin: true,
		RoleSlugs:       []string{"org-admin"},
		OrgSlug:         "acme",
		OrgID:           &orgID,
	}
	if !userIsOrgAdmin(admin) {
		t.Fatal("expected org admin user")
	}

	baseServer := &Server{}
	page := baseServer.pageBaseForUser(admin, "dashboard_body", "workflow", "Workflow")
	if !page.ShowOrgsLink || !page.ShowMyOrgLink {
		t.Fatalf("expected both nav flags true, got %+v", page)
	}

	unassigned := &AccountUser{
		RoleSlugs: []string{"org-admin"},
	}
	if !userIsOrgAdmin(unassigned) {
		t.Fatal("expected unassigned org admin user")
	}
	pageUnassigned := baseServer.pageBaseForUser(unassigned, "dashboard_body", "", "")
	if !pageUnassigned.ShowMyOrgLink {
		t.Fatalf("expected my-org nav flag for unassigned org admin, got %+v", pageUnassigned)
	}

	non := &AccountUser{RoleSlugs: []string{"dep1"}}
	if userIsOrgAdmin(non) {
		t.Fatal("expected non org admin user")
	}
	page2 := baseServer.pageBaseForUser(non, "dashboard_body", "", "")
	if page2.ShowOrgsLink || page2.ShowMyOrgLink {
		t.Fatalf("expected nav flags false, got %+v", page2)
	}
}

func TestRequireAuthenticatedAndOrgAdminGuards(t *testing.T) {
	server := &Server{
		store:       NewMemoryStore(),
		enforceAuth: false,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	user, _, ok := server.requireAuthenticatedPage(rec, req)
	if !ok || !user.ID.IsZero() {
		t.Fatalf("expected legacy auth user, got ok=%v user=%+v", ok, user)
	}

	now := time.Now().UTC()
	authServer := &Server{
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				if sessionSecret != "plain-session" {
					return IdentitySession{}, ErrIdentityUnauthorized
				}
				return fakeIdentitySession(sessionSecret, "user-plain", now.Add(24*time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{
					ID:      "user-plain",
					Email:   "plain@example.com",
					OrgSlug: "acme",
					Labels:  []string{encodeIdentityRoleLabel("dep1")},
					Status:  "active",
				}, nil
			},
		},
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/org-admin/users", nil)
	req2.AddCookie(&http.Cookie{Name: "attesta_session", Value: "plain-session"})
	if _, ok := authServer.requireOrgAdmin(rec2, req2); ok {
		t.Fatal("expected requireOrgAdmin to reject non-org-admin")
	}
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec2.Code, http.StatusForbidden)
	}

	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/process/start", nil)
	if _, _, ok := authServer.requireAuthenticatedPost(rec3, req3); ok {
		t.Fatal("expected requireAuthenticatedPost to fail without cookie")
	}
	if rec3.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec3.Code, http.StatusUnauthorized)
	}

}

func TestIsDuplicateSlugError(t *testing.T) {
	if isDuplicateSlugError(nil) {
		t.Fatal("expected nil error to be non-duplicate")
	}

	dupErr := mongo.WriteException{
		WriteErrors: []mongo.WriteError{{Code: 11000, Message: "E11000 duplicate key"}},
	}
	if !isDuplicateSlugError(dupErr) {
		t.Fatal("expected mongo duplicate-key error to be recognized")
	}

	if !isDuplicateSlugError(errors.New("slug already exists")) {
		t.Fatal("expected slug conflict message to be recognized")
	}
	if !isDuplicateSlugError(errors.New("role already exists")) {
		t.Fatal("expected role conflict message to be recognized")
	}
	if !isDuplicateSlugError(errors.New("duplicate key")) {
		t.Fatal("expected duplicate key message to be recognized")
	}
	if isDuplicateSlugError(errors.New("something else")) {
		t.Fatal("expected unrelated error to be non-duplicate")
	}
}

func TestRequestedRoleSlugs(t *testing.T) {
	form := url.Values{}
	form["roles"] = []string{" org-admin ", "org_admin", "dep1", "dep1"}
	got := requestedRoleSlugs(form)
	if len(got) != 2 || got[0] != "org-admin" || got[1] != "dep1" {
		t.Fatalf("requestedRoleSlugs roles = %#v, want [org-admin dep1]", got)
	}

	legacy := url.Values{}
	legacy.Set("role", " dep2 ")
	got = requestedRoleSlugs(legacy)
	if len(got) != 1 || got[0] != "dep2" {
		t.Fatalf("requestedRoleSlugs legacy role = %#v, want [dep2]", got)
	}

	empty := requestedRoleSlugs(url.Values{})
	if len(empty) != 0 {
		t.Fatalf("requestedRoleSlugs empty = %#v, want []", empty)
	}
}

func TestAccountMatchesOrg(t *testing.T) {
	orgID := primitive.NewObjectID()
	otherID := primitive.NewObjectID()
	if accountMatchesOrg(nil, orgID, "acme") {
		t.Fatal("expected nil user mismatch")
	}
	if accountMatchesOrg(&AccountUser{}, orgID, "acme") {
		t.Fatal("expected missing user org mismatch")
	}
	if accountMatchesOrg(&AccountUser{OrgID: &otherID, OrgSlug: "acme"}, orgID, "acme") {
		t.Fatal("expected different org ID mismatch")
	}
	if accountMatchesOrg(&AccountUser{OrgID: &orgID, OrgSlug: "acme"}, orgID, "other") {
		t.Fatal("expected different org slug mismatch")
	}
	if !accountMatchesOrg(&AccountUser{OrgID: &orgID, OrgSlug: " acme "}, orgID, "acme") {
		t.Fatal("expected matching org ID and slug")
	}
}
