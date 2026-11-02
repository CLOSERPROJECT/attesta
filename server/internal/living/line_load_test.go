package living

import (
	"strings"
	"testing"
)

func lineLoadTask(line Job) TaskConfig {
	v, _ := TaskByID(line.Tasks, "lineLoad")
	return v
}

func commitLineShift(t *testing.T, c *Committer, shift string) Domain {
	t.Helper()
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/shift", JobInstanceID: shift,
		Value: map[string]any{"shift": shift},
		Mint:  startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Domain
}

func TestFixtureLineLoadReadsDomainProgress(t *testing.T) {
	line := lineJob(t)
	task := lineLoadTask(line)
	dyn := task["dynamicProperties"].(map[string]any)
	progress := dyn["progress"].(map[string]any)
	if progress["read"] != "rows" {
		t.Fatalf("progress.read = %#v", progress["read"])
	}
}

func TestLineLoadSubmissionSatisfiesStartsWhen(t *testing.T) {
	line := lineJob(t)
	gate := &StartsWhen{
		Task:       StartsWhenRef{JobID: "line", TaskID: "lineLoad"},
		Quantifier: "some",
	}
	subs := []Submission{sub("line", "lineLoad", "1st")}
	instances := []JobInstance{{JobID: "line", ID: "1st", Label: "Turno: 1st"}}
	if !IsStartsWhenSatisfied(gate, subs, instances, lineLoadTask(line)) {
		t.Fatal("lineLoad Submission must satisfy startsWhen")
	}
}

func TestCommitLineLoadSubmissionUnlocksNext(t *testing.T) {
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
	if result.Domain.Kind(kindPallet)[intakeID+".1"]["progress"] != "done" {
		t.Fatalf("pallet .1 = %#v", result.Domain.Kind(kindPallet)[intakeID+".1"])
	}
	if result.Domain.Kind(kindPallet)[intakeID+".2"]["progress"] != "partial" {
		t.Fatalf("pallet .2 = %#v", result.Domain.Kind(kindPallet)[intakeID+".2"])
	}
	rows := lineLoadRowIDs(result.Domain)
	if len(rows) != 1 || rows[0] != intakeID+".2" {
		t.Fatalf("rows after batch = %#v", rows)
	}

	line := lineJob(t)
	tasks := []TaskConfig{lineLoadTask(line), {"id": "next"}}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: []string{"lineLoad", "next"}, JobID: "line", TaskID: "next",
		JobInstanceID: "1st", Submissions: result.Domain.Submissions, Tasks: tasks,
	}) {
		t.Fatal("lineLoad Submission must unlock next Task")
	}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(line.Tasks), JobID: "line", TaskID: "containerReg",
		JobInstanceID: "1st", Submissions: result.Domain.Submissions, Tasks: line.Tasks,
	}) {
		t.Fatal("containerReg must unlock after lineLoad Submission")
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

	if _, has := second.Domain.Kind(kindPallet)[intakeID+".2"]["progress"]; has {
		t.Fatalf("second instance clear must unset progress: %#v", second.Domain.Kind(kindPallet)[intakeID+".2"])
	}
	if second.Domain.Kind(kindPallet)[intakeID+".1"]["progress"] != "done" {
		t.Fatalf("first instance done must remain: %#v", second.Domain.Kind(kindPallet)[intakeID+".1"])
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

	rows := lineLoadRowIDs(second.Domain)
	if len(rows) != 1 || rows[0] != intakeID+".2" {
		t.Fatalf("cleared pallet must re-enter rows: %#v", rows)
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

func TestFixtureLineLoadAdditionalPropertiesIsProgress(t *testing.T) {
	task := map[string]any(lineLoadTask(lineJob(t)))
	add := additionalPropertiesSchema(task, "progress")
	if !additionalPropertiesIsProgress(add) {
		t.Fatalf("additionalPropertiesIsProgress = false (null|enum anyOf must still classify as progress); schema=%#v", add)
	}
}
