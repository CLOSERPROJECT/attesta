package living

import (
	"strings"
	"testing"
)

func TestCommitWeighRejectsPartialRequiredMap(t *testing.T) {
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)

	_, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      GroupTaskPath("line", "palletWeight", ProvisionalGroupInstanceID, "weigh"),
		JobInstanceID: lineID,
		Value: palletWeightValue(map[string]any{
			loadID + ".1": weighRow(11),
		}),
		Mint: spawningNewMint(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(ce.Message, "required map keys") {
		t.Fatalf("message = %q", ce.Message)
	}

	domain, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range domain.Submissions {
		if s.TaskID == "weigh" {
			t.Fatalf("partial must not append weigh Submission: %#v", domain.Submissions)
		}
	}
	if _, has := domain.Kind("pallet")[loadID+".1"]["weight"]; has {
		t.Fatal("partial reject must not patch pallets")
	}
}

func TestCommitWeighCompleteThenCloseUnlocksLineLoad(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)

	result := commitWeigh(t, c, lineID, map[string]any{
		loadID + ".1": weighRow(11),
		loadID + ".2": weighRow(22),
	})
	if result.Domain.Kind("pallet")[loadID+".1"]["weight"] != 11.0 {
		t.Fatalf("pallet .1 = %#v", result.Domain.Kind("pallet")[loadID+".1"])
	}
	if result.Domain.Kind("pallet")[loadID+".2"]["weight"] != 22.0 {
		t.Fatalf("pallet .2 = %#v", result.Domain.Kind("pallet")[loadID+".2"])
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "weigh" || last.GroupID != "palletWeight" {
		t.Fatalf("weigh submission = %#v", last)
	}

	closePalletWeight(t, c, lineID)
	stored, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
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
