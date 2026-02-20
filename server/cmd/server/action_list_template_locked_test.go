package main

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

func TestActionListTemplateLockedFormataHooks(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	view := ActionListView{
		WorkflowKey: "workflow",
		ProcessID:   "process-1",
		Actions: []ActionView{
			{
				ProcessID:    "process-1",
				SubstepID:    "1.2",
				Title:        "Locked Formata",
				InputKey:     "payload",
				InputType:    "formata",
				FormSchema:   `{"type":"object"}`,
				FormUISchema: `{"type":"VerticalLayout","elements":[]}`,
				Status:       "locked",
				Disabled:     true,
				Reason:       "Locked by sequence",
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "action_list.html", view); err != nil {
		t.Fatalf("render action list template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, "action-card action-locked") {
		t.Fatalf("expected locked action class hook, got body: %s", body)
	}
	if !strings.Contains(body, `data-formata-disabled="true"`) {
		t.Fatalf("expected disabled formata data hook, got body: %s", body)
	}
}
