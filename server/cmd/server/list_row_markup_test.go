package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestOrgAdminListRowMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := OrgAdminView{
		Header: PageHeaderView{
			Title:       "Organization admin dashboard",
			Description: "Manage organization settings, roles, and members",
			BackHref:    "/",
		},
		Organization: Organization{Name: "Acme Org", Slug: "acme-org"},
		RoleRows: []OrgAdminRoleRow{
			{Slug: "qa-reviewer", Name: "QA Reviewer", Palette: "emerald", InUse: false},
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
		t.Fatalf("render org_admin_body: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="list-rows"`,
		`class="list-row"`,
		`class="list-row-main"`,
		`class="list-row-actions"`,
		`class="user-email"`,
		`class="user-tags"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in org admin markup, got:\n%s", want, body)
		}
	}

	for _, legacy := range []string{
		`class="roles-list"`,
		`class="roles-item"`,
		`class="users-list"`,
		`class="users-item"`,
		`class="user-main"`,
		`class="user-actions"`,
	} {
		if strings.Contains(body, legacy) {
			t.Fatalf("did not expect legacy class %q in org admin markup", legacy)
		}
	}

	// Roles pill lives inside list-row-main (not a bare first child beside actions).
	rowIdx := strings.Index(body, `class="list-row"`)
	if rowIdx < 0 {
		t.Fatal("expected list-row")
	}
	snippet := body[rowIdx:]
	mainIdx := strings.Index(snippet, `class="list-row-main"`)
	actionsIdx := strings.Index(snippet, `class="list-row-actions"`)
	pillIdx := strings.Index(snippet, `class="pill pill-lg role-pill"`)
	if mainIdx < 0 || actionsIdx < 0 || pillIdx < 0 {
		t.Fatal("expected list-row-main, list-row-actions, and role pill")
	}
	if !(mainIdx < pillIdx && pillIdx < actionsIdx) {
		t.Fatalf("expected role pill inside list-row-main before list-row-actions")
	}
}

func TestPlatformAdminListRowMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := PlatformAdminView{
		Header: PageHeaderView{
			Title:       "Platform admin dashboard",
			Description: "Create and manage organizations",
			BackHref:    "/",
		},
		Organizations: []PlatformAdminOrganizationRow{
			{
				Name:                    "Accepted Org",
				Slug:                    "accepted",
				OrgAdminStatus:          "At least one org admin accepted",
				OrgAdminStatusClassName: "accepted",
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_results", view); err != nil {
		t.Fatalf("render platform_admin_results: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="list-rows"`,
		`class="list-row"`,
		`class="list-row-main"`,
		`class="list-row-actions"`,
		`class="platform-admin-item-copy"`,
		`class="platform-admin-item-name"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in platform admin markup, got:\n%s", want, body)
		}
	}

	for _, legacy := range []string{
		`class="platform-admin-list"`,
		`class="platform-admin-item"`,
		`class="platform-admin-item-main"`,
		`class="user-actions"`,
	} {
		if strings.Contains(body, legacy) {
			t.Fatalf("did not expect legacy class %q in platform admin markup", legacy)
		}
	}
}
