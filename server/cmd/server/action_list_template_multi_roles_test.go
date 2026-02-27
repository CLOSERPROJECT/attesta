package main

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

func TestActionListTemplateShowsAllRoleBadges(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	view := ActionListView{
		WorkflowKey: "workflow",
		ProcessID:   "process-1",
		Actions: []ActionView{
			{
				ProcessID: "process-1",
				SubstepID: "1.1",
				Title:     "Multi Role Substep",
				InputKey:  "value",
				InputType: "string",
				Status:    "available",
				RoleBadges: []ActionRoleBadge{
					{ID: "dep1", Label: "Department 1", Color: "#aaaaaa", Border: "#111111"},
					{ID: "dep2", Label: "Department 2", Color: "#bbbbbb", Border: "#222222"},
				},
				MatchingRoles: []string{"dep1", "dep2"},
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "action_list.html", view); err != nil {
		t.Fatalf("render action list template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, "Department 1") || !strings.Contains(body, "Department 2") {
		t.Fatalf("expected both role labels in template, got body: %s", body)
	}
}
