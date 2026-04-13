package main

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoginTemplateShowsForgotPasswordLink(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

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
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "login_body", LoginView{Confirmation: "Password reset successfully. Now you can enter with your new credentials."}); err != nil {
		t.Fatalf("render login template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, "Password reset successfully. Now you can enter with your new credentials.") {
		t.Fatalf("expected confirmation copy, got: %s", body)
	}
}
