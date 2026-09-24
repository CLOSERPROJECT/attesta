package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoginTemplateShowsForgotPasswordLink(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "login_body", LoginView{}); err != nil {
		t.Fatalf("render login template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `href="/reset"`) {
		t.Fatalf("expected reset action link, got: %s", body)
	}
	if !strings.Contains(body, "Forgot password?") {
		t.Fatalf("expected forgot password copy, got: %s", body)
	}
}

func TestLoginTemplateShowsConfirmation(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "login_body", LoginView{Confirmation: "Password reset successfully. Now you can enter with your new credentials."}); err != nil {
		t.Fatalf("render login template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, "Password reset successfully. Now you can enter with your new credentials.") {
		t.Fatalf("expected confirmation copy, got: %s", body)
	}
}

func TestLoginTemplateShowsPlaceholdersAndSignupSwitch(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "login_body", LoginView{ShowSignup: true}); err != nil {
		t.Fatalf("render login template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `placeholder="you@example.com"`) {
		t.Fatalf("expected email placeholder, got: %s", body)
	}
	if !strings.Contains(body, `placeholder="Your password"`) {
		t.Fatalf("expected password placeholder, got: %s", body)
	}
	if !strings.Contains(body, `data-target="password"`) {
		t.Fatalf("expected password toggle target, got: %s", body)
	}
	if !strings.Contains(body, `class="u-divider"`) {
		t.Fatalf("expected separator under CTA, got: %s", body)
	}
	if !strings.Contains(body, `class="muted auth-switch"`) {
		t.Fatalf("expected centered auth switch, got: %s", body)
	}
	if !strings.Contains(body, `href="/signup"`) || !strings.Contains(body, "Sign up") {
		t.Fatalf("expected signup cross-link, got: %s", body)
	}
}

func TestSignupTemplateShowsConfirmPasswordAndLoginSwitch(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "signup_body", SignupView{}); err != nil {
		t.Fatalf("render signup template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `name="confirm_password"`) {
		t.Fatalf("expected confirm password field, got: %s", body)
	}
	if !strings.Contains(body, `data-target="signup-password"`) {
		t.Fatalf("expected password toggle target, got: %s", body)
	}
	if !strings.Contains(body, `data-target="confirm-password"`) {
		t.Fatalf("expected confirm password toggle target, got: %s", body)
	}
	if !strings.Contains(body, `placeholder="Your name"`) {
		t.Fatalf("expected name placeholder, got: %s", body)
	}
	if !strings.Contains(body, `placeholder="you@example.com"`) {
		t.Fatalf("expected email placeholder, got: %s", body)
	}
	if !strings.Contains(body, `class="u-divider"`) {
		t.Fatalf("expected separator under CTA, got: %s", body)
	}
	if !strings.Contains(body, `class="muted auth-switch"`) {
		t.Fatalf("expected centered auth switch, got: %s", body)
	}
	if !strings.Contains(body, `href="/login"`) || !strings.Contains(body, "Log in") {
		t.Fatalf("expected login cross-link, got: %s", body)
	}
	if !strings.Contains(body, "passwords do not match") {
		t.Fatalf("expected client-side mismatch message, got: %s", body)
	}
}
