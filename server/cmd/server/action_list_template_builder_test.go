package main

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

func TestActionListTemplateLockedFormataBuilderHint(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	view := ActionListView{
		WorkflowKey: "workflow",
		ProcessID:   "process-1",
		Actions: []ActionView{
			{
				ProcessID:  "process-1",
				SubstepID:  "1.2",
				Title:      "Formata",
				InputKey:   "payload",
				InputType:  "formata",
				FormSchema: `{"type":"object"}`,
				Status:     "locked",
				Disabled:   true,
				Reason:     "Locked by sequence",
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "action_list.html", view); err != nil {
		t.Fatalf("render action list template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, "Status: locked - Locked by sequence") {
		t.Fatalf("expected locked helper text, got body: %s", body)
	}
}
