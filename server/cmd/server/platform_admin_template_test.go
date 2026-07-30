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
		CategoriesEditor: CategoriesEditorView{
			GroupCount: 1,
			LeafCount:  1,
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
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_body", view); err != nil {
		t.Fatalf("render platform_admin_body: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`id="platform-admin-categories"`,
		"Manage stream discovery taxonomy",
		"1 groups · 1 categories",
		`class="sidebar-nav-link is-active"`,
		`href="/admin/categories"`,
		`class="platform-admin-taxonomy-list"`,
		`class="platform-admin-taxonomy-icon"`,
		"Supply Chain",
		"supply-chain",
		"/static/taxonomy/batch-traceability.svg",
		"Procurement",
		"procurement-workflow",
		"PO management",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in categories panel markup, got:\n%s", want, body)
		}
	}

	if strings.Contains(body, `class="panel-head-actions"`) && strings.Contains(body, `name="invite_email"`) {
		t.Fatalf("did not expect org invite fields on categories panel")
	}
}

func TestPlatformAdminCategoriesPanelEmptyState(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := PlatformAdminView{
		ActivePanel:      "categories",
		CategoriesEditor: CategoriesEditorView{},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_body", view); err != nil {
		t.Fatalf("render platform_admin_body: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, "No categories yet") {
		t.Fatalf("expected empty state message, got:\n%s", body)
	}
	if !strings.Contains(body, "0 groups · 0 categories") {
		t.Fatalf("expected empty meta pill counts, got:\n%s", body)
	}
}

func TestPlatformAdminCategoriesPanelNewGroupForm(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := PlatformAdminView{
		ActivePanel: "categories",
		CategoriesEditor: CategoriesEditorView{
			GroupCount: 1,
			LeafCount:  1,
			Form: CategoriesEditorForm{
				Open:  true,
				Level: "group",
				Mode:  "create",
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_categories_panel", view); err != nil {
		t.Fatalf("render platform_admin_categories_panel: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `name="intent"`) || !strings.Contains(body, `value="create"`) {
		t.Fatalf("expected create intent in group form, got:\n%s", body)
	}
	if strings.Contains(body, `name="description"`) || strings.Contains(body, "categories-editor-description") {
		t.Fatalf("group form must not include description field, got:\n%s", body)
	}
}

func TestPlatformAdminCategoriesPanelHTMXActions(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := PlatformAdminView{
		ActivePanel: "categories",
		CategoriesEditor: CategoriesEditorView{
			GroupCount: 1,
			LeafCount:  1,
			IconKeys:   []string{"weee", "batch-traceability"},
			Categories: []TaxonomyCategoryNode{
				{
					Name:        "Supply Chain",
					Slug:        "supply-chain",
					Icon:        "batch-traceability",
					IconURL:     "/static/taxonomy/batch-traceability.svg",
					CanDelete:   false,
					DeleteReason: "Has subcategories",
					CanMoveDown: false,
					SubCategories: []TaxonomySubCategoryNode{
						{
							Name:        "Procurement",
							Slug:        "procurement",
							Icon:        "procurement-workflow",
							IconURL:     "/static/taxonomy/procurement-workflow.svg",
							Description: "PO management",
							CanDelete:   true,
							CanMoveUp:   false,
							CanMoveDown: false,
						},
					},
				},
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_categories_panel", view); err != nil {
		t.Fatalf("render platform_admin_categories_panel: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`hx-get="/admin/categories?new=group"`,
		`hx-target="#platform-admin-categories"`,
		`hx-swap="outerHTML"`,
		`hx-get="/admin/categories?new=sub&amp;parent=supply-chain"`,
		`hx-get="/admin/categories?edit=group&amp;slug=supply-chain"`,
		`hx-get="/admin/categories?edit=sub&amp;parent=supply-chain&amp;slug=procurement"`,
		`hx-post="/admin/categories"`,
		`hx-post="/admin/categories/supply-chain/subcategories"`,
		`hx-sync="#platform-admin-categories:queue all"`,
		`hx-disabled-elt=".platform-admin-taxonomy-reorder"`,
		`name="direction"`,
		`value="up"`,
		`value="down"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in categories panel HTMX markup, got:\n%s", want, body)
		}
	}
}

func TestPlatformAdminCategoriesPanelDeleteDialog(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := PlatformAdminView{
		ActivePanel: "categories",
		CategoriesEditor: CategoriesEditorView{
			GroupCount: 1,
			LeafCount:  1,
			Categories: []TaxonomyCategoryNode{
				{
					Name:      "Empty Group",
					Slug:      "empty-group",
					Icon:      "weee",
					IconURL:   "/static/taxonomy/weee.svg",
					CanDelete: true,
					SubCategories: []TaxonomySubCategoryNode{
						{
							Name:        "Leaf",
							Slug:        "leaf",
							Icon:        "weee",
							IconURL:     "/static/taxonomy/weee.svg",
							CanDelete:   false,
							DeleteReason: "Referenced by a stream",
						},
					},
				},
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_categories_panel", view); err != nil {
		t.Fatalf("render platform_admin_categories_panel: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `<dialog id="delete-group-empty-group" class="dialog">`) {
		t.Fatalf("expected delete dialog for deletable group, got:\n%s", body)
	}
	if strings.Contains(body, `<dialog id="delete-sub-empty-group-leaf" class="dialog">`) {
		t.Fatalf("expected no delete dialog for blocked leaf, got:\n%s", body)
	}
	if !strings.Contains(body, `title="Referenced by a stream"`) {
		t.Fatalf("expected disabled delete title for blocked leaf, got:\n%s", body)
	}
}

func TestPlatformAdminCategoriesPanelFormIconGridAndCancel(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := PlatformAdminView{
		ActivePanel: "categories",
		CategoriesEditor: CategoriesEditorView{
			Form: CategoriesEditorForm{
				Open:  true,
				Level: "leaf",
				Mode:  "create",
				ParentSlug: "supply-chain",
			},
			IconKeys: []string{"weee", "batch-traceability"},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_categories_form", view.CategoriesEditor); err != nil {
		t.Fatalf("render platform_admin_categories_form: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`data-taxonomy-icon-picker`,
		`/static/taxonomy/weee.svg`,
		`/static/taxonomy/batch-traceability.svg`,
		`name="description"`,
		`hx-get="/admin/categories"`,
		`hx-target="#platform-admin-categories"`,
		`id="categories-editor-form"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in leaf create form, got:\n%s", want, body)
		}
	}
}

func TestPlatformAdminCategoriesPanelErrorWhenFormClosed(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := PlatformAdminView{
		ActivePanel: "categories",
		CategoriesEditor: CategoriesEditorView{
			Form: CategoriesEditorForm{
				Open:  false,
				Error: "cannot move further",
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_categories_panel", view); err != nil {
		t.Fatalf("render platform_admin_categories_panel: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `class="error platform-admin-taxonomy-panel-error"`) ||
		!strings.Contains(body, "cannot move further") {
		t.Fatalf("expected panel-level error when form is closed, got:\n%s", body)
	}
	if strings.Contains(body, `id="categories-editor-form"`) {
		t.Fatalf("did not expect open form when showing panel error, got:\n%s", body)
	}
}
