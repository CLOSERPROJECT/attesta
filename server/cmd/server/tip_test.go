package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestTipTemplateRendersIconHost(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	data := map[string]any{
		"Tooltip": "Completed at",
		"Class":   "stream-step-summary-meta-icon",
		"Icon":    "icon-clock",
	}
	if err := tmpl.ExecuteTemplate(&out, "tip", data); err != nil {
		t.Fatalf("render tip template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	for _, want := range []string{
		`class="tip stream-step-summary-meta-icon"`,
		`data-tooltip="Completed at"`,
		`aria-label="Completed at"`,
		`tabindex="0"`,
		`class="icon-svg icon-svg-md"`,
	} {
		if !strings.Contains(compactBody, want) {
			t.Fatalf("expected %q in tip markup, got: %s", want, body)
		}
	}
}

func TestTipTemplateRendersImageHost(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	data := map[string]any{
		"Tooltip":  "Organization",
		"ImgSrc":   "https://example.com/logo.png",
		"ImgClass": "stream-step-summary-org-mark",
	}
	if err := tmpl.ExecuteTemplate(&out, "tip", data); err != nil {
		t.Fatalf("render tip template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="tip"`,
		`data-tooltip="Organization"`,
		`src="https://example.com/logo.png"`,
		`class="stream-step-summary-org-mark"`,
		`alt=""`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in tip markup, got: %s", want, body)
		}
	}
}

func TestTipTemplateRendersArbitraryIconDefine(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	data := map[string]any{
		"Tooltip": "Search",
		"Icon":    "icon-search",
	}
	if err := tmpl.ExecuteTemplate(&out, "tip", data); err != nil {
		t.Fatalf("render tip template: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `class="icon-svg`) {
		t.Fatalf("expected icon-search svg via render, got: %s", body)
	}
}

func TestTipTemplateUnknownIconErrors(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	data := map[string]any{
		"Tooltip": "Missing",
		"Icon":    "icon-does-not-exist",
	}
	if err := tmpl.ExecuteTemplate(&out, "tip", data); err == nil {
		t.Fatal("expected error for unknown icon define")
	}
}
