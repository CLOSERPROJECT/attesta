package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

func TestHandleHomeRedirectsToLoginWhenUnauthenticated(t *testing.T) {
	server := &Server{
		store: NewMemoryStore(),
		tmpl:  testTemplates(),
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	server.handleHome(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	location := rec.Header().Get("Location")
	if location != "/login?next=%2F" {
		t.Fatalf("location = %q, want /login?next=%%2F", location)
	}
}

func TestHandleCompleteSubstepUnauthenticatedReturnsUnauthorized(t *testing.T) {
	server := &Server{
		store: NewMemoryStore(),
		tmpl:  testTemplates(),
	}
	req := httptest.NewRequest(http.MethodPost, "/w/workflow/process/abc/substep/1.1/complete", strings.NewReader("value=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleCompleteSubstep(rec, req, primitive.NewObjectID().Hex(), "1.1")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandleLoginCreatesSessionCookie(t *testing.T) {
	store := NewMemoryStore()
	hash, err := bcrypt.GenerateFromPassword([]byte("secure-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := store.CreateUser(t.Context(), AccountUser{
		UserID:       "u1",
		Email:        "u1@example.com",
		PasswordHash: string(hash),
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	server := &Server{
		store: store,
		tmpl:  testTemplates(),
		now: func() time.Time {
			return time.Date(2026, 2, 26, 15, 0, 0, 0, time.UTC)
		},
	}

	form := url.Values{}
	form.Set("email", "u1@example.com")
	form.Set("password", "secure-password")
	form.Set("next", "/w/workflow/")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleLogin(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/w/workflow/" {
		t.Fatalf("location = %q, want /w/workflow/", rec.Header().Get("Location"))
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "attesta_session" {
		t.Fatalf("expected attesta_session cookie, got %#v", cookies)
	}
	if cookies[0].HttpOnly != true {
		t.Fatal("expected HttpOnly session cookie")
	}
	if _, err := store.LoadSessionByID(t.Context(), cookies[0].Value); err != nil {
		t.Fatalf("LoadSessionByID error: %v", err)
	}

	updated, err := store.GetUserByUserID(t.Context(), user.UserID)
	if err != nil {
		t.Fatalf("GetUserByUserID error: %v", err)
	}
	if updated.LastLoginAt == nil {
		t.Fatal("expected user lastLoginAt to be updated")
	}
}

func TestHandleLogoutClearsSession(t *testing.T) {
	store := NewMemoryStore()
	session, err := store.CreateSession(t.Context(), Session{
		SessionID:   "session-1",
		UserID:      "u1",
		UserMongoID: primitive.NewObjectID(),
		CreatedAt:   time.Now().UTC(),
		LastLoginAt: time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	server := &Server{store: store, now: time.Now}
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: session.SessionID})
	rec := httptest.NewRecorder()

	server.handleLogout(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/login" {
		t.Fatalf("location = %q, want /login", rec.Header().Get("Location"))
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "attesta_session" || cookies[0].Value != "" {
		t.Fatalf("expected cleared attesta_session cookie, got %#v", cookies)
	}
	if _, err := store.LoadSessionByID(t.Context(), session.SessionID); err == nil {
		t.Fatal("expected session to be deleted")
	}
}
