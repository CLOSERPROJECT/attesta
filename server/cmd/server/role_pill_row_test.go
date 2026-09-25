package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRolePillRowTemplate(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := RolePillRowView{
		Label: "Required roles:",
		Size:  "sm",
		Class: "u-m-0",
		Pills: []RolePillView{
			{Label: "QA", Palette: "emerald"},
			{Label: "Ops", Palette: "orange"},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "role_pill_row", view); err != nil {
		t.Fatalf("render role_pill_row: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`class="role-pill-row u-m-0"`,
		`class="role-pill-label"`,
		"Required roles:",
		`class="pill role-pill pill-sm"`,
		`data-role-palette="emerald"`,
		">QA<",
		`data-role-palette="orange"`,
		">Ops<",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in role_pill_row, got:\n%s", want, body)
		}
	}
}

func TestRolePillRowTemplateEmptyPillsRendersNothing(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "role_pill_row", RolePillRowView{}); err != nil {
		t.Fatalf("render role_pill_row: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Fatalf("expected empty output, got: %q", out.String())
	}
}

func TestRolePillRowFromOrgAdminOptions(t *testing.T) {
	got := rolePillRowFromOrgAdminOptions([]OrgAdminRoleOption{
		{Slug: "qa", Name: "QA Reviewer", Palette: "emerald"},
		{Slug: " ", Name: "", Palette: "red"},
	}, "lg")
	if got.Size != "lg" || len(got.Pills) != 1 || got.Pills[0].Label != "QA Reviewer" {
		t.Fatalf("unexpected row: %#v", got)
	}
}

func TestRolePillRowSelectedFromOrgAdminOptions(t *testing.T) {
	got := rolePillRowSelectedFromOrgAdminOptions([]OrgAdminRoleOption{
		{Slug: "qa", Name: "QA", Palette: "emerald", Selected: true},
		{Slug: "ops", Name: "Ops", Palette: "orange", Selected: false},
	}, "sm")
	if got.Size != "sm" || len(got.Pills) != 1 || got.Pills[0].Label != "QA" {
		t.Fatalf("unexpected selected row: %#v", got)
	}
}

func TestRolePillRowFromSubstepBody(t *testing.T) {
	required := rolePillRowFromSubstepBody(SubstepBodyView{
		RoleBadges: []SubstepRoleBadge{{ID: "qa", Label: "QA", Palette: "emerald"}},
	})
	if required.Label != "Required role:" || required.Size != "sm" || required.Class != "u-m-0" {
		t.Fatalf("unexpected required row: %#v", required)
	}
	done := rolePillRowFromSubstepBody(SubstepBodyView{
		Status:     "done",
		RoleBadges: []SubstepRoleBadge{{ID: "qa", Label: "QA", Palette: "emerald"}},
	})
	if done.Label != "Completed by role:" {
		t.Fatalf("unexpected done label: %q", done.Label)
	}
	multi := rolePillRowFromSubstepBody(SubstepBodyView{
		RoleBadges: []SubstepRoleBadge{
			{ID: "qa", Label: "QA", Palette: "emerald"},
			{ID: "ops", Label: "Ops", Palette: "orange"},
		},
	})
	if multi.Label != "Required roles:" {
		t.Fatalf("unexpected multi label: %q", multi.Label)
	}
}
