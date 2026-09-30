package main

// Email verification HTTP handler tests (waiting path, confirm, gate helpers).

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func emailVerificationTemplates() *template.Template {
	return template.Must(template.New("verify-test").Parse(`
{{define "layout.html"}}{{if eq .Body "verify_email_body"}}{{template "verify_email_body" .}}{{else if eq .Body "home_picker_body"}}HOME{{end}}{{end}}
{{define "verify_email_body"}}VERIFY_EMAIL {{.Email}}{{if .Confirmation}} {{.Confirmation}}{{end}}{{if .Error}} {{.Error}}{{end}}{{if .ResendDisabled}} RESEND_DISABLED{{end}}{{if .ResendAvailableAt}} RESEND_AT={{.ResendAvailableAt}}{{end}}{{if .ResendRemainingSeconds}} RESEND_IN={{.ResendRemainingSeconds}}{{end}}{{end}}
{{define "verify_email.html"}}{{template "layout.html" .}}{{end}}
{{define "home.html"}}{{template "layout.html" .}}{{end}}
{{define "home_picker_body"}}HOME{{end}}
`))
}

func TestEmailVerificationUnverifiedMyRedirectsToWaiting(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-unverified"
	identity := testIdentityForSessionsRespectingEmailVerified(now, map[string]AccountUser{
		sessionID: {
			IdentityUserID: "user-1",
			Email:          "unverified@example.com",
			Status:         "active",
			EmailVerified:  false,
		},
	})
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        emailVerificationTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
		authorizer:  fakeAuthorizer{},
	}

	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.newMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != emailVerificationPath() {
		t.Fatalf("location = %q, want %s", loc, emailVerificationPath())
	}
}

func TestEmailVerificationWaitingPathRenders(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-waiting"
	identity := testIdentityForSessionsRespectingEmailVerified(now, map[string]AccountUser{
		sessionID: {
			IdentityUserID: "user-1",
			Email:          "waiting@example.com",
			Status:         "active",
			EmailVerified:  false,
		},
	})
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        emailVerificationTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, emailVerificationPath(), nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleEmailVerificationWaiting(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "VERIFY_EMAIL") || !strings.Contains(body, "waiting@example.com") {
		t.Fatalf("body = %q", body)
	}
	if strings.Contains(body, "RESEND_DISABLED") {
		t.Fatalf("expected resend enabled without cooldown cookie, body = %q", body)
	}
}

func TestEmailVerificationWaitingPathRendersResendCooldown(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-waiting-cooldown"
	identity := testIdentityForSessionsRespectingEmailVerified(now, map[string]AccountUser{
		sessionID: {
			IdentityUserID: "user-1",
			Email:          "waiting@example.com",
			Status:         "active",
			EmailVerified:  false,
		},
	})
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        emailVerificationTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, emailVerificationPath(), nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	req.AddCookie(&http.Cookie{
		Name:  emailVerificationResendCookie,
		Value: strconv.FormatInt(now.Add(-30*time.Second).Unix(), 10),
	})
	rec := httptest.NewRecorder()
	server.handleEmailVerificationWaiting(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	wantAt := now.Add(30 * time.Second).Unix()
	for _, want := range []string{
		"RESEND_DISABLED",
		"RESEND_AT=" + strconv.FormatInt(wantAt, 10),
		"RESEND_IN=30",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in body = %q", want, body)
		}
	}
}

func TestEmailVerificationConfirmSuccess(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	completeCalled := false
	server := &Server{
		identity: &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Email: "user@example.com", EmailVerified: false}, nil
			},
			completeEmailVerificationFunc: func(_ context.Context, userID, secret string) error {
				completeCalled = true
				if userID != "user-1" || secret != "secret-1" {
					t.Fatalf("CompleteEmailVerification args = %q %q", userID, secret)
				}
				return nil
			},
		},
		store: NewMemoryStore(),
		tmpl:  emailVerificationTemplates(),
		now:   func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, emailVerificationConfirmPath()+"?userId=user-1&secret=secret-1", nil)
	rec := httptest.NewRecorder()
	server.handleEmailVerificationConfirm(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != pathWithNotice(onboardingPath(), noticeEmailVerified) {
		t.Fatalf("location = %q, want %s", loc, pathWithNotice(onboardingPath(), noticeEmailVerified))
	}
	if !completeCalled {
		t.Fatal("expected CompleteEmailVerification to be called")
	}
}

func TestEmailVerificationConfirmSuccessAffiliatedGoesHome(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	server := &Server{
		identity: &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Email: "member@example.com", OrgSlug: "acme", EmailVerified: false}, nil
			},
			completeEmailVerificationFunc: func(_ context.Context, userID, secret string) error {
				return nil
			},
		},
		store: NewMemoryStore(),
		tmpl:  emailVerificationTemplates(),
		now:   func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodGet, emailVerificationConfirmPath()+"?userId=user-1&secret=secret-1", nil)
	rec := httptest.NewRecorder()
	server.handleEmailVerificationConfirm(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	want := pathWithNotice(appHomePath, noticeEmailVerified)
	if loc := rec.Header().Get("Location"); loc != want {
		t.Fatalf("location = %q, want %s", loc, want)
	}
}

func TestVerifyRedirectURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://attesta.local/", nil)
	if got := verifyRedirectURL(req); got != "http://attesta.local/verify/confirm" {
		t.Fatalf("verifyRedirectURL = %q", got)
	}
	t.Setenv("APPWRITE_VERIFY_REDIRECT_URL", "https://app.example/verify/confirm")
	if got := verifyRedirectURL(req); got != "https://app.example/verify/confirm" {
		t.Fatalf("configured verify redirect = %q", got)
	}
}

func TestEmailVerificationWaitingAlreadyVerifiedRedirects(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		user    AccountUser
		wantLoc string
	}{
		{
			name: "unaffiliated goes onboarding",
			user: AccountUser{
				IdentityUserID: "user-1",
				Email:          "verified@example.com",
				Status:         "active",
				EmailVerified:  true,
			},
			wantLoc: onboardingPath(),
		},
		{
			name: "affiliated goes home",
			user: AccountUser{
				IdentityUserID: "user-2",
				Email:          "member@example.com",
				OrgSlug:        "acme",
				Status:         "active",
				EmailVerified:  true,
			},
			wantLoc: appHomePath,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sessionID := "session-" + tc.name
			server := &Server{
				identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: tc.user}),
				store:       NewMemoryStore(),
				tmpl:        emailVerificationTemplates(),
				enforceAuth: true,
				now:         func() time.Time { return now },
			}
			req := httptest.NewRequest(http.MethodGet, emailVerificationPath(), nil)
			req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
			rec := httptest.NewRecorder()
			server.handleEmailVerificationWaiting(rec, req)
			if rec.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
			}
			if loc := rec.Header().Get("Location"); loc != tc.wantLoc {
				t.Fatalf("location = %q, want %s", loc, tc.wantLoc)
			}
		})
	}
}

func TestEmailVerificationWaitingResendSuccess(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-resend-ok"
	startCalled := false
	identity := testIdentityForSessionsRespectingEmailVerified(now, map[string]AccountUser{
		sessionID: {
			IdentityUserID: "user-1",
			Email:          "waiting@example.com",
			Status:         "active",
			EmailVerified:  false,
		},
	})
	identity.createEmailVerificationFunc = func(_ context.Context, sessionSecret, redirectURL string) error {
		startCalled = true
		if sessionSecret != sessionID {
			t.Fatalf("sessionSecret = %q", sessionSecret)
		}
		if !strings.Contains(redirectURL, emailVerificationConfirmPath()) {
			t.Fatalf("redirectURL = %q", redirectURL)
		}
		return nil
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        emailVerificationTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, emailVerificationPath(), strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleEmailVerificationWaiting(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	want := emailVerificationPath() + "?notice=" + url.QueryEscape(noticeVerificationSent)
	if loc := rec.Header().Get("Location"); loc != want {
		t.Fatalf("location = %q, want %s", loc, want)
	}
	if !startCalled {
		t.Fatal("expected CreateEmailVerification to be called")
	}
	foundCookie := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == emailVerificationResendCookie {
			foundCookie = true
			if c.Value != strconv.FormatInt(now.Unix(), 10) {
				t.Fatalf("resend cookie = %q", c.Value)
			}
		}
	}
	if !foundCookie {
		t.Fatal("expected resend cooldown cookie")
	}
}

func TestEmailVerificationWaitingResendCooldown(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-resend-wait"
	startCalled := false
	identity := testIdentityForSessionsRespectingEmailVerified(now, map[string]AccountUser{
		sessionID: {
			IdentityUserID: "user-1",
			Email:          "waiting@example.com",
			Status:         "active",
			EmailVerified:  false,
		},
	})
	identity.createEmailVerificationFunc = func(context.Context, string, string) error {
		startCalled = true
		return nil
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        emailVerificationTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, emailVerificationPath(), strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	req.AddCookie(&http.Cookie{
		Name:  emailVerificationResendCookie,
		Value: strconv.FormatInt(now.Add(-30*time.Second).Unix(), 10),
	})
	rec := httptest.NewRecorder()
	server.handleEmailVerificationWaiting(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, emailVerificationNoticeMessage(noticeVerificationResendWait)) {
		t.Fatalf("body = %q", body)
	}
	if startCalled {
		t.Fatal("did not expect CreateEmailVerification during cooldown")
	}
}

func TestEmailVerificationWaitingResendFailure(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-resend-fail"
	identity := testIdentityForSessionsRespectingEmailVerified(now, map[string]AccountUser{
		sessionID: {
			IdentityUserID: "user-1",
			Email:          "waiting@example.com",
			Status:         "active",
			EmailVerified:  false,
		},
	})
	identity.createEmailVerificationFunc = func(context.Context, string, string) error {
		return errors.New("send failed")
	}
	server := &Server{
		identity:    identity,
		store:       NewMemoryStore(),
		tmpl:        emailVerificationTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}

	req := httptest.NewRequest(http.MethodPost, emailVerificationPath(), strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleEmailVerificationWaiting(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, emailVerificationNoticeMessage(noticeVerificationSendFailed)) {
		t.Fatalf("body = %q", body)
	}
}

func TestEmailVerificationWaitingMethodNotAllowed(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-method"
	server := &Server{
		identity: testIdentityForSessionsRespectingEmailVerified(now, map[string]AccountUser{
			sessionID: {
				IdentityUserID: "user-1",
				Email:          "waiting@example.com",
				Status:         "active",
				EmailVerified:  false,
			},
		}),
		store:       NewMemoryStore(),
		tmpl:        emailVerificationTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodPut, emailVerificationPath(), nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleEmailVerificationWaiting(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestEmailVerificationWaitingGETShowsNoticeAndCooldown(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-notice"
	server := &Server{
		identity: testIdentityForSessionsRespectingEmailVerified(now, map[string]AccountUser{
			sessionID: {
				IdentityUserID: "user-1",
				Email:          "waiting@example.com",
				Status:         "active",
				EmailVerified:  false,
			},
		}),
		store:       NewMemoryStore(),
		tmpl:        emailVerificationTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodGet, emailVerificationPath()+"?notice="+url.QueryEscape(noticeVerificationSent), nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	req.AddCookie(&http.Cookie{
		Name:  emailVerificationResendCookie,
		Value: strconv.FormatInt(now.Unix(), 10),
	})
	rec := httptest.NewRecorder()
	server.handleEmailVerificationWaiting(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, emailVerificationNoticeMessage(noticeVerificationSent)) {
		t.Fatalf("missing confirmation in body = %q", body)
	}
	if !strings.Contains(body, "RESEND_DISABLED") {
		t.Fatalf("expected RESEND_DISABLED in body = %q", body)
	}
}

func TestEmailVerificationConfirmFailurePaths(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	failedNotice := url.QueryEscape(noticeVerificationFailed)

	t.Run("missing params redirects to login", func(t *testing.T) {
		server := &Server{
			identity:    &fakeIdentityStore{},
			store:       NewMemoryStore(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, emailVerificationConfirmPath(), nil)
		rec := httptest.NewRecorder()
		server.handleEmailVerificationConfirm(rec, req)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d", rec.Code)
		}
		want := "/login?notice=" + failedNotice
		if loc := rec.Header().Get("Location"); loc != want {
			t.Fatalf("location = %q, want %s", loc, want)
		}
	})

	t.Run("logged in failure goes to waiting", func(t *testing.T) {
		sessionID := "session-fail-logged-in"
		server := &Server{
			identity: testIdentityForSessionsRespectingEmailVerified(now, map[string]AccountUser{
				sessionID: {
					IdentityUserID: "user-1",
					Email:          "waiting@example.com",
					Status:         "active",
					EmailVerified:  false,
				},
			}),
			store:       NewMemoryStore(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, emailVerificationConfirmPath()+"?userId=user-1", nil)
		req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
		rec := httptest.NewRecorder()
		server.handleEmailVerificationConfirm(rec, req)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d", rec.Code)
		}
		want := emailVerificationPath() + "?notice=" + failedNotice
		if loc := rec.Header().Get("Location"); loc != want {
			t.Fatalf("location = %q, want %s", loc, want)
		}
	})

	t.Run("complete error redirects to login", func(t *testing.T) {
		server := &Server{
			identity: &fakeIdentityStore{
				getUserByIDFunc: func(context.Context, string) (IdentityUser, error) {
					return IdentityUser{ID: "user-1", EmailVerified: false}, nil
				},
				completeEmailVerificationFunc: func(context.Context, string, string) error {
					return errors.New("bad secret")
				},
			},
			store:       NewMemoryStore(),
			enforceAuth: true,
			now:         func() time.Time { return now },
		}
		req := httptest.NewRequest(http.MethodGet, emailVerificationConfirmPath()+"?userId=user-1&secret=bad", nil)
		rec := httptest.NewRecorder()
		server.handleEmailVerificationConfirm(rec, req)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("status = %d", rec.Code)
		}
		want := "/login?notice=" + failedNotice
		if loc := rec.Header().Get("Location"); loc != want {
			t.Fatalf("location = %q, want %s", loc, want)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		server := &Server{identity: &fakeIdentityStore{}, store: NewMemoryStore()}
		req := httptest.NewRequest(http.MethodPost, emailVerificationConfirmPath(), nil)
		rec := httptest.NewRecorder()
		server.handleEmailVerificationConfirm(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("nil identity not found", func(t *testing.T) {
		server := &Server{store: NewMemoryStore()}
		req := httptest.NewRequest(http.MethodGet, emailVerificationConfirmPath()+"?userId=u&secret=s", nil)
		rec := httptest.NewRecorder()
		server.handleEmailVerificationConfirm(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d", rec.Code)
		}
	})
}

func TestDenyIfUnverifiedPost(t *testing.T) {
	server := &Server{identity: &fakeIdentityStore{}, enforceAuth: true}
	unverified := AccountUser{Email: "u@example.com", EmailVerified: false}
	rec := httptest.NewRecorder()
	if !server.denyIfUnverifiedPost(rec, unverified) {
		t.Fatal("expected deny for unverified user")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	verified := AccountUser{Email: "v@example.com", EmailVerified: true}
	recOK := httptest.NewRecorder()
	if server.denyIfUnverifiedPost(recOK, verified) {
		t.Fatal("did not expect deny for verified user")
	}
}

func TestRequireVerifiedPostDeniesUnverified(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-post-deny"
	server := &Server{
		identity: testIdentityForSessionsRespectingEmailVerified(now, map[string]AccountUser{
			sessionID: {
				IdentityUserID: "user-1",
				Email:          "waiting@example.com",
				Status:         "active",
				EmailVerified:  false,
			},
		}),
		store:       NewMemoryStore(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodPost, "/anything", strings.NewReader(""))
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	if _, _, ok := server.requireVerifiedPost(rec, req); ok {
		t.Fatal("expected requireVerifiedPost to deny unverified user")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestNestedMyHandlersGateUnverifiedInIsolation(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-nested-gate"
	unverified := AccountUser{
		IdentityUserID: "user-1",
		Email:          "waiting@example.com",
		Status:         "active",
		EmailVerified:  false,
	}
	server := &Server{
		identity:    testIdentityForSessionsRespectingEmailVerified(now, map[string]AccountUser{sessionID: unverified}),
		store:       NewMemoryStore(),
		tmpl:        emailVerificationTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	pageReq := httptest.NewRequest(http.MethodGet, "/streams/demo/", nil)
	pageReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	pageReq = pageReq.WithContext(context.WithValue(pageReq.Context(), workflowContextKey{}, workflowContextValue{
		Key: "demo",
		Cfg: testRuntimeConfig(),
	}))
	pageRec := httptest.NewRecorder()
	server.handleWorkflowHome(pageRec, pageReq)
	if pageRec.Code != http.StatusSeeOther {
		t.Fatalf("workflow home status = %d, want %d", pageRec.Code, http.StatusSeeOther)
	}
	if loc := pageRec.Header().Get("Location"); loc != emailVerificationPath() {
		t.Fatalf("workflow home location = %q, want %s", loc, emailVerificationPath())
	}

	postReq := httptest.NewRequest(http.MethodPost, leaveOrganizationPath(), nil)
	postReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	postRec := httptest.NewRecorder()
	server.handleLeaveOrganization(postRec, postReq)
	if postRec.Code != http.StatusForbidden {
		t.Fatalf("leave org status = %d, want %d", postRec.Code, http.StatusForbidden)
	}
}

func TestEmailVerificationResendTooSoonEdges(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	server := &Server{now: func() time.Time { return now }}

	noCookie := httptest.NewRequest(http.MethodGet, "/", nil)
	if server.emailVerificationResendTooSoon(noCookie) {
		t.Fatal("no cookie should not be too soon")
	}

	bad := httptest.NewRequest(http.MethodGet, "/", nil)
	bad.AddCookie(&http.Cookie{Name: emailVerificationResendCookie, Value: "not-a-number"})
	if server.emailVerificationResendTooSoon(bad) {
		t.Fatal("invalid cookie should not be too soon")
	}

	expired := httptest.NewRequest(http.MethodGet, "/", nil)
	expired.AddCookie(&http.Cookie{
		Name:  emailVerificationResendCookie,
		Value: strconv.FormatInt(now.Add(-2*time.Minute).Unix(), 10),
	})
	if server.emailVerificationResendTooSoon(expired) {
		t.Fatal("expired cooldown should allow resend")
	}

	active := httptest.NewRequest(http.MethodGet, "/", nil)
	active.AddCookie(&http.Cookie{
		Name:  emailVerificationResendCookie,
		Value: strconv.FormatInt(now.Add(-25*time.Second).Unix(), 10),
	})
	state := server.emailVerificationResendState(active)
	if !state.Disabled || state.RemainingSeconds != 35 || state.AvailableAt != now.Add(35*time.Second).Unix() {
		t.Fatalf("active cooldown state = %+v", state)
	}
}

func TestVerifiedLandingPath(t *testing.T) {
	server := &Server{store: NewMemoryStore()}

	cases := []struct {
		name string
		user *AccountUser
		next string
		want string
	}{
		{name: "nil user", user: nil, want: emailVerificationPath()},
		{
			name: "unverified goes verify",
			user: &AccountUser{Email: "u@example.com", EmailVerified: false},
			next: appHomePath,
			want: emailVerificationPath(),
		},
		{
			name: "unaffiliated empty next goes onboarding",
			user: &AccountUser{Email: "u@example.com", EmailVerified: true},
			want: onboardingPath(),
		},
		{
			name: "unaffiliated app home goes onboarding",
			user: &AccountUser{Email: "u@example.com", EmailVerified: true},
			next: appHomePath,
			want: onboardingPath(),
		},
		{
			name: "unaffiliated app home slash goes onboarding",
			user: &AccountUser{Email: "u@example.com", EmailVerified: true},
			next: appHomePath + "/",
			want: onboardingPath(),
		},
		{
			name: "unaffiliated explicit next wins",
			user: &AccountUser{Email: "u@example.com", EmailVerified: true},
			next: "/my/streams/workflow/",
			want: "/my/streams/workflow/",
		},
		{
			name: "affiliated empty next goes home",
			user: &AccountUser{Email: "m@example.com", OrgSlug: "acme", EmailVerified: true},
			want: appHomePath,
		},
		{
			name: "affiliated app home stays home",
			user: &AccountUser{Email: "m@example.com", OrgSlug: "acme", EmailVerified: true},
			next: appHomePath,
			want: appHomePath,
		},
		{
			name: "affiliated explicit next wins",
			user: &AccountUser{Email: "m@example.com", OrgSlug: "acme", EmailVerified: true},
			next: "/admin/organizations",
			want: "/admin/organizations",
		},
		{
			name: "unsafe next unaffiliated goes onboarding",
			user: &AccountUser{Email: "u@example.com", EmailVerified: true},
			next: "https://evil.example/",
			want: onboardingPath(),
		},
		{
			name: "unsafe next affiliated goes home",
			user: &AccountUser{Email: "m@example.com", OrgSlug: "acme", EmailVerified: true},
			next: "https://evil.example/",
			want: appHomePath,
		},
		{
			name: "platform admin unverified unaffiliated goes onboarding",
			user: &AccountUser{Email: "admin@example.com", IsPlatformAdmin: true, EmailVerified: false},
			want: onboardingPath(),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := server.verifiedLandingPath(tc.user, tc.next); got != tc.want {
				t.Fatalf("verifiedLandingPath = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPostVerificationAppPathForUserIDFallback(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{
			getUserByIDFunc: func(context.Context, string) (IdentityUser, error) {
				return IdentityUser{}, errors.New("missing")
			},
		},
		store: NewMemoryStore(),
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := server.postVerificationAppPathForUserID(req, "missing"); got != appHomePath {
		t.Fatalf("got %q, want %s", got, appHomePath)
	}

	nilIdentity := &Server{store: NewMemoryStore()}
	if got := nilIdentity.postVerificationAppPathForUserID(req, "any"); got != appHomePath {
		t.Fatalf("nil identity got %q, want %s", got, appHomePath)
	}
}

func TestMailboxProofSuccessPath(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Email: "user@example.com", EmailVerified: true}, nil
			},
		},
		store: NewMemoryStore(),
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if got := server.mailboxProofSuccessPath(req, mailboxProofConfirm, "user-1", false); got != pathWithNotice(onboardingPath(), noticeEmailVerified) {
		t.Fatalf("confirm = %q", got)
	}
	if got := server.mailboxProofSuccessPath(req, mailboxProofInviteAccept, "user-1", false); got != appHomePath {
		t.Fatalf("invite home = %q", got)
	}
	if got := server.mailboxProofSuccessPath(req, mailboxProofInviteAccept, "user-1", true); got != "/invite/password" {
		t.Fatalf("invite password = %q", got)
	}
	if got := server.mailboxProofSuccessPath(req, mailboxProofPasswordRecovery, "user-1", false); got != pathWithNotice("/login", noticePasswordResetSuccess) {
		t.Fatalf("recovery = %q", got)
	}
}

func TestEmailVerificationWaitingUnauthenticatedRedirects(t *testing.T) {
	server := &Server{
		identity:    &fakeIdentityStore{},
		store:       NewMemoryStore(),
		tmpl:        emailVerificationTemplates(),
		enforceAuth: true,
		now:         time.Now,
	}
	req := httptest.NewRequest(http.MethodGet, emailVerificationPath(), nil)
	rec := httptest.NewRecorder()
	server.handleEmailVerificationWaiting(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "/login") {
		t.Fatalf("location = %q, want login redirect", loc)
	}
}

func TestEmailVerificationWaitingResendUnauthorizedWithoutSession(t *testing.T) {
	// enforceAuth off returns a nil session from requireAuthenticatedPage.
	server := &Server{
		identity:    &fakeIdentityStore{},
		store:       NewMemoryStore(),
		tmpl:        emailVerificationTemplates(),
		enforceAuth: false,
		now:         time.Now,
	}
	req := httptest.NewRequest(http.MethodPost, emailVerificationPath(), strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleEmailVerificationWaiting(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestEmailVerificationWaitingResendParseFormError(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-resend-parse"
	server := &Server{
		identity: testIdentityForSessionsRespectingEmailVerified(now, map[string]AccountUser{
			sessionID: {
				IdentityUserID: "user-1",
				Email:          "waiting@example.com",
				Status:         "active",
				EmailVerified:  false,
			},
		}),
		store:       NewMemoryStore(),
		tmpl:        emailVerificationTemplates(),
		enforceAuth: true,
		now:         func() time.Time { return now },
	}
	req := httptest.NewRequest(http.MethodPost, emailVerificationPath(), errReadCloser{})
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleEmailVerificationWaiting(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRenderEmailVerificationWaitingTemplateError(t *testing.T) {
	server := &Server{tmpl: template.New("empty"), now: time.Now}
	rec := httptest.NewRecorder()
	server.renderEmailVerificationWaiting(rec, httptest.NewRequest(http.MethodGet, "/", nil), &AccountUser{
		Email: "waiting@example.com",
	}, "", "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
