package org

import (
	"testing"
)

func TestLotSessionStaysOpenUntilCloseLot(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	_, loadID := plantIntakeOneLot(t, c)
	_ = plantCommitStartLot(t, c, loadID)
	if mustDomain(t, c).Kind("lot")[loadID]["status"] != "open" {
		t.Fatalf("after startLot = %#v", mustDomain(t, c).Kind("lot")[loadID])
	}
	if len(mustDomain(t, c).Kind("shift")) != 0 || len(mustDomain(t, c).Kind("workingDay")) != 0 {
		t.Fatalf("domain must omit shift/workingDay: %#v", mustDomain(t, c))
	}

	_ = plantCommitWeigh(t, c, loadID, loadID+".1", 11)
	mid := mustDomain(t, c)
	if mid.Kind("lot")[loadID]["status"] != "open" {
		t.Fatalf("lot must stay open mid-session = %#v (no parked)", mid.Kind("lot")[loadID])
	}
	for _, lot := range mid.Kind("lot") {
		if lot["status"] == "parked" {
			t.Fatalf("lot status parked is forbidden: %#v", mid.Kind("lot"))
		}
	}

	after, err := loader.Load(t.Context(), FeragameScope(), adminOp())
	if err != nil {
		t.Fatal(err)
	}
	lp, ok := worklistJobByID(after.Worklist, "lot_processing")
	if !ok {
		t.Fatal("missing lot_processing")
	}
	inst, ok := worklistInstance(lp, loadID)
	if !ok {
		t.Fatal("missing lot_processing instance")
	}
	if inst.Tasks["weigh"] != TaskStatusActionable {
		t.Fatalf("weigh mid-session = %q, want actionable", inst.Tasks["weigh"])
	}
	if inst.Tasks["markProgress"] != TaskStatusActionable {
		t.Fatalf("markProgress after one weigh = %q, want actionable", inst.Tasks["markProgress"])
	}
	if _, has := inst.Tasks["openTurno"]; has {
		t.Fatalf("openTurno must not project: %#v", inst.Tasks)
	}
	if _, has := inst.Tasks["closeTurno"]; has {
		t.Fatalf("closeTurno must not project: %#v", inst.Tasks)
	}

	_ = plantCommitWeigh(t, c, loadID, loadID+".2", 22)
	_ = plantCommitMarkProgress(t, c, loadID, map[string]any{
		loadID + ".1": "done",
		loadID + ".2": "done",
	})
	closed := plantCommitCloseLot(t, c, loadID)
	if closed.Domain.Kind("lot")[loadID]["status"] != "fullyProcessed" {
		t.Fatalf("closeLot = %#v, want fullyProcessed", closed.Domain.Kind("lot")[loadID])
	}
}

func TestCloseLotDeniedUntilWeighedAndLoaded(t *testing.T) {
	c, _, _ := testTaskCommitter(t, readyWhenGateAuthorizer{})
	_, loadID := plantIntakeOneLot(t, c)
	_ = plantCommitStartLot(t, c, loadID)

	assertCloseLotDenied := func(when string) {
		t.Helper()
		_, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
			TaskPath: "lot_processing/closeLot", JobInstanceID: loadID, Value: map[string]any{},
		})
		if err == nil {
			t.Fatalf("closeLot %s must fail", when)
		}
		wl := ProjectWorklist(mustActiveConfig(t, c), mustDomain(t, c), adminOp())
		inst, _ := worklistInstance(worklistMustJob(t, wl, "lot_processing"), loadID)
		if inst.Tasks["closeLot"] != TaskStatusUnavailable {
			t.Fatalf("closeLot %s status = %q, want unavailable", when, inst.Tasks["closeLot"])
		}
	}

	assertCloseLotDenied("before weigh+progress")

	_ = plantCommitWeigh(t, c, loadID, loadID+".1", 11)
	_ = plantCommitWeigh(t, c, loadID, loadID+".2", 22)
	assertCloseLotDenied("after weigh only")

	_ = plantCommitMarkProgress(t, c, loadID, map[string]any{
		loadID + ".1": "done",
		loadID + ".2": "partial",
	})
	assertCloseLotDenied("with partial progress")

	_ = plantCommitMarkProgress(t, c, loadID, map[string]any{
		loadID + ".2": "done",
	})
	closed := plantCommitCloseLot(t, c, loadID)
	if closed.Domain.Kind("lot")[loadID]["status"] != "fullyProcessed" {
		t.Fatalf("closeLot = %#v", closed.Domain.Kind("lot")[loadID])
	}
}

func TestContainerRegDoesNotSetFullyProcessed(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	loadID := plantSeedOpenLotReadyForContainer(t, c)

	result, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath:      "lot_processing/containerReg",
		JobInstanceID: loadID,
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
	if last.TaskID != "containerReg" {
		t.Fatalf("containerReg submission = %#v", last)
	}
}
