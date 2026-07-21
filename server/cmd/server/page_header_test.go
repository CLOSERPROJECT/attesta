package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPageHeaderBackRendersHrefAndLabel(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "page_header_back", "/w/workflow/"); err != nil {
		t.Fatalf("render page_header_back template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="page-header-back"`,
		`href="/w/workflow/"`,
		"Back",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered back link, got: %s", want, body)
		}
	}
}
