package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
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

	id := randomUserIDFromEmail("  USER.Name+tag@example.com ")
	if !strings.HasPrefix(id, "user-name-tag-") {
		t.Fatalf("unexpected randomUserIDFromEmail prefix: %q", id)
	}
	idFallback := randomUserIDFromEmail("   @example.com ")
	if !strings.HasPrefix(idFallback, "user-") {
		t.Fatalf("unexpected randomUserIDFromEmail fallback prefix: %q", idFallback)
	}
}

func TestReadSessionAndCurrentUser(t *testing.T) {
	store := NewMemoryStore()
	now := time.Date(2026, 2, 26, 20, 0, 0, 0, time.UTC)
	user, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "u-session",
		Email:     "u-session@example.com",
		Status:    "active",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	_, err = store.CreateSession(t.Context(), Session{
		SessionID:   "session-valid",
		UserID:      user.UserID,
		UserMongoID: user.ID,
		CreatedAt:   now,
		LastLoginAt: now,
		ExpiresAt:   now.Add(2 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession valid error: %v", err)
	}
	_, err = store.CreateSession(t.Context(), Session{
		SessionID:   "session-expired",
		UserID:      user.UserID,
		UserMongoID: user.ID,
		CreatedAt:   now.Add(-4 * time.Hour),
		LastLoginAt: now.Add(-4 * time.Hour),
		ExpiresAt:   now.Add(-1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession expired error: %v", err)
	}

	server := &Server{
		store: store,
		now:   func() time.Time { return now },
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
		if _, err := store.LoadSessionByID(t.Context(), "session-expired"); err == nil {
			t.Fatal("expected expired session to be deleted")
		}
	})

	t.Run("current user success and failure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-valid"})
		gotUser, gotSession, err := server.currentUser(req)
		if err != nil {
			t.Fatalf("currentUser error: %v", err)
		}
		if gotUser.UserID != "u-session" || gotSession.SessionID != "session-valid" {
			t.Fatalf("unexpected currentUser result user=%+v session=%+v", gotUser, gotSession)
		}

		_ = store.DeleteSession(t.Context(), "session-valid")
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-valid"})
		if _, _, err := server.currentUser(req2); err == nil {
			t.Fatal("expected currentUser error after deleting session")
		}
	})
}

func TestUserOrgAdminAndPageBaseFlags(t *testing.T) {
	orgID := primitive.NewObjectID()
	admin := &AccountUser{
		UserID:          "admin",
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

	non := &AccountUser{UserID: "non", RoleSlugs: []string{"dep1"}}
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
	if !ok || user.UserID != "legacy-user" {
		t.Fatalf("expected legacy auth user, got ok=%v user=%+v", ok, user)
	}

	store := NewMemoryStore()
	now := time.Now().UTC()
	plainUser, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "plain-user",
		Email:     "plain@example.com",
		Status:    "active",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	session, err := store.CreateSession(t.Context(), Session{
		SessionID:   "plain-session",
		UserID:      plainUser.UserID,
		UserMongoID: plainUser.ID,
		CreatedAt:   now,
		LastLoginAt: now,
		ExpiresAt:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	authServer := &Server{
		store:       store,
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/org-admin/users", nil)
	req2.AddCookie(&http.Cookie{Name: "attesta_session", Value: session.SessionID})
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
