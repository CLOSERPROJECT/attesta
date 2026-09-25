package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerifyTurnstilePostsTokenAndClientIP(t *testing.T) {
	verification := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("secret"); got != "secret-key" {
			t.Errorf("secret = %q", got)
		}
		if got := r.Form.Get("response"); got != "response-token" {
			t.Errorf("response = %q", got)
		}
		if got := r.Form.Get("remoteip"); got != "203.0.113.10" {
			t.Errorf("remoteip = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer verification.Close()

	server := &Server{
		turnstileSiteKey:   "site-key",
		turnstileSecretKey: "secret-key",
		turnstileClient:    verification.Client(),
		turnstileVerifyURL: verification.URL,
	}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("cf-turnstile-response=response-token"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("CF-Connecting-IP", "203.0.113.10")
	if err := req.ParseForm(); err != nil {
		t.Fatalf("parse form: %v", err)
	}

	if err := server.verifyTurnstile(req); err != nil {
		t.Fatalf("verify Turnstile: %v", err)
	}
}

func TestHandleLoginRejectsInvalidTurnstileBeforeIdentity(t *testing.T) {
	called := false
	server := &Server{
		identity: &fakeIdentityStore{createEmailPasswordSessionFunc: func(context.Context, string, string) (IdentitySession, error) {
			called = true
			return IdentitySession{}, nil
		}},
		tmpl:               testTemplates(),
		turnstileSiteKey:   "site-key",
		turnstileSecretKey: "secret-key",
	}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("email=user%40example.com&password=password"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleLogin(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if called {
		t.Fatal("identity login called after a failed Turnstile verification")
	}
}

func TestHandleSignupRejectsInvalidTurnstileBeforeAccountCreation(t *testing.T) {
	called := false
	server := &Server{
		identity: &fakeIdentityStore{createAccountFunc: func(context.Context, string, string, string) (IdentityUser, error) {
			called = true
			return IdentityUser{}, nil
		}},
		tmpl:               testTemplates(),
		turnstileSiteKey:   "site-key",
		turnstileSecretKey: "secret-key",
	}
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "true")
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("name=User&email=user%40example.com&password=long-enough-password&confirm_password=long-enough-password"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleSignup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if called {
		t.Fatal("account creation called after a failed Turnstile verification")
	}
	if !strings.Contains(rec.Body.String(), "Verification failed. Please try again.") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestTurnstileSiteKeyRequiresBothKeys(t *testing.T) {
	if got := (&Server{turnstileSiteKey: "site-key"}).configuredTurnstileSiteKey(); got != "" {
		t.Fatalf("site key without secret = %q", got)
	}
	if got := (&Server{turnstileSecretKey: "secret-key"}).configuredTurnstileSiteKey(); got != "" {
		t.Fatalf("site key without public key = %q", got)
	}
	if got := (&Server{turnstileSiteKey: "site-key", turnstileSecretKey: "secret-key"}).configuredTurnstileSiteKey(); got != "site-key" {
		t.Fatalf("site key = %q", got)
	}
}

func TestAuthTemplatesRenderTurnstileWhenConfigured(t *testing.T) {
	tmpl := parseTestTemplates(t)
	for name, view := range map[string]any{
		"login_body":  LoginView{PageBase: PageBase{TurnstileSiteKey: "site-key"}},
		"signup_body": SignupView{PageBase: PageBase{TurnstileSiteKey: "site-key"}},
	} {
		var out bytes.Buffer
		if err := tmpl.ExecuteTemplate(&out, name, view); err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		body := out.String()
		if !strings.Contains(body, `data-turnstile-gate`) || !strings.Contains(body, `class="turnstile-gate-widget cf-turnstile"`) {
			t.Fatalf("%s missing Turnstile widget: %s", name, body)
		}
		if !strings.Contains(body, `name="cf-turnstile-response"`) || !strings.Contains(body, `hidden`) {
			t.Fatalf("%s should hide the form until verification: %s", name, body)
		}
		if !strings.Contains(body, "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit") {
			t.Fatalf("%s missing Turnstile script: %s", name, body)
		}
	}
}
