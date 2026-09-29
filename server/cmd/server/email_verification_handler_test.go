package main

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func boolPtr(v bool) *bool { return &v }

func emailVerificationTemplates() *template.Template {
	return template.Must(template.New("verify-test").Parse(`
{{define "layout.html"}}{{if eq .Body "verify_email_body"}}{{template "verify_email_body" .}}{{else if eq .Body "home_picker_body"}}HOME{{end}}{{end}}
{{define "verify_email_body"}}VERIFY_EMAIL {{.Email}}{{if .Confirmation}} {{.Confirmation}}{{end}}{{if .Error}} {{.Error}}{{end}}{{if .ResendDisabled}} RESEND_DISABLED{{end}}{{end}}
{{define "verify_email.html"}}{{template "layout.html" .}}{{end}}
{{define "home.html"}}{{template "layout.html" .}}{{end}}
{{define "home_picker_body"}}HOME{{end}}
`))
}

func TestEmailVerificationUnverifiedMyRedirectsToWaiting(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-unverified"
	identity := testIdentityForSessions(now, map[string]AccountUser{
		sessionID: {
			IdentityUserID: "user-1",
			Email:          "unverified@example.com",
			Status:         "active",
			EmailVerified:  boolPtr(false),
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
	identity := testIdentityForSessions(now, map[string]AccountUser{
		sessionID: {
			IdentityUserID: "user-1",
			Email:          "waiting@example.com",
			Status:         "active",
			EmailVerified:  boolPtr(false),
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
	if loc := rec.Header().Get("Location"); loc != onboardingPath() {
		t.Fatalf("location = %q, want %s", loc, onboardingPath())
	}
	if !completeCalled {
		t.Fatal("expected CompleteEmailVerification to be called")
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
