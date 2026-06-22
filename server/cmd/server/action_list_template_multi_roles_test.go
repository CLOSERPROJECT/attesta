package main

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

func TestActionListTemplateShowsRoleChoiceDialogWhilePostingSlugs(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	action := ActionView{
		WorkflowKey: "workflow",
		ProcessID:   "process-1",
		SubstepID:   "1.1",
		Title:       "Multi Role Substep",
		InputKey:    "value",
		InputType:   "formata",
		Status:      "available",
		RoleBadges: []ActionRoleBadge{
			{ID: "dep1", Label: "Department 1", Color: template.CSS("#aaaaaa"), Border: template.CSS("#111111")},
			{ID: "dep2", Label: "Department 2", Color: template.CSS("#bbbbbb"), Border: template.CSS("#222222")},
		},
		MatchingRoles: []ActionRoleOption{
			{Slug: "dep1", Label: "Department 1"},
			{Slug: "dep2", Label: "Department 2"},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "action_detail_content.html", action); err != nil {
		t.Fatalf("render action detail template: %v", err)
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
}
