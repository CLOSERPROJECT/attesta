package living

import (
	"testing"
)

func TestOrderedTaskIDsPreservesAuthoredOrder(t *testing.T) {
	tasks := newOrderedObject()
	tasks.Set("register", map[string]any{})
	tasks.Set("codiceFir", map[string]any{})
	tasks.Set("palletWeight", map[string]any{})
	got := OrderedTaskIDs(tasks)
	want := []string{"register", "codiceFir", "palletWeight"}
	if len(got) != len(want) {
		t.Fatalf("got = %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got = %#v want %#v", got, want)
		}
	}
}

func TestIsTaskUnlockedFirstAlwaysOpen(t *testing.T) {
	tasks := intakeUnlockTasks()
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs:  []string{"register", "codiceFir", "palletWeight"},
		JobID:       "intake",
		TaskID:      "register",
		Submissions: nil,
		Tasks:       tasks,
	}) {
		t.Fatal("first Task must be unlocked with no Submissions")
	}
}

func TestIsTaskUnlockedRequiresPriorSubmissionOnSameInstance(t *testing.T) {
	tasks := intakeUnlockTasks()
	order := []string{"register", "codiceFir", "palletWeight", "weightCheck"}

	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "codiceFir",
		JobInstanceID: "load-1", Submissions: nil, Tasks: tasks,
	}) {
		t.Fatal("codiceFir locked before register Submission")
	}

	afterRegister := []Submission{sub("intake", "register", "load-1")}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "codiceFir",
		JobInstanceID: "load-1", Submissions: afterRegister, Tasks: tasks,
	}) {
		t.Fatal("codiceFir should unlock after register on same instance")
	}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "palletWeight",
		JobInstanceID: "load-1", Submissions: afterRegister, Tasks: tasks,
	}) {
		t.Fatal("palletWeight must stay locked until codiceFir")
	}

	afterFir := append(afterRegister, sub("intake", "codiceFir", "load-1"))
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "palletWeight",
		JobInstanceID: "load-1", Submissions: afterFir, Tasks: tasks,
	}) {
		t.Fatal("palletWeight unlocks after codiceFir (simple gate into later intake)")
	}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: "load-1", Submissions: afterFir, Tasks: tasks,
	}) {
		t.Fatal("weightCheck still locked")
	}
}

func TestIsTaskUnlockedIgnoresOtherJobInstance(t *testing.T) {
	tasks := intakeUnlockTasks()
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs:    []string{"register", "codiceFir"},
		JobID:         "intake",
		TaskID:        "codiceFir",
		JobInstanceID: "load-1",
		Submissions:   []Submission{sub("intake", "register", "load-2")},
		Tasks:         tasks,
	}) {
		t.Fatal("Submission on another Job instance must not unlock")
	}
}

func TestIsTaskUnlockedDistinctFromStartsWhen(t *testing.T) {
	// Unlock order only reads prior Task Submissions on this Job instance —
	// not Job.startsWhen (line opens from intake/weightCheck) or readyWhen.
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		t.Fatal(err)
	}
	lineVal, ok := cfg.Jobs.Get("line")
	if !ok {
		t.Fatal("line job missing")
	}
	line := lineVal.(Job)
	if line.StartsWhen == nil || line.StartsWhen.Task.JobID != "intake" {
		t.Fatalf("fixture startsWhen = %#v", line.StartsWhen)
	}

	intakeVal := cfg.Jobs.Values["intake"].(Job)
	order := OrderedTaskIDs(intakeVal.Tasks)
	// A Submission of the Job-level startsWhen target must not unlock in-Job Tasks
	// on a different instance — Unlock order only looks at the immediate predecessor.
	subs := []Submission{sub("intake", line.StartsWhen.Task.TaskID, "other-load")}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "codiceFir",
		JobInstanceID: "load-1", Submissions: subs, Tasks: intakeVal.Tasks,
	}) {
		t.Fatal("Unlock must not treat startsWhen targets as in-Job predecessors")
	}
}

func TestIsTaskUnlockedCompletenessSomeNeverGatesNext(t *testing.T) {
	tasks := newOrderedObject()
	tasks.Set("lineLoad", map[string]any{
		"dynamicProperties": map[string]any{
			"progress": map[string]any{"read": "leftovers", "completeness": "some"},
		},
	})
	tasks.Set("next", map[string]any{})
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: []string{"lineLoad", "next"}, JobID: "line", TaskID: "next",
		JobInstanceID: "line", Submissions: []Submission{sub("line", "lineLoad", "line")},
		Tasks: tasks,
	}) {
		t.Fatal("completeness some predecessor must not unlock next")
	}
}

func intakeUnlockTasks() OrderedObject {
	tasks := newOrderedObject()
	tasks.Set("register", map[string]any{})
	tasks.Set("codiceFir", map[string]any{})
	tasks.Set("palletWeight", map[string]any{
		"dynamicProperties": map[string]any{
			"weights": map[string]any{"read": "leftovers", "completeness": "all"},
		},
	})
	tasks.Set("weightCheck", map[string]any{})
	return tasks
}

func sub(jobID, taskID, instanceID string) Submission {
	return Submission{
		JobID:         jobID,
		TaskID:        taskID,
		JobInstanceID: instanceID,
		Timestamp:     "2026-10-29T15:00:00Z",
		UserID:        "ada",
		Facts:         map[string]any{},
	}
}
