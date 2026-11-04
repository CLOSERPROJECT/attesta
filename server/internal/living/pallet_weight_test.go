package living

import (
	"testing"
)

func TestCommitWeighPatchesOnePalletLeavesOtherUnweighted(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)

	result := commitWeigh(t, c, lineID, loadID+".1", 11)
	if result.Domain.Kind("pallet")[loadID+".1"]["weight"] != 11.0 {
		t.Fatalf("pallet .1 = %#v", result.Domain.Kind("pallet")[loadID+".1"])
	}
	if _, has := result.Domain.Kind("pallet")[loadID+".2"]["weight"]; has {
		t.Fatalf("pallet .2 must stay unweighted; got %#v", result.Domain.Kind("pallet")[loadID+".2"])
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "weigh" || last.GroupID != "palletWeight" {
		t.Fatalf("weigh submission = %#v", last)
	}
	if last.Facts["palletId"] != loadID+".1" {
		t.Fatalf("weigh facts = %#v, want palletId %s.1", last.Facts, loadID)
	}
	if _, has := last.Facts["averagePanelWeight"]; !has {
		t.Fatal("averagePanelWeight must stay Submission fact")
	}
}

func TestLineLoadUnlockFromOpenTurnoReadyWhenWeighNotClose(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)
	line := lineJob(t)
	lineOp := OperatorIdentity{UserID: "line", Org: "feragame", Roles: []string{"line"}}

	afterOpen, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(line.Tasks), JobID: "line", TaskID: "lineLoad",
		JobInstanceID: lineID, Submissions: afterOpen.Submissions, Tasks: line.Tasks,
		ClosedGroups: afterOpen.ClosedGroups,
	}) {
		t.Fatal("lineLoad must unlock from openTurno (not palletWeight Close)")
	}
	cfg := mustActiveConfig(t, c)
	openWL := ProjectWorklist(cfg, afterOpen, lineOp)
	openLine, _ := worklistJobByID(openWL, "line")
	openInst, ok := worklistInstance(openLine, lineID)
	if !ok || openInst.Tasks["lineLoad"] != TaskStatusUnavailable {
		t.Fatalf("openTurno only, no weighed pallets: lineLoad = %#v, want unavailable", openInst)
	}
	if openInst.Tasks["palletWeight"] != TaskStatusAvailable {
		t.Fatalf("palletWeight after openTurno = %q, want available", openInst.Tasks["palletWeight"])
	}

	_ = commitWeigh(t, c, lineID, loadID+".1", 11)
	weighed, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if weighed.Kind("pallet")[loadID+".1"]["weight"] != 11.0 {
		t.Fatalf("pallet .1 = %#v", weighed.Kind("pallet")[loadID+".1"])
	}
	weighWL := ProjectWorklist(cfg, weighed, lineOp)
	weighLine, _ := worklistJobByID(weighWL, "line")
	weighInst, ok := worklistInstance(weighLine, lineID)
	if !ok || weighInst.Tasks["lineLoad"] != TaskStatusActionable {
		t.Fatalf("weighed leftover + open weigh group: lineLoad = %#v, want actionable", weighInst)
	}
	if weighInst.Tasks["palletWeight"] != TaskStatusAvailable {
		t.Fatalf("palletWeight still open = %q, want available", weighInst.Tasks["palletWeight"])
	}

	_ = commitWeigh(t, c, lineID, loadID+".2", 22)
	closePalletWeight(t, c, lineID)
	closed, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if closed.Kind("pallet")[loadID+".2"]["weight"] != 22.0 {
		t.Fatalf("pallet .2 = %#v", closed.Kind("pallet")[loadID+".2"])
	}
	closeWL := ProjectWorklist(cfg, closed, lineOp)
	closeLine, _ := worklistJobByID(closeWL, "line")
	closeInst, ok := worklistInstance(closeLine, lineID)
	if !ok || closeInst.Tasks["lineLoad"] != TaskStatusActionable {
		t.Fatalf("after Close weigh: lineLoad = %#v (Close must not special-case)", closeInst)
	}
	if closeInst.Tasks["palletWeight"] != TaskStatusDone {
		t.Fatalf("palletWeight after close = %q, want done", closeInst.Tasks["palletWeight"])
	}
}

func TestClosePalletWeightRequiresChildSubmission(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, _, _ = commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)

	_, err := c.CloseGroup(t.Context(), "feragame", adminOp(), CloseGroupRequest{
		JobID: "line", JobInstanceID: lineID, GroupID: "palletWeight",
	})
	var ce *CommitError
	// Committer gates on CanClose (false with zero children) before Domain CloseGroup.
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("err = %v, want prep cannot-close", err)
	}
}
