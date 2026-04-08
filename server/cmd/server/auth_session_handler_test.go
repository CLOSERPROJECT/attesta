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
)

type loginStatusError struct {
	status int
}

func (e loginStatusError) Error() string {
	return "login status error"
}

func (e loginStatusError) GetStatusCode() int {
	return e.status
}

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
		now:  func() time.Time { return now },
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

func TestHandleLoginCreatesPlatformAdminSessionCookie(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	now := time.Date(2026, 2, 26, 15, 0, 0, 0, time.UTC)
	server := &Server{
		tmpl: testTemplates(),
		now:  func() time.Time { return now },
	}

	form := url.Values{}
	form.Set("email", "admin@example.com")
	form.Set("password", "change-me")
	form.Set("next", "/admin/orgs")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleLogin(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/admin/orgs" {
		t.Fatalf("location = %q, want /admin/orgs", rec.Header().Get("Location"))
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "attesta_session" {
		t.Fatalf("expected attesta_session cookie, got %#v", cookies)
	}
	if cookies[0].Value != platformAdminSessionValue() {
		t.Fatalf("session cookie value = %q, want %q", cookies[0].Value, platformAdminSessionValue())
	}
}

func TestHandleLoginPlatformAdminRejectsInvalidPassword(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	server := &Server{tmpl: testTemplates(), now: time.Now}
	form := url.Values{}
	form.Set("email", "admin@example.com")
	form.Set("password", "wrong-password")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleLogin(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
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

func TestHandleLogoutSkipsIdentityDeleteForPlatformAdminSession(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	var deleteCalls int
	server := &Server{
		identity: &fakeIdentityStore{
			deleteSessionFunc: func(ctx context.Context, sessionSecret string) error {
				deleteCalls++
				return nil
			},
		},
		now: time.Now,
	}
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: platformAdminSessionValue()})
	rec := httptest.NewRecorder()

	server.handleLogout(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if deleteCalls != 0 {
		t.Fatalf("delete calls = %d, want 0", deleteCalls)
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
		now:  time.Now,
	}
	form := url.Values{}
	form.Set("email", "u1@example.com")
	form.Set("password", "bad-password")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleLogin(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(rec.Body.String(), "Invalid email or password.") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHandleLoginBadRequestIdentityErrorRendersFormError(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	server := &Server{
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				return IdentitySession{}, loginStatusError{status: http.StatusBadRequest}
			},
		},
		tmpl: tmpl,
		now:  time.Now,
	}
	form := url.Values{}
	form.Set("email", "u1@example.com")
	form.Set("password", "short")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleLogin(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(rec.Body.String(), `<p class="error">Invalid email or password.</p>`) {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHandleLoginIdentityUnavailable(t *testing.T) {
	server := &Server{tmpl: testTemplates(), now: time.Now}
	form := url.Values{}
	form.Set("email", "u1@example.com")
	form.Set("password", "secure-password")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleLogin(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleLogoutWithoutCookieStillRedirects(t *testing.T) {
	server := &Server{identity: &fakeIdentityStore{}, now: time.Now}
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()
	server.handleLogout(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
}

func TestHandleLogoutBranches(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, now: time.Now}
		req := httptest.NewRequest(http.MethodGet, "/logout", nil)
		rec := httptest.NewRecorder()

		server.handleLogout(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("secure cookie follows forwarded proto", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, now: time.Now}
		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		rec := httptest.NewRecorder()

		server.handleLogout(rec, req)

		cookies := rec.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("expected logout cookie")
		}
		if !cookies[0].Secure {
			t.Fatalf("cookie secure = %v, want true", cookies[0].Secure)
		}
	})
}

func TestHandleLoginFailureReturnsServerError(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				return IdentitySession{}, errors.New("boom")
			},
		},
		tmpl: testTemplates(),
		now:  time.Now,
	}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("email=u1%40example.com&password=secret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleLogin(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandleLoginMethodNotAllowed(t *testing.T) {
	server := &Server{tmpl: testTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodPut, "/login", nil)
	rec := httptest.NewRecorder()

	server.handleLogin(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleLoginWriteSessionCookieFailure(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{
			createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
				return IdentitySession{UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
		},
		tmpl: testTemplates(),
		now:  time.Now,
	}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("email=u1%40example.com&password=secret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleLogin(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandleSignupDisabledReturnsNotFound(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "false")
	server := &Server{tmpl: testTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodGet, "/signup", nil)
	rec := httptest.NewRecorder()

	server.handleSignup(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleSignupPageRendersWhenEnabled(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "true")
	server := &Server{tmpl: testTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodGet, "/signup", nil)
	rec := httptest.NewRecorder()

	server.handleSignup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "SIGNUP") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHandleSignupRedirectsAuthenticatedUser(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "true")
	now := time.Date(2026, 2, 26, 15, 0, 0, 0, time.UTC)
	server := &Server{
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return fakeIdentitySession(sessionSecret, "user-1", now.Add(time.Hour)), nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "u1@example.com", Status: "active"}, nil
			},
		},
		tmpl:        testTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, "/signup", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-auth-signup"})
	rec := httptest.NewRecorder()

	server.handleSignup(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if rec.Header().Get("Location") != "/" {
		t.Fatalf("location = %q, want /", rec.Header().Get("Location"))
	}
}

func TestHandleSignupRejectsWeakPassword(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "true")
	server := &Server{identity: &fakeIdentityStore{}, tmpl: testTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=u1%40example.com&password=short"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleSignup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "password") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestHandleSignupIdentityUnavailable(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "true")
	server := &Server{tmpl: testTemplates(), now: time.Now}
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=u1%40example.com&password=secure-password"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleSignup(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleSignupCreatesSessionAndRedirectsByOrgMembership(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "true")
	now := time.Date(2026, 2, 26, 15, 0, 0, 0, time.UTC)

	t.Run("without organization redirects to org admin bootstrap", func(t *testing.T) {
		var createdEmail string
		var createdPassword string
		var sessionEmail string
		var sessionPassword string
		server := &Server{
			identity: &fakeIdentityStore{
				createAccountFunc: func(ctx context.Context, email, password, name string) (IdentityUser, error) {
					createdEmail = email
					createdPassword = password
					return IdentityUser{ID: "user-1", Email: email, Status: "active"}, nil
				},
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					sessionEmail = email
					sessionPassword = password
					return fakeIdentitySession("signup-session", "user-1", now.Add(24*time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "user-1", Email: "new@example.com", Status: "active"}, nil
				},
			},
			tmpl: testTemplates(),
			now:  func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=New%40Example.com&password=secure-password"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		server.handleSignup(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
		if rec.Header().Get("Location") != "/org-admin/users" {
			t.Fatalf("location = %q, want /org-admin/users", rec.Header().Get("Location"))
		}
		if createdEmail != "new@example.com" || createdPassword != "secure-password" {
			t.Fatalf("create account args = %q/%q", createdEmail, createdPassword)
		}
		if sessionEmail != "new@example.com" || sessionPassword != "secure-password" {
			t.Fatalf("create session args = %q/%q", sessionEmail, sessionPassword)
		}
		cookies := rec.Result().Cookies()
		if len(cookies) == 0 || cookies[0].Name != "attesta_session" || cookies[0].Value != "signup-session" {
			t.Fatalf("session cookies = %#v", cookies)
		}
	})

	t.Run("existing org membership redirects home", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createAccountFunc: func(ctx context.Context, email, password, name string) (IdentityUser, error) {
					return IdentityUser{}, ErrIdentityUnauthorized
				},
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("signup-session-org", "user-2", now.Add(24*time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{ID: "user-2", Email: "member@example.com", OrgSlug: "acme", Status: "active"}, nil
				},
			},
			tmpl: testTemplates(),
			now:  func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=member%40example.com&password=secure-password"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		server.handleSignup(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
		if rec.Header().Get("Location") != "/" {
			t.Fatalf("location = %q, want /", rec.Header().Get("Location"))
		}
	})
}

func TestHandleSignupReturnsServerErrors(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "true")
	now := time.Date(2026, 2, 26, 15, 0, 0, 0, time.UTC)

	t.Run("create account failure", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createAccountFunc: func(ctx context.Context, email, password, name string) (IdentityUser, error) {
					return IdentityUser{}, errors.New("boom")
				},
			},
			tmpl: testTemplates(),
			now:  func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=u1%40example.com&password=secure-password"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		server.handleSignup(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("load current user failure", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createAccountFunc: func(ctx context.Context, email, password, name string) (IdentityUser, error) {
					return IdentityUser{ID: "user-1", Email: email, Status: "active"}, nil
				},
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return fakeIdentitySession("signup-session", "user-1", now.Add(24*time.Hour)), nil
				},
				getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
					return IdentityUser{}, errors.New("boom")
				},
			},
			tmpl: testTemplates(),
			now:  func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=u1%40example.com&password=secure-password"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		server.handleSignup(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}

func TestHandleSignupAdditionalBranches(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "true")
	now := time.Date(2026, 2, 26, 15, 0, 0, 0, time.UTC)

	t.Run("method not allowed", func(t *testing.T) {
		server := &Server{tmpl: testTemplates(), now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPut, "/signup", nil)
		rec := httptest.NewRecorder()

		server.handleSignup(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("invalid form", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, tmpl: testTemplates(), now: func() time.Time { return now }}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("%zz"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		server.handleSignup(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("create session failure", func(t *testing.T) {
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
			now:  func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=u1%40example.com&password=this-is-strong-enough"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		server.handleSignup(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("write session cookie failure", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				createAccountFunc: func(ctx context.Context, email, password, name string) (IdentityUser, error) {
					return IdentityUser{ID: "user-1", Email: email, Status: "active"}, nil
				},
				createEmailPasswordSessionFunc: func(ctx context.Context, email, password string) (IdentitySession, error) {
					return IdentitySession{UserID: "user-1", ExpiresAt: now.Add(24 * time.Hour)}, nil
				},
			},
			tmpl: testTemplates(),
			now:  func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("email=u1%40example.com&password=this-is-strong-enough"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		server.handleSignup(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}
