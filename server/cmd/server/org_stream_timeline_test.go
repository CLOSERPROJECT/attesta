package main

import (
	"strings"
	"testing"

	"github.com/CLOSERPROJECT/attesta/server/internal/org"
)

func TestOrgJobSeedFieldFromInstanceID(t *testing.T) {
	if got := orgJobSeedField(org.Job{InstanceID: "$.lotId"}); got != "lotId" {
		t.Fatalf("got %q", got)
	}
	if got := orgJobSeedField(org.Job{InstanceID: "$lookup($domain.lot, $.id)"}); got != "" {
		t.Fatalf("complex expr must not seed, got %q", got)
	}
}

func TestAppOrgSeededStartTaskPath(t *testing.T) {
	got := appOrgSeededStartTaskPath("recycling", "T1", "lot_processing", "startLot", "lotId", "L-wait")
	want := "/app/recycling/i/T1/lot_processing/new/startLot?seed.lotId=L-wait"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildOrgTimelineStepsEachGroupAndSeed(t *testing.T) {
	cfg := org.OrgConfig{
		Jobs: org.OrderedObject{
			Keys: []string{"intake", "lot_processing"},
			Values: map[string]any{
				"intake": org.Job{
					Title: "Ingresso",
					Tasks: []org.TaskConfig{
						{"id": "arrive", "title": "Arrive"},
						{"id": "check", "title": "Check"},
					},
				},
				"lot_processing": org.Job{
					Title:      "Lavorazione",
					InstanceID: "$.lotId",
					Tasks: []org.TaskConfig{
						{"id": "startLot", "title": "Apri lotto"},
						{"id": "weigh", "title": "Pesa"},
					},
				},
			},
		},
	}
	domain := org.EmptyDomain()
	domain.JobInstances = []org.JobInstance{
		{JobID: "intake", ID: "T1", Label: "T1"},
		{JobID: "lot_processing", ID: "L-open", Label: "Lotto: L-open"},
	}
	timeline := org.StreamTimeline{Steps: []org.StreamTimelineStep{
		{
			JobID: "intake", Title: "Ingresso",
			Tasks: []org.StreamTimelineTask{
				{ID: "arrive", Title: "Arrive", State: org.StreamSubstepDone},
				{ID: "check", Title: "Check", State: org.StreamSubstepAvailable},
			},
		},
		{
			JobID: "lot_processing", Title: "Lavorazione",
			Rows: []org.StreamTimelineRow{
				{
					ID: "L-open", Label: "Lotto: L-open",
					Tasks: []org.StreamTimelineTask{
						{ID: "startLot", Title: "Apri lotto", State: org.StreamSubstepDone},
						{ID: "weigh", Title: "Pesa", State: org.StreamSubstepAvailable},
					},
				},
				{
					ID: "L-wait", Label: "Lotto: L-wait",
					Tasks: []org.StreamTimelineTask{
						{
							ID: "startLot", Title: "Apri lotto", State: org.StreamSubstepLocked,
							BlockedReason: "Linea occupata: lotto L-open",
						},
						{ID: "weigh", Title: "Pesa", State: org.StreamSubstepLocked},
					},
				},
			},
		},
	}}

	steps := buildOrgTimelineSteps(cfg, domain, timeline, "recycling", "T1")
	if len(steps) != 2 {
		t.Fatalf("steps = %#v", steps)
	}
	if steps[0].Summary.Title != "Ingresso" || len(steps[0].Substeps) != 2 || len(steps[0].Rows) != 0 {
		t.Fatalf("once step = %#v", steps[0])
	}
	if steps[0].Substeps[0].Href != "/app/recycling/i/T1/intake/T1/arrive" {
		t.Fatalf("once href = %q", steps[0].Substeps[0].Href)
	}
	if steps[1].Summary.Title != "Lavorazione ×2" || len(steps[1].Rows) != 2 {
		t.Fatalf("each step = %#v", steps[1])
	}
	open := steps[1].Rows[0]
	if open.Substeps[0].Href != "/app/recycling/i/T1/lot_processing/L-open/startLot" {
		t.Fatalf("open href = %q", open.Substeps[0].Href)
	}
	wait := steps[1].Rows[1]
	if wait.Substeps[0].BlockedReason != "Linea occupata: lotto L-open" {
		t.Fatalf("blocked = %q", wait.Substeps[0].BlockedReason)
	}
	wantSeed := "/app/recycling/i/T1/lot_processing/new/startLot?seed.lotId=L-wait"
	if wait.Substeps[0].Href != wantSeed {
		t.Fatalf("seed href = %q, want %q", wait.Substeps[0].Href, wantSeed)
	}
	if wait.Substeps[1].Href != "" {
		t.Fatalf("provisional non-start must not link, got %q", wait.Substeps[1].Href)
	}
}

func TestStreamTimelineTemplateRendersEachRows(t *testing.T) {
	tmpl := parseTestTemplates(t)
	view := StreamTimelineView{
		Timeline: []TimelineStep{{
			Summary: StepSummaryView{
				StepID:       "2",
				Title:        "Lavorazione ×2",
				HideOrgMark:  true,
				SubstepCount: 2,
			},
			Expanded: true,
			Rows: []TimelineRowGroup{{
				ID: "L-wait", Label: "Lotto: L-wait", Expanded: true,
				Substeps: []TimelineSubstep{{
					SubstepID:     "lot_processing/startLot",
					Title:         "Apri lotto",
					Status:        "locked",
					BlockedReason: "Linea occupata: lotto L-open",
					Href:          "/app/recycling/i/T1/lot_processing/new/startLot?seed.lotId=L-wait",
				}},
			}},
		}},
	}
	var out strings.Builder
	if err := tmpl.ExecuteTemplate(&out, "stream_timeline", view); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	for _, want := range []string{
		`class="stream-timeline-rows"`,
		`class="stream-timeline-row"`,
		"Lavorazione ×2",
		"Lotto: L-wait",
		"Linea occupata: lotto L-open",
		`href="/app/recycling/i/T1/lot_processing/new/startLot?seed.lotId=L-wait"`,
		`class="substep substep-locked"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in:\n%s", want, body)
		}
	}
}
