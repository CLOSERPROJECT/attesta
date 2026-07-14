package main

import (
	"bytes"
	"strings"
	"testing"
)

func testStreamTimelineView() StreamInstanceDetailView {
	return StreamInstanceDetailView{
		WorkflowKey: "workflow",
		ProcessID:   "process-1",
		Timeline: []TimelineStep{{
			Summary: StepSummaryView{
				StepID:           "1",
				Title:            "Production",
				OrganizationName: "Acme Org",
				OrgLogoURL:       "https://example.com/logo.png",
				SubstepCount:     1,
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
					FormSchema:  `{"type":"object"}`,
				},
			}},
		}},
	}
}

func TestStreamTimelineTemplateRendersStepsAndSubsteps(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "stream_timeline", testStreamTimelineView()); err != nil {
		t.Fatalf("render stream_timeline template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	for _, want := range []string{
		`class="stream-timeline-list"`,
		`class="stream-timeline-step"`,
		`class="stream-timeline-step-summary"`,
		`class="stream-step-summary-org-mark"`,
		`src="https://example.com/logo.png"`,
		`class="stream-timeline-substeps"`,
		`class="substep substep-available"`,
		`class="substep-accordion js-process-substep-panel"`,
		`data-substep-id="1.1"`,
		`class="substep-details"`,
		`class="stream-step-summary-meta"`,
		`<span class="status">available</span>`,
		"Production",
		"<strong>Organization:</strong>",
		"Acme Org",
		"<strong>Substeps:</strong>",
		"Capture batch data",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered timeline, got: %s", want, body)
		}
	}
	if !strings.Contains(compactBody, `class="substep-accordion js-process-substep-panel" data-substep-id="1.1" open`) {
		t.Fatalf("expected selected substep accordion open, got: %s", body)
	}
}

func TestStreamTimelineTemplateHidesStatusWhenHideStatus(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := testStreamTimelineView()
	view.HideStatus = true

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "stream_timeline", view); err != nil {
		t.Fatalf("render stream_timeline template: %v", err)
	}
	body := out.String()

	if strings.Contains(body, `<span class="status">`) {
		t.Fatalf("did not expect status pill when HideStatus is true, got: %s", body)
	}
	if strings.Contains(body, `class="substep substep-available"`) {
		t.Fatalf("did not expect status-colored substep class when HideStatus is true, got: %s", body)
	}
	if !strings.Contains(body, `class="substep"`) {
		t.Fatalf("expected bare substep class, got: %s", body)
	}
}

func TestStreamTimelineTemplateRendersOrgLogoFallback(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := testStreamTimelineView()
	view.Timeline[0].Summary.OrgLogoURL = ""

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "stream_timeline", view); err != nil {
		t.Fatalf("render stream_timeline template: %v", err)
	}
	body := out.String()

	if strings.Contains(body, `src="https://example.com/logo.png"`) {
		t.Fatalf("did not expect org logo img when URL empty, got: %s", body)
	}
	if !strings.Contains(body, `class="stream-step-summary-org-mark"`) {
		t.Fatalf("expected org mark fallback container, got: %s", body)
	}
	if !strings.Contains(body, `class="icon-svg"`) {
		t.Fatalf("expected icon-no-org fallback svg, got: %s", body)
	}
}

func TestStreamTimelineTemplateRendersMissingBodyMessage(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := testStreamTimelineView()
	view.Timeline[0].Substeps[0].Body = nil

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "stream_timeline", view); err != nil {
		t.Fatalf("render stream_timeline template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, "No data form configured for this substep.") {
		t.Fatalf("expected missing-action fallback copy, got: %s", body)
	}
}

func TestStreamTimelineTemplateRendersDoneSubstepMetaClasses(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := testStreamTimelineView()
	view.Timeline[0].Substeps[0].Status = "done"
	view.Timeline[0].Substeps[0].DoneAt = "5 Mar 2026 at 14:30 UTC"
	view.Timeline[0].Substeps[0].DoneBy = "alice@example.com"

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "stream_timeline", view); err != nil {
		t.Fatalf("render stream_timeline template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="substep-meta"`,
		"<strong>Completed at:</strong>",
		"5 Mar 2026 at 14:30 UTC",
		"<strong>Operator:</strong>",
		"alice@example.com",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered timeline, got: %s", want, body)
		}
	}
	if strings.Contains(body, `class="time"`) || strings.Contains(body, `class="actor"`) {
		t.Fatalf("expected labeled meta markup, got: %s", body)
	}
}

func TestStreamTimelineTemplateUsesSubstepTitleHeadingClass(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "stream_timeline", testStreamTimelineView()); err != nil {
		t.Fatalf("render stream_timeline template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `class="substep-title-heading"`) {
		t.Fatalf("expected substep title heading class, got: %s", body)
	}
	if strings.Contains(body, `class="stream-step-summary-title">Capture batch data`) {
		t.Fatalf("substep title must not reuse stream-step-summary-title, got: %s", body)
	}
}
