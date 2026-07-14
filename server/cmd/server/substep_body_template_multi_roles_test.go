package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSubstepBodyTemplateShowsRoleChoiceDialogWhilePostingSlugs(t *testing.T) {
	tmpl := parseTestTemplates(t)

	action := SubstepBodyView{
		WorkflowKey: "workflow",
		ProcessID:   "process-1",
		SubstepID:   "1.1",
		Title:       "Multi Role Substep",
		InputKey:    "value",
		InputType:   "formata",
		Status:      "available",
		Mode:        SubstepBodyModeActionable,
		RoleBadges: []SubstepRoleBadge{
			{ID: "dep1", Label: "Department 1", Palette: "red"},
			{ID: "dep2", Label: "Department 2", Palette: "orange"},
		},
		MatchingRoles: []SubstepRoleOption{
			{Slug: "dep1", Label: "Department 1"},
			{Slug: "dep2", Label: "Department 2"},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_body", action); err != nil {
		t.Fatalf("render substep_body template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `name="activeRole"`) {
		t.Fatalf("expected active role hidden input in template, got body: %s", body)
	}
	if !strings.Contains(body, `data-active-role-input="true"`) {
		t.Fatalf("expected active role input to be marked for modal selection, got body: %s", body)
	}
	if !strings.Contains(body, `data-active-role-dialog="active-role-dialog-process-1-1.1"`) {
		t.Fatalf("expected form to reference active role dialog, got body: %s", body)
	}
	if strings.Contains(body, `<select`) {
		t.Fatalf("expected no inline role selector, got body: %s", body)
	}
	if !strings.Contains(body, `data-active-role-option="true"`) {
		t.Fatalf("expected dialog role options to be marked for JS selection, got body: %s", body)
	}
	if !strings.Contains(body, `value="dep1"`) || !strings.Contains(body, `value="dep2"`) {
		t.Fatalf("expected matching role radio values in template, got body: %s", body)
	}
	if !strings.Contains(body, `>Department 1</span>`) || !strings.Contains(body, `>Department 2</span>`) {
		t.Fatalf("expected dialog labels to use role names, got body: %s", body)
	}
	if !strings.Contains(body, `data-role-palette="red"`) || !strings.Contains(body, `data-role-palette="orange"`) {
		t.Fatalf("expected role palette attributes on badges, got body: %s", body)
	}
	if strings.Contains(body, `style="--pill-bg:`) {
		t.Fatalf("expected no inline role pill styles, got body: %s", body)
	}
}
