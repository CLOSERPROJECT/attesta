package main

import (
	"context"
	"errors"
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
	now := time.Date(2026, 2, 26, 15, 0, 0, 0, time.UTC)
	var loginEmail string
	var loginPassword string
	server := &Server{
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				loginEmail = email
				loginPassword = password
				return fakeIdentitySession("session-secret", "user-1", now.Add(24*time.Hour)), nil
			},
		},
		tmpl: testTemplates(),
		now: func() time.Time {
			return now
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
	if cookies[0].Value != "session-secret" {
		t.Fatalf("session cookie value = %q, want session-secret", cookies[0].Value)
	}
	if !cookies[0].Expires.Equal(now.Add(24 * time.Hour)) {
		t.Fatalf("session cookie expiry = %s, want %s", cookies[0].Expires, now.Add(24*time.Hour))
	}
	if loginEmail != "u1@example.com" || loginPassword != "secure-password" {
		t.Fatalf("login credentials = %q/%q", loginEmail, loginPassword)
	}
}

func TestHandleLoginLegacyFallbackCreatesStoredSession(t *testing.T) {
	store := NewMemoryStore()
	hash, err := bcrypt.GenerateFromPassword([]byte("secure-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user, err := store.CreateUser(t.Context(), AccountUser{
		Email:        "legacy@example.com",
		PasswordHash: string(hash),
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	server := &Server{
		store: store,
		tmpl:  testTemplates(),
		now:   func() time.Time { return time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC) },
	}
	form := url.Values{}
	form.Set("email", "legacy@example.com")
	form.Set("password", "secure-password")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleLogin(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session cookie, got %#v", cookies)
	}
	session, err := store.LoadSessionByID(t.Context(), cookies[0].Value)
	if err != nil {
		t.Fatalf("LoadSessionByID error: %v", err)
	}
	if session.UserMongoID != user.ID {
		t.Fatalf("session user id = %s, want %s", session.UserMongoID.Hex(), user.ID.Hex())
	}
}

func TestHandleLogoutClearsSession(t *testing.T) {
	var deletedSecret string
	server := &Server{
		identity: &fakeIdentityStore{
			deleteSessionFunc: func(ctx context.Context, sessionSecret string) error {
				deletedSecret = sessionSecret
				return nil
			},
		},
		now: time.Now,
	}
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-1"})
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
	if deletedSecret != "session-1" {
		t.Fatalf("deleted secret = %q, want session-1", deletedSecret)
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
	if !strings.Contains(body, `class="toggle-password"`) && !strings.Contains(body, `id="password-toggle"`) {
		t.Fatalf("expected password toggle control in login page, got %q", body)
	}
}

func TestHandleLoginPageShowsSignupWhenEnabled(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "true")
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	server := &Server{identity: &fakeIdentityStore{}, tmpl: tmpl}

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	server.handleLogin(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `href="/signup"`) {
		t.Fatalf("expected signup link, got %q", rec.Body.String())
	}
}

func TestHandleLoginRedirectsAuthenticatedUserToHome(t *testing.T) {
	now := time.Date(2026, 2, 26, 15, 0, 0, 0, time.UTC)
	server := &Server{
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				if sessionSecret != "session-auth-login" {
					return IdentitySession{}, ErrIdentityUnauthorized
				}
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(24*time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "u-auth-login@example.com", Status: "active"}, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-auth-login"})
	rec := httptest.NewRecorder()

	server.handleLogin(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/" {
		t.Fatalf("location = %q, want /", rec.Header().Get("Location"))
	}
}

func TestHandleLoginRejectsInvalidCredentials(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	server := &Server{
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				return IdentitySession{}, ErrIdentityUnauthorized
			},
		},
		tmpl: tmpl,
	}
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
	server := &Server{identity: &fakeIdentityStore{}, tmpl: testTemplates()}
	req := httptest.NewRequest(http.MethodPut, "/login", nil)
	rec := httptest.NewRecorder()
	server.handleLogin(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleLoginInvalidFormAndUnknownUser(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				return IdentitySession{}, ErrIdentityUnauthorized
			},
		},
		tmpl: testTemplates(),
	}

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

func TestHandleLoginIdentityInternalFailure(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				return IdentitySession{}, errors.New("boom")
			},
		},
		tmpl: testTemplates(),
	}

	form := url.Values{}
	form.Set("email", "user@example.com")
	form.Set("password", "secure-password")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleLogin(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandleLoginLegacyInvalidCredentials(t *testing.T) {
	store := NewMemoryStore()
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := store.CreateUser(t.Context(), AccountUser{
		Email:        "legacy-invalid@example.com",
		PasswordHash: string(hash),
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}

	server := &Server{store: store, tmpl: testTemplates()}
	form := url.Values{}
	form.Set("email", "legacy-invalid@example.com")
	form.Set("password", "wrong-password")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleLogin(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandleLoginLegacyUnknownUser(t *testing.T) {
	server := &Server{store: NewMemoryStore(), tmpl: testTemplates()}
	form := url.Values{}
	form.Set("email", "missing-legacy@example.com")
	form.Set("password", "whatever-password")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleLogin(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandleLogoutMethodAndNoCookie(t *testing.T) {
	server := &Server{identity: &fakeIdentityStore{}, now: time.Now}

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

func TestHandleSignupHiddenWhenDisabled(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "false")
	server := &Server{identity: &fakeIdentityStore{}, tmpl: testTemplates()}

	req := httptest.NewRequest(http.MethodGet, "/signup", nil)
	rec := httptest.NewRecorder()
	server.handleSignup(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleSignupCreatesUserAndSession(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "true")
	now := time.Date(2026, 2, 26, 18, 0, 0, 0, time.UTC)
	var createdEmail string
	var createdPassword string
	var sessionEmail string
	server := &Server{
		identity: &fakeIdentityStore{
			createAccountFunc: func(ctx context.Context, email, password, name string) (IdentityUser, error) {
				createdEmail = email
				createdPassword = password
				return IdentityUser{ID: "user-signup", Email: email, Status: "active"}, nil
			},
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				sessionEmail = email
				return fakeIdentitySession("signup-session", "user-signup", now.Add(24*time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-signup", Email: "signup@example.com", Status: "active"}, nil
			},
		},
		tmpl: testTemplates(),
		now:  func() time.Time { return now },
	}

	form := url.Values{}
	form.Set("email", "signup@example.com")
	form.Set("password", "very-secure-password")
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleSignup(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/org-admin/users" {
		t.Fatalf("location = %q, want /org-admin/users", rec.Header().Get("Location"))
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Value != "signup-session" {
		t.Fatalf("cookies = %#v", cookies)
	}
	if createdEmail != "signup@example.com" || createdPassword != "very-secure-password" || sessionEmail != "signup@example.com" {
		t.Fatalf("signup inputs create=%q/%q session=%q", createdEmail, createdPassword, sessionEmail)
	}
}

func TestHandleSignupGetRendersForm(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "true")
	server := &Server{identity: &fakeIdentityStore{}, tmpl: testTemplates()}

	req := httptest.NewRequest(http.MethodGet, "/signup", nil)
	rec := httptest.NewRecorder()
	server.handleSignup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleSignupErrorPaths(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "true")

	t.Run("method not allowed", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: testTemplates()}
		req := httptest.NewRequest(http.MethodPut, "/signup", nil)
		rec := httptest.NewRecorder()
		server.handleSignup(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("invalid form", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: testTemplates()}
		req := httptest.NewRequest(http.MethodPost, "/signup?bad=%zz", strings.NewReader("email=u%40e.com&password=very-secure-password"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleSignup(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("identity unavailable", func(t *testing.T) {
		server := &Server{tmpl: testTemplates()}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=u%40e.com&password=very-secure-password"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleSignup(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("short password", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: testTemplates()}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=u%40e.com&password=short"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleSignup(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("authenticated get redirects", func(t *testing.T) {
		now := time.Date(2026, 2, 26, 19, 0, 0, 0, time.UTC)
		server := &Server{
			identity: &fakeIdentityStore{
				getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
					return fakeIdentitySession(sessionSecret, "user-1", now.Add(24*time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "user-1", Email: "auth@example.com", Status: "active"}, nil
				},
			},
			tmpl:        testTemplates(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, "/signup", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "signup-auth"})
		rec := httptest.NewRecorder()
		server.handleSignup(rec, req)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
			t.Fatalf("status/location = %d %q", rec.Code, rec.Header().Get("Location"))
		}
	})

	t.Run("create account failure", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createAccountFunc: func(ctx context.Context, email, password, name string) (IdentityUser, error) {
					return IdentityUser{}, errors.New("boom")
				},
			},
			tmpl: testTemplates(),
		}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=u%40e.com&password=very-secure-password"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleSignup(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("session creation failure", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createAccountFunc: func(ctx context.Context, email, password, name string) (IdentityUser, error) {
					return IdentityUser{ID: "user-1", Email: email, Status: "active"}, nil
				},
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return IdentitySession{}, errors.New("boom")
				},
			},
			tmpl: testTemplates(),
		}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=u%40e.com&password=very-secure-password"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleSignup(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("current user failure after session", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createAccountFunc: func(ctx context.Context, email, password, name string) (IdentityUser, error) {
					return IdentityUser{ID: "user-1", Email: email, Status: "active"}, nil
				},
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("session-1", "user-1", time.Now().Add(24*time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{}, errors.New("boom")
				},
			},
			tmpl: testTemplates(),
		}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=u%40e.com&password=very-secure-password"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleSignup(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("create account unauthorized still signs in", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createAccountFunc: func(ctx context.Context, email, password, name string) (IdentityUser, error) {
					return IdentityUser{}, ErrIdentityUnauthorized
				},
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("session-1", "user-1", time.Now().Add(24*time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "user-1", Email: "u@e.com", OrgSlug: "acme", Status: "active"}, nil
				},
			},
			tmpl: testTemplates(),
		}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=u%40e.com&password=very-secure-password"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleSignup(rec, req)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
			t.Fatalf("status/location = %d %q", rec.Code, rec.Header().Get("Location"))
		}
	})

	t.Run("existing org redirects home", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createAccountFunc: func(ctx context.Context, email, password, name string) (IdentityUser, error) {
					return IdentityUser{ID: "user-1", Email: email, Status: "active"}, nil
				},
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("session-1", "user-1", time.Now().Add(24*time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "user-1", Email: "u@e.com", OrgSlug: "acme", Status: "active"}, nil
				},
			},
			tmpl: testTemplates(),
		}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=u%40e.com&password=very-secure-password"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.handleSignup(rec, req)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
			t.Fatalf("status/location = %d %q", rec.Code, rec.Header().Get("Location"))
		}
	})
}
