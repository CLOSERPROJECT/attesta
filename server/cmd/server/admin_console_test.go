package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWantsAdminConsolePartial(t *testing.T) {
	t.Parallel()

	full := httptest.NewRequest(http.MethodGet, "/admin/orgs", nil)
	if wantsAdminConsolePartial(full) {
		t.Fatal("non-HTMX must be false")
	}

	results := httptest.NewRequest(http.MethodGet, "/admin/orgs?q=a", nil)
	results.Header.Set("HX-Request", "true")
	results.Header.Set("HX-Target", "platform-admin-results")
	if wantsAdminConsolePartial(results) {
		t.Fatal("orgs search HTMX must stay on results partial")
	}

	console := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	console.Header.Set("HX-Request", "true")
	console.Header.Set("HX-Target", "admin-console")
	if !wantsAdminConsolePartial(console) {
		t.Fatal("sidebar soft-nav must request admin_console partial")
	}
}

func TestPlatformAdminConsoleNav(t *testing.T) {
	t.Parallel()
	view := PlatformAdminView{ActivePanel: "categories", Breadcrumbs: buildPlatformAdminBreadcrumbs("categories")}
	c := platformAdminConsole(view)
	if c.MainTemplate != "platform_admin_main" {
		t.Fatalf("MainTemplate = %q", c.MainTemplate)
	}
	if len(c.NavItems) != 2 || !c.NavItems[1].Active || c.NavItems[1].Href != "/admin/categories" {
		t.Fatalf("unexpected nav: %+v", c.NavItems)
	}
	if c.Subtitle != "Manage stream discovery taxonomy" {
		t.Fatalf("Subtitle = %q", c.Subtitle)
	}
	if c.NavItems[1].Copy != "Manage stream discovery taxonomy" {
		t.Fatalf("Categories nav Copy = %q", c.NavItems[1].Copy)
	}
}

func TestOrgAdminConsoleNav(t *testing.T) {
	t.Parallel()
	view := OrgAdminView{ActivePanel: "roles", Breadcrumbs: buildOrgAdminBreadcrumbs("roles")}
	c := orgAdminConsole(view)
	if c.MainTemplate != "org_admin_main" {
		t.Fatalf("MainTemplate = %q", c.MainTemplate)
	}
	if len(c.NavItems) != 3 || !c.NavItems[1].Active || c.NavItems[1].Href != organizationPath("roles") {
		t.Fatalf("unexpected nav: %+v", c.NavItems)
	}
}
