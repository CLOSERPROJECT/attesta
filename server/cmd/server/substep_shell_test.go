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
				Palette:     "blue",
				FormSchema:  `{"type":"object"}`,
			},
		},
		HideStatus: false,
	}
}

func TestSubstepShellTemplateRendersTerminatedStatus(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := testSubstepShellView()
	view.Substep.Status = processStatusTerminated
	view.Substep.Body.Status = processStatusTerminated

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", view); err != nil {
		t.Fatalf("render substep_shell template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="substep substep-terminated"`,
		`<span class="status" aria-label="Status: TERMINATED">TERMINATED</span>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered substep shell, got: %s", want, body)
		}
	}
}

func TestSubstepShellTemplateRendersSkippedStatus(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := testSubstepShellView()
	view.Substep.Status = "skipped"
	view.Substep.Body.Status = "skipped"

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", view); err != nil {
		t.Fatalf("render substep_shell template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="substep substep-skipped"`,
		`<span class="status" aria-label="Status: skipped">skipped</span>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered substep shell, got: %s", want, body)
		}
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
		`<span class="status" aria-label="Status: available">available</span>`,
		"Capture batch data",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered substep shell, got: %s", want, body)
		}
	}
	if !strings.Contains(compactBody, `class="substep-accordion js-process-substep-panel" data-substep-id="1.1" aria-labelledby="substep-1-1-heading" open`) {
		t.Fatalf("expected selected substep accordion open, got: %s", body)
	}
}

func TestSubstepShellTemplateAccessibleAccordionName(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", testSubstepShellView()); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`id="substep-1-1-heading"`,
		`aria-labelledby="substep-1-1-heading"`,
		`Capture batch data`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q, got: %s", want, body)
		}
	}
}

func TestSubstepShellSummaryIsFocusable(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", testSubstepShellView()); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `class="substep-accordion-summary"`) {
		t.Fatalf("expected substep summary, got: %s", body)
	}
	if strings.Contains(body, `tabindex="-1"`) {
		t.Fatalf("substep summary must stay in tab order; got tabindex=-1: %s", body)
	}
}

func TestSubstepShellStatusHasAccessibleLabel(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", testSubstepShellView()); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `<span class="status" aria-label="Status: available">`) {
		t.Fatalf("expected status aria-label, got: %s", body)
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

func TestSubstepShellTemplateRendersDoneSubstepMetaFromBody(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := testSubstepShellView()
	view.Substep.Status = "locked"
	view.Substep.DoneAt = ""
	view.Substep.DoneBy = ""
	view.Substep.Body.Status = "done"
	view.Substep.Body.DoneAt = "5 Mar 2026 at 14:30 UTC"
	view.Substep.Body.DoneAtISO = "2026-03-05T14:30:00Z"
	view.Substep.Body.DoneBy = "alice@example.com"

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", view); err != nil {
		t.Fatalf("render substep_shell template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="substep-meta"`,
		`class="js-local-datetime"`,
		`datetime="2026-03-05T14:30:00Z"`,
		"5 Mar 2026 at 14:30 UTC",
		"alice@example.com",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered substep shell, got: %s", want, body)
		}
	}
}

func TestSubstepShellTemplateRendersDoneSubstepMetaClasses(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := testSubstepShellView()
	view.Substep.Body = nil
	view.Substep.Status = "done"
	view.Substep.DoneAt = "5 Mar 2026 at 14:30 UTC"
	view.Substep.DoneAtISO = "2026-03-05T14:30:00Z"
	view.Substep.DoneBy = "alice@example.com"

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", view); err != nil {
		t.Fatalf("render substep_shell template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="substep-meta"`,
		`class="substep-meta-time"`,
		`class="substep-meta-actor"`,
		"<strong>Completed at:</strong>",
		`class="js-local-datetime"`,
		`datetime="2026-03-05T14:30:00Z"`,
		"5 Mar 2026 at 14:30 UTC",
		"<strong>Operator:</strong>",
		"alice@example.com",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered substep shell, got: %s", want, body)
		}
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

	if !strings.Contains(body, "No data form configured for this substep") {
		t.Fatalf("expected missing-body fallback copy, got: %s", body)
	}
}
