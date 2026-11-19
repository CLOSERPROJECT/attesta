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
	domain.Submissions = []org.Submission{
		{JobID: "intake", TaskID: "arrive", JobInstanceID: "T1", Timestamp: "2026-11-19T10:00:00Z", UserID: "ops@test"},
		{JobID: "intake", TaskID: "check", JobInstanceID: "T1", Timestamp: "2026-11-19T11:00:00Z", UserID: "ops@test"},
		{JobID: "intake", TaskID: "check", JobInstanceID: "T1", Timestamp: "2026-11-19T12:00:00Z", UserID: "ops@test"},
		{JobID: "lot_processing", TaskID: "startLot", JobInstanceID: "L-open", Timestamp: "2026-11-19T12:30:00Z", UserID: "line@test"},
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
							ID: "startLot", Title: "Apri lotto", State: org.StreamSubstepAvailable,
							BlockedReason: "",
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
	if steps[0].Substeps[0].OrgActionable || !steps[0].Substeps[1].OrgActionable {
		t.Fatalf("org actionable flags = done=%v available=%v", steps[0].Substeps[0].OrgActionable, steps[0].Substeps[1].OrgActionable)
	}
	if len(steps[0].Substeps[0].Submissions) != 1 {
		t.Fatalf("once submissions = %#v", steps[0].Substeps[0].Submissions)
	}
	if len(steps[0].Substeps[1].Submissions) != 2 {
		t.Fatalf("many submissions = %#v", steps[0].Substeps[1].Submissions)
	}
	if steps[0].Substeps[1].Submissions[0].Timestamp != "2026-11-19T11:00:00Z" {
		t.Fatalf("many must be oldest-first, got %#v", steps[0].Substeps[1].Submissions)
	}
	if steps[1].Summary.Title != "Lavorazione ×2" || len(steps[1].Rows) != 2 {
		t.Fatalf("each step = %#v", steps[1])
	}
	open := steps[1].Rows[0]
	if open.Substeps[0].Href != "/app/recycling/i/T1/lot_processing/L-open/startLot" {
		t.Fatalf("open href = %q", open.Substeps[0].Href)
	}
	if len(open.Substeps[0].Submissions) != 1 || open.Substeps[0].OrgActionable {
		t.Fatalf("open startLot submissions/actionable = %#v", open.Substeps[0])
	}
	if !open.Substeps[1].OrgActionable || len(open.Substeps[1].Submissions) != 0 {
		t.Fatalf("weigh actionable/empty submissions = %#v", open.Substeps[1])
	}
	wait := steps[1].Rows[1]
	wantSeed := "/app/recycling/i/T1/lot_processing/new/startLot?seed.lotId=L-wait"
	if wait.Substeps[0].Href != wantSeed {
		t.Fatalf("seed href = %q, want %q", wait.Substeps[0].Href, wantSeed)
	}
	if !wait.Substeps[0].OrgActionable {
		t.Fatalf("provisional can-start must be org-actionable")
	}
	if len(wait.Substeps[0].Submissions) != 0 {
		t.Fatalf("provisional must not attach submissions, got %#v", wait.Substeps[0].Submissions)
	}
	if wait.Substeps[1].Href != "" {
		t.Fatalf("provisional non-start must not link, got %q", wait.Substeps[1].Href)
	}
}

func TestOrgTimelineSubmissionsSkipsEmpty(t *testing.T) {
	if got := orgTimelineSubmissions(org.EmptyDomain(), "intake", "T1", "arrive"); got != nil {
		t.Fatalf("empty = %#v", got)
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

func TestSubstepShellRendersOrgActionableAndSubmissions(t *testing.T) {
	tmpl := parseTestTemplates(t)
	view := StreamTimelineSubstepView{
		Substep: TimelineSubstep{
			SubstepID:     "intake.check",
			Title:         "Check",
			Status:        "available",
			OrgActionable: true,
			Href:          "/app/recycling/i/T1/intake/T1/check",
			Submissions: []TimelineSubmissionView{
				{Timestamp: "2026-11-19T11:00:00Z", TimestampHuman: "19 Nov 2026 at 11:00 UTC", UserID: "ops@test"},
				{Timestamp: "2026-11-19T12:00:00Z", TimestampHuman: "19 Nov 2026 at 12:00 UTC", UserID: "ops@test"},
			},
		},
	}
	var out strings.Builder
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", view); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	for _, want := range []string{
		`data-org-actionable="true"`,
		"Submissions (2)",
		`datetime="2026-11-19T11:00:00Z"`,
		"ops@test",
		`href="/app/recycling/i/T1/intake/T1/check"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in:\n%s", want, body)
		}
	}
}

func TestSubstepShellOmitsEmptySubmissionsShell(t *testing.T) {
	tmpl := parseTestTemplates(t)
	view := StreamTimelineSubstepView{
		Substep: TimelineSubstep{
			SubstepID: "intake.check",
			Title:     "Check",
			Status:    "available",
			Href:      "/app/recycling/i/T1/intake/T1/check",
		},
	}
	var out strings.Builder
	if err := tmpl.ExecuteTemplate(&out, "substep_shell", view); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	if strings.Contains(body, "substep-org-submissions") || strings.Contains(body, "Submissions (") {
		t.Fatalf("empty submissions must omit shell, got:\n%s", body)
	}
	if strings.Contains(body, `data-org-actionable="true"`) {
		t.Fatalf("Formata/non-actionable must omit data-org-actionable, got:\n%s", body)
	}
}
