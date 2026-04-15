package main

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

func TestHomeTemplateRendersSidebarAndReadOnlyPreview(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	view := HomeView{
		PageBase: PageBase{
			WorkflowKey:  "workflow",
			WorkflowPath: "/w/workflow",
			WorkflowName: "Demo workflow",
		},
		WorkflowDescription: "Previewable workflow",
		Processes: []ProcessListItem{
			{ID: "process-1", Status: "active", Percent: 25, DoneSubsteps: 1, TotalSubsteps: 4, CreatedAt: "1 Mar 2026 at 10:00 UTC"},
		},
		Preview: ActionListView{
			Timeline: []TimelineStep{
				{
					StepID:   "1",
					Title:    "Collect",
					Expanded: true,
					Substeps: []TimelineSubstep{
						{
							SubstepID: "1.1",
							Title:     "Record input",
							Status:    "available",
							Selected:  true,
							Action: &ActionView{
								WorkflowKey:   "workflow",
								SubstepID:     "1.1",
								Title:         "Record input",
								Status:        "available",
								FormSchema:    `{"type":"object","properties":{"value":{"type":"string"}}}`,
								ReadOnly:      true,
								Reason:        "Preview only. Start an instance to submit data.",
								MatchingRoles: []string{"dep1"},
							},
						},
					},
				},
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "home_body", view); err != nil {
		t.Fatalf("render home template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	for _, marker := range []string{
		`data-home-nav="instances"`,
		`data-home-nav="preview"`,
		`id="home-panel-instances"`,
		`id="home-panel-preview"`,
		`data-home-root`,
		`data-formata-disabled="true"`,
		`Preview only. Start an instance to submit data.`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected marker %q in output, got: %s", marker, body)
		}
	}
	if !strings.Contains(compactBody, `data-home-default-panel=" instances "`) &&
		!strings.Contains(compactBody, `data-home-default-panel="instances"`) {
		t.Fatalf("expected instances to be default when processes exist, got: %s", body)
	}

	if strings.Contains(body, `/process//substep/1.1/complete`) {
		t.Fatalf("did not expect preview to render a submit form action, got: %s", body)
	}
}

func TestHomeTemplateDefaultsToPreviewWithoutInstances(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	view := HomeView{
		PageBase: PageBase{
			WorkflowKey:  "workflow",
			WorkflowPath: "/w/workflow",
			WorkflowName: "Demo workflow",
		},
		WorkflowDescription: "Previewable workflow",
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "home_body", view); err != nil {
		t.Fatalf("render home template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	if !strings.Contains(compactBody, `data-home-default-panel=" preview "`) &&
		!strings.Contains(compactBody, `data-home-default-panel="preview"`) {
		t.Fatalf("expected preview to be default without instances, got: %s", body)
	}
}
