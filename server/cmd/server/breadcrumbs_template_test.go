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
