package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLayoutLoadsGoogleFontsWithSwap(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var rendered bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", PageBase{}); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := rendered.String()

	for _, want := range []string{
		`rel="preconnect" href="https://fonts.googleapis.com"`,
		`rel="preconnect" href="https://fonts.gstatic.com" crossorigin`,
		`fonts.googleapis.com/css2?`,
		`family=Inter:wght@400;500;600;700`,
		`family=Space+Grotesk:wght@600;700`,
		`family=JetBrains+Mono:wght@400;500`,
		`display=swap`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected layout to include %q, got:\n%s", want, body)
		}
	}

	if strings.Contains(body, "Source+Code+Pro") {
		t.Fatalf("expected Source Code Pro to be removed from layout fonts, got:\n%s", body)
	}
}

func TestLayoutShowsAccountMenuWhenLoggedIn(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var rendered bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", PageBase{ShowLogout: true}); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := rendered.String()
	if !strings.Contains(body, `account-menu`) {
		t.Fatalf("expected account menu when logged in, got:\n%s", body)
	}
	if !strings.Contains(body, `href="/my"`) || !strings.Contains(body, "Dashboard") {
		t.Fatalf("expected Dashboard under account menu when logged in, got:\n%s", body)
	}
	if strings.Contains(body, `class="btn btn-ghost btn-lg nav-action">Dashboard`) {
		t.Fatalf("expected Dashboard inside menu, not topbar button, got:\n%s", body)
	}
	if strings.Contains(body, `>Login</a>`) {
		t.Fatalf("expected no Login link when logged in, got:\n%s", body)
	}
}
