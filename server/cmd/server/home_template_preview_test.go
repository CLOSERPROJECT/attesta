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
	if strings.Contains(body, `id="home-panel-preview" data-home-panel="preview" hidden`) {
		t.Fatalf("expected preview panel to remain visible, got: %s", body)
	}
	if !strings.Contains(body, `scrollIntoView({ behavior, block: "start" })`) {
		t.Fatalf("expected nav click to scroll to panel, got: %s", body)
	}

	if strings.Contains(body, `/process//substep/1.1/complete`) {
		t.Fatalf("did not expect preview to render a submit form action, got: %s", body)
	}
}

func TestHomeTemplateDefaultsToInstancesWithoutInstances(t *testing.T) {
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

	if !strings.Contains(compactBody, `data-home-default-panel="instances"`) &&
		!strings.Contains(compactBody, `data-home-default-panel=" instances "`) {
		t.Fatalf("expected instances to be default without instances, got: %s", body)
	}
}

func TestHomeTemplateRendersInstancesPagination(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	view := HomeView{
		PageBase: PageBase{
			WorkflowKey:  "workflow",
			WorkflowPath: "/w/workflow",
			WorkflowName: "Demo workflow",
		},
		Sort:            "status",
		StatusFilter:    "active",
		CurrentPage:     2,
		TotalPages:      3,
		PageNumbers:     []int{1, 2, 3},
		HasPreviousPage: true,
		HasNextPage:     true,
		PreviousPage:    1,
		NextPage:        3,
		Processes: []ProcessListItem{
			{ID: "process-13", Status: "active", Percent: 25, DoneSubsteps: 1, TotalSubsteps: 4, CreatedAt: "1 Mar 2026 at 10:00 UTC"},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "home_body", view); err != nil {
		t.Fatalf("render home template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	if !strings.Contains(compactBody, `aria-label="Stream instances pagination"`) {
		t.Fatalf("expected stream instances pagination nav, got: %s", body)
	}
	if !strings.Contains(compactBody, `/w/workflow/?sort=status&filter=active`) {
		t.Fatalf("expected pagination to preserve sort/filter, got: %s", body)
	}
	if !strings.Contains(compactBody, `#home-panel-instances`) {
		t.Fatalf("expected pagination links to target instances panel, got: %s", body)
	}
	if !strings.Contains(compactBody, `page=3`) {
		t.Fatalf("expected next page link, got: %s", body)
	}
}

func TestHomeTemplateHighlightsProcessWhenItIsUsersTurn(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	view := HomeView{
		PageBase: PageBase{
			WorkflowKey:  "workflow",
			WorkflowPath: "/w/workflow",
			WorkflowName: "Demo workflow",
		},
		Processes: []ProcessListItem{
			{
				ID:              "process-1",
				Status:          "active",
				Percent:         25,
				DoneSubsteps:    1,
				TotalSubsteps:   4,
				CreatedAt:       "1 Mar 2026 at 10:00 UTC",
				HasUserTurn:     true,
				UserTurnSubstep: "1.2",
				UserTurnTitle:   "Record input",
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "home_body", view); err != nil {
		t.Fatalf("render home template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `class="process-item process-user-turn"`) {
		t.Fatalf("expected highlighted process class, got: %s", body)
	}
	if !strings.Contains(body, `status-tag status-tag-compact status-active`) ||
		!strings.Contains(body, `Your turn`) {
		t.Fatalf("expected user turn status tag, got: %s", body)
	}
}
