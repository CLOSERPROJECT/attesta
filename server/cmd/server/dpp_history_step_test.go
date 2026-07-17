package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDPPHistoryStepTemplateRendersRailAndTimelineStep(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	step := StreamTimelineStepView{
		Step: TimelineStep{
			Summary: StepSummaryView{
				StepID: "1",
				Title:  "Harvest",
			},
			Expanded: true,
		},
		HideStatus: true,
	}
	if err := tmpl.ExecuteTemplate(&out, "dpp_history_step", step); err != nil {
		t.Fatalf("render dpp_history_step template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="dpp-history-item"`,
		`class="dpp-history-rail"`,
		`class="dpp-history-dot"`,
		`class="stream-timeline-step"`,
		"Harvest",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered dpp_history_step, got: %s", want, body)
		}
	}
}
