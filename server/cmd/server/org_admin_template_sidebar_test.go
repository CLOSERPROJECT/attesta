package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestOrgAdminTemplateRendersSidebarPanels(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := OrgAdminView{
		Header: PageHeaderView{
			Title:       "Organization admin dashboard",
			Description: "Create and manage roles and users",
			BackHref:    "/",
		},
		Organization: Organization{
			Name: "Acme Org",
			Slug: "acme-org",
		},
		OrganizationLogoURL: "/org-admin/logo/logo-1",
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
		`data-org-admin-nav="profile"`,
		`data-org-admin-nav="roles"`,
		`data-org-admin-nav="members"`,
		`id="org-admin-panel-profile"`,
		`id="org-admin-panel-roles"`,
		`id="org-admin-panel-members"`,
		`data-org-admin-shell`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected marker %q in output, got: %s", marker, body)
		}
	}

	if count := strings.Count(body, `name="intent" value="update_org"`); count != 1 {
		t.Fatalf("expected one update_org form, got %d in %s", count, body)
	}
}
