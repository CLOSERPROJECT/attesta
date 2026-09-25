package main

import (
	"bytes"
	"net/url"
	"strings"
	"testing"
)

func TestRolesPickerTemplateRendersRolePills(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := RolesPickerView{
		Options: []RolesPickerOption{
			{Slug: "viewer", Name: "Viewer", Palette: "blue"},
			{Slug: "editor", Name: "Editor", Palette: "emerald", Selected: true},
		},
		RequireSelection: true,
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "roles_picker", view); err != nil {
		t.Fatalf("render roles_picker: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`data-role-picker`,
		`data-role-picker-require-selection`,
		`roles-picker-search-field`,
		`data-role-picker-search-clear`,
		`data-value="viewer"`,
		`data-value="editor"`,
		`data-label="Viewer"`,
		`data-palette="blue"`,
		`data-role-palette="emerald"`,
		`data-selected="true"`,
		`name="roles" value="editor"`,
		"No roles match your search",
		"Try a different name.",
		"Editor",
		"Viewer",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in roles_picker output, got: %s", want, body)
		}
	}
	if strings.Contains(body, "Viewer (viewer)") || strings.Contains(body, "Editor (editor)") {
		t.Fatalf("roles_picker must show pills, not name+slug text, got: %s", body)
	}
	if strings.Contains(body, ">Selected<") {
		t.Fatalf("roles_picker must use check icon, not Selected text, got: %s", body)
	}
	if !strings.Contains(body, `roles-picker-option-check`) || !strings.Contains(body, `M20 6 9 17l-5-5`) {
		t.Fatalf("expected check icon in roles_picker option, got: %s", body)
	}
	if strings.Contains(body, "Select roles") {
		t.Fatalf("toggle should show selected pill, not placeholder, got: %s", body)
	}
}

func TestRolesPickerTemplateEmptyOptions(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "roles_picker", RolesPickerView{}); err != nil {
		t.Fatalf("render roles_picker: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, "No roles available") {
		t.Fatalf("expected empty-state toggle label, got: %s", body)
	}
	if strings.Contains(body, `data-role-picker-menu`) {
		t.Fatalf("empty picker must not render menu, got: %s", body)
	}
}

func TestRolesPickerFromRolesOmitsOrgAdminStanding(t *testing.T) {
	view := rolesPickerFromRoles([]Role{
		{Slug: "org-admin", Name: "Org Admin"},
		{Slug: "viewer", Name: "Viewer", Palette: "sky"},
	}, false)
	if len(view.Options) != 1 || view.Options[0].Slug != "viewer" {
		t.Fatalf("roles picker options = %#v, want viewer only", view.Options)
	}

	selected := rolesPickerFromOrgAdminOptions([]OrgAdminRoleOption{
		{Slug: "org-admin", Name: "Org Admin", Selected: true},
		{Slug: "viewer", Name: "Viewer", Palette: "sky", Selected: true},
	}, false)
	if len(selected.Options) != 1 || selected.Options[0].Slug != "viewer" || !selected.Options[0].Selected {
		t.Fatalf("org-admin options picker = %#v, want selected viewer only", selected.Options)
	}
}

func TestFormRequestsOrgAdmin(t *testing.T) {
	if formRequestsOrgAdmin(nil) {
		t.Fatal("nil form must not request Org admin")
	}
	for _, raw := range []string{"1", "true", "ON", "yes"} {
		if !formRequestsOrgAdmin(url.Values{"is_org_admin": {raw}}) {
			t.Fatalf("expected Org admin for %q", raw)
		}
	}
	if formRequestsOrgAdmin(url.Values{"is_org_admin": {"0"}}) {
		t.Fatal("0 must not request Org admin")
	}
}

func TestOnboardingJoinDialogDisablesSubmitUntilRoles(t *testing.T) {
	tmpl := parseTestTemplates(t)
	view := OnboardingJoinView{
		SelectedOrgSlug:  "acme",
		SelectedOrgName:  "Acme Org",
		SelectedOrgRoles: []Role{{Slug: "viewer", Name: "Viewer", Palette: "sky"}},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "onboarding_join_dialog", view); err != nil {
		t.Fatalf("render onboarding_join_dialog: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`data-role-picker-require-selection`,
		`data-role-picker-submit`,
		`data-role-palette="sky"`,
		`disabled`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in join dialog, got: %s", want, body)
		}
	}
	if strings.Contains(body, "Viewer (viewer)") {
		t.Fatalf("join dialog must use role pills, got: %s", body)
	}
}
