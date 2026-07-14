package main

import (
	"bytes"
	"strings"
	"testing"
)

func testSubstepShellView() StreamTimelineSubstepView {
	return StreamTimelineSubstepView{
		Substep: TimelineSubstep{
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
		},
		HideStatus: false,
	}
}

func TestSubstepShellTemplateRendersAccordion(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", testSubstepShellView()); err != nil {
		t.Fatalf("render substep_shell template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	for _, want := range []string{
		`class="substep substep-available"`,
		`class="substep-accordion js-process-substep-panel"`,
		`data-substep-id="1.1"`,
		`class="substep-details"`,
		`<span class="status">available</span>`,
		"Capture batch data",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered substep shell, got: %s", want, body)
		}
	}
	if !strings.Contains(compactBody, `class="substep-accordion js-process-substep-panel" data-substep-id="1.1" open`) {
		t.Fatalf("expected selected substep accordion open, got: %s", body)
	}
}

func TestSubstepShellTemplateHidesStatusWhenHideStatus(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := testSubstepShellView()
	view.HideStatus = true

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", view); err != nil {
		t.Fatalf("render substep_shell template: %v", err)
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

func TestSubstepShellTemplateDispatchesToSubstepBody(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", testSubstepShellView()); err != nil {
		t.Fatalf("render substep_shell template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `class="substep-body-form"`) {
		t.Fatalf("expected substep_body form markup, got: %s", body)
	}
}

func TestSubstepShellTemplateRendersMissingBodyMessage(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := testSubstepShellView()
	view.Substep.Body = nil

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", view); err != nil {
		t.Fatalf("render substep_shell template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, "No data form configured for this substep.") {
		t.Fatalf("expected missing-body fallback copy, got: %s", body)
	}
}
