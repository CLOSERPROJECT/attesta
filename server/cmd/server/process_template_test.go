package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestProcessTemplateRendersAccordionSubstepContent(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := ProcessPageView{
		PageBase: PageBase{
			Body:         "process_body",
			WorkflowKey:  "workflow",
			WorkflowName: "Main Workflow",
			WorkflowPath: "/my/streams/workflow",
		},
		ProcessID:    "process-1",
		InstanceName: "Pilot batch",
		Detail: StreamInstanceDetailView{
			WorkflowKey: "workflow",
			ProcessID:   "process-1",
			ProcessDone: true,
			SelectedBody: &SubstepBodyView{
				WorkflowKey: "workflow",
				ProcessID:   "process-1",
				SubstepID:   "1.1",
				Title:       "Capture batch data",
				Status:      "available",
				FormSchema:  `{"type":"object","properties":{"status":{"type":"string"}}}`,
			},
			Timeline: []TimelineStep{{
				Summary: StepSummaryView{
					StepID:           "1",
					Title:            "Production",
					OrganizationName: "Acme Org",
				},
				Expanded: true,
				Substeps: []TimelineSubstep{{
					SubstepID: "1.1",
					Title:     "Capture batch data",
					Status:    "available",
					Selected:  true,
					Palette:   "blue",
					Body: &SubstepBodyView{
						WorkflowKey: "workflow",
						ProcessID:   "process-1",
						SubstepID:   "1.1",
						Title:       "Capture batch data",
						Status:      "available",
						FormSchema:  `{"type":"object","properties":{"status":{"type":"string"}}}`,
					},
				}},
			}},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "process_body", view); err != nil {
		t.Fatalf("render process template: %v", err)
	}

	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")
	if !strings.Contains(body, `id="process-page"`) || !strings.Contains(body, `id="process-page-content"`) {
		t.Fatalf("expected process page wrapper and content target, got: %s", body)
	}
	if !strings.Contains(body, `Pilot batch`) || !strings.Contains(body, `process-1`) {
		t.Fatalf("expected instance name and process id, got: %s", body)
	}
	if !strings.Contains(body, `class="substep-accordion js-process-substep-panel"`) {
		t.Fatalf("expected process accordion substep panel, got: %s", body)
	}
	if !strings.Contains(compactBody, `class="substep-accordion js-process-substep-panel" data-substep-id="1.1" aria-labelledby="substep-1-1-heading" open`) {
		t.Fatalf("expected selected accordion substep to render open, got: %s", body)
	}
	if !strings.Contains(body, `class="process-resources-grid"`) || !strings.Contains(body, `class="stream-timeline-list"`) {
		t.Fatalf("expected process resources grid with timeline, got: %s", body)
	}
	if !strings.Contains(body, `class="substep-details"`) {
		t.Fatalf("expected embedded accordion substep details, got: %s", body)
	}
	if strings.Contains(body, `id="timeline"`) || strings.Contains(body, `id="action-area"`) {
		t.Fatalf("expected legacy split-panel targets to be removed, got: %s", body)
	}
}
