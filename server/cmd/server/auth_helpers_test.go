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

func TestRequestedRoleSlugs(t *testing.T) {
	form := url.Values{}
	form["roles"] = []string{" qa-reviewer ", "org-admin"}
	got := requestedRoleSlugs(form)
	if len(got) != 2 || got[0] != "qa-reviewer" || got[1] != "org-admin" {
		t.Fatalf("requestedRoleSlugs = %#v", got)
	}

	legacy := url.Values{}
	legacy.Set("role", " dep2 ")
	got = requestedRoleSlugs(legacy)
	if len(got) != 1 || got[0] != "dep2" {
		t.Fatalf("requestedRoleSlugs legacy role = %#v, want [dep2]", got)
	}
}

func TestAccountUserFromIdentity(t *testing.T) {
	server := &Server{}
	user := server.accountUserFromIdentity(context.Background(), IdentityUser{
		ID:         "user-1",
		Email:      "legacy@example.com",
		OrgSlug:    "acme",
		Labels:     []string{encodeIdentityRoleLabel("qa-reviewer")},
		IsOrgAdmin: false,
		Status:     "pending",
	})
	if user.Email != "legacy@example.com" || user.OrgSlug != "acme" || len(user.RoleSlugs) != 1 || user.RoleSlugs[0] != "qa-reviewer" || user.Status != "pending" {
		t.Fatalf("user = %#v", user)
	}
}

func TestReadSessionAndCurrentUserIdentityOnly(t *testing.T) {
	now := time.Now().UTC()
	server := &Server{
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				if sessionSecret == "expired" {
					return fakeIdentitySession(sessionSecret, "user-1", now.Add(-time.Hour)), nil
				}
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "user@example.com", OrgSlug: "acme", Labels: []string{encodeIdentityRoleLabel("qa-reviewer")}, Status: "active"}, nil
			},
		},
		now: func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "ok"})
	user, session, err := server.currentUser(req)
	if err != nil {
		t.Fatalf("currentUser error: %v", err)
	}
	if session.Secret != "ok" || user.Email != "user@example.com" || user.OrgSlug != "acme" {
		t.Fatalf("user/session = %#v %#v", user, session)
	}

	expiredReq := httptest.NewRequest(http.MethodGet, "/", nil)
	expiredReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: "expired"})
	if _, err := server.readSession(expiredReq); !errors.Is(err, ErrIdentityUnauthorized) {
		t.Fatalf("expired readSession error = %v", err)
	}
}

func TestReadSessionRequiresIdentity(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	if _, err := server.readSession(req); !errors.Is(err, ErrIdentityUnauthorized) {
		t.Fatalf("readSession error = %v", err)
	}
}

func TestCurrentUserPropagatesIdentityLookupError(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", time.Now().Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{}, errors.New("boom")
			},
		},
		now: time.Now,
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})

	_, _, err := server.currentUser(req)

	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("currentUser error = %v, want boom", err)
	}
}

func TestRequestURLsAndCookieHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://attesta.local/", nil)
	if got := requestBaseURL(req); got != "http://attesta.local" {
		t.Fatalf("requestBaseURL = %q", got)
	}
	if got := resetRedirectURL(req); got != "http://attesta.local/reset/confirm" {
		t.Fatalf("resetRedirectURL = %q", got)
	}
	if got := inviteRedirectURL(req); got != "http://attesta.local/invite/accept" {
		t.Fatalf("inviteRedirectURL = %q", got)
	}

	t.Setenv("APPWRITE_RESET_REDIRECT_URL", "https://app.example/reset/confirm")
	if got := resetRedirectURL(req); got != "https://app.example/reset/confirm" {
		t.Fatalf("configured reset redirect = %q", got)
	}
	t.Setenv("APPWRITE_INVITE_REDIRECT_URL", "https://app.example/invite/accept")
	if got := inviteRedirectURL(req); got != "https://app.example/invite/accept" {
		t.Fatalf("configured invite redirect = %q", got)
	}

	secureReq := httptest.NewRequest(http.MethodGet, "http://attesta.local/", nil)
	secureReq.Header.Set("X-Forwarded-Proto", "https")
	if got := requestBaseURL(secureReq); got != "https://attesta.local" {
		t.Fatalf("secure requestBaseURL = %q", got)
	}

	hostlessReq := httptest.NewRequest(http.MethodGet, "/", nil)
	hostlessReq.Host = ""
	if got := requestBaseURL(hostlessReq); got != "http://localhost:3000" {
		t.Fatalf("hostless requestBaseURL = %q", got)
	}
}

func TestSessionSecretFromRequest(t *testing.T) {
	if _, err := sessionSecretFromRequest(httptest.NewRequest(http.MethodGet, "/", nil)); err == nil {
		t.Fatal("expected missing cookie error")
	}
	reqEmpty := httptest.NewRequest(http.MethodGet, "/", nil)
	reqEmpty.AddCookie(&http.Cookie{Name: "attesta_session", Value: "   "})
	if _, err := sessionSecretFromRequest(reqEmpty); !errors.Is(err, ErrIdentityUnauthorized) {
		t.Fatalf("empty cookie error = %v", err)
	}
}

func TestStableIdentityHelpers(t *testing.T) {
	orgID := stableOrgObjectID("Acme")
	userID := stableIdentityUserObjectID("user-1")
	if orgID == primitive.NilObjectID || userID == primitive.NilObjectID {
		t.Fatal("stable object ids should not be nil")
	}
}

func TestResetTTLHrsDefaultsInvalidValues(t *testing.T) {
	t.Setenv("RESET_TTL_HOURS", "48")
	if got := resetTTLHrs(); got != 48 {
		t.Fatalf("resetTTLHrs = %d, want 48", got)
	}

	t.Setenv("RESET_TTL_HOURS", "0")
	if got := resetTTLHrs(); got != 24 {
		t.Fatalf("resetTTLHrs zero = %d, want 24", got)
	}
}

func TestAccountMatchesOrg(t *testing.T) {
	orgID := stableOrgObjectID("acme")
	user := &AccountUser{OrgID: &orgID, OrgSlug: "acme"}
	if !accountMatchesOrg(user, orgID, " acme ") {
		t.Fatal("expected account to match org")
	}
	if accountMatchesOrg(nil, orgID, "acme") {
		t.Fatal("nil account should not match")
	}
	otherOrgID := stableOrgObjectID("other")
	if accountMatchesOrg(user, otherOrgID, "acme") {
		t.Fatal("different org id should not match")
	}
	if accountMatchesOrg(user, orgID, "other") {
		t.Fatal("different org slug should not match")
	}
}

func TestIsSameAccount(t *testing.T) {
	accountID := stableIdentityUserObjectID("user-1")
	if !isSameAccount(&AccountUser{ID: accountID}, &AccountUser{ID: accountID}) {
		t.Fatal("expected same account ids to match")
	}
	if isSameAccount(nil, &AccountUser{ID: accountID}) {
		t.Fatal("nil account should not match")
	}
	if isSameAccount(&AccountUser{}, &AccountUser{ID: accountID}) {
		t.Fatal("zero account id should not match")
	}
}

func TestRequireOrgAdmin(t *testing.T) {
	now := time.Now().UTC()
	orgID := stableOrgObjectID("acme")
	admin := AccountUser{ID: primitive.NewObjectID(), OrgID: &orgID, OrgSlug: "acme", Email: "admin@example.com", RoleSlugs: []string{"org-admin"}, Status: "active"}
	member := AccountUser{ID: primitive.NewObjectID(), OrgID: &orgID, OrgSlug: "acme", Email: "member@example.com", RoleSlugs: []string{"qa-reviewer"}, Status: "active"}
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{"session-admin": admin, "session-member": member}),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	t.Run("unauthenticated redirects", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/org-admin/users", nil)
		rec := httptest.NewRecorder()
		if _, ok := server.requireOrgAdmin(rec, req); ok {
			t.Fatal("expected requireOrgAdmin to fail")
		}
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
	})

	t.Run("non admin forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/org-admin/users", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-member"})
		rec := httptest.NewRecorder()
		if _, ok := server.requireOrgAdmin(rec, req); ok {
			t.Fatal("expected requireOrgAdmin to reject non-admin")
		}
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("admin allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/org-admin/users", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-admin"})
		rec := httptest.NewRecorder()
		user, ok := server.requireOrgAdmin(rec, req)
		if !ok {
			t.Fatal("expected requireOrgAdmin to allow admin")
		}
		if user == nil || user.Email != "admin@example.com" {
			t.Fatalf("user = %#v", user)
		}
	})
}

func TestSessionTTLDaysAndNewSessionID(t *testing.T) {
	t.Setenv("SESSION_TTL_DAYS", "45")
	if got := sessionTTLDays(); got != 45 {
		t.Fatalf("sessionTTLDays = %d, want 45", got)
	}

	t.Setenv("SESSION_TTL_DAYS", "0")
	if got := sessionTTLDays(); got != 30 {
		t.Fatalf("sessionTTLDays zero = %d, want 30", got)
	}

	first, err := newSessionID()
	if err != nil {
		t.Fatalf("newSessionID error: %v", err)
	}
	second, err := newSessionID()
	if err != nil {
		t.Fatalf("newSessionID second error: %v", err)
	}
	if first == "" || second == "" || first == second {
		t.Fatalf("session ids = %q %q", first, second)
	}
}

func TestWorkflowValidationAndRoleStyleHelpers(t *testing.T) {
	if got := (&WorkflowRefValidationError{}).Error(); got != "workflow references are invalid" {
		t.Fatalf("empty workflow validation error = %q", got)
	}
	errText := (&WorkflowRefValidationError{Messages: []string{"missing org", "missing role"}}).Error()
	if errText != "workflow references are invalid: missing org; missing role" {
		t.Fatalf("workflow validation error = %q", errText)
	}

	if got := resolveRolePaletteStyle("sky"); got.Color == "" || got.Border == "" {
		t.Fatalf("expected known palette style, got %#v", got)
	}
	if got := resolveRolePaletteStyle("unknown"); got != rolePaletteStyles["red"] {
		t.Fatalf("fallback palette style = %#v, want %#v", got, rolePaletteStyles["red"])
	}
}

func TestIsDuplicateSlugError(t *testing.T) {
	if isDuplicateSlugError(nil) {
		t.Fatal("nil error should not be duplicate slug")
	}
	if !isDuplicateSlugError(errors.New("slug already exists")) {
		t.Fatal("expected slug already exists to match")
	}
	if !isDuplicateSlugError(errors.New("duplicate key")) {
		t.Fatal("expected duplicate key to match")
	}
	if isDuplicateSlugError(errors.New("boom")) {
		t.Fatal("unexpected duplicate slug match")
	}
}
