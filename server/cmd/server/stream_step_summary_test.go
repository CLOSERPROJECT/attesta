package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestStepSummaryTemplateRendersExtendedMetadata(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	summary := StepSummaryView{
		StepID:           "1",
		Title:            "Production",
		OrganizationName: "Acme Org",
		OrgLogoURL:       "https://example.com/logo.png",
		CompletedAt:      "2026-03-05T14:30:00Z",
		CompletedAtHuman: "5 Mar 2026 at 14:30 UTC",
		SubstepCount:     3,
	}
	if err := tmpl.ExecuteTemplate(&out, "stream_step_summary", summary); err != nil {
		t.Fatalf("render stream_step_summary template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	for _, want := range []string{
		`class="stream-step-summary-main"`,
		`class="stream-step-summary-org-mark"`,
		`src="https://example.com/logo.png"`,
		`class="tip"`,
		`data-tooltip="Organization"`,
		`class="stream-step-summary-title"`,
		`class="stream-step-summary-meta"`,
		`stream-step-summary-meta-icon`,
		"Acme Org",
		`data-tooltip="Completed at"`,
		`aria-label="Completed at"`,
		`class="js-local-datetime"`,
		`datetime="2026-03-05T14:30:00Z"`,
		"5 Mar 2026 at 14:30 UTC",
		`data-tooltip="Steps"`,
		`aria-label="Steps"`,
		"Production",
	} {
		if !strings.Contains(compactBody, want) {
			t.Fatalf("expected %q in rendered step summary, got: %s", want, body)
		}
	}
	if !strings.Contains(compactBody, `</span> 3 </span>`) {
		t.Fatalf("expected substep count 3 after Steps icon, got: %s", body)
	}
}

func TestStepSummaryTemplateHidesOrgMarkWhenFlagSet(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	summary := StepSummaryView{
		StepID:           "1",
		Title:            "Production",
		OrganizationName: "Acme Org",
		OrgLogoURL:       "https://example.com/logo.png",
		HideOrgMark:      true,
	}
	if err := tmpl.ExecuteTemplate(&out, "stream_step_summary", summary); err != nil {
		t.Fatalf("render stream_step_summary template: %v", err)
	}
	body := out.String()

	if strings.Contains(body, `class="stream-step-summary-org-mark"`) {
		t.Fatalf("did not expect org mark when HideOrgMark is true, got: %s", body)
	}
	if !strings.Contains(body, "Acme Org") {
		t.Fatalf("expected organization name in meta, got: %s", body)
	}
}

func TestStepSummaryTemplateOmitsOptionalFields(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	summary := StepSummaryView{
		StepID: "2",
		Title:  "Review",
	}
	if err := tmpl.ExecuteTemplate(&out, "stream_step_summary", summary); err != nil {
		t.Fatalf("render stream_step_summary template: %v", err)
	}
	body := out.String()

	if strings.Contains(body, `data-tooltip="Completed at"`) {
		t.Fatalf("did not expect completed-at meta, got: %s", body)
	}
	if strings.Contains(body, `data-tooltip="Steps"`) {
		t.Fatalf("did not expect substep count meta, got: %s", body)
	}
	if !strings.Contains(body, "No organization") {
		t.Fatalf("expected no-organization fallback, got: %s", body)
	}
	if strings.Contains(body, `class="stream-step-summary-org-mark"`) {
		t.Fatalf("did not expect org mark square for building fallback, got: %s", body)
	}
	if !strings.Contains(body, `class="icon-svg icon-svg-md"`) {
		t.Fatalf("expected building icon fallback, got: %s", body)
	}
}

func TestBuildStepSummaryRollsUpCompletedAt(t *testing.T) {
	doneAt := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)
	def := WorkflowDef{
		Steps: []WorkflowStep{{
			StepID:           "1",
			Title:            "Review materials",
			Order:            1,
			OrganizationSlug: "org-a",
			Substep: []WorkflowSub{{
				SubstepID: "1.1",
				Title:     "Check batch",
				Order:     1,
			}},
		}},
	}
	process := &Process{
		Progress: map[string]ProcessStep{
			"1.1": {
				State:  "done",
				DoneAt: &doneAt,
			},
		},
	}

	summary := buildStepSummary(def.Steps[0], sortedSubsteps(def.Steps[0]), process, map[string]string{"org-a": "Acme Org"})
	if summary.OrganizationName != "Acme Org" {
		t.Fatalf("organization name = %q, want Acme Org", summary.OrganizationName)
	}
	if summary.CompletedAt != "2026-03-05T14:30:00Z" {
		t.Fatalf("completedAt = %q, want RFC3339 timestamp", summary.CompletedAt)
	}
	if summary.CompletedAtHuman != "5 Mar 2026 at 14:30 UTC" {
		t.Fatalf("completedAtHuman = %q, want human-readable time", summary.CompletedAtHuman)
	}
	if summary.SubstepCount != 1 {
		t.Fatalf("substepCount = %d, want 1", summary.SubstepCount)
	}
}
