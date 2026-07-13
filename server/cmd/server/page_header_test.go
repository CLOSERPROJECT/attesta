package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPageHeaderTemplateRendersOptionalFields(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	header := PageHeaderView{
		Title:    "Main Workflow",
		BackHref: "/w/workflow/",
		Subtitle: "Pilot batch",
		Meta:     "process-1",
	}
	if err := tmpl.ExecuteTemplate(&out, "page_header", header); err != nil {
		t.Fatalf("render page_header template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="page-header"`,
		`class="page-header-back"`,
		`class="page-header-body"`,
		`class="page-header-titles"`,
		`class="page-header-subtitle"`,
		`class="page-header-meta-id"`,
		"Main Workflow",
		"Pilot batch",
		"process-1",
		`href="/w/workflow/"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered header, got: %s", want, body)
		}
	}
}

func TestPageHeaderTemplateOmitsBackLinkWhenUnset(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	header := PageHeaderView{
		Title:       "Choose a stream",
		Description: "Select a stream to start or continue process tracking.",
	}
	if err := tmpl.ExecuteTemplate(&out, "page_header", header); err != nil {
		t.Fatalf("render page_header template: %v", err)
	}
	body := out.String()

	if strings.Contains(body, `class="page-header-back"`) {
		t.Fatalf("expected no back link, got: %s", body)
	}
	if !strings.Contains(body, "Choose a stream") {
		t.Fatalf("expected title copy, got: %s", body)
	}
}

func TestPageHeaderTemplateOmitsEmptyOptionalFields(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	header := PageHeaderView{
		Title:    "Main Workflow",
		BackHref: "/w/workflow/",
	}
	if err := tmpl.ExecuteTemplate(&out, "page_header", header); err != nil {
		t.Fatalf("render page_header template: %v", err)
	}
	body := out.String()

	if strings.Contains(body, `class="page-header-subtitle"`) {
		t.Fatalf("expected no subtitle block, got: %s", body)
	}
	if strings.Contains(body, `class="page-header-meta-id"`) {
		t.Fatalf("expected no meta block, got: %s", body)
	}
}
