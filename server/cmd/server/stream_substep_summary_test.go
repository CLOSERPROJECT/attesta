package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestStreamSubstepSummaryTemplateRendersDoneMeta(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	data := map[string]any{
		"Substep": TimelineSubstep{
			SubstepID: "1.1",
			Title:     "Capture batch data",
			Status:    "done",
			DoneAt:    "5 Mar 2026 at 14:30 UTC",
			DoneBy:    "alice@example.com",
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "stream_substep_summary", data); err != nil {
		t.Fatalf("render stream_substep_summary template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	for _, want := range []string{
		`class="stream-substep-summary-main"`,
		`class="stream-substep-summary-copy"`,
		`class="stream-substep-summary-title"`,
		`class="stream-substep-summary-meta"`,
		`class="stream-substep-summary-meta-time"`,
		`class="stream-substep-summary-meta-actor"`,
		"Capture batch data",
		"Completed at 5 Mar 2026 at 14:30 UTC",
		"by alice@example.com",
	} {
		if !strings.Contains(compactBody, want) {
			t.Fatalf("expected %q in rendered substep summary, got: %s", want, body)
		}
	}
}

func TestStreamSubstepSummaryTemplateOmitsDoneMetaWhenNotDone(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	data := map[string]any{
		"Substep": TimelineSubstep{
			SubstepID: "1.1",
			Title:     "Capture batch data",
			Status:    "available",
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "stream_substep_summary", data); err != nil {
		t.Fatalf("render stream_substep_summary template: %v", err)
	}
	body := out.String()

	if strings.Contains(body, `class="stream-substep-summary-meta"`) {
		t.Fatalf("did not expect done meta for non-done substep, got: %s", body)
	}
	if !strings.Contains(body, "Capture batch data") {
		t.Fatalf("expected substep title in summary, got: %s", body)
	}
}

func TestStreamSubstepSummaryTemplateRendersCompletedFallbackWithoutActor(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	data := map[string]any{
		"Substep": TimelineSubstep{
			SubstepID: "1.1",
			Title:     "Capture batch data",
			Status:    "done",
			DoneAt:    "5 Mar 2026 at 14:30 UTC",
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "stream_substep_summary", data); err != nil {
		t.Fatalf("render stream_substep_summary template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `class="stream-substep-summary-meta-actor">completed`) {
		t.Fatalf("expected completed fallback actor label, got: %s", body)
	}
}
