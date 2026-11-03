package living

import (
	"testing"
	"time"
)

func TestFixtureLineHasNoStartsWhen(t *testing.T) {
	line := lineJob(t)
	if line.StartsWhen != nil {
		t.Fatalf("line must not author startsWhen, got %#v", line.StartsWhen)
	}
	if !line.Repeatable {
		t.Fatal("line must stay repeatable")
	}
	want := []string{"openTurno", "palletWeight", "lineLoad", "containerReg", "loadedExtracted"}
	got := OrderedTaskIDs(line.Tasks)
	if len(got) != len(want) {
		t.Fatalf("line tasks = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line tasks = %v, want %v", got, want)
		}
	}
}

func TestCanStartJobLineWithoutStartsWhen(t *testing.T) {
	line := lineJob(t)
	empty := EmptyDomain()
	if !CanStartJob(line, "line", empty.Submissions, empty.JobInstances, nil) {
		t.Fatal("line must be startable with no startsWhen gate")
	}
}

func TestCommitOpenTurnoViaCerbosOnEmptyDomain(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, _, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/openTurno",
		Value:    openTurnoValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantID := "2026-10-19-1st"
	if result.JobInstanceID != wantID {
		t.Fatalf("jobInstanceId = %q, want %q", result.JobInstanceID, wantID)
	}
	found := false
	for _, inst := range result.Domain.JobInstances {
		if inst.JobID == "line" && inst.ID == wantID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("line Job instance missing: %#v", result.Domain.JobInstances)
	}
}

func TestSequenceOkForCommitLineWithoutStartsWhen(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	empty := EmptyDomain()
	if !sequenceOkForCommit(cfg, empty, "line", "openTurno", "") {
		t.Fatal("sequenceOk must allow openTurno without startsWhen")
	}
}
