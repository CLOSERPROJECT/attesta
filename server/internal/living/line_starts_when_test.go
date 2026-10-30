package living

import (
	"testing"
	"time"
)

func completeIntakeThroughPalletWeight(t *testing.T, c *Committer, instanceID string) Domain {
	t.Helper()
	commitThroughCodiceFir(t, c, instanceID)
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/palletWeight", JobInstanceID: instanceID,
		Value: palletWeightValue(map[string]any{
			instanceID + ".1": weighRow(11),
			instanceID + ".2": weighRow(22),
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Domain
}

func TestFixtureLineStartsWhenSomeQuantifier(t *testing.T) {
	line := lineJob(t)
	if line.StartsWhen == nil {
		t.Fatal("line startsWhen missing")
	}
	if line.StartsWhen.Quantifier != "some" {
		t.Fatalf("fixture quantifier = %q, want some", line.StartsWhen.Quantifier)
	}
	if line.StartsWhen.Task.JobID != "intake" || line.StartsWhen.Task.TaskID != "palletWeight" {
		t.Fatalf("startsWhen.task = %#v", line.StartsWhen.Task)
	}
	if !line.Repeatable {
		t.Fatal("line must be repeatable so one gate Submission keeps start open")
	}
}

func TestCanStartJobBeforeAndAfterPalletWeightGate(t *testing.T) {
	line := lineJob(t)
	intake := intakeTasks(t)
	gateTask := palletWeightTask(intake)

	empty := EmptyDomain()
	if CanStartJob(line, "line", empty.Submissions, empty.JobInstances, gateTask) {
		t.Fatal("line must not start before intake gate Submission")
	}

	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	domain := completeIntakeThroughPalletWeight(t, c, "NIR-261019-NO-1")
	if !IsStartsWhenSatisfied(line.StartsWhen, domain.Submissions, domain.JobInstances, gateTask) {
		t.Fatal("fixture some must satisfy after one palletWeight Submission")
	}
	if !CanStartJob(line, "line", domain.Submissions, domain.JobInstances, gateTask) {
		t.Fatal("line Job must be startable after gate")
	}
}

func TestCommitLineShiftDeniedViaCerbosBeforeStartsWhen(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, domains, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	_, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "line/shift",
		JobInstanceID: "1st",
		Value:         map[string]any{"shift": "1st"},
		Mint:          startingNewMint(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("err = %v", err)
	}
	if _, getErr := domains.Get(t.Context(), "feragame"); getErr == nil {
		t.Fatal("domain must stay unset when startsWhen denies first line Task")
	}

	line := lineJob(t)
	if CanStartJob(line, "line", nil, nil, palletWeightTask(intakeTasks(t))) {
		t.Fatal("CanStartJob must stay false without gate Submission")
	}
}

func TestCommitLineShiftAllowedViaCerbosAfterStartsWhen(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, _, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	intakeID := "NIR-261019-NO-1"
	domain := completeIntakeThroughPalletWeight(t, c, intakeID)

	line := lineJob(t)
	if line.StartsWhen.Quantifier != "some" {
		t.Fatalf("quantifier = %q", line.StartsWhen.Quantifier)
	}
	if !CanStartJob(line, "line", domain.Submissions, domain.JobInstances, palletWeightTask(intakeTasks(t))) {
		t.Fatal("line must be startable after palletWeight gate")
	}

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "line/shift",
		JobInstanceID: "1st",
		Value:         map[string]any{"shift": "1st"},
		Mint:          startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.JobInstanceID != "1st" {
		t.Fatalf("jobInstanceId = %q", result.JobInstanceID)
	}
	found := false
	for _, inst := range result.Domain.JobInstances {
		if inst.JobID == "line" && inst.ID == "1st" && inst.Label == "Turno: 1st" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("line Job instance missing: %#v", result.Domain.JobInstances)
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.JobID != "line" || last.TaskID != "shift" || last.JobInstanceID != "1st" {
		t.Fatalf("shift Submission = %#v", last)
	}
}

func TestSequenceOkForCommitIncludesStartsWhen(t *testing.T) {
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		t.Fatal(err)
	}
	empty := EmptyDomain()
	if sequenceOkForCommit(cfg, empty, "line", "shift", "1st") {
		t.Fatal("sequenceOk must be false before startsWhen gate")
	}

	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	domain := completeIntakeThroughPalletWeight(t, c, "NIR-261019-NO-1")
	if !sequenceOkForCommit(cfg, domain, "line", "shift", "1st") {
		t.Fatal("sequenceOk must be true after fixture some gate")
	}
	// Intake Jobs without startsWhen stay gated only by Unlock.
	if !sequenceOkForCommit(cfg, domain, "intake", "weightCheck", "NIR-261019-NO-1") {
		t.Fatal("intake Unlock must still pass independently of line startsWhen")
	}
}
