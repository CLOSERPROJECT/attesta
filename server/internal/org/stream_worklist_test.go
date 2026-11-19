package org

import (
	"testing"
)

func streamWorklistCfg() OrgConfig {
	cfg := validStreamCfg()
	intake := cfg.Jobs.Values["intake"].(Job)
	intake.Tasks = []TaskConfig{
		{"id": "arrive", "roles": []any{"ops"}, "submit": "once"},
		{"id": "register", "roles": []any{"ops"}, "submit": "many"},
		{"id": "check", "roles": []any{"ops"}, "submit": "once"},
	}
	cfg.Jobs.Values["intake"] = intake

	lp := cfg.Jobs.Values["lot_processing"].(Job)
	lp.Tasks = []TaskConfig{
		{
			"id": "start", "roles": []any{"ops"},
			"readyWhen":     `$count($domain.lot.*[truckId = $stream.id and $not(status = "open")]) >= 1`,
			"blockedReason": `'no eligible lots on ' & $stream.id`,
		},
		{"id": "weigh", "roles": []any{"ops"}, "submit": "many"},
		{"id": "close", "roles": []any{"ops"}, "submit": "once"},
	}
	cfg.Jobs.Values["lot_processing"] = lp
	return cfg
}

func TestProjectWorklistFinishedMatchesSPARule(t *testing.T) {
	op := OperatorIdentity{UserID: "u", Org: "synth", Roles: []string{"ops"}}
	cfg := streamWorklistCfg()

	t.Run("partial_once_not_finished", func(t *testing.T) {
		domain := EmptyDomain()
		domain.JobInstances = []JobInstance{{JobID: "intake", ID: "T1", Label: "T1"}}
		domain.Submissions = []Submission{
			sub("intake", "arrive", "T1"),
		}
		wl := ProjectWorklist(cfg, domain, op, "T1")
		inst, ok := worklistInstance(worklistMustJob(t, wl, "intake"), "T1")
		if !ok {
			t.Fatal("missing instance")
		}
		if inst.Finished {
			t.Fatal("want finished=false when once Task still open")
		}
	})

	t.Run("all_once_done_finished", func(t *testing.T) {
		domain := EmptyDomain()
		domain.JobInstances = []JobInstance{{JobID: "intake", ID: "T1", Label: "T1"}}
		domain.Submissions = []Submission{
			sub("intake", "arrive", "T1"),
			sub("intake", "check", "T1"),
		}
		wl := ProjectWorklist(cfg, domain, op, "T1")
		inst, ok := worklistInstance(worklistMustJob(t, wl, "intake"), "T1")
		if !ok {
			t.Fatal("missing instance")
		}
		if !inst.Finished {
			t.Fatal("want finished=true when every once Task is done")
		}
	})

	t.Run("finished_ignores_submit_many", func(t *testing.T) {
		domain := EmptyDomain()
		domain.JobInstances = []JobInstance{{JobID: "intake", ID: "T1", Label: "T1"}}
		domain.Submissions = []Submission{
			sub("intake", "arrive", "T1"),
			sub("intake", "check", "T1"),
		}
		wl := ProjectWorklist(cfg, domain, op, "T1")
		inst, ok := worklistInstance(worklistMustJob(t, wl, "intake"), "T1")
		if !ok {
			t.Fatal("missing instance")
		}
		if inst.Tasks["register"] != TaskStatusActionable {
			t.Fatalf("many leaf = %q, want actionable", inst.Tasks["register"])
		}
		if !inst.Finished {
			t.Fatal("want finished=true; submit:many must not block")
		}
	})
}

func TestProjectWorklistBlockedReasonWhenReadyWhenFalse(t *testing.T) {
	op := OperatorIdentity{UserID: "u", Org: "synth", Roles: []string{"ops"}}
	cfg := streamWorklistCfg()
	lp := cfg.Jobs.Values["lot_processing"].(Job)
	lp.Tasks = []TaskConfig{
		{"id": "start", "roles": []any{"ops"}},
		{
			"id": "weigh", "roles": []any{"ops"}, "submit": "many",
			"readyWhen":     `$lookup($domain.lot, $instance.id).status = "open"`,
			"blockedReason": `'lot ' & $instance.id & ' not open on ' & $stream.id`,
		},
		{"id": "close", "roles": []any{"ops"}},
	}
	cfg.Jobs.Values["lot_processing"] = lp

	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"lot": {"L1": {"truckId": "T-A", "status": "fullyProcessed"}},
	}
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T-A", Label: "T-A"},
		{JobID: "lot_processing", ID: "L1", Label: "L1"},
	}
	domain.Submissions = []Submission{sub("lot_processing", "start", "L1")}

	wl := ProjectWorklist(cfg, domain, op, "T-A")
	inst, ok := worklistInstance(worklistMustJob(t, wl, "lot_processing"), "L1")
	if !ok {
		t.Fatal("missing lot instance")
	}
	if inst.Tasks["weigh"] != TaskStatusUnavailable {
		t.Fatalf("weigh = %q, want unavailable", inst.Tasks["weigh"])
	}
	got := inst.BlockedReasons["weigh"]
	want := "lot L1 not open on T-A"
	if got != want {
		t.Fatalf("blockedReason = %q, want %q", got, want)
	}
	if _, ok := inst.BlockedReasons["start"]; ok {
		t.Fatalf("ready Task must not carry blockedReason: %#v", inst.BlockedReasons)
	}
}

func TestProjectWorklistFiltersToStreamInstance(t *testing.T) {
	op := OperatorIdentity{UserID: "u", Org: "synth", Roles: []string{"ops"}}
	cfg := streamWorklistCfg()
	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"lot": {
			"L-A1": {"truckId": "T-A", "status": "registered"},
			"L-B1": {"truckId": "T-B", "status": "registered"},
		},
	}
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T-A", Label: "T-A"},
		{JobID: "intake", ID: "T-B", Label: "T-B"},
		{JobID: "lot_processing", ID: "L-A1", Label: "L-A1"},
		{JobID: "lot_processing", ID: "L-B1", Label: "L-B1"},
	}
	domain.Submissions = []Submission{
		sub("intake", "arrive", "T-A"),
		sub("intake", "arrive", "T-B"),
		sub("lot_processing", "start", "L-A1"),
		sub("lot_processing", "start", "L-B1"),
	}

	wlA := ProjectWorklist(cfg, domain, op, "T-A")
	intakeA := worklistMustJob(t, wlA, "intake")
	if len(intakeA.Instances) != 1 || intakeA.Instances[0].ID != "T-A" {
		t.Fatalf("intake instances = %#v, want only T-A", intakeA.Instances)
	}
	lpA := worklistMustJob(t, wlA, "lot_processing")
	if len(lpA.Instances) != 1 || lpA.Instances[0].ID != "L-A1" {
		t.Fatalf("lot_processing instances = %#v, want only L-A1", lpA.Instances)
	}

	wlB := ProjectWorklist(cfg, domain, op, "T-B")
	lpB := worklistMustJob(t, wlB, "lot_processing")
	if len(lpB.Instances) != 1 || lpB.Instances[0].ID != "L-B1" {
		t.Fatalf("lot_processing instances = %#v, want only L-B1", lpB.Instances)
	}
}

func TestProjectWorklistEachStartUsesStreamBinding(t *testing.T) {
	op := OperatorIdentity{UserID: "u", Org: "synth", Roles: []string{"ops"}}
	cfg := streamWorklistCfg()
	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"lot": {
			"L-B1": {"truckId": "T-B", "status": "registered"},
		},
	}
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T-A", Label: "T-A"},
		{JobID: "intake", ID: "T-B", Label: "T-B"},
	}

	// Truck A has no eligible lots → start readyWhen false → canStart false + blockedReason on start chrome path.
	wlA := ProjectWorklist(cfg, domain, op, "T-A")
	lpA := worklistMustJob(t, wlA, "lot_processing")
	if lpA.CanStart {
		t.Fatal("T-A canStart want false (no lots on this truck)")
	}

	wlB := ProjectWorklist(cfg, domain, op, "T-B")
	lpB := worklistMustJob(t, wlB, "lot_processing")
	if !lpB.CanStart {
		t.Fatal("T-B canStart want true (eligible lot on this truck)")
	}
}

func TestEvaluateReadyWhenBindsStream(t *testing.T) {
	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"lot": {"L1": {"truckId": "T-A"}},
	}
	ok, err := evaluateReadyWhen(domain, `$count($domain.lot.*[truckId = $stream.id]) = 1`, "", nil, &streamBinding{Key: "recycling", ID: "T-A"})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("want readyWhen true with $stream.id = T-A")
	}
	ok, err = evaluateReadyWhen(domain, `$count($domain.lot.*[truckId = $stream.id]) = 1`, "", nil, &streamBinding{Key: "recycling", ID: "T-B"})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("want readyWhen false with $stream.id = T-B")
	}
}
