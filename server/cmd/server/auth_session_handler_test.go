package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

func TestHandleHomeRedirectsToLoginWhenUnauthenticated(t *testing.T) {
	server := &Server{
		store:       NewMemoryStore(),
		tmpl:        testTemplates(),
		enforceAuth: true,
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
		store:       NewMemoryStore(),
		tmpl:        testTemplates(),
		enforceAuth: true,
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

func TestHandleLoginPageHidesAdminTopbarLinks(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	server := &Server{
		store: NewMemoryStore(),
		tmpl:  tmpl,
	}

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	server.handleLogin(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if strings.Contains(body, `href="/admin/orgs"`) || strings.Contains(body, `href="/org-admin/users"`) {
		t.Fatalf("expected login page without admin nav links, got %q", body)
	}
}

func TestHandleLoginRejectsInvalidCredentials(t *testing.T) {
	store := NewMemoryStore()
	hash, err := bcrypt.GenerateFromPassword([]byte("valid-password-value"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := store.CreateUser(t.Context(), AccountUser{
		UserID:       "u-invalid-login",
		Email:        "u-invalid-login@example.com",
		PasswordHash: string(hash),
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	server := &Server{store: store, tmpl: tmpl}
	form := url.Values{}
	form.Set("email", "u-invalid-login@example.com")
	form.Set("password", "wrong-password")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleLogin(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(rec.Body.String(), "Invalid email or password.") {
		t.Fatalf("expected invalid credentials message, got %q", rec.Body.String())
	}
}

func TestHandleLoginMethodNotAllowed(t *testing.T) {
	server := &Server{store: NewMemoryStore(), tmpl: testTemplates()}
	req := httptest.NewRequest(http.MethodPut, "/login", nil)
	rec := httptest.NewRecorder()
	server.handleLogin(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleLoginInvalidFormAndUnknownUser(t *testing.T) {
	server := &Server{store: NewMemoryStore(), tmpl: testTemplates()}

	reqParse := httptest.NewRequest(http.MethodPost, "/login?bad=%zz", strings.NewReader("email=u%40example.com&password=pw"))
	reqParse.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recParse := httptest.NewRecorder()
	server.handleLogin(recParse, reqParse)
	if recParse.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recParse.Code, http.StatusBadRequest)
	}

	form := url.Values{}
	form.Set("email", "missing@example.com")
	form.Set("password", "irrelevant-password")
	reqMissing := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	reqMissing.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recMissing := httptest.NewRecorder()
	server.handleLogin(recMissing, reqMissing)
	if recMissing.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recMissing.Code, http.StatusUnauthorized)
	}
}

func TestHandleLogoutMethodAndNoCookie(t *testing.T) {
	server := &Server{store: NewMemoryStore(), now: time.Now}

	reqMethod := httptest.NewRequest(http.MethodGet, "/logout", nil)
	recMethod := httptest.NewRecorder()
	server.handleLogout(recMethod, reqMethod)
	if recMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recMethod.Code, http.StatusMethodNotAllowed)
	}

	reqNoCookie := httptest.NewRequest(http.MethodPost, "/logout", nil)
	recNoCookie := httptest.NewRecorder()
	server.handleLogout(recNoCookie, reqNoCookie)
	if recNoCookie.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", recNoCookie.Code, http.StatusSeeOther)
	}
	if recNoCookie.Header().Get("Location") != "/login" {
		t.Fatalf("location = %q, want /login", recNoCookie.Header().Get("Location"))
	}
}
