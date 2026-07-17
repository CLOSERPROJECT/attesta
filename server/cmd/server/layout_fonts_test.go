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
		`family=Lato:wght@400;700;900`,
		`family=Space+Grotesk:wght@600;700`,
		`family=JetBrains+Mono:wght@500`,
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
