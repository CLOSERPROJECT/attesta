package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestBackLinkTemplateRendersHrefAndLabel(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := BackLinkView{Href: onboardingPath(), Label: "Get started"}
	if err := tmpl.ExecuteTemplate(&out, "back_link", view); err != nil {
		t.Fatalf("render back_link: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`class="back-link"`,
		`href="/my/onboarding"`,
		"Get started",
		`class="icon-svg`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in back_link, got:\n%s", want, body)
		}
	}
}

func TestBackLinkTemplateEmptyFieldsRenderNothing(t *testing.T) {
	tmpl := parseTestTemplates(t)

	cases := []BackLinkView{
		{},
		{Href: "/my/onboarding"},
		{Label: "Get started"},
	}
	for _, view := range cases {
		var out bytes.Buffer
		if err := tmpl.ExecuteTemplate(&out, "back_link", view); err != nil {
			t.Fatalf("render back_link %+v: %v", view, err)
		}
		if strings.TrimSpace(out.String()) != "" {
			t.Fatalf("expected empty output for %+v, got: %q", view, out.String())
		}
	}
}
