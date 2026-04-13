package main

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

func TestResetSetTemplateShowsConfirmPasswordAndToggles(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "reset_set_body", ResetSetView{
		Title:       "Set New Password",
		SubmitLabel: "Update password",
	}); err != nil {
		t.Fatalf("render reset set template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `name="confirm_password"`) {
		t.Fatalf("expected confirm password field, got: %s", body)
	}
	if !strings.Contains(body, `data-target="password"`) {
		t.Fatalf("expected password toggle target, got: %s", body)
	}
	if !strings.Contains(body, `data-target="confirm-password"`) {
		t.Fatalf("expected confirm password toggle target, got: %s", body)
	}
	if !strings.Contains(body, "icon-eye") || !strings.Contains(body, "icon-eye-off") {
		t.Fatalf("expected eye icons, got: %s", body)
	}
}
