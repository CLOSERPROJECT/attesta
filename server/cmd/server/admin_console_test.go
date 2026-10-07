package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTMXTargetIDNilRequest(t *testing.T) {
	t.Parallel()
	if got := htmxTargetID(nil); got != "" {
		t.Fatalf("htmxTargetID(nil) = %q, want empty", got)
	}
}

func TestWantsAdminConsolePartial(t *testing.T) {
	t.Parallel()

	full := httptest.NewRequest(http.MethodGet, "/admin/organizations", nil)
	if wantsAdminConsolePartial(full) {
		t.Fatal("non-HTMX must be false")
	}

	results := httptest.NewRequest(http.MethodGet, "/admin/organizations?q=a", nil)
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
	if len(c.NavItems) != 3 {
		t.Fatalf("unexpected nav length: %+v", c.NavItems)
	}
	if c.NavItems[0].Title != "Organizations" || c.NavItems[0].Href != "/admin/organizations" || c.NavItems[0].Active {
		t.Fatalf("unexpected Organizations nav: %+v", c.NavItems[0])
	}
	if c.NavItems[1].Title != "Streams" || c.NavItems[1].Href != "/admin/streams" || c.NavItems[1].Active {
		t.Fatalf("unexpected Streams nav: %+v", c.NavItems[1])
	}
	if c.NavItems[2].Title != "Categories" || c.NavItems[2].Href != "/admin/categories" || !c.NavItems[2].Active {
		t.Fatalf("unexpected Categories nav: %+v", c.NavItems[2])
	}
	if c.Subtitle != "Manage stream discovery taxonomy" {
		t.Fatalf("Subtitle = %q", c.Subtitle)
	}
	if c.NavItems[2].Copy != "Manage stream discovery taxonomy" {
		t.Fatalf("Categories nav Copy = %q", c.NavItems[2].Copy)
	}
	if c.NavItems[0].HasAttention {
		t.Fatal("Organizations soft-nav must not light Attention without HasOrgCreationAttention")
	}

	streams := platformAdminConsole(PlatformAdminView{ActivePanel: "streams"})
	if streams.Subtitle != "Browse and manage platform streams" {
		t.Fatalf("streams Subtitle = %q", streams.Subtitle)
	}
	if !streams.NavItems[1].Active || streams.NavItems[0].Active || streams.NavItems[2].Active {
		t.Fatalf("streams panel must mark only Streams active: %+v", streams.NavItems)
	}
	if streams.NavItems[1].Copy != "Browse and manage platform streams" {
		t.Fatalf("Streams nav Copy = %q", streams.NavItems[1].Copy)
	}

	withOrgCreation := PlatformAdminView{
		PageBase:    PageBase{HasOrgCreationAttention: true},
		ActivePanel: "organizations",
	}
	orgs := platformAdminConsole(withOrgCreation)
	if !orgs.NavItems[0].HasAttention || orgs.NavItems[0].Title != "Organizations" {
		t.Fatalf("Organizations soft-nav should carry org-creation Attention: %+v", orgs.NavItems[0])
	}
	if !orgs.NavItems[0].Active || orgs.NavItems[1].Active || orgs.NavItems[2].Active {
		t.Fatalf("organizations panel must mark only Organizations active: %+v", orgs.NavItems)
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
	if c.NavItems[2].HasAttention {
		t.Fatal("Members soft-nav must not light Attention without HasJoinRequestAttention")
	}

	defaultPanel := orgAdminConsole(OrgAdminView{})
	if !defaultPanel.NavItems[0].Active || defaultPanel.NavItems[0].Title != "Organization profile" {
		t.Fatalf("empty ActivePanel must default to profile: %+v", defaultPanel.NavItems)
	}

	withJoin := OrgAdminView{
		PageBase:    PageBase{HasJoinRequestAttention: true},
		ActivePanel: "members",
	}
	members := orgAdminConsole(withJoin)
	if !members.NavItems[2].HasAttention || members.NavItems[2].Title != "Members" {
		t.Fatalf("Members soft-nav should carry Join-request Attention: %+v", members.NavItems[2])
	}
}

func TestOrganizationDialogAutoOpenUnknownAction(t *testing.T) {
	t.Parallel()
	view := PlatformAdminView{
		OrganizationDialogAction: "unknown",
		OrganizationError:        "boom",
		OrganizationDialogSlug:   "acme",
	}
	if view.OrganizationDialogAutoOpen("unknown", "acme") {
		t.Fatal("unknown dialog action must not auto-open")
	}
}
