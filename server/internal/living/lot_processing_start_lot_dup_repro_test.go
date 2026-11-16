package living

import (
	"testing"
)

// Two startingNew Commits with the same evaluated lotId must not append a
// duplicate lot_processing JobInstance (JobPanel keyed-each would crash).
func TestCommitStartLotStartingNewTwiceDeniesDuplicate(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID := plantIntakeOneLot(t, c)

	op := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}
	first, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "lot_processing/startLot",
		Value:    map[string]any{"lotId": loadID},
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.JobInstanceID != loadID {
		t.Fatalf("first mint = %q, want %q", first.JobInstanceID, loadID)
	}

	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "lot_processing/startLot",
		Value:    map[string]any{"lotId": loadID},
		Mint:     startingNewMint(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("second startingNew must prep-deny: %v", err)
	}

	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	var count int
	for _, inst := range domain.JobInstances {
		if inst.JobID == "lot_processing" && inst.ID == loadID {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("lot_processing JobInstances with id %s = %d (want 1); all=%#v", loadID, count, domain.JobInstances)
	}
}

func TestMutateRejectsDuplicateJobInstance(t *testing.T) {
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "lot_processing", ID: "CO-261029-1", Label: "Lotto: CO-261029-1"}}
	_, err := Mutate(domain, MutateRequest{
		UserID:         "ada",
		JobID:          "lot_processing",
		TaskID:         "startLot",
		JobInstanceID:  "CO-261029-1",
		Timestamp:      "2026-10-30T12:00:00Z",
		Facts:          map[string]any{"lotId": "CO-261029-1"},
		NewJobInstance: &JobInstance{JobID: "lot_processing", ID: "CO-261029-1", Label: "Lotto: CO-261029-1"},
	})
	var me MutateError
	if err == nil || !asMutateError(err, &me) || me.Kind != "duplicate-job-instance" {
		t.Fatalf("err = %v", err)
	}
}
