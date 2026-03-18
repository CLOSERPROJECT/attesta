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

type authFailingStore struct {
	*MemoryStore
	failSetLastLogin  bool
	failCreateSession bool
}

func (s *authFailingStore) SetUserLastLogin(ctx context.Context, userMongoID primitive.ObjectID, lastLoginAt time.Time) error {
	if s.failSetLastLogin {
		return errors.New("set last login failed")
	}
	return s.MemoryStore.SetUserLastLogin(ctx, userMongoID, lastLoginAt)
}

func (s *authFailingStore) CreateSession(ctx context.Context, session Session) (Session, error) {
	if s.failCreateSession {
		return Session{}, errors.New("create session failed")
	}
	return s.MemoryStore.CreateSession(ctx, session)
}

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

func TestReadSessionAndCurrentUserLegacy(t *testing.T) {
	now := time.Date(2026, 2, 26, 20, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	user, err := store.CreateUser(t.Context(), AccountUser{
		Email:     "legacy@example.com",
		Status:    "active",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	validSession, err := store.CreateSession(t.Context(), Session{
		SessionID:   "legacy-valid",
		UserMongoID: user.ID,
		CreatedAt:   now,
		LastLoginAt: now,
		ExpiresAt:   now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	if _, err := store.CreateSession(t.Context(), Session{
		SessionID:   "legacy-expired",
		UserMongoID: user.ID,
		CreatedAt:   now,
		LastLoginAt: now,
		ExpiresAt:   now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("CreateSession expired error: %v", err)
	}

	server := &Server{
		store: store,
		now:   func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: validSession.SessionID})
	gotUser, gotSession, err := server.currentUser(req)
	if err != nil {
		t.Fatalf("currentUser error: %v", err)
	}
	if gotUser.ID != user.ID || gotSession.Secret != validSession.SessionID {
		t.Fatalf("user/session = %#v %#v", gotUser, gotSession)
	}

	expiredReq := httptest.NewRequest(http.MethodGet, "/", nil)
	expiredReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: "legacy-expired"})
	if _, err := server.readSession(expiredReq); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("readSession error = %v, want %v", err, mongo.ErrNoDocuments)
	}
	if _, err := store.LoadSessionByID(t.Context(), "legacy-expired"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("expired session still exists: %v", err)
	}
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

	underscore := &AccountUser{RoleSlugs: []string{"org_admin"}}
	if !userIsOrgAdmin(underscore) {
		t.Fatal("expected underscore org admin role to count")
	}
	if userHasOrganizationContext(&AccountUser{OrgID: &orgID, OrgSlug: "   "}) {
		t.Fatal("expected blank org slug to fail organization context")
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

func TestAccountUserFromIdentity(t *testing.T) {
	t.Run("without store fallback", func(t *testing.T) {
		server := &Server{}
		user := server.accountUserFromIdentity(t.Context(), IdentityUser{
			Email:      "user@example.com",
			OrgSlug:    "acme",
			Labels:     []string{identityOrgAdminLabel, encodeIdentityRoleLabel("qa-reviewer")},
			IsOrgAdmin: true,
			Status:     "active",
		})
		if user.Email != "user@example.com" || user.OrgSlug != "acme" || !containsRole(user.RoleSlugs, "org-admin") || !containsRole(user.RoleSlugs, "qa-reviewer") {
			t.Fatalf("user = %#v", user)
		}
	})

	t.Run("merges onto legacy user", func(t *testing.T) {
		store := NewMemoryStore()
		legacy, err := store.CreateUser(t.Context(), AccountUser{
			Email:     "legacy@example.com",
			RoleSlugs: []string{"dep1"},
			Status:    "active",
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}
		server := &Server{store: store}
		user := server.accountUserFromIdentity(t.Context(), IdentityUser{
			Email:   "legacy@example.com",
			OrgSlug: "acme",
			Labels:  []string{encodeIdentityRoleLabel("qa-reviewer")},
			Status:  "pending",
		})
		if user.ID != legacy.ID || user.OrgSlug != "acme" || len(user.RoleSlugs) != 1 || user.RoleSlugs[0] != "qa-reviewer" || user.Status != "pending" {
			t.Fatalf("user = %#v legacy=%#v", user, legacy)
		}
	})
}

func TestResetURLHelpers(t *testing.T) {
	t.Run("base URL prefers forwarded https", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/reset", nil)
		req.Host = "attesta.local"
		req.Header.Set("X-Forwarded-Proto", "https")
		if got := requestBaseURL(req); got != "https://attesta.local" {
			t.Fatalf("requestBaseURL = %q, want https://attesta.local", got)
		}
	})

	t.Run("configured redirect overrides request", func(t *testing.T) {
		t.Setenv("APPWRITE_RESET_REDIRECT_URL", "https://app.example/reset/confirm")
		req := httptest.NewRequest(http.MethodGet, "/reset", nil)
		if got := resetRedirectURL(req); got != "https://app.example/reset/confirm" {
			t.Fatalf("resetRedirectURL = %q", got)
		}
	})

	t.Run("base URL falls back to localhost", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/reset", nil)
		req.Host = ""
		if got := requestBaseURL(req); got != "http://localhost:3000" {
			t.Fatalf("requestBaseURL = %q, want http://localhost:3000", got)
		}
	})
}

func TestWriteSessionCookieAndLegacySessionErrors(t *testing.T) {
	server := &Server{store: NewMemoryStore(), now: time.Now}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	if err := server.writeSessionCookie(rec, req, IdentitySession{}); err == nil {
		t.Fatal("expected writeSessionCookie error for missing secret")
	}

	secureReq := httptest.NewRequest(http.MethodGet, "/", nil)
	secureReq.Header.Set("X-Forwarded-Proto", "https")
	secureRec := httptest.NewRecorder()
	expiresAt := time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC)
	if err := server.writeSessionCookie(secureRec, secureReq, IdentitySession{Secret: "session-1", ExpiresAt: expiresAt}); err != nil {
		t.Fatalf("writeSessionCookie error: %v", err)
	}
	cookies := secureRec.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure {
		t.Fatalf("cookies = %#v", cookies)
	}

	if err := server.issueLegacySession(httptest.NewRecorder(), req, nil); err == nil {
		t.Fatal("expected issueLegacySession nil user error")
	}

	user := &AccountUser{ID: primitive.NewObjectID()}
	failLogin := &Server{
		store: &authFailingStore{MemoryStore: NewMemoryStore(), failSetLastLogin: true},
		now:   time.Now,
	}
	if err := failLogin.issueLegacySession(httptest.NewRecorder(), req, user); err == nil || err.Error() != "set last login failed" {
		t.Fatalf("issueLegacySession set last login error = %v", err)
	}

	createSessionStore := NewMemoryStore()
	storedUser, err := createSessionStore.CreateUser(t.Context(), AccountUser{
		Email:     "legacy@example.com",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	failCreate := &Server{
		store: &authFailingStore{MemoryStore: createSessionStore, failCreateSession: true},
		now:   time.Now,
	}
	if err := failCreate.issueLegacySession(httptest.NewRecorder(), req, &storedUser); err == nil || err.Error() != "create session failed" {
		t.Fatalf("issueLegacySession create session error = %v", err)
	}
}

func TestLoadActivePasswordResetAndIsSameAccount(t *testing.T) {
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	server := &Server{
		store: store,
		now:   func() time.Time { return now },
	}

	if _, err := store.CreatePasswordReset(t.Context(), PasswordReset{
		Email:       "used@example.com",
		UserMongoID: primitive.NewObjectID(),
		TokenHash:   "used-token",
		ExpiresAt:   now.Add(time.Hour),
		CreatedAt:   now,
		UsedAt:      ptrTime(now),
	}); err != nil {
		t.Fatalf("CreatePasswordReset error: %v", err)
	}

	if _, err := server.loadActivePasswordReset(t.Context(), "used-token"); err == nil || err.Error() != "reset token already used" {
		t.Fatalf("loadActivePasswordReset error = %v", err)
	}

	if isSameAccount(nil, nil) {
		t.Fatal("expected nil accounts to differ")
	}
	if isSameAccount(&AccountUser{}, &AccountUser{}) {
		t.Fatal("expected zero-value accounts to differ")
	}
	id := primitive.NewObjectID()
	if !isSameAccount(&AccountUser{ID: id}, &AccountUser{ID: id}) {
		t.Fatal("expected matching IDs to compare equal")
	}
}
