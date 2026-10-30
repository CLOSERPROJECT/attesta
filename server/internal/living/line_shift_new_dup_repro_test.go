package living

import (
	"testing"
)

// Nuovo turno: two startingNew Commits with the same evaluated shift id must not
// append a duplicate line/1st JobInstance (JobPanel keyed-each would crash).
func TestCommitLineShiftStartingNewTwiceDeniesDuplicate(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	completeIntakeThroughPalletWeight(t, c, "NIR-261019-NO-1")

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	first, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/shift",
		Value:    map[string]any{"shift": "1st"},
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.JobInstanceID != "1st" {
		t.Fatalf("first mint = %q", first.JobInstanceID)
	}

	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/shift",
		Value:    map[string]any{"shift": "1st"},
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
		if inst.JobID == "line" && inst.ID == "1st" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("line JobInstances with id 1st = %d (want 1); all=%#v", count, domain.JobInstances)
	}
}

func TestMutateRejectsDuplicateJobInstance(t *testing.T) {
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "line", ID: "1st", Label: "Turno: 1st"}}
	_, err := Mutate(domain, MutateRequest{
		UserID:         "ada",
		JobID:          "line",
		TaskID:         "shift",
		JobInstanceID:  "1st",
		Timestamp:      "2026-10-30T12:00:00Z",
		Facts:          map[string]any{"shift": "1st"},
		NewJobInstance: &JobInstance{JobID: "line", ID: "1st", Label: "Turno: 1st"},
	})
	var me MutateError
	if err == nil || !asMutateError(err, &me) || me.Kind != "duplicate-job-instance" {
		t.Fatalf("err = %v", err)
	}
}
