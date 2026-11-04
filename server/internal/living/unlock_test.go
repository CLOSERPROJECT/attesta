package living

import (
	"testing"
)

func TestOrderedTaskIDsPreservesAuthoredOrder(t *testing.T) {
	tasks := []TaskConfig{
		{"id": "truckArrive"},
		{"id": "lots", "kind": "group", "tasks": []TaskConfig{{"id": "register"}}},
		{"id": "weightCheck"},
	}
	got := OrderedTaskIDs(tasks)
	want := []string{"truckArrive", "lots", "weightCheck"}
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
		OrderedIDs:  []string{"truckArrive", "lots", "weightCheck"},
		JobID:       "intake",
		TaskID:      "truckArrive",
		Submissions: nil,
		Tasks:       tasks,
	}) {
		t.Fatal("first Task must be unlocked with no Submissions")
	}
}

func TestIsTaskUnlockedRequiresPriorSubmissionOnSameInstance(t *testing.T) {
	tasks := intakeUnlockTasks()
	order := []string{"truckArrive", "lots", "weightCheck"}

	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "lots",
		JobInstanceID: "T1", Submissions: nil, Tasks: tasks,
	}) {
		t.Fatal("lots locked before truckArrive Submission")
	}

	afterArrive := []Submission{sub("intake", "truckArrive", "T1")}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "lots",
		JobInstanceID: "T1", Submissions: afterArrive, Tasks: tasks,
	}) {
		t.Fatal("lots should unlock after truckArrive on same instance")
	}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: "T1", Submissions: afterArrive, Tasks: tasks,
	}) {
		t.Fatal("weightCheck must stay locked until lots close")
	}

	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: "T1", Submissions: afterArrive, Tasks: tasks,
		ClosedGroups: []ClosedGroup{{JobID: "intake", JobInstanceID: "T1", GroupID: "lots"}},
	}) {
		t.Fatal("weightCheck unlocks after lots close")
	}
}

func TestIsTaskUnlockedIgnoresOtherJobInstance(t *testing.T) {
	tasks := intakeUnlockTasks()
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs:    []string{"truckArrive", "lots"},
		JobID:         "intake",
		TaskID:        "lots",
		JobInstanceID: "T1",
		Submissions:   []Submission{sub("intake", "truckArrive", "T2")},
		Tasks:         tasks,
	}) {
		t.Fatal("Submission on another Job instance must not unlock")
	}
}

func TestIsTaskUnlockedDistinctFromStartsWhen(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	lineVal, ok := cfg.Jobs.Get("line")
	if !ok {
		t.Fatal("line job missing")
	}
	line := lineVal.(Job)
	if line.StartsWhen != nil {
		t.Fatalf("fixture must not author line startsWhen, got %#v", line.StartsWhen)
	}

	intakeVal := cfg.Jobs.Values["intake"].(Job)
	order := OrderedTaskIDs(intakeVal.Tasks)
	subs := []Submission{sub("line", "openTurno", "2026-10-19-1st")}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "lots",
		JobInstanceID: "T1", Submissions: subs, Tasks: intakeVal.Tasks,
	}) {
		t.Fatal("Unlock must not treat other Job Submissions as in-Job predecessors")
	}
}

func TestIsTaskUnlockedSubmissionGatesNext(t *testing.T) {
	tasks := []TaskConfig{
		{
			"id": "lineLoad",
			"dynamicProperties": map[string]any{
				"progress": map[string]any{"read": "rows"},
			},
		},
		{"id": "next"},
	}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: []string{"lineLoad", "next"}, JobID: "line", TaskID: "next",
		JobInstanceID: "line", Submissions: nil, Tasks: tasks,
	}) {
		t.Fatal("next must stay locked without predecessor Submission")
	}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: []string{"lineLoad", "next"}, JobID: "line", TaskID: "next",
		JobInstanceID: "line", Submissions: []Submission{sub("line", "lineLoad", "line")},
		Tasks: tasks,
	}) {
		t.Fatal("predecessor Submission must unlock next")
	}
}

func TestIsTaskUnlockedFanOutFromSharedPredecessor(t *testing.T) {
	// Synthetic: after open Submission, linear sibling mid and unlockFrom sibling side both unlock.
	tasks := []TaskConfig{
		{"id": "open"},
		{"id": "mid"},
		{"id": "side", "unlockFrom": "open"},
	}
	order := OrderedTaskIDs(tasks)

	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "job", TaskID: "mid",
		JobInstanceID: "J1", Submissions: nil, Tasks: tasks,
	}) || IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "job", TaskID: "side",
		JobInstanceID: "J1", Submissions: nil, Tasks: tasks,
	}) {
		t.Fatal("successors locked before open Submission")
	}

	afterOpen := []Submission{sub("job", "open", "J1")}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "job", TaskID: "mid",
		JobInstanceID: "J1", Submissions: afterOpen, Tasks: tasks,
	}) {
		t.Fatal("mid unlocks linearly after open")
	}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "job", TaskID: "side",
		JobInstanceID: "J1", Submissions: afterOpen, Tasks: tasks,
	}) {
		t.Fatal("side unlocks via unlockFrom open without waiting for mid")
	}

	bothFromOpen := []TaskConfig{
		{"id": "open"},
		{"id": "alpha", "unlockFrom": "open"},
		{"id": "beta", "unlockFrom": "open"},
	}
	bothOrder := OrderedTaskIDs(bothFromOpen)
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: bothOrder, JobID: "job", TaskID: "alpha",
		JobInstanceID: "J1", Submissions: afterOpen, Tasks: bothFromOpen,
	}) || !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: bothOrder, JobID: "job", TaskID: "beta",
		JobInstanceID: "J1", Submissions: afterOpen, Tasks: bothFromOpen,
	}) {
		t.Fatal("both unlockFrom successors unlock after shared predecessor Submission")
	}
}

func TestIsTaskUnlockedFanOutGroupPredecessorStillRequiresClose(t *testing.T) {
	tasks := []TaskConfig{
		{"id": "open"},
		{
			"id":   "batch",
			"kind": "group",
			"tasks": []TaskConfig{
				{"id": "item"},
			},
		},
		{"id": "follow", "unlockFrom": "batch"},
		{"id": "linearAfter"},
	}
	order := OrderedTaskIDs(tasks)
	afterOpen := []Submission{sub("job", "open", "J1")}
	childOnly := append(afterOpen, Submission{
		JobID: "job", TaskID: "item", JobInstanceID: "J1",
		GroupID: "batch", GroupInstanceID: "B1",
		Timestamp: "2026-11-04T10:00:00Z", UserID: "ada", Facts: map[string]any{},
	})

	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "job", TaskID: "follow",
		JobInstanceID: "J1", Submissions: childOnly, Tasks: tasks,
	}) {
		t.Fatal("unlockFrom group predecessor must not unlock on child Submission")
	}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "job", TaskID: "linearAfter",
		JobInstanceID: "J1", Submissions: childOnly, Tasks: tasks,
	}) {
		t.Fatal("linear successor of unlockFrom leaf stays locked until that leaf submits")
	}

	closed := []ClosedGroup{{JobID: "job", JobInstanceID: "J1", GroupID: "batch"}}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "job", TaskID: "follow",
		JobInstanceID: "J1", Submissions: childOnly, ClosedGroups: closed, Tasks: tasks,
	}) {
		t.Fatal("unlockFrom group predecessor unlocks on Close")
	}
}

func intakeUnlockTasks() []TaskConfig {
	return []TaskConfig{
		{"id": "truckArrive"},
		{
			"id":   "lots",
			"kind": "group",
			"tasks": []TaskConfig{
				{"id": "register"},
			},
		},
		{"id": "weightCheck"},
	}
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
