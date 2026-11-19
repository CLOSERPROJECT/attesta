package org

import (
	"encoding/json"
	"testing"
)

func TestCommitTruckArriveAdvancesToRegister(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}

	arrived, err := c.Commit(t.Context(), FeragameScope(), admin, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if arrived.Navigation != CommitNextTask {
		t.Fatalf("after truckArrive Navigation = %q, want task", arrived.Navigation)
	}
	if arrived.NextTaskFormBundle == nil || arrived.NextTaskFormBundle.TaskID != "register" {
		t.Fatalf("nextTaskFormBundle = %#v, want register", arrived.NextTaskFormBundle)
	}
	if arrived.ContinueNavigation != "" {
		t.Fatalf("advance Commit must omit continueNavigation, got %q", arrived.ContinueNavigation)
	}
}

func TestCommitStartLotAdvancesToWeigh(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID, loadID := plantIntakeOneLot(t, c)
	logistic := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic", "line"}}

	started, err := c.Commit(t.Context(), FeragameScope(), logistic, CommitRequest{
		TaskPath:         "lot_processing/startLot",
		Value:            map[string]any{"lotId": loadID},
		Mint:             startingNewMint(),
		StreamInstanceID: truckID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if started.Navigation != CommitNextTask {
		t.Fatalf("after startLot Navigation = %q, want task", started.Navigation)
	}
	if started.NextTaskFormBundle == nil || started.NextTaskFormBundle.TaskID != "weigh" {
		t.Fatalf("nextTaskFormBundle = %#v, want weigh", started.NextTaskFormBundle)
	}
}

func TestNextAdvanceTaskRefSkipsAdvanceTargetFalse(t *testing.T) {
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"demo"},
			Values: map[string]any{
				"demo": Job{
					Repeatable: true,
					Tasks: []TaskConfig{
						{
							"id": "first", "roles": []any{"administration"},
							"schema": map[string]any{"type": "object"},
						},
						{
							"id": "park", "roles": []any{"administration"},
							"onCommit": map[string]any{"advanceTarget": false},
							"schema":   map[string]any{"type": "object"},
						},
					},
				},
			},
		},
	}
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "demo", ID: "X1", Label: "x"}}
	domain.Submissions = []Submission{sub("demo", "first", "X1")}
	op := OperatorIdentity{UserID: "ada", Org: "test", Roles: []string{"administration"}}
	wl := projectWorklistUnfiltered(cfg, domain, op)
	inst, ok := worklistInstance(worklistMustJob(t, wl, "demo"), "X1")
	if !ok {
		t.Fatal("missing demo instance")
	}
	if inst.Tasks["park"] != TaskStatusActionable {
		t.Fatalf("park = %q, want actionable so skip can be observed", inst.Tasks["park"])
	}
	if _, ok := NextAdvanceTaskRef(cfg, wl, "demo", "X1"); ok {
		t.Fatal("NextAdvanceTaskRef must skip park when advanceTarget is false")
	}
}

func TestCloseLotStayOmitsTurnoTasks(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	loadID, _ := seedLotReadyAfterWeigh(t, c)
	_ = commitMarkProgress(t, c, loadID, map[string]any{
		loadID + ".1": "done",
		loadID + ".2": "done",
	})
	_ = commitContainerReg(t, c, loadID, containerRegValue(loadID, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 50.0}},
	}))
	closed := commitCloseLot(t, c, loadID)
	lp, ok := worklistJobByID(closed.Worklist, "lot_processing")
	if !ok {
		t.Fatal("missing lot_processing")
	}
	inst, ok := worklistInstance(lp, loadID)
	if !ok {
		t.Fatal("missing lot_processing instance")
	}
	if _, has := inst.Tasks["closeTurno"]; has {
		t.Fatalf("closeTurno must not project: %#v", inst.Tasks)
	}
	if _, has := inst.Tasks["openTurno"]; has {
		t.Fatalf("openTurno must not project: %#v", inst.Tasks)
	}
	if closed.Navigation != CommitNextStay {
		t.Fatalf("closeLot Navigation = %q, want stay", closed.Navigation)
	}
}

func TestCommitLinkStayContinueNavigationWire(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	truckID := commitTruckArrive(t, c)

	lot, err := c.Commit(t.Context(), FeragameScope(), admin, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: truckID,
		Value:         registerLotValue(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if lot.Navigation != CommitNextStay {
		t.Fatalf("link register Navigation = %q, want stay", lot.Navigation)
	}
	if lot.ContinueNavigation != CommitNextTask {
		t.Fatalf("ContinueNavigation = %q, want task (weightCheck unlocked after register)", lot.ContinueNavigation)
	}

	raw, err := json.Marshal(lot)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if _, has := wire["next"]; has {
		t.Fatalf("Commit JSON must not ship legacy next: %s", string(raw))
	}
	if wire["navigation"] != string(CommitNextStay) {
		t.Fatalf("JSON navigation = %#v", wire["navigation"])
	}
	if wire["continueNavigation"] != string(CommitNextTask) {
		t.Fatalf("JSON continueNavigation = %#v", wire["continueNavigation"])
	}
}
