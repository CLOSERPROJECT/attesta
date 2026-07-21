package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestBreadcrumbsTemplateRendersLinksAndCurrent(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: "Alpha", Href: "/w/alpha/"},
		{Label: "Instance: Batch 1", Href: ""},
	}}
	if err := tmpl.ExecuteTemplate(&out, "breadcrumbs", view); err != nil {
		t.Fatalf("render breadcrumbs: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`aria-label="Breadcrumb"`,
		`class="breadcrumbs"`,
		`href="/"`,
		">Streams<",
		`href="/w/alpha/"`,
		">Alpha<",
		`aria-current="page"`,
		"Instance: Batch 1",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in breadcrumbs, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, `href="">`) {
		t.Fatalf("current crumb must not be an empty-href link, got:\n%s", body)
	}
}

func TestBreadcrumbsTemplateEmptyItemsRendersNothing(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "breadcrumbs", BreadcrumbsView{}); err != nil {
		t.Fatalf("render breadcrumbs: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Fatalf("expected empty output, got: %q", out.String())
	}
}

func TestProcessPageHeaderRendersBreadcrumbs(t *testing.T) {
	tmpl := parseTestTemplates(t)
	view := ProcessPageView{
		PageBase: PageBase{
			Body:         "process_body",
			WorkflowKey:  "wf-a",
			WorkflowName: "Alpha Stream",
		},
		ProcessID:    "abc123",
		InstanceName: "Batch 1",
		Breadcrumbs:  buildProcessBreadcrumbs("wf-a", "Alpha Stream", "Batch 1", "abc123"),
		Detail: StreamInstanceDetailView{
			WorkflowKey: "wf-a",
			ProcessID:   "abc123",
		},
	}
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "process_body", view); err != nil {
		t.Fatalf("render process_body: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`class="breadcrumbs"`,
		">Streams<",
		">Alpha Stream<",
		"Instance: Batch 1",
		`aria-current="page"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in process header, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, "page-header-back") || strings.Contains(body, ">Back<") {
		t.Fatalf("expected Back link removed, got:\n%s", body)
	}
}
