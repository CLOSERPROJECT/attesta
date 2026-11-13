package living

import (
	"testing"
)

func TestLotSessionParkAcrossCloseOpenTurno(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	line1 := commitOpenTurno(t, c)
	_ = commitStartLot(t, c, line1, loadID)
	if mustDomain(t, c).Kind("lot")[loadID]["status"] != "open" {
		t.Fatalf("after startLot = %#v", mustDomain(t, c).Kind("lot")[loadID])
	}

	commitCloseTurno(t, c, line1)
	parked := mustDomain(t, c)
	if parked.Kind("lot")[loadID]["status"] != "parked" {
		t.Fatalf("after closeTurno = %#v, want parked", parked.Kind("lot")[loadID])
	}
	if parked.Kind("shift")[line1]["status"] != "closed" {
		t.Fatalf("shift after closeTurno = %#v", parked.Kind("shift")[line1])
	}

	cfg := mustActiveConfig(t, c)
	wl := ProjectWorklist(cfg, parked, adminOp())
	lineJobWL, _ := worklistJobByID(wl, "line")
	inst, ok := worklistInstance(lineJobWL, line1)
	if !ok {
		t.Fatal("missing closed line instance")
	}
	for _, id := range []string{"weigh", "markProgress", "containerReg", "closeLot", "startLot"} {
		if st := inst.Tasks[id]; st == TaskStatusActionable {
			t.Fatalf("%s on closed shift = %q, want not actionable", id, st)
		}
	}

	line2 := commitOpenTurnoWith(t, c, map[string]any{"date": "2026-10-19", "turno": "2nd"})
	resumed := mustDomain(t, c)
	if resumed.Kind("lot")[loadID]["status"] != "open" {
		t.Fatalf("after openTurno resume = %#v, want open", resumed.Kind("lot")[loadID])
	}
	after, err := loader.Load(t.Context(), "feragame", adminOp())
	if err != nil {
		t.Fatal(err)
	}
	lineJobWL, _ = worklistJobByID(after.Worklist, "line")
	inst2, ok := worklistInstance(lineJobWL, line2)
	if !ok {
		t.Fatal("missing second line instance")
	}
	if inst2.Tasks["weigh"] != TaskStatusActionable {
		t.Fatalf("weigh after resume = %q", inst2.Tasks["weigh"])
	}
	if inst2.Tasks["markProgress"] != TaskStatusUnavailable {
		t.Fatalf("markProgress after resume (unweighed) = %q, want unavailable", inst2.Tasks["markProgress"])
	}
	if inst2.Tasks["startLot"] != TaskStatusUnavailable {
		t.Fatalf("startLot while resumed open lot = %q", inst2.Tasks["startLot"])
	}

	_ = commitWeigh(t, c, line2, loadID+".1", 11)
	afterWeigh, err := loader.Load(t.Context(), "feragame", adminOp())
	if err != nil {
		t.Fatal(err)
	}
	lineJobWL, _ = worklistJobByID(afterWeigh.Worklist, "line")
	inst3, ok := worklistInstance(lineJobWL, line2)
	if !ok {
		t.Fatal("missing second line instance after weigh")
	}
	if inst3.Tasks["markProgress"] != TaskStatusActionable {
		t.Fatalf("markProgress after resume+weigh = %q", inst3.Tasks["markProgress"])
	}
}

func TestCloseLotDeniedUntilWeighedAndLoaded(t *testing.T) {
	// AllowAll ignores readyWhen; gate authorizer mirrors Cerbos readyWhenOk.
	c, _, _ := testCommitter(t, readyWhenGateAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)
	_ = commitStartLot(t, c, lineID, loadID)

	assertCloseLotDenied := func(when string) {
		t.Helper()
		_, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
			TaskPath: "line/closeLot", JobInstanceID: lineID, Value: map[string]any{},
		})
		if err == nil {
			t.Fatalf("closeLot %s must fail", when)
		}
		wl := ProjectWorklist(mustActiveConfig(t, c), mustDomain(t, c), adminOp())
		inst, _ := worklistInstance(worklistMustJob(t, wl, "line"), lineID)
		if inst.Tasks["closeLot"] != TaskStatusUnavailable {
			t.Fatalf("closeLot %s status = %q, want unavailable", when, inst.Tasks["closeLot"])
		}
	}

	assertCloseLotDenied("before weigh+progress")

	_ = commitWeigh(t, c, lineID, loadID+".1", 11)
	_ = commitWeigh(t, c, lineID, loadID+".2", 22)
	assertCloseLotDenied("after weigh only")

	_ = commitMarkProgress(t, c, lineID, map[string]any{
		loadID + ".1": "done",
		loadID + ".2": "partial",
	})
	assertCloseLotDenied("with partial progress")

	_ = commitMarkProgress(t, c, lineID, map[string]any{
		loadID + ".2": "done",
	})
	closed := commitCloseLot(t, c, lineID)
	if closed.Domain.Kind("lot")[loadID]["status"] != "fullyProcessed" {
		t.Fatalf("closeLot = %#v", closed.Domain.Kind("lot")[loadID])
	}
}

func TestContainerRegDoesNotSetFullyProcessed(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)

	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      "line/containerReg",
		JobInstanceID: lineID,
		Value: map[string]any{
			"sourceLotId": loadID,
			"polvereDiVetro": []any{
				map[string]any{"weight": 120.0},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Kind("lot")[loadID]["status"] != "open" {
		t.Fatalf("containerReg must leave lot open, got %#v", result.Domain.Kind("lot")[loadID])
	}
	if len(result.Domain.Kind("container")) == 0 {
		t.Fatal("expected minted containers")
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "containerReg" || last.GroupID != "" {
		t.Fatalf("containerReg submission = %#v", last)
	}
}
