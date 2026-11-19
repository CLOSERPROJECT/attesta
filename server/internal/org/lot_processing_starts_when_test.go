package org

import (
	"testing"
	"time"
)

func TestFixtureLotProcessingHasNoStartsWhen(t *testing.T) {
	lp := lotProcessingJob(t)
	if lp.StartsWhen != nil {
		t.Fatalf("lot_processing must not author startsWhen, got %#v", lp.StartsWhen)
	}
	if !lp.Repeatable {
		t.Fatal("lot_processing must stay repeatable")
	}
	want := []string{"startLot", "weigh", "markProgress", "containerReg", "closeLot", "lock"}
	got := OrderedTaskIDs(lp.Tasks)
	if len(got) != len(want) {
		t.Fatalf("lot_processing tasks = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("lot_processing tasks = %v, want %v", got, want)
		}
	}
}

func TestCanStartJobLotProcessingWithoutStartsWhen(t *testing.T) {
	lp := lotProcessingJob(t)
	empty := EmptyDomain()
	if !CanStartJob(lp, "lot_processing", empty.Submissions, empty.JobInstances, nil) {
		t.Fatal("lot_processing must be startable with no startsWhen gate")
	}
}

func TestCommitStartLotViaCerbosAfterIntake(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, _, _ := testTaskCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	_, loadID := plantIntakeOneLot(t, c)
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	result, err := c.Commit(t.Context(), FeragameScope(), op, CommitRequest{
		TaskPath: "lot_processing/startLot",
		Value:    map[string]any{"lotId": loadID},
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.JobInstanceID != loadID {
		t.Fatalf("jobInstanceId = %q, want %q", result.JobInstanceID, loadID)
	}
	found := false
	for _, inst := range result.Domain.JobInstances {
		if inst.JobID == "lot_processing" && inst.ID == loadID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("lot_processing Job instance missing: %#v", result.Domain.JobInstances)
	}
}

func TestSequenceOkForCommitLotProcessingWithoutStartsWhen(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	empty := EmptyDomain()
	if !sequenceOkForCommit(cfg, empty, "lot_processing", "startLot", "") {
		t.Fatal("sequenceOk must allow startLot without startsWhen")
	}
}
