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
	server := &Server{
		authorizer: fakeAuthorizer{}}
	user := server.accountUserFromIdentity(context.Background(), IdentityUser{
		ID:         "user-1",
		Email:      "legacy@example.com",
		OrgSlug:    "acme",
		Labels:     []string{encodeIdentityRoleLabel("qa-reviewer")},
		IsOrgAdmin: false,
		Status:     "pending",
	})
	if user.IdentityUserID != "user-1" || user.Email != "legacy@example.com" || user.OrgSlug != "acme" || len(user.RoleSlugs) != 1 || user.RoleSlugs[0] != "qa-reviewer" || user.Status != "pending" {
		t.Fatalf("user = %#v", user)
	}
	if !user.ID.IsZero() {
		t.Fatalf("user ID = %s, want zero value for Appwrite-backed account", user.ID.Hex())
	}
}

func TestReadSessionAndCurrentUserIdentityOnly(t *testing.T) {
	now := time.Now().UTC()
	server := &Server{
		authorizer: fakeAuthorizer{},
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
	server := &Server{
		authorizer: fakeAuthorizer{}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
	if _, err := server.readSession(req); !errors.Is(err, ErrIdentityUnauthorized) {
		t.Fatalf("readSession error = %v", err)
	}
}

func TestCurrentUserPropagatesIdentityLookupError(t *testing.T) {
	server := &Server{
		authorizer: fakeAuthorizer{},
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

func TestCurrentUserReturnsPlatformAdminFromSession(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	server := &Server{
		authorizer: fakeAuthorizer{}, now: func() time.Time { return now }}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})

	user, session, err := server.currentUser(req)

	if err != nil {
		t.Fatalf("currentUser error: %v", err)
	}
	if session.Secret != platformAdminSessionValue() {
		t.Fatalf("session secret = %q", session.Secret)
	}
	if !user.IsPlatformAdmin || user.Email != "admin@example.com" {
		t.Fatalf("user = %#v", user)
	}
}

func TestStableOrgHelpers(t *testing.T) {
	orgID := stableOrgObjectID("Acme")
	if orgID == primitive.NilObjectID {
		t.Fatal("stable org object id should not be nil")
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
	if !isSameAccount(&AccountUser{IdentityUserID: "user-1"}, &AccountUser{IdentityUserID: "user-1"}) {
		t.Fatal("expected same identity user ids to match")
	}
	if isSameAccount(nil, &AccountUser{IdentityUserID: "user-1"}) {
		t.Fatal("nil account should not match")
	}
	if isSameAccount(&AccountUser{}, &AccountUser{IdentityUserID: "user-1"}) {
		t.Fatal("missing identity user id should not match")
	}
	legacyID := primitive.NewObjectID()
	if !isSameAccount(&AccountUser{ID: legacyID}, &AccountUser{ID: legacyID}) {
		t.Fatal("expected legacy account ids to match")
	}
}

func TestAppwriteActorHelpers(t *testing.T) {
	if got := accountActorID(nil); got != "legacy-user" {
		t.Fatalf("accountActorID(nil) = %q", got)
	}
	if got := accountActorID(&AccountUser{}); got != "legacy-user" {
		t.Fatalf("accountActorID(empty) = %q", got)
	}
	userID := primitive.NewObjectID()
	if got := accountActorID(&AccountUser{ID: userID}); got != userID.Hex() {
		t.Fatalf("accountActorID(legacy) = %q, want %q", got, userID.Hex())
	}
	if got := accountActorID(&AccountUser{ID: userID, IdentityUserID: "user-1"}); got != "appwrite:user-1" {
		t.Fatalf("accountActorID(appwrite) = %q", got)
	}
	if got := appwriteActorID(" user-2 "); got != "appwrite:user-2" {
		t.Fatalf("appwriteActorID = %q", got)
	}
	if _, ok := parseAppwriteActorID("legacy-user"); ok {
		t.Fatal("legacy actor id should not parse as appwrite")
	}
	if parsed, ok := parseAppwriteActorID("appwrite:user-2"); !ok || parsed != "user-2" {
		t.Fatalf("parseAppwriteActorID = %q, %v", parsed, ok)
	}
}

func TestRequireOrgAdmin(t *testing.T) {
	now := time.Now().UTC()
	orgID := stableOrgObjectID("acme")
	admin := AccountUser{ID: primitive.NewObjectID(), OrgID: &orgID, OrgSlug: "acme", Email: "admin@example.com", RoleSlugs: []string{"org-admin"}, Status: "active"}
	member := AccountUser{ID: primitive.NewObjectID(), OrgID: &orgID, OrgSlug: "acme", Email: "member@example.com", RoleSlugs: []string{"qa-reviewer"}, Status: "active"}
	server := &Server{
		authorizer:  fakeAuthorizer{},
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

func TestRequirePlatformAdmin(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Now().UTC()
	member := AccountUser{ID: primitive.NewObjectID(), Email: "member@example.com", Status: "active"}
	server := &Server{
		authorizer:  fakeAuthorizer{},
		identity:    testIdentityForSessions(now, map[string]AccountUser{"session-member": member}),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	t.Run("unauthenticated redirects", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/orgs", nil)
		rec := httptest.NewRecorder()
		if _, ok := server.requirePlatformAdmin(rec, req); ok {
			t.Fatal("expected requirePlatformAdmin to fail")
		}
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
	})

	t.Run("platform admin allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/orgs", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
		rec := httptest.NewRecorder()
		user, ok := server.requirePlatformAdmin(rec, req)
		if !ok {
			t.Fatal("expected requirePlatformAdmin to allow platform admin")
		}
		if user == nil || !user.IsPlatformAdmin {
			t.Fatalf("user = %#v", user)
		}
	})

	t.Run("non platform admin forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/orgs", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-member"})
		rec := httptest.NewRecorder()
		if _, ok := server.requirePlatformAdmin(rec, req); ok {
			t.Fatal("expected requirePlatformAdmin to reject non-platform admin")
		}
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
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

func TestCookieAndPlatformAdminHelpers(t *testing.T) {
	t.Run("should secure cookie from env forwarded proto and tls", func(t *testing.T) {
		t.Setenv("COOKIE_SECURE", "true")
		req := httptest.NewRequest(http.MethodGet, "http://attesta.local/", nil)
		if !shouldSecureCookie(req) {
			t.Fatal("expected secure cookie from env")
		}

		t.Setenv("COOKIE_SECURE", "false")
		req = httptest.NewRequest(http.MethodGet, "http://attesta.local/", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		if !shouldSecureCookie(req) {
			t.Fatal("expected secure cookie from forwarded proto")
		}

		req = httptest.NewRequest(http.MethodGet, "https://attesta.local/", nil)
		if !shouldSecureCookie(req) {
			t.Fatal("expected secure cookie from tls request")
		}
	})

	t.Run("platform admin helpers require credentials", func(t *testing.T) {
		t.Setenv("ADMIN_EMAIL", "")
		t.Setenv("ADMIN_PASSWORD", "")
		if user := platformAdminAccountUser(); user != nil {
			t.Fatalf("platformAdminAccountUser = %#v, want nil", user)
		}
		server := &Server{
			authorizer: fakeAuthorizer{}, now: time.Now}
		if session, user, ok := server.platformAdminSession(); ok || session != nil || user != nil {
			t.Fatalf("platformAdminSession = %#v %#v %t", session, user, ok)
		}
	})

	t.Run("platform admin helpers derive session and user", func(t *testing.T) {
		t.Setenv("ADMIN_EMAIL", "ADMIN@EXAMPLE.COM")
		t.Setenv("ADMIN_PASSWORD", "change-me")
		server := &Server{
			authorizer: fakeAuthorizer{}, now: func() time.Time { return time.Date(2026, time.March, 24, 9, 0, 0, 0, time.UTC) }}
		user := platformAdminAccountUser()
		if user == nil || user.Email != "admin@example.com" || !user.IsPlatformAdmin {
			t.Fatalf("platformAdminAccountUser = %#v", user)
		}
		session, sessionUser, ok := server.platformAdminSession()
		if !ok || session == nil || sessionUser == nil {
			t.Fatalf("platformAdminSession = %#v %#v %t", session, sessionUser, ok)
		}
		if session.Secret != platformAdminSessionValue() || session.UserID != platformAdminStreamUserID() || sessionUser.Email != user.Email {
			t.Fatalf("platformAdminSession = %#v %#v", session, sessionUser)
		}
	})
}

func TestWorkflowValidationAndRoleStyleHelpers(t *testing.T) {
	if got := (&WorkflowRefValidationError{}).Error(); got != "workflow references are invalid" {
		t.Fatalf("empty workflow validation error = %q", got)
	}
	errText := (&WorkflowRefValidationError{Messages: []string{"missing org", "missing role"}}).Error()
	if errText != "workflow references are invalid:\n- missing org\n- missing role" {
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
