package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPlatformAdminTemplateOrganizationInviteAndPagination(t *testing.T) {
	tmpl := parseTestTemplates(t)

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

func TestPlatformAdminCategoriesPanelMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := PlatformAdminView{
		ActivePanel: "categories",
		Categories: []TaxonomyCategoryNode{
			{
				Name:    "Supply Chain",
				Slug:    "supply-chain",
				Icon:    "batch-traceability",
				IconURL: "/static/taxonomy/batch-traceability.svg",
				SubCategories: []TaxonomySubCategoryNode{
					{
						Name:        "Procurement",
						Slug:        "procurement",
						Icon:        "procurement-workflow",
						IconURL:     "/static/taxonomy/procurement-workflow.svg",
						Description: "PO management",
					},
				},
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_body", view); err != nil {
		t.Fatalf("render platform_admin_body: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		"Browse stream discovery categories",
		`class="sidebar-nav-link is-active"`,
		`href="/admin/categories"`,
		`class="platform-admin-taxonomy-list"`,
		`class="platform-admin-taxonomy-icon"`,
		"Supply Chain",
		"supply-chain",
		"batch-traceability",
		"/static/taxonomy/batch-traceability.svg",
		"Procurement",
		"procurement-workflow",
		"PO management",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in categories panel markup, got:\n%s", want, body)
		}
	}

	if strings.Contains(body, `class="panel-head-actions"`) {
		t.Fatalf("did not expect orgs panel-head-actions on categories panel")
	}
}

func TestPlatformAdminCategoriesPanelEmptyState(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := PlatformAdminView{
		ActivePanel: "categories",
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_body", view); err != nil {
		t.Fatalf("render platform_admin_body: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, "No categories yet") {
		t.Fatalf("expected empty state message, got:\n%s", body)
	}
}
