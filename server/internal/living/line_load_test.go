package living

import (
	"strings"
	"testing"
)

func lineLoadGroup(line Job) TaskConfig {
	v, _ := TaskByID(line.Tasks, "lineLoad")
	return v
}

func lineLoadMarkTask(line Job) TaskConfig {
	v, _ := TaskByID(lineLoadGroup(line).ChildTasks(), "mark")
	return v
}

func commitLineLoadMark(t *testing.T, c *Committer, lineID string, progress map[string]any) CommitResult {
	t.Helper()
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      GroupTaskPath("line", "lineLoad", ProvisionalGroupInstanceID, "mark"),
		JobInstanceID: lineID,
		Value:         map[string]any{"progress": progress},
		Mint:          spawningNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestFixtureLineLoadReadsDomainProgress(t *testing.T) {
	line := lineJob(t)
	group := lineLoadGroup(line)
	if !group.IsGroup() {
		t.Fatalf("lineLoad must be a group, got %#v", group)
	}
	task := lineLoadMarkTask(line)
	dyn := task["dynamicProperties"].(map[string]any)
	progress := dyn["progress"].(map[string]any)
	if progress["read"] != "rows" {
		t.Fatalf("progress.read = %#v", progress["read"])
	}
}

func TestLineLoadMarkSubmissionDoesNotSatisfyGroupStartsWhen(t *testing.T) {
	line := lineJob(t)
	gate := &StartsWhen{
		Task:       StartsWhenRef{JobID: "line", TaskID: "lineLoad"},
		Quantifier: "some",
	}
	instances := []JobInstance{{JobID: "line", ID: "2026-10-19-1st", Label: "2026-10-19 Turno: 1°"}}
	subs := []Submission{{
		JobID: "line", TaskID: "mark", JobInstanceID: "2026-10-19-1st",
		GroupID: "lineLoad", GroupInstanceID: "lineLoad-1",
	}}
	if IsStartsWhenSatisfied(gate, subs, instances, lineLoadGroup(line)) {
		t.Fatal("child mark Submission must not satisfy startsWhen targeting group lineLoad")
	}
}

func TestCommitLineLoadSubmissionUnlocksNext(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)

	result := commitLineLoadMark(t, c, lineID, map[string]any{
		loadID + ".1": "done",
		loadID + ".2": "partial",
	})
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
	// Group predecessor unlocks the next sibling only after Close.
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: []string{"lineLoad", "next"}, JobID: "line", TaskID: "next",
		JobInstanceID: lineID, Submissions: result.Domain.Submissions, Tasks: []TaskConfig{lineLoadGroup(line), {"id": "next"}},
		ClosedGroups: result.Domain.ClosedGroups,
	}) {
		t.Fatal("next must stay locked until lineLoad Close")
	}
	closed, err := c.CloseGroup(t.Context(), "feragame", adminOp(), CloseGroupRequest{
		JobID: "line", JobInstanceID: lineID, GroupID: "lineLoad",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: []string{"lineLoad", "next"}, JobID: "line", TaskID: "next",
		JobInstanceID: lineID, Submissions: closed.Domain.Submissions, Tasks: []TaskConfig{lineLoadGroup(line), {"id": "next"}},
		ClosedGroups: closed.Domain.ClosedGroups,
	}) {
		t.Fatal("lineLoad Close must unlock next Task")
	}
	// containerReg unlocks from openTurno (path), not lineLoad; still unlocked after mark.
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(line.Tasks), JobID: "line", TaskID: "containerReg",
		JobInstanceID: lineID, Submissions: result.Domain.Submissions, Tasks: line.Tasks,
		ClosedGroups: result.Domain.ClosedGroups,
	}) {
		t.Fatal("containerReg must stay unlocked via openTurno after lineLoad mark")
	}
}

func TestCommitLineLoadMultiInstanceIndependentProgress(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, line1 := seedLineReadyForLineLoad(t, c)

	line := lineJob(t)
	if !CanStartJob(line, "line", nil, nil, nil) {
		t.Fatal("repeatable line must stay startable")
	}

	first := commitLineLoadMark(t, c, line1, map[string]any{
		loadID + ".1": "done",
		loadID + ".2": "partial",
	})

	line2 := commitOpenTurnoWith(t, c, map[string]any{"date": "2026-10-19", "turno": "2nd"})
	// closeRequiresChildSubmission: need a weigh leaf on this Job instance before Close.
	_ = commitWeigh(t, c, line2, loadID+".1", 11)
	closePalletWeight(t, c, line2)

	second := commitLineLoadMark(t, c, line2, map[string]any{
		loadID + ".2": nil,
	})

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

	var markOn1st, markOn2nd int
	for _, s := range second.Domain.Submissions {
		if s.JobID != "line" || s.TaskID != "mark" || s.GroupID != "lineLoad" {
			continue
		}
		switch s.JobInstanceID {
		case line1:
			markOn1st++
		case line2:
			markOn2nd++
		}
	}
	if markOn1st != 1 || markOn2nd != 1 {
		t.Fatalf("mark Submissions per instance: 1st=%d 2nd=%d", markOn1st, markOn2nd)
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
		TaskPath:      GroupTaskPath("line", "lineLoad", ProvisionalGroupInstanceID, "mark"),
		JobInstanceID: lineID,
		Value:         map[string]any{"progress": map[string]any{}},
		Mint:          spawningNewMint(),
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

func TestCommitLineLoadAllowsMultipleMarks(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)

	_ = commitLineLoadMark(t, c, lineID, map[string]any{
		loadID + ".1": "partial",
		loadID + ".2": "partial",
	})
	second := commitLineLoadMark(t, c, lineID, map[string]any{
		loadID + ".1": "done",
		loadID + ".2": "done",
	})
	var marks int
	for _, s := range second.Domain.Submissions {
		if s.JobID == "line" && s.GroupID == "lineLoad" && s.TaskID == "mark" && s.JobInstanceID == lineID {
			marks++
		}
	}
	if marks != 2 {
		t.Fatalf("mark submissions = %d, want 2", marks)
	}
	wl := ProjectWorklist(mustActiveConfig(t, c), second.Domain, adminOp())
	inst, ok := worklistInstance(worklistMustJob(t, wl, "line"), lineID)
	if !ok {
		t.Fatal("missing line instance")
	}
	group := inst.Groups["lineLoad"]
	if group.CanSpawn {
		t.Fatal("CanSpawn must be false when no unfinished weighed pallets remain")
	}
	if len(group.Instances) != 2 {
		t.Fatalf("group instances = %#v, want 2 marks", group.Instances)
	}
	wantLabel := loadID + ".1, " + loadID + ".2"
	if group.Instances[1].Label != wantLabel {
		t.Fatalf("second mark label = %q, want pallet ids %q", group.Instances[1].Label, wantLabel)
	}
}

func TestFixtureLineLoadAdditionalPropertiesIsCellMap(t *testing.T) {
	task := map[string]any(lineLoadMarkTask(lineJob(t)))
	add := additionalPropertiesSchema(task, "progress")
	if !additionalPropertiesIsCellMap(add) {
		t.Fatalf("additionalPropertiesIsCellMap = false (null|enum anyOf must still classify as cell map); schema=%#v", add)
	}
}
