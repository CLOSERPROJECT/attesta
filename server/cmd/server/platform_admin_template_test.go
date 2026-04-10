package main

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlatformAdminTemplateOrganizationInviteAndPagination(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	view := PlatformAdminView{
		CurrentPage: 1,
		TotalPages:  3,
		PageNumbers: []int{1, 2, 3},
		HasNextPage: true,
		NextPage:    2,
		Organizations: []PlatformAdminOrganizationRow{
			{
				Name:                    "Accepted Org",
				Slug:                    "accepted",
				OrgAdminEmails:          []string{"owner@example.com"},
				PendingOrgAdminEmails:   []string{"pending@example.com"},
				OrgAdminStatus:          "At least one org admin accepted",
				OrgAdminStatusClassName: "accepted",
			},
			{
				Name:                    "Pending Org",
				Slug:                    "pending",
				OrgAdminStatus:          "All org admin invites pending",
				OrgAdminStatusClassName: "pending",
			},
			{
				Name:                    "Missing Org",
				Slug:                    "missing",
				OrgAdminStatus:          "No org admin",
				OrgAdminStatusClassName: "missing",
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_body", view); err != nil {
		t.Fatalf("render platform admin template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	if !strings.Contains(body, `name="invite_email"`) || !strings.Contains(body, "Invite the first org admin now, or add more org admins later") {
		t.Fatalf("expected optional create invite field and helper text, got: %s", body)
	}
	if !strings.Contains(body, "At least one org admin accepted") || !strings.Contains(body, "All org admin invites pending") || !strings.Contains(body, "No org admin") {
		t.Fatalf("expected organization status labels, got: %s", body)
	}
	if !strings.Contains(body, "Current org admins") || !strings.Contains(body, "owner@example.com") {
		t.Fatalf("expected org admin list in invite dialog, got: %s", body)
	}
	if !strings.Contains(compactBody, `aria-label="Organizations pagination"`) ||
		!strings.Contains(compactBody, `m15 18-6-6 6-6`) ||
		!strings.Contains(compactBody, `m9 18 6-6-6-6`) ||
		!strings.Contains(compactBody, `?page=2`) ||
		!strings.Contains(compactBody, `?page=3`) {
		t.Fatalf("expected pagination controls and pages, got: %s", body)
	}
}
