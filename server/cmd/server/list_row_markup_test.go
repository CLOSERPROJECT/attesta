package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestOrgAdminListRowMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := OrgAdminView{
		ActivePanel:  "roles",
		Organization: Organization{Name: "Acme Org", Slug: "acme-org"},
		RoleRows: []OrgAdminRoleRow{
			{Slug: "qa-reviewer", Name: "QA Reviewer", Palette: "emerald", InUse: false, CanDelete: true},
		},
		Users: []OrgAdminUserRow{
			{
				UserID:    "user-1",
				Email:     "member@example.com",
				Activated: true,
				CanDelete: true,
				RoleOptions: []OrgAdminRoleOption{
					{Slug: "qa-reviewer", Name: "QA Reviewer", Palette: "emerald", Selected: true},
				},
			},
		},
	}

	var rolesOut bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rolesOut, "org_admin_body", view); err != nil {
		t.Fatalf("render org_admin_body roles: %v", err)
	}
	rolesBody := rolesOut.String()

	for _, want := range []string{
		`class="list-rows"`,
		`class="list-row"`,
		`class="list-row-main"`,
		`class="list-row-actions"`,
	} {
		if !strings.Contains(rolesBody, want) {
			t.Fatalf("expected %q in roles markup, got:\n%s", want, rolesBody)
		}
	}

	view.ActivePanel = "members"
	var membersOut bytes.Buffer
	if err := tmpl.ExecuteTemplate(&membersOut, "org_admin_body", view); err != nil {
		t.Fatalf("render org_admin_body members: %v", err)
	}
	membersBody := membersOut.String()
	for _, want := range []string{
		`class="user-email"`,
		`class="role-pill-row"`,
	} {
		if !strings.Contains(membersBody, want) {
			t.Fatalf("expected %q in members markup, got:\n%s", want, membersBody)
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
		if strings.Contains(rolesBody, legacy) || strings.Contains(membersBody, legacy) {
			t.Fatalf("did not expect legacy class %q in org admin markup", legacy)
		}
	}

	// Roles pill lives inside list-row-main (not a bare first child beside actions).
	rowIdx := strings.Index(rolesBody, `class="list-row"`)
	if rowIdx < 0 {
		t.Fatal("expected list-row")
	}
	snippet := rolesBody[rowIdx:]
	mainIdx := strings.Index(snippet, `class="list-row-main"`)
	actionsIdx := strings.Index(snippet, `class="list-row-actions"`)
	pillIdx := strings.Index(snippet, `class="pill role-pill"`)
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
		Organizations: []PlatformAdminOrganizationRow{
			{
				Name:                    "Accepted Org",
				Slug:                    "accepted",
				OrgAdminStatus:          "At least one org admin accepted",
				OrgAdminStatusClassName: "accepted",
			},
		},
		PendingOrgCreationRequests: []PlatformAdminOrgCreationRequestRow{
			{
				ID:             "req-1",
				ProposedName:   "Fresh Org",
				ProposedSlug:   "fresh-org",
				RequesterEmail: "newbie@example.com",
				CreatedAt:      "1 Mar 2026 at 12:00 UTC",
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

	var pendingOut bytes.Buffer
	if err := tmpl.ExecuteTemplate(&pendingOut, "platform_admin_main", view); err != nil {
		t.Fatalf("render platform_admin_main: %v", err)
	}
	pendingBody := pendingOut.String()
	for _, want := range []string{
		`class="list-row-main list-row-main-stack"`,
		`aria-label="Approve"`,
		`aria-label="Reject"`,
		`name="intent" value="approve_org_creation"`,
		`name="intent" value="reject_org_creation"`,
		"Fresh Org",
	} {
		if !strings.Contains(pendingBody, want) {
			t.Fatalf("expected %q in pending org creation markup, got:\n%s", want, pendingBody)
		}
	}
	for _, legacy := range []string{
		`>Approve</button>`,
		`>Reject</button>`,
	} {
		if strings.Contains(pendingBody, legacy) {
			t.Fatalf("did not expect labeled approve/reject button %q; use icon buttons", legacy)
		}
	}
}
