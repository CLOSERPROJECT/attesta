package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestStreamInstanceCardTemplateRendersDetailLink(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	card := StreamInstanceCard{
		ID:            "process-1",
		Name:          "Pilot batch",
		Status:        "available",
		StatusLabel:   "Available",
		DetailHref:    "/w/workflow/process/process-1",
		Percent:       25,
		DoneSubsteps:  1,
		TotalSubsteps: 4,
		CreatedAt:     "1 Mar 2026 at 10:00 UTC",
	}
	if err := tmpl.ExecuteTemplate(&out, "stream_instance_card", card); err != nil {
		t.Fatalf("render stream_instance_card template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="stream-instance-card stream-instance-card-available"`,
		`class="stream-instance-card-link"`,
		`class="stream-instance-card-body"`,
		`class="stream-instance-card-head"`,
		`class="stream-instance-card-title"`,
		`class="stream-instance-card-name"`,
		`class="stream-instance-card-id"`,
		`class="stream-instance-card-progress-line"`,
		`class="stream-instance-card-progress-fill"`,
		`class="stream-instance-card-meta"`,
		`class="stream-instance-card-meta-primary"`,
		`href="/w/workflow/process/process-1"`,
		"Pilot batch",
		"process-1",
		"complete (25%)",
		"1 /",
		"4",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered card, got: %s", want, body)
		}
	}
}

func TestStreamInstanceCardTemplateRendersStatusIcon(t *testing.T) {
	tmpl := parseTestTemplates(t)

	tests := []struct {
		status string
		marker string
	}{
		{status: "done", marker: `class="stream-instance-card-icon status-done"`},
		{status: "terminated", marker: `class="stream-instance-card-icon status-terminated"`},
		{status: "active", marker: `class="stream-instance-card-icon status-active"`},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			var out bytes.Buffer
			card := StreamInstanceCard{
				ID:         "process-1",
				Status:     tc.status,
				DetailHref: "/w/workflow/process/process-1",
			}
			if err := tmpl.ExecuteTemplate(&out, "stream_instance_card", card); err != nil {
				t.Fatalf("render stream_instance_card template: %v", err)
			}
			if !strings.Contains(out.String(), tc.marker) {
				t.Fatalf("expected %q in rendered card, got: %s", tc.marker, out.String())
			}
		})
	}
}
