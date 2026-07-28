package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestAdminConsoleTemplateSoftNavContract(t *testing.T) {
	tmpl := parseTestTemplates(t)

	mainCrumbs := BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "MainSlot", Href: "/main-slot", Current: true},
	}}
	view := AdminConsoleView{
		ID:       "admin-console",
		NavLabel: "Test sections",
		Title:    "Test dashboard",
		Subtitle: "Test subtitle",
		Breadcrumbs: BreadcrumbsView{Items: []BreadcrumbItem{
			{Label: "Dashboard", Href: "/my"},
			{Label: "Test", Href: "/test", Current: true},
		}},
		NavItems: []AdminConsoleNavItem{
			{Href: "/test/a", Title: "Alpha", Copy: "First", Active: true},
			{Href: "/test/b", Title: "Beta", Copy: "Second", Active: false},
		},
		MainTemplate: "breadcrumbs",
		MainData:     mainCrumbs,
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "admin_console", view); err != nil {
		t.Fatalf("render admin_console: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`id="admin-console"`,
		`class="page-header"`,
		`class="breadcrumbs"`,
		"<h1>Test dashboard</h1>",
		"Test subtitle",
		`aria-label="Test sections"`,
		`class="sidebar-nav"`,
		`hx-get="/test/a"`,
		`hx-get="/test/b"`,
		`hx-target="#admin-console"`,
		`hx-select="#admin-console"`,
		`hx-swap="outerHTML"`,
		`hx-push-url="true"`,
		`class="sidebar-nav-link is-active"`,
		`aria-current="page"`,
		"MainSlot",
		`href="/main-slot"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in admin_console, got:\n%s", want, body)
		}
	}
}
