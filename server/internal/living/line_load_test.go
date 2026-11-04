package living

import (
	"strings"
	"testing"
)

func lineLoadTask(line Job) TaskConfig {
	v, _ := TaskByID(line.Tasks, "lineLoad")
	return v
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
	subs := []Submission{sub("line", "lineLoad", "2026-10-19-1st")}
	instances := []JobInstance{{JobID: "line", ID: "2026-10-19-1st", Label: "Turno: 2026-10-19 1°"}}
	if !IsStartsWhenSatisfied(gate, subs, instances, lineLoadTask(line)) {
		t.Fatal("lineLoad Submission must satisfy startsWhen")
	}
}

func TestCommitLineLoadSubmissionUnlocksNext(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: lineID,
		Value: map[string]any{
			"progress": map[string]any{
				loadID + ".1": "done",
				loadID + ".2": "partial",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Kind("pallet")[loadID+".1"]["progress"] != "done" {
		t.Fatalf("pallet .1 = %#v", result.Domain.Kind("pallet")[loadID+".1"])
	}
	if result.Domain.Kind("pallet")[loadID+".2"]["progress"] != "partial" {
		t.Fatalf("pallet .2 = %#v", result.Domain.Kind("pallet")[loadID+".2"])
	}
	rows := lineLoadRowIDs(result.Domain)
	if len(rows) != 1 || rows[0] != loadID+".2" {
		t.Fatalf("rows after batch = %#v", rows)
	}

	line := lineJob(t)
	tasks := []TaskConfig{lineLoadTask(line), {"id": "next"}}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: []string{"lineLoad", "next"}, JobID: "line", TaskID: "next",
		JobInstanceID: lineID, Submissions: result.Domain.Submissions, Tasks: tasks,
	}) {
		t.Fatal("lineLoad Submission must unlock next Task")
	}
	// containerReg unlocks from openTurno (path), not lineLoad; still unlocked after lineLoad.
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(line.Tasks), JobID: "line", TaskID: "containerReg",
		JobInstanceID: lineID, Submissions: result.Domain.Submissions, Tasks: line.Tasks,
		ClosedGroups: result.Domain.ClosedGroups,
	}) {
		t.Fatal("containerReg must stay unlocked via openTurno after lineLoad Submission")
	}
}

func TestCommitLineLoadMultiInstanceIndependentProgress(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, line1 := seedLineReadyForLineLoad(t, c)

	line := lineJob(t)
	if !CanStartJob(line, "line", nil, nil, nil) {
		t.Fatal("repeatable line must stay startable")
	}

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	first, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: line1,
		Value: map[string]any{
			"progress": map[string]any{
				loadID + ".1": "done",
				loadID + ".2": "partial",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	line2 := commitOpenTurnoWith(t, c, map[string]any{"date": "2026-10-19", "turno": "2nd"})
	// closeRequiresChildSubmission: need a weigh leaf on this Job instance before Close.
	_ = commitWeigh(t, c, line2, loadID+".1", 11)
	closePalletWeight(t, c, line2)

	second, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: line2,
		Value: map[string]any{
			"progress": map[string]any{
				loadID + ".2": nil,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, has := second.Domain.Kind("pallet")[loadID+".2"]["progress"]; has {
		t.Fatalf("second instance clear must unset progress: %#v", second.Domain.Kind("pallet")[loadID+".2"])
	}
	if second.Domain.Kind("pallet")[loadID+".1"]["progress"] != "done" {
		t.Fatalf("first instance done must remain: %#v", second.Domain.Kind("pallet")[loadID+".1"])
	}

	var saw1st, saw2nd bool
	for _, inst := range second.Domain.JobInstances {
		if inst.JobID != "line" {
			continue
		}
		if inst.ID == line1 {
			saw1st = true
		}
		if inst.ID == line2 {
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
		case line1:
			lineLoadOn1st++
		case line2:
			lineLoadOn2nd++
		}
	}
	if lineLoadOn1st != 1 || lineLoadOn2nd != 1 {
		t.Fatalf("lineLoad Submissions per instance: 1st=%d 2nd=%d", lineLoadOn1st, lineLoadOn2nd)
	}

	rows := lineLoadRowIDs(second.Domain)
	if len(rows) != 1 || rows[0] != loadID+".2" {
		t.Fatalf("cleared pallet must re-enter rows: %#v", rows)
	}
	_ = first
}

func TestCommitLineLoadRejectsEmptyBatch(t *testing.T) {
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	_, lineID := seedLineReadyForLineLoad(t, c)
	before, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: lineID,
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

func TestFixtureLineLoadAdditionalPropertiesIsCellMap(t *testing.T) {
	task := map[string]any(lineLoadTask(lineJob(t)))
	add := additionalPropertiesSchema(task, "progress")
	if !additionalPropertiesIsCellMap(add) {
		t.Fatalf("additionalPropertiesIsCellMap = false (null|enum anyOf must still classify as cell map); schema=%#v", add)
	}
}
