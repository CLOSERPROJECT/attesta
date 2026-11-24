package org

import (
	"testing"
)

func TestMapWorklistStatusToStreamState(t *testing.T) {
	cases := []struct {
		in   TaskStatus
		want StreamSubstepState
	}{
		{TaskStatusActionable, StreamSubstepAvailable},
		{TaskStatusAvailable, StreamSubstepPending},
		{TaskStatusUnavailable, StreamSubstepLocked},
		{TaskStatusDone, StreamSubstepDone},
	}
	for _, tc := range cases {
		if got := MapWorklistStatusToStreamState(tc.in); got != tc.want {
			t.Fatalf("%q → %q, want %q", tc.in, got, tc.want)
		}
	}
}

func mustTimelineInstance(t *testing.T, id string) TimelineFocus {
	t.Helper()
	focus, err := TimelineInstance(id)
	if err != nil {
		t.Fatalf("TimelineInstance(%q): %v", id, err)
	}
	return focus
}

func TestProjectStreamTimelineBootEmptyDomainAnchorAvailable(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	op := FixtureAllRolesOp()

	tl := ProjectStreamTimeline(cfg, EmptyDomain(), op, TimelineBoot())
	if len(tl.Steps) != 2 {
		t.Fatalf("steps = %#v, want intake + lot_processing", tl.Steps)
	}

	intake := tl.Steps[0]
	if intake.JobID != "intake" || len(intake.Rows) != 0 {
		t.Fatalf("intake step = %#v", intake)
	}
	if len(intake.Tasks) == 0 {
		t.Fatal("intake tasks empty")
	}
	if intake.Tasks[0].State != StreamSubstepAvailable {
		t.Fatalf("anchor first task = %q (%s), want available", intake.Tasks[0].State, intake.Tasks[0].ID)
	}
	for i := 1; i < len(intake.Tasks); i++ {
		if intake.Tasks[i].State != StreamSubstepLocked {
			t.Fatalf("intake task[%d] = %#v, want locked", i, intake.Tasks[i])
		}
	}

	lots := tl.Steps[1]
	if lots.JobID != "lot_processing" {
		t.Fatalf("lots job = %q", lots.JobID)
	}
	if len(lots.Rows) != 0 {
		t.Fatalf("boot each rows = %#v, want none", lots.Rows)
	}
	if len(lots.Tasks) != 0 {
		t.Fatalf("boot each tasks = %#v, want none", lots.Tasks)
	}
}

func streamProjectionCfg() OrgConfig {
	cfg := streamWorklistCfg()
	intake := cfg.Jobs.Values["intake"].(Job)
	intake.Title = "Ingresso"
	cfg.Jobs.Values["intake"] = intake
	lp := cfg.Jobs.Values["lot_processing"].(Job)
	lp.Title = "Lavorazione"
	lp.Label = "'Lotto ' & $.lotId"
	cfg.Jobs.Values["lot_processing"] = lp
	return cfg
}

func TestListStreamInstancesListsIdLabelCreatedStatusProgress(t *testing.T) {
	op := OperatorIdentity{UserID: "u", Org: "synth", Roles: []string{"ops"}}
	cfg := streamProjectionCfg()
	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"lot": {
			"L1": {"truckId": "T1", "status": "registered"},
			"L2": {"truckId": "T1", "status": "registered"},
		},
	}
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T1", Label: "Truck T1"},
		{JobID: "intake", ID: "T2", Label: "Truck T2"},
	}
	domain.Submissions = []Submission{
		{
			JobID: "intake", TaskID: "arrive", JobInstanceID: "T1",
			Timestamp: "2026-11-19T10:00:00Z", UserID: "u", Facts: map[string]any{},
		},
		{
			JobID: "intake", TaskID: "check", JobInstanceID: "T1",
			Timestamp: "2026-11-19T11:00:00Z", UserID: "u", Facts: map[string]any{},
		},
		{
			JobID: "intake", TaskID: "arrive", JobInstanceID: "T2",
			Timestamp: "2026-11-19T12:00:00Z", UserID: "u", Facts: map[string]any{},
		},
	}

	list := ListStreamInstances(cfg, domain, op)
	if len(list) != 2 {
		t.Fatalf("len = %d, want 2", len(list))
	}
	a := list[0]
	if a.ID != "T1" || a.Label != "Truck T1" {
		t.Fatalf("first = %#v", a)
	}
	if a.CreatedAt != "2026-11-19T10:00:00Z" {
		t.Fatalf("CreatedAt = %q", a.CreatedAt)
	}
	if a.Status != StreamInstanceActive {
		t.Fatalf("T1 status = %q, want active (each rows unfinished)", a.Status)
	}
	if len(a.Steps) != 2 {
		t.Fatalf("steps = %#v", a.Steps)
	}
	if a.Steps[0].JobID != "intake" || a.Steps[0].Done != 1 || a.Steps[0].Total != 1 {
		t.Fatalf("intake progress = %#v", a.Steps[0])
	}
	if a.Steps[1].JobID != "lot_processing" || a.Steps[1].Done != 0 || a.Steps[1].Total != 2 {
		t.Fatalf("lot progress = %#v, want Done=0 Total=2", a.Steps[1])
	}

	b := list[1]
	if b.ID != "T2" || b.Status != StreamInstanceActive {
		t.Fatalf("T2 = %#v", b)
	}
	if b.CreatedAt != "2026-11-19T12:00:00Z" {
		t.Fatalf("T2 CreatedAt = %q", b.CreatedAt)
	}
	if b.Steps[0].Done != 0 || b.Steps[1].Total != 0 {
		t.Fatalf("T2 progress = %#v", b.Steps)
	}
}

func TestListStreamInstancesDoneWhenAnchorAndEachFinished(t *testing.T) {
	op := OperatorIdentity{UserID: "u", Org: "synth", Roles: []string{"ops"}}
	cfg := streamProjectionCfg()
	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"lot": {"L1": {"truckId": "T1", "status": "fullyProcessed"}},
	}
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T1", Label: "T1"},
		{JobID: "lot_processing", ID: "L1", Label: "Lotto L1"},
	}
	domain.Submissions = []Submission{
		sub("intake", "arrive", "T1"),
		sub("intake", "check", "T1"),
		sub("lot_processing", "start", "L1"),
		sub("lot_processing", "close", "L1"),
	}

	list := ListStreamInstances(cfg, domain, op)
	if len(list) != 1 {
		t.Fatalf("len = %d", len(list))
	}
	if list[0].Status != StreamInstanceDone {
		t.Fatalf("status = %q, want done", list[0].Status)
	}
	if list[0].Steps[1].Done != 1 || list[0].Steps[1].Total != 1 {
		t.Fatalf("lot progress = %#v", list[0].Steps[1])
	}
}

func TestProjectStreamTimelineOnceAndEachNotYetStarted(t *testing.T) {
	op := OperatorIdentity{UserID: "u", Org: "synth", Roles: []string{"ops"}}
	cfg := streamProjectionCfg()
	// Eligible lots exist → start readyWhen true on provisional each rows.
	lp := cfg.Jobs.Values["lot_processing"].(Job)
	lp.Tasks = []TaskConfig{
		{
			"id": "start", "roles": []any{"ops"}, "title": "Apri lotto",
			"readyWhen":     `$count($domain.lot.*[truckId = $stream.id and $not(status = "open")]) >= 1`,
			"blockedReason": `'no eligible lots on ' & $stream.id`,
		},
		{"id": "weigh", "roles": []any{"ops"}, "submit": "many", "title": "Pesa"},
		{"id": "close", "roles": []any{"ops"}, "title": "Chiudi"},
	}
	cfg.Jobs.Values["lot_processing"] = lp

	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"lot": {
			"L-A1": {"truckId": "T-A", "status": "registered"},
			"L-A2": {"truckId": "T-A", "status": "registered"},
		},
	}
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T-A", Label: "T-A"},
	}
	domain.Submissions = []Submission{
		sub("intake", "arrive", "T-A"),
	}

	tl := ProjectStreamTimeline(cfg, domain, op, mustTimelineInstance(t, "T-A"))
	if len(tl.Steps) != 2 {
		t.Fatalf("steps = %#v", tl.Steps)
	}
	intake := tl.Steps[0]
	if intake.JobID != "intake" || len(intake.Rows) != 0 {
		t.Fatalf("intake step = %#v", intake)
	}
	if len(intake.Tasks) != 3 {
		t.Fatalf("intake tasks = %#v", intake.Tasks)
	}
	if intake.Tasks[0].State != StreamSubstepDone {
		t.Fatalf("arrive = %q, want done", intake.Tasks[0].State)
	}
	if intake.Tasks[2].State != StreamSubstepAvailable {
		t.Fatalf("check = %q, want available (actionable→available)", intake.Tasks[2].State)
	}

	lots := tl.Steps[1]
	if lots.JobID != "lot_processing" || len(lots.Tasks) != 0 {
		t.Fatalf("lots step = %#v", lots)
	}
	if len(lots.Rows) != 2 {
		t.Fatalf("rows = %#v, want 2 not-yet-started", lots.Rows)
	}
	for _, row := range lots.Rows {
		if row.Finished {
			t.Fatalf("row %s should not be finished", row.ID)
		}
		wantLabel := "Lotto " + row.ID
		if row.Label != wantLabel {
			t.Fatalf("label = %q, want %q", row.Label, wantLabel)
		}
		if len(row.Tasks) != 3 {
			t.Fatalf("row tasks = %#v", row.Tasks)
		}
		if row.Tasks[0].State != StreamSubstepAvailable {
			t.Fatalf("start on %s = %q, want available", row.ID, row.Tasks[0].State)
		}
		if row.Tasks[1].State != StreamSubstepLocked || row.Tasks[2].State != StreamSubstepLocked {
			t.Fatalf("later tasks on %s = %#v", row.ID, row.Tasks)
		}
	}
}

func TestProjectStreamTimelineNotYetStartedShowsBlockedReason(t *testing.T) {
	op := OperatorIdentity{UserID: "u", Org: "synth", Roles: []string{"ops"}}
	cfg := streamProjectionCfg()
	lp := cfg.Jobs.Values["lot_processing"].(Job)
	lp.Tasks = []TaskConfig{
		{
			"id": "start", "roles": []any{"ops"},
			"readyWhen":     `$count($domain.lot.*[truckId = $stream.id and status = "open"]) = 0 and $count($domain.lot.*[truckId = $stream.id and $not(status = "fullyProcessed") and $not(status = "open")]) >= 1`,
			"blockedReason": `'Linea occupata on ' & $stream.id`,
		},
		{"id": "weigh", "roles": []any{"ops"}, "submit": "many"},
		{"id": "close", "roles": []any{"ops"}},
	}
	cfg.Jobs.Values["lot_processing"] = lp

	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"lot": {
			"L-open": {"truckId": "T-A", "status": "open"},
			"L-wait": {"truckId": "T-A", "status": "registered"},
		},
	}
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T-A", Label: "T-A"},
		{JobID: "lot_processing", ID: "L-open", Label: "Lotto L-open"},
	}
	domain.Submissions = []Submission{
		sub("intake", "arrive", "T-A"),
		sub("lot_processing", "start", "L-open"),
	}

	tl := ProjectStreamTimeline(cfg, domain, op, mustTimelineInstance(t, "T-A"))
	lots := tl.Steps[1]
	var waiting *StreamTimelineRow
	for i := range lots.Rows {
		if lots.Rows[i].ID == "L-wait" {
			waiting = &lots.Rows[i]
			break
		}
	}
	if waiting == nil {
		t.Fatalf("missing L-wait row: %#v", lots.Rows)
	}
	start := waiting.Tasks[0]
	if start.State != StreamSubstepLocked {
		t.Fatalf("start state = %q, want locked", start.State)
	}
	if start.BlockedReason != "Linea occupata on T-A" {
		t.Fatalf("blockedReason = %q", start.BlockedReason)
	}
}

func TestProjectStreamTimelineMapsEveryWorklistStatusIncludingBlockedReason(t *testing.T) {
	op := OperatorIdentity{UserID: "u", Org: "synth", Roles: []string{"ops"}}
	other := OperatorIdentity{UserID: "v", Org: "synth", Roles: []string{"office"}}
	cfg := streamProjectionCfg()
	intake := cfg.Jobs.Values["intake"].(Job)
	intake.Tasks = []TaskConfig{
		{"id": "arrive", "roles": []any{"ops"}},
		{"id": "register", "roles": []any{"ops"}, "submit": "many"},
		{
			"id": "office", "roles": []any{"office"},
		},
		{
			"id": "check", "roles": []any{"ops"},
			"readyWhen":     `false`,
			"blockedReason": `'waiting on weigh for ' & $stream.id`,
		},
	}
	cfg.Jobs.Values["intake"] = intake
	cfg.Stream.Steps = []StreamStep{{Job: "intake"}}

	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "intake", ID: "T1", Label: "T1"}}
	domain.Submissions = []Submission{sub("intake", "arrive", "T1")}

	tlOps := ProjectStreamTimeline(cfg, domain, op, mustTimelineInstance(t, "T1"))
	tasks := tlOps.Steps[0].Tasks
	byID := map[string]StreamTimelineTask{}
	for _, task := range tasks {
		byID[task.ID] = task
	}
	if byID["arrive"].State != StreamSubstepDone {
		t.Fatalf("arrive = %q", byID["arrive"].State)
	}
	if byID["register"].State != StreamSubstepAvailable {
		t.Fatalf("register actionable→available = %q", byID["register"].State)
	}
	if byID["office"].State != StreamSubstepPending {
		t.Fatalf("office available→pending = %q", byID["office"].State)
	}
	if byID["check"].State != StreamSubstepLocked {
		t.Fatalf("check unavailable→locked = %q", byID["check"].State)
	}
	if byID["check"].BlockedReason != "waiting on weigh for T1" {
		t.Fatalf("blockedReason = %q", byID["check"].BlockedReason)
	}

	tlOffice := ProjectStreamTimeline(cfg, domain, other, mustTimelineInstance(t, "T1"))
	officeTasks := tlOffice.Steps[0].Tasks
	officeByID := map[string]StreamTimelineTask{}
	for _, task := range officeTasks {
		officeByID[task.ID] = task
	}
	if officeByID["office"].State != StreamSubstepAvailable {
		t.Fatalf("office for office role = %q, want available", officeByID["office"].State)
	}
	if officeByID["register"].State != StreamSubstepPending {
		t.Fatalf("register for office = %q, want pending", officeByID["register"].State)
	}
}

func TestProjectStreamTimelineExistingEachRowFinished(t *testing.T) {
	op := OperatorIdentity{UserID: "u", Org: "synth", Roles: []string{"ops"}}
	cfg := streamProjectionCfg()
	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"lot": {"L1": {"truckId": "T1", "status": "fullyProcessed"}},
	}
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T1", Label: "T1"},
		{JobID: "lot_processing", ID: "L1", Label: "Lotto L1"},
	}
	domain.Submissions = []Submission{
		sub("intake", "arrive", "T1"),
		sub("intake", "check", "T1"),
		sub("lot_processing", "start", "L1"),
		sub("lot_processing", "close", "L1"),
	}

	tl := ProjectStreamTimeline(cfg, domain, op, mustTimelineInstance(t, "T1"))
	row := tl.Steps[1].Rows[0]
	if row.ID != "L1" || !row.Finished {
		t.Fatalf("row = %#v", row)
	}
	if row.Label != "Lotto L1" {
		t.Fatalf("label = %q", row.Label)
	}
	if row.Tasks[0].State != StreamSubstepDone || row.Tasks[2].State != StreamSubstepDone {
		t.Fatalf("tasks = %#v", row.Tasks)
	}
}
