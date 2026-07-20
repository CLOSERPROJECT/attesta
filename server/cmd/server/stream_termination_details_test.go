package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestStreamTerminationDetailsTemplateRendersFields(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := StreamTerminationDetailsView{
		EndedAtHuman: "5 Mar 2026 at 14:30 UTC",
		EndedBy:      "operator@example.com",
		SubstepID:    "2.1",
		Reason:       "Pilot ended early",
	}
	if err := tmpl.ExecuteTemplate(&out, "stream_termination_details", view); err != nil {
		t.Fatalf("render stream_termination_details template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="warning warning--rich stream-termination-details"`,
		`class="stream-termination-details-title"`,
		"Stream ended early",
		`d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"`,
		`class="stream-termination-details-fields"`,
		">Ended at</dt>",
		"5 Mar 2026 at 14:30 UTC",
		">Ended by</dt>",
		"operator@example.com",
		">Current substep</dt>",
		"2.1",
		">Reason</dt>",
		"Pilot ended early",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered termination details, got:\n%s", want, body)
		}
	}

	if strings.Contains(body, `class="panel"`) {
		t.Fatalf("termination details must use warning banner, not panel, got:\n%s", body)
	}
}

func TestStreamTerminationDetailsTemplateShowsMissingReason(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := StreamTerminationDetailsView{
		EndedAtHuman: "5 Mar 2026 at 14:30 UTC",
		EndedBy:      "user-1",
		SubstepID:    "1.2",
	}
	if err := tmpl.ExecuteTemplate(&out, "stream_termination_details", view); err != nil {
		t.Fatalf("render stream_termination_details template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `class="muted">No reason provided</dd>`) {
		t.Fatalf("expected muted missing-reason copy, got:\n%s", body)
	}
}

func TestStreamTerminationDetailsTemplateOmitsEmptyOptionalFields(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := StreamTerminationDetailsView{
		Reason: "Supplier cancelled",
	}
	if err := tmpl.ExecuteTemplate(&out, "stream_termination_details", view); err != nil {
		t.Fatalf("render stream_termination_details template: %v", err)
	}
	body := out.String()

	for _, absent := range []string{
		">Ended at</dt>",
		">Ended by</dt>",
		">Current substep</dt>",
	} {
		if strings.Contains(body, absent) {
			t.Fatalf("expected no %q block, got:\n%s", absent, body)
		}
	}
}
