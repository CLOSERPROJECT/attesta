package living

import (
	"testing"
)

// Nuovo turno: two startingNew Commits with the same evaluated date+turno must not
// append a duplicate line JobInstance (JobPanel keyed-each would crash).
func TestCommitOpenTurnoStartingNewTwiceDeniesDuplicate(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	ensureOpenWorkingDay(t, c)

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	first, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    openTurnoValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantID := "2026-10-19-1st"
	if first.JobInstanceID != wantID {
		t.Fatalf("first mint = %q", first.JobInstanceID)
	}

	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    openTurnoValue(),
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
		if inst.JobID == "line" && inst.ID == wantID {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("line JobInstances with id %s = %d (want 1); all=%#v", wantID, count, domain.JobInstances)
	}
}

func TestMutateRejectsDuplicateJobInstance(t *testing.T) {
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "line", ID: "2026-10-19-1st", Label: "2026-10-19 Turno: 1°"}}
	_, err := Mutate(domain, MutateRequest{
		UserID:         "ada",
		JobID:          "line",
		TaskID:         "openTurno",
		JobInstanceID:  "2026-10-19-1st",
		Timestamp:      "2026-10-30T12:00:00Z",
		Facts:          map[string]any{"date": "2026-10-19", "turno": "1st"},
		NewJobInstance: &JobInstance{JobID: "line", ID: "2026-10-19-1st", Label: "2026-10-19 Turno: 1°"},
	})
	var me MutateError
	if err == nil || !asMutateError(err, &me) || me.Kind != "duplicate-job-instance" {
		t.Fatalf("err = %v", err)
	}
}
