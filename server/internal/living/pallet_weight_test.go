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

func TestCommitWeighThenCloseUnlocksLineLoad(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)

	_ = commitWeigh(t, c, lineID, loadID+".1", 11)
	_ = commitWeigh(t, c, lineID, loadID+".2", 22)

	closePalletWeight(t, c, lineID)
	stored, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Kind("pallet")[loadID+".1"]["weight"] != 11.0 {
		t.Fatalf("pallet .1 = %#v", stored.Kind("pallet")[loadID+".1"])
	}
	if stored.Kind("pallet")[loadID+".2"]["weight"] != 22.0 {
		t.Fatalf("pallet .2 = %#v", stored.Kind("pallet")[loadID+".2"])
	}
	line := lineJob(t)
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(line.Tasks), JobID: "line", TaskID: "lineLoad",
		JobInstanceID: lineID, Submissions: stored.Submissions, Tasks: line.Tasks,
		ClosedGroups: stored.ClosedGroups,
	}) {
		t.Fatal("closing palletWeight must unlock lineLoad")
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
