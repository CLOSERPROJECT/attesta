package org

import (
	"strings"
	"testing"
)

func TestBuildOrgTimelineStepsEachGroupAndSeed(t *testing.T) {
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"intake", "lot_processing"},
			Values: map[string]any{
				"intake": Job{
					Title: "Ingresso",
					Tasks: []TaskConfig{
						{"id": "arrive", "title": "Arrive"},
						{"id": "check", "title": "Check", "submit": "many"},
					},
				},
				"lot_processing": Job{
					Title:      "Lavorazione",
					InstanceID: "$.lotId",
					Tasks: []TaskConfig{
						{"id": "startLot", "title": "Apri lotto"},
						{"id": "weigh", "title": "Pesa", "submit": "many"},
					},
				},
			},
		},
	}
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T1", Label: "T1"},
		{JobID: "lot_processing", ID: "L-open", Label: "Lotto L-open"},
	}
	domain.Submissions = []Submission{
		{JobID: "intake", TaskID: "arrive", JobInstanceID: "T1", Timestamp: "2026-11-19T10:00:00Z", UserID: "ops@test"},
		{JobID: "intake", TaskID: "check", JobInstanceID: "T1", Timestamp: "2026-11-19T11:00:00Z", UserID: "ops@test"},
		{JobID: "intake", TaskID: "check", JobInstanceID: "T1", Timestamp: "2026-11-19T12:00:00Z", UserID: "ops@test"},
		{JobID: "lot_processing", TaskID: "startLot", JobInstanceID: "L-open", Timestamp: "2026-11-19T12:30:00Z", UserID: "line@test"},
	}
	timeline := StreamTimeline{Steps: []StreamTimelineStep{
		{
			JobID: "intake", Title: "Ingresso",
			Tasks: []StreamTimelineTask{
				{ID: "arrive", Title: "Arrive", State: StreamSubstepDone},
				{ID: "check", Title: "Check", State: StreamSubstepAvailable},
			},
		},
		{
			JobID: "lot_processing", Title: "Lavorazione",
			Rows: []StreamTimelineRow{
				{
					ID: "L-open", Label: "Lotto L-open",
					Tasks: []StreamTimelineTask{
						{ID: "startLot", Title: "Apri lotto", State: StreamSubstepDone},
						{ID: "weigh", Title: "Pesa", State: StreamSubstepAvailable},
					},
				},
				{
					ID: "L-wait", Label: "Lotto L-wait",
					Tasks: []StreamTimelineTask{
						{ID: "startLot", Title: "Apri lotto", State: StreamSubstepAvailable},
						{ID: "weigh", Title: "Pesa", State: StreamSubstepLocked},
					},
				},
			},
		},
	}}

	hrefs := TimelineHrefPolicyFromFocus("recycling", mustTimelineInstance(t, "T1"))
	steps := BuildOrgTimelineSteps(cfg, domain, timeline, hrefs)
	if len(steps) != 2 {
		t.Fatalf("steps = %#v", steps)
	}
	if steps[0].Title != "Ingresso" || len(steps[0].Substeps) != 2 || len(steps[0].Rows) != 0 {
		t.Fatalf("once step = %#v", steps[0])
	}
	if steps[0].Substeps[0].Href != "/app/recycling/i/T1/intake/T1/arrive" {
		t.Fatalf("once href = %q", steps[0].Substeps[0].Href)
	}
	if steps[0].Substeps[0].OrgActionable || !steps[0].Substeps[1].OrgActionable {
		t.Fatalf("org actionable flags = done=%v available=%v", steps[0].Substeps[0].OrgActionable, steps[0].Substeps[1].OrgActionable)
	}
	if steps[0].Substeps[0].SubmitMany || !steps[0].Substeps[1].SubmitMany {
		t.Fatalf("submitMany flags = arrive=%v check=%v", steps[0].Substeps[0].SubmitMany, steps[0].Substeps[1].SubmitMany)
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
	if steps[1].Title != "Lavorazione ×2" || len(steps[1].Rows) != 2 {
		t.Fatalf("each step = %#v", steps[1])
	}
	open := steps[1].Rows[0]
	if open.Substeps[0].Href != "/app/recycling/i/T1/lot_processing/L-open/startLot" {
		t.Fatalf("open href = %q", open.Substeps[0].Href)
	}
	wait := steps[1].Rows[1]
	wantSeed := "/app/recycling/i/T1/lot_processing/new/startLot?seed.lotId=L-wait"
	if wait.Substeps[0].Href != wantSeed {
		t.Fatalf("seed href = %q, want %q", wait.Substeps[0].Href, wantSeed)
	}
	if !wait.Substeps[0].OrgActionable {
		t.Fatalf("provisional can-start must be org-actionable")
	}
	if wait.Substeps[1].Href != "" {
		t.Fatalf("provisional non-start must not link, got %q", wait.Substeps[1].Href)
	}
}

func TestBuildOrgTimelineStepsStatusRollup(t *testing.T) {
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"once_mixed", "once_done", "each_action", "each_done", "each_empty", "each_finished"},
			Values: map[string]any{
				"once_mixed": Job{
					Title: "Once mixed",
					Tasks: []TaskConfig{
						{"id": "a", "title": "A"},
						{"id": "b", "title": "B"},
					},
				},
				"once_done": Job{
					Title: "Once done",
					Tasks: []TaskConfig{
						{"id": "a", "title": "A"},
						{"id": "b", "title": "B"},
					},
				},
				"each_action": Job{
					Title:      "Each action",
					InstanceID: "$.lotId",
					Tasks: []TaskConfig{
						{"id": "start", "title": "Start"},
						{"id": "weigh", "title": "Weigh", "submit": "many"},
					},
				},
				"each_done": Job{
					Title:      "Each done",
					InstanceID: "$.lotId",
					Tasks: []TaskConfig{
						{"id": "start", "title": "Start"},
						{"id": "weigh", "title": "Weigh", "submit": "many"},
					},
				},
				"each_empty": Job{
					Title:      "Each empty",
					InstanceID: "$.lotId",
					Tasks: []TaskConfig{
						{"id": "start", "title": "Start"},
					},
				},
				"each_finished": Job{
					Title:      "Each finished",
					InstanceID: "$.lotId",
					Tasks: []TaskConfig{
						{"id": "start", "title": "Start"},
						{"id": "weigh", "title": "Weigh", "submit": "many"},
					},
				},
			},
		},
	}
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{
		{JobID: "once_mixed", ID: "T1"},
		{JobID: "once_done", ID: "T1"},
		{JobID: "each_action", ID: "L1"},
		{JobID: "each_done", ID: "L1"},
		{JobID: "each_finished", ID: "L1"},
	}
	timeline := StreamTimeline{Steps: []StreamTimelineStep{
		{
			JobID: "once_mixed", Title: "Once mixed",
			Tasks: []StreamTimelineTask{
				{ID: "a", Title: "A", State: StreamSubstepDone},
				{ID: "b", Title: "B", State: StreamSubstepAvailable},
			},
		},
		{
			JobID: "once_done", Title: "Once done",
			Tasks: []StreamTimelineTask{
				{ID: "a", Title: "A", State: StreamSubstepDone},
				{ID: "b", Title: "B", State: StreamSubstepDone},
			},
		},
		{
			JobID: "each_action", Title: "Each action",
			Rows: []StreamTimelineRow{
				{
					ID: "L1", Label: "L1",
					Tasks: []StreamTimelineTask{
						{ID: "start", Title: "Start", State: StreamSubstepDone},
						{ID: "weigh", Title: "Weigh", State: StreamSubstepAvailable},
					},
				},
				{
					ID: "L2", Label: "L2",
					Tasks: []StreamTimelineTask{
						{ID: "start", Title: "Start", State: StreamSubstepDone},
						{ID: "weigh", Title: "Weigh", State: StreamSubstepDone},
					},
				},
			},
		},
		{
			JobID: "each_done", Title: "Each done",
			Rows: []StreamTimelineRow{
				{
					ID: "L1", Label: "L1", Finished: true,
					Tasks: []StreamTimelineTask{
						{ID: "start", Title: "Start", State: StreamSubstepDone},
						{ID: "weigh", Title: "Weigh", State: StreamSubstepDone},
					},
				},
			},
		},
		{
			JobID: "each_empty", Title: "Each empty",
			Rows: []StreamTimelineRow{},
		},
		{
			JobID: "each_finished", Title: "Each finished",
			Rows: []StreamTimelineRow{
				{
					ID: "L1", Label: "L1", Finished: true,
					Tasks: []StreamTimelineTask{
						{ID: "start", Title: "Start", State: StreamSubstepDone},
						{ID: "weigh", Title: "Weigh", State: StreamSubstepAvailable},
					},
				},
			},
		},
	}}

	hrefs := TimelineHrefPolicyFromFocus("recycling", mustTimelineInstance(t, "T1"))
	steps := BuildOrgTimelineSteps(cfg, domain, timeline, hrefs)
	if len(steps) != 6 {
		t.Fatalf("steps = %#v", steps)
	}

	if steps[0].Status != string(StreamSubstepAvailable) {
		t.Fatalf("once-step mixed → available, got %q", steps[0].Status)
	}
	if steps[1].Status != string(StreamSubstepDone) {
		t.Fatalf("all done → done, got %q", steps[1].Status)
	}
	if steps[2].Status != string(StreamSubstepAvailable) {
		t.Fatalf("each-step one actionable row → available, got %q", steps[2].Status)
	}
	if steps[2].Rows[0].Status != string(StreamSubstepAvailable) {
		t.Fatalf("actionable row status = %q", steps[2].Rows[0].Status)
	}
	if steps[3].Status != string(StreamSubstepDone) {
		t.Fatalf("each-step all rows done → done, got %q", steps[3].Status)
	}
	if steps[3].Rows[0].Status != string(StreamSubstepDone) {
		t.Fatalf("done row status = %q", steps[3].Rows[0].Status)
	}
	if steps[4].Status != "" || steps[4].StatusLabel != "" {
		t.Fatalf("each-step 0 rows → empty status, got status=%q label=%q", steps[4].Status, steps[4].StatusLabel)
	}
	if steps[5].Rows[0].Status != string(StreamSubstepAvailable) {
		t.Fatalf("finished-ish row still with available child → available, got %q", steps[5].Rows[0].Status)
	}
	if steps[5].Status != string(StreamSubstepAvailable) {
		t.Fatalf("finished-ish step rollup → available, got %q", steps[5].Status)
	}
	if steps[0].StatusLabel != string(StreamSubstepAvailable) {
		t.Fatalf("StatusLabel parity = %q", steps[0].StatusLabel)
	}
}

func TestTimelineAggregateStreamSubstepStates(t *testing.T) {
	if _, ok := aggregateStreamSubstepStates(nil); ok {
		t.Fatal("empty must omit")
	}
	got, ok := aggregateStreamSubstepStates([]StreamSubstepState{StreamSubstepDone, StreamSubstepAvailable, StreamSubstepLocked})
	if !ok || got != StreamSubstepAvailable {
		t.Fatalf("attention-first available = %q ok=%v", got, ok)
	}
	got, ok = aggregateStreamSubstepStates([]StreamSubstepState{StreamSubstepLocked, StreamSubstepPending})
	if !ok || got != StreamSubstepPending {
		t.Fatalf("pending over locked = %q ok=%v", got, ok)
	}
	got, ok = aggregateStreamSubstepStates([]StreamSubstepState{StreamSubstepDone, StreamSubstepLocked})
	if !ok || got != StreamSubstepLocked {
		t.Fatalf("locked over done = %q ok=%v", got, ok)
	}
	got, ok = aggregateStreamSubstepStates([]StreamSubstepState{StreamSubstepDone, StreamSubstepDone})
	if !ok || got != StreamSubstepDone {
		t.Fatalf("all done = %q ok=%v", got, ok)
	}
}

func TestOrgTimelineSubmissionsSkipsEmpty(t *testing.T) {
	if got := OrgTimelineSubmissions(EmptyDomain(), "intake", "T1", "arrive"); got != nil {
		t.Fatalf("empty = %#v", got)
	}
}

func TestAppOrgSeededStartTaskPath(t *testing.T) {
	got := AppOrgSeededStartTaskPath("recycling", "T1", "lot_processing", "startLot", "lotId", "L-wait")
	want := "/app/recycling/i/T1/lot_processing/new/startLot?seed.lotId=L-wait"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAppOrgPlayTaskPath(t *testing.T) {
	got := AppOrgPlayTaskPath("recycling", "intake", "", "truckArrive")
	want := "/app/recycling/new/intake/new/truckArrive"
	if got != want {
		t.Fatalf("blank jobInstance → new: got %q, want %q", got, want)
	}
	got = AppOrgPlayTaskPath("recycling", "intake", "T1", "truckArrive")
	want = "/app/recycling/new/intake/T1/truckArrive"
	if got != want {
		t.Fatalf("explicit jobInstance: got %q, want %q", got, want)
	}
}

func TestBuildOrgTimelineViewBootActionableHref(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	op := FixtureAllRolesOp()
	view := BuildOrgTimelineView(cfg, EmptyDomain(), op, "recycling", TimelineBoot())
	if view.Mode != TimelineModeBoot {
		t.Fatalf("mode = %q, want %q", view.Mode, TimelineModeBoot)
	}
	if view.Instance != "" {
		t.Fatalf("boot instance = %q, want empty", view.Instance)
	}
	if view.Stream != "recycling" {
		t.Fatalf("stream = %q", view.Stream)
	}
	if len(view.Steps) == 0 || len(view.Steps[0].Substeps) == 0 {
		t.Fatalf("boot steps = %#v", view.Steps)
	}
	start := view.Steps[0].Substeps[0]
	if !start.OrgActionable {
		t.Fatalf("boot start must be orgActionable: %#v", start)
	}
	taskID := strings.TrimPrefix(start.SubstepID, "intake.")
	wantHref := AppOrgPlayTaskPath("recycling", "intake", "", taskID)
	if start.Href != wantHref {
		t.Fatalf("boot href = %q, want %q", start.Href, wantHref)
	}
}

func TestBuildOrgTimelineViewInstanceMode(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	op := FixtureAllRolesOp()
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "intake", ID: "T1", Label: "Truck T1"}}
	view := BuildOrgTimelineView(cfg, domain, op, "recycling", mustTimelineInstance(t, "T1"))
	if view.Mode != TimelineModeInstance {
		t.Fatalf("mode = %q, want %q", view.Mode, TimelineModeInstance)
	}
	if view.Instance != "T1" {
		t.Fatalf("instance = %q", view.Instance)
	}
	if len(view.Steps) == 0 || len(view.Steps[0].Substeps) == 0 {
		t.Fatalf("instance steps = %#v", view.Steps)
	}
	href := view.Steps[0].Substeps[0].Href
	if !strings.HasPrefix(href, "/app/recycling/i/T1/") {
		t.Fatalf("instance href = %q", href)
	}
}

func TestBuildOrgTimelineRoleBadgesFromRoleDefs(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	op := FixtureAllRolesOp()
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "intake", ID: "T1", Label: "Truck T1"}}
	view := BuildOrgTimelineView(cfg, domain, op, "recycling", mustTimelineInstance(t, "T1"))
	if len(view.Steps) == 0 {
		t.Fatal("expected intake step")
	}
	var truckArrive *TimelineSubstepDTO
	for i := range view.Steps[0].Substeps {
		sub := &view.Steps[0].Substeps[i]
		if sub.SubstepID == "intake.truckArrive" {
			truckArrive = sub
			break
		}
	}
	if truckArrive == nil {
		t.Fatalf("truckArrive missing: %#v", view.Steps[0].Substeps)
	}
	if len(truckArrive.RoleBadges) != 1 {
		t.Fatalf("truckArrive roleBadges = %#v, want 1", truckArrive.RoleBadges)
	}
	badge := truckArrive.RoleBadges[0]
	if badge.ID != "administration" || badge.Label != "Administration" || badge.Palette != "blue" {
		t.Fatalf("truckArrive badge = %#v, want administration/Administration/blue", badge)
	}

	cfg.Jobs = OrderedObject{
		Keys: []string{"intake"},
		Values: map[string]any{
			"intake": Job{
				Title: "Ingresso",
				Tasks: []TaskConfig{
					{"id": "weigh", "title": "Pesa", "roles": []any{"line", "logistic"}},
				},
			},
		},
	}
	badges := roleBadgesForPath(cfg, TaskPathRef{JobID: "intake", TaskID: "weigh"})
	if len(badges) != 2 {
		t.Fatalf("multi badges = %#v", badges)
	}
	if badges[0].ID != "line" || badges[0].Palette != "orange" {
		t.Fatalf("line badge = %#v", badges[0])
	}
	if badges[1].ID != "logistic" || badges[1].Palette != "cyan" {
		t.Fatalf("logistic badge = %#v", badges[1])
	}
}
