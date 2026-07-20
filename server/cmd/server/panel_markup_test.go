package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestProcessDownloadsPanelMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := ProcessPageView{
		PageBase:  PageBase{WorkflowPath: "/w/demo"},
		ProcessID: "proc-1",
	}
	if err := tmpl.ExecuteTemplate(&out, "process_downloads", view); err != nil {
		t.Fatalf("render process_downloads: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`id="process-downloads"`,
		`class="panel"`,
		`class="panel-head-actions"`,
		`class="panel-heading"`,
		"<h2>Downloads</h2>",
		"Export attachments and notarized data for this stream",
		`class="btn btn-secondary js-download-link"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in process_downloads markup, got:\n%s", want, body)
		}
	}

	headIdx := strings.Index(body, `class="panel-head-actions"`)
	headingIdx := strings.Index(body, `class="panel-heading"`)
	btnIdx := strings.Index(body, `class="btn btn-secondary js-download-link"`)
	if headIdx == -1 || headingIdx == -1 || btnIdx == -1 {
		t.Fatal("expected panel-head-actions, panel-heading, and action button")
	}
	if !(headIdx < headingIdx && headingIdx < btnIdx) {
		t.Fatalf("expected panel-heading before action button inside panel-head-actions block")
	}
}

func TestProcessDPPPanelMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := ProcessPageView{
		PageBase: PageBase{WorkflowPath: "/w/demo"},
		DPPURL:   "https://example.com/01/01234567890123/10/LOT1/21/SN1",
		DPPGS1:   "01/01234567890123/10/LOT1/21/SN1",
	}
	if err := tmpl.ExecuteTemplate(&out, "process_dpp", view); err != nil {
		t.Fatalf("render process_dpp: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="panel dpp-downloads-section"`,
		`class="panel-head-actions"`,
		`class="panel-heading"`,
		"<h2>Digital Product Passport</h2>",
		"GS1 Digital Link and DPP data for this stream",
		`class="btn btn-primary"`,
		"View DPP data",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in process_dpp markup, got:\n%s", want, body)
		}
	}

	headIdx := strings.Index(body, `class="panel-head-actions"`)
	headingIdx := strings.Index(body, `class="panel-heading"`)
	btnIdx := strings.Index(body, `class="btn btn-primary"`)
	if headIdx == -1 || headingIdx == -1 || btnIdx == -1 {
		t.Fatal("expected panel-head-actions, panel-heading, and action button")
	}
	if !(headIdx < headingIdx && headingIdx < btnIdx) {
		t.Fatalf("expected panel-heading before action button inside panel-head-actions block")
	}
}

func TestProcessTerminationDetailsPanelMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := StreamInstanceDetailView{
		Termination: &ProcessTerminationView{
			Reason:       "Pilot ended early",
			EndedAtHuman: "5 Mar 2026 at 14:30 UTC",
			EndedBy:      "appwrite:user-1",
			SubstepID:    "2.1",
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "process_termination_details", view); err != nil {
		t.Fatalf("render process_termination_details: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="panel"`,
		`class="panel-heading"`,
		"<h2>Stream ended early</h2>",
		"Ended at",
		"5 Mar 2026 at 14:30 UTC",
		"while substep 2.1 was available",
		"Pilot ended early",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in process_termination_details markup, got:\n%s", want, body)
		}
	}

	if strings.Contains(body, `class="panel-head-actions"`) {
		t.Fatalf("heading-only panel must not use panel-head-actions, got:\n%s", body)
	}
}

func TestDPPBodyPanelMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := DPPPageView{
		Header: PageHeaderView{
			Title:    "Demo workflow",
			BackHref: "/",
		},
		DigitalLink: "https://example.com/01/09506000134352/10/LOT1/21/SN1",
		GTIN:        "09506000134352",
		Lot:         "LOT1",
		Serial:      "SN1",
		Workflow: WorkflowDef{
			Name: "Demo workflow",
		},
		Integrity: DPPIntegrityView{
			Root: DPPIntegrityHashView{
				Full:  "abc123full",
				Short: "abc123",
			},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "dpp_body", view); err != nil {
		t.Fatalf("render dpp_body: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="panel"`,
		`class="panel-head-actions"`,
		`class="panel-heading"`,
		"<h2>Demo workflow</h2>",
		`class="btn btn-primary js-share-link"`,
		"Share DPP link",
		"GTIN|09506000134352",
		"Serial|SN1",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in dpp_body markup, got:\n%s", want, body)
		}
	}

	headIdx := strings.Index(body, `class="panel-head-actions"`)
	headingIdx := strings.Index(body, `class="panel-heading"`)
	btnIdx := strings.Index(body, `class="btn btn-primary js-share-link"`)
	if headIdx == -1 || headingIdx == -1 || btnIdx == -1 {
		t.Fatal("expected panel-head-actions, panel-heading, and action button")
	}
	if !(headIdx < headingIdx && headingIdx < btnIdx) {
		t.Fatalf("expected panel-heading before action button inside panel-head-actions block")
	}
}

func TestPlatformAdminPanelMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := PlatformAdminView{
		Header: PageHeaderView{
			Title:       "Platform admin dashboard",
			Description: "Create and manage organizations",
			BackHref:    "/",
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_body", view); err != nil {
		t.Fatalf("render platform_admin_body: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="panel"`,
		`class="panel-head-actions"`,
		`class="panel-heading"`,
		"<h2>Organizations</h2>",
		`class="btn btn-primary"`,
		"Add organization",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in platform_admin_body markup, got:\n%s", want, body)
		}
	}

	headIdx := strings.Index(body, `class="panel-head-actions"`)
	headingIdx := strings.Index(body, `class="panel-heading"`)
	btnIdx := strings.Index(body, `onclick="document.getElementById('create-org-dialog').showModal()"`)
	if headIdx == -1 || headingIdx == -1 || btnIdx == -1 {
		t.Fatal("expected panel-head-actions, panel-heading, and action button")
	}
	if !(headIdx < headingIdx && headingIdx < btnIdx) {
		t.Fatalf("expected panel-heading before action button inside panel-head-actions block")
	}
}

func TestStreamHomeBodyPanelMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := HomeView{
		PageBase: PageBase{
			WorkflowKey:  "workflow",
			WorkflowPath: "/w/workflow",
			WorkflowName: "Demo workflow",
		},
		Header: PageHeaderView{
			Title:       "Demo workflow",
			Description: "Previewable workflow",
			BackHref:    "/",
		},
		ProcessGroups: testHomeProcessGroups(),
	}
	if err := tmpl.ExecuteTemplate(&out, "home_body", view); err != nil {
		t.Fatalf("render home_body: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="rail-layout rail-layout-ready"`,
		`class="panel panel-sticky"`,
		`class="sidebar-nav"`,
		`class="sidebar-nav-link is-active"`,
		`class="sidebar-nav-title"`,
		`class="panel-head-actions"`,
		`class="panel-actions"`,
		"<h2>Stream instances</h2>",
		"View and manage all instances of this stream",
		"View preview",
		"New instance",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in home_body markup, got:\n%s", want, body)
		}
	}

	sidebarIdx := strings.Index(body, `class="panel panel-sticky"`)
	headIdx := strings.Index(body, `class="panel-head-actions"`)
	actionsIdx := strings.Index(body, `class="panel-actions"`)
	headingIdx := strings.Index(body, "<h2>Stream instances</h2>")
	if sidebarIdx == -1 || headIdx == -1 || actionsIdx == -1 || headingIdx == -1 {
		t.Fatal("expected sidebar panel, panel-head-actions, panel-actions, and stream instances heading")
	}
	if !(sidebarIdx < headIdx && headIdx < headingIdx && headingIdx < actionsIdx) {
		t.Fatalf("expected sidebar before instances header; panel-heading before panel-actions inside panel-head-actions")
	}

	railIdx := strings.Index(body, `class="rail-layout rail-layout-ready"`)
	if railIdx == -1 || !(railIdx < sidebarIdx) {
		t.Fatalf("expected rail-layout wrapping sticky sidebar")
	}

	mainStart := strings.Index(body, `class="rail-layout-main home-workflow-panel-main"`)
	mainEnd := strings.Index(body, `class="stream-status-sections"`)
	if mainStart == -1 || mainEnd == -1 || mainStart >= mainEnd {
		t.Fatal("expected rail-layout-main wrapping instances header")
	}
	mainBlock := body[mainStart:mainEnd]
	if strings.Contains(mainBlock, `<section class="panel"`) || strings.Contains(mainBlock, `<section class="panel `) {
		t.Fatalf("rail-layout-main must not wrap instances header in section.panel, got:\n%s", mainBlock)
	}
}

func TestOrgAdminRolesPanelMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := OrgAdminView{
		Header: PageHeaderView{
			Title:       "Organization admin dashboard",
			Description: "Manage organization settings, roles, and members",
			BackHref:    "/",
		},
		Organization: Organization{
			Name: "Acme Org",
			Slug: "acme-org",
		},
		RoleRows: []OrgAdminRoleRow{
			{
				Slug:    "qa-reviewer",
				Name:    "QA Reviewer",
				Palette: "emerald",
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "org_admin_body", view); err != nil {
		t.Fatalf("render org_admin_body: %v", err)
	}
	body := out.String()

	rolesStart := strings.Index(body, `id="org-admin-panel-roles"`)
	if rolesStart == -1 {
		t.Fatalf("expected org-admin roles panel section, got:\n%s", body)
	}
	rolesEnd := strings.Index(body[rolesStart:], `id="org-admin-panel-members"`)
	if rolesEnd == -1 {
		t.Fatalf("expected org-admin members panel section after roles, got:\n%s", body)
	}
	rolesSection := body[rolesStart : rolesStart+rolesEnd]

	for _, want := range []string{
		`class="org-admin-panel-section"`,
		`class="panel-head-actions"`,
		`class="panel-heading"`,
		"<h2>Roles</h2>",
		"Add role",
	} {
		if !strings.Contains(rolesSection, want) {
			t.Fatalf("expected %q in roles panel markup, got:\n%s", want, rolesSection)
		}
	}

	headIdx := strings.Index(rolesSection, `class="panel-head-actions"`)
	headingIdx := strings.Index(rolesSection, `class="panel-heading"`)
	btnIdx := strings.Index(rolesSection, "Add role")
	if headIdx == -1 || headingIdx == -1 || btnIdx == -1 {
		t.Fatal("expected panel-head-actions, panel-heading, and Add role button in roles section")
	}
	if !(headIdx < headingIdx && headingIdx < btnIdx) {
		t.Fatalf("expected panel-heading before Add role button inside panel-head-actions block")
	}
}

func TestLoginPanelMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "login_body", LoginView{}); err != nil {
		t.Fatalf("render login_body: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="panel login"`,
		`class="panel-heading"`,
		"<h1>Login</h1>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in login panel markup, got:\n%s", want, body)
		}
	}

	if strings.Contains(body, `class="panel-head-actions"`) {
		t.Fatalf("heading-only login panel must not use panel-head-actions, got:\n%s", body)
	}
}
