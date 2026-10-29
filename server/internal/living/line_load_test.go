package living

import (
	"strings"
	"testing"
)

func lineLoadTask(line Job) any {
	v, _ := line.Tasks.Get("lineLoad")
	return v
}

func commitLineShift(t *testing.T, c *Committer, shift string) Domain {
	t.Helper()
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/shift", JobInstanceID: shift,
		Value: map[string]any{"shift": shift},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Domain
}

func TestFixtureLineLoadCompletenessSome(t *testing.T) {
	line := lineJob(t)
	task := lineLoadTask(line)
	if taskMayBeUnlockGate(task) {
		t.Fatal("fixture lineLoad completeness some must not be an unlock / startsWhen gate")
	}
	m := task.(map[string]any)
	dyn := m["dynamicProperties"].(map[string]any)
	progress := dyn["progress"].(map[string]any)
	if progress["completeness"] != "some" {
		t.Fatalf("completeness = %#v", progress["completeness"])
	}
}

func TestLineLoadSomeNeverSatisfiesStartsWhen(t *testing.T) {
	line := lineJob(t)
	gate := &StartsWhen{
		Task:       StartsWhenRef{JobID: "line", TaskID: "lineLoad"},
		Quantifier: "some",
	}
	subs := []Submission{sub("line", "lineLoad", "1st")}
	instances := []JobInstance{{JobID: "line", ID: "1st", Label: "Turno: 1st"}}
	if IsStartsWhenSatisfied(gate, subs, instances, lineLoadTask(line)) {
		t.Fatal("lineLoad completeness some must never satisfy startsWhen")
	}
}

func TestCommitLineLoadBatchSomeDoesNotUnlockNext(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	intakeID := "NIR-261019-NO-1"
	completeIntakeThroughPalletWeight(t, c, intakeID)
	commitLineShift(t, c, "1st")

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: "1st",
		Value: map[string]any{
			"progress": map[string]any{
				intakeID + ".1": "done",
				intakeID + ".2": "partial",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Pallet[intakeID+".1"]["progress"] != "done" {
		t.Fatalf("pallet .1 = %#v", result.Domain.Pallet[intakeID+".1"])
	}
	if result.Domain.Pallet[intakeID+".2"]["progress"] != "partial" {
		t.Fatalf("pallet .2 = %#v", result.Domain.Pallet[intakeID+".2"])
	}
	leftovers := lineLoadLeftoverIDs(result.Domain)
	if len(leftovers) != 1 || leftovers[0] != intakeID+".2" {
		t.Fatalf("leftovers after batch = %#v", leftovers)
	}

	line := lineJob(t)
	if taskMayBeUnlockGate(lineLoadTask(line)) {
		t.Fatal("lineLoad must stay a non-gate after Submission")
	}
	// Synthetic successor without unlockFrom:instance — Submission of some must not open it.
	tasks := newOrderedObject()
	tasks.Set("lineLoad", lineLoadTask(line))
	tasks.Set("next", map[string]any{})
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: []string{"lineLoad", "next"}, JobID: "line", TaskID: "next",
		JobInstanceID: "1st", Submissions: result.Domain.Submissions, Tasks: tasks,
	}) {
		t.Fatal("completeness some Submission must not unlock next Task")
	}
	// Fixture containerReg opens with the Job instance, not via lineLoad as unlock gate.
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(line.Tasks), JobID: "line", TaskID: "containerReg",
		JobInstanceID: "1st", Submissions: result.Domain.Submissions, Tasks: line.Tasks,
	}) {
		t.Fatal("containerReg unlockFrom instance must open with Job instance despite some predecessor")
	}
}

func TestCommitLineLoadMultiInstanceIndependentProgress(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	intakeID := "NIR-261019-NO-1"
	domain := completeIntakeThroughPalletWeight(t, c, intakeID)

	line := lineJob(t)
	if !CanStartJob(line, "line", domain.Submissions, domain.JobInstances, palletWeightTask(intakeTasks(t))) {
		t.Fatal("line must be startable after intake gate")
	}

	commitLineShift(t, c, "1st")
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	first, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: "1st",
		Value: map[string]any{
			"progress": map[string]any{
				intakeID + ".1": "done",
				intakeID + ".2": "partial",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Repeatable line stays startable after the first Job instance exists.
	if !CanStartJob(line, "line", first.Domain.Submissions, first.Domain.JobInstances, palletWeightTask(intakeTasks(t))) {
		t.Fatal("repeatable line must stay startable for a second Job instance")
	}

	commitLineShift(t, c, "2nd")
	second, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: "2nd",
		Value: map[string]any{
			"progress": map[string]any{
				intakeID + ".2": nil,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, has := second.Domain.Pallet[intakeID+".2"]["progress"]; has {
		t.Fatalf("second instance clear must unset progress: %#v", second.Domain.Pallet[intakeID+".2"])
	}
	if second.Domain.Pallet[intakeID+".1"]["progress"] != "done" {
		t.Fatalf("first instance done must remain: %#v", second.Domain.Pallet[intakeID+".1"])
	}

	var saw1st, saw2nd bool
	for _, inst := range second.Domain.JobInstances {
		if inst.JobID != "line" {
			continue
		}
		if inst.ID == "1st" {
			saw1st = true
		}
		if inst.ID == "2nd" {
			saw2nd = true
		}
	}
	if !saw1st || !saw2nd {
		t.Fatalf("job instances = %#v", second.Domain.JobInstances)
	}

	var lineLoadOn1st, lineLoadOn2nd int
	for _, s := range second.Domain.Submissions {
		if s.JobID != "line" || s.TaskID != "lineLoad" {
			continue
		}
		switch s.JobInstanceID {
		case "1st":
			lineLoadOn1st++
		case "2nd":
			lineLoadOn2nd++
		}
	}
	if lineLoadOn1st != 1 || lineLoadOn2nd != 1 {
		t.Fatalf("lineLoad Submissions per instance: 1st=%d 2nd=%d", lineLoadOn1st, lineLoadOn2nd)
	}

	leftovers := lineLoadLeftoverIDs(second.Domain)
	if len(leftovers) != 1 || leftovers[0] != intakeID+".2" {
		t.Fatalf("cleared pallet must re-enter leftovers: %#v", leftovers)
	}
}

func TestCommitLineLoadRejectsEmptyBatch(t *testing.T) {
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	intakeID := "NIR-261019-NO-1"
	completeIntakeThroughPalletWeight(t, c, intakeID)
	before := commitLineShift(t, c, "1st")

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	_, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: "1st",
		Value: map[string]any{"progress": map[string]any{}},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(ce.Message, "nothing to submit") {
		t.Fatalf("message = %q", ce.Message)
	}

	domain, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(domain.Submissions) != len(before.Submissions) {
		t.Fatal("empty some batch must not append Submission")
	}
}
