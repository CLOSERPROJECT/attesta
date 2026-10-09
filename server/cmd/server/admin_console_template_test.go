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
		ID:         "admin-console",
		NavLabel:   "Test sections",
		NavHeading: "Platform settings",
		NavItems: []AdminConsoleNavItem{
			{Href: "/test/a", Title: "Alpha", Icon: "icon-building-grid", Active: true},
			{Href: "/test/b", Title: "Beta", Icon: "icon-layers-2", Active: false},
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
		`class="admin-console"`,
		`class="admin-console-nav-heading"`,
		">Platform settings<",
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
		`class="icon-svg`,
		"Alpha",
		"Beta",
		"MainSlot",
		`href="/main-slot"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in admin_console, got:\n%s", want, body)
		}
	}
	for _, banned := range []string{
		`class="page-header"`,
		"sidebar-nav-copy",
		"admin-console-compact",
	} {
		if strings.Contains(body, banned) {
			t.Fatalf("did not expect %q in admin_console, got:\n%s", banned, body)
		}
	}
}
