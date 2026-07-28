package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestOrgAdminTemplateRendersSidebarPanels(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := OrgAdminView{
		ActivePanel: "profile",
		Organization: Organization{
			Name: "Acme Org",
			Slug: "acme-org",
		},
		OrganizationLogoURL: "/my/organization/logo/logo-1",
		Roles: []Role{
			{Slug: "qa-reviewer", Name: "QA Reviewer"},
		},
		RoleRows: []OrgAdminRoleRow{
			{
				Slug:    "qa-reviewer",
				Name:    "QA Reviewer",
				Palette: "emerald",
			},
		},
		Users: []OrgAdminUserRow{
			{
				UserID:    "user-1",
				Email:     "member@example.com",
				Activated: true,
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "org_admin_body", view); err != nil {
		t.Fatalf("render org admin template: %v", err)
	}
	body := out.String()

	for _, marker := range []string{
		`id="admin-console"`,
		`class="sidebar-nav"`,
		`class="sidebar-nav-link is-active"`,
		`hx-get="/my/organization/profile"`,
		`hx-get="/my/organization/roles"`,
		`hx-get="/my/organization/members"`,
		`hx-target="#admin-console"`,
		`id="org-admin-panel-profile"`,
		`name="intent" value="update_org"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected marker %q in output, got: %s", marker, body)
		}
	}

	for _, banned := range []string{
		`data-org-admin-nav=`,
		`data-org-admin-shell`,
		`id="org-admin-panel-roles"`,
		`id="org-admin-panel-members"`,
		`history.pushState`,
	} {
		if strings.Contains(body, banned) {
			t.Fatalf("did not expect %q in single-panel profile render, got: %s", banned, body)
		}
	}

	if count := strings.Count(body, `name="intent" value="update_org"`); count != 1 {
		t.Fatalf("expected one update_org form, got %d in %s", count, body)
	}
}

func TestOrgAdminTemplateRendersRolesPanelOnly(t *testing.T) {
	tmpl := parseTestTemplates(t)
	view := OrgAdminView{
		ActivePanel: "roles",
		Organization: Organization{
			Name: "Acme Org",
			Slug: "acme-org",
		},
		RoleRows: []OrgAdminRoleRow{
			{Slug: "qa-reviewer", Name: "QA Reviewer", Palette: "emerald"},
		},
	}
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "org_admin_body", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `id="org-admin-panel-roles"`) {
		t.Fatalf("expected roles panel, got: %s", body)
	}
	if strings.Contains(body, `id="org-admin-panel-profile"`) || strings.Contains(body, `id="org-admin-panel-members"`) {
		t.Fatalf("roles render must not include other panels, got: %s", body)
	}
}
