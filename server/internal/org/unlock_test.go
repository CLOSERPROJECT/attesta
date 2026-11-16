package org

import (
	"testing"
)

func TestOrderedTaskIDsPreservesAuthoredOrder(t *testing.T) {
	tasks := []TaskConfig{
		{"id": "truckArrive"},
		{"id": "register"},
		{"id": "weightCheck"},
	}
	got := OrderedTaskIDs(tasks)
	want := []string{"truckArrive", "register", "weightCheck"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order[%d]=%q want %q", i, got[i], want[i])
		}
	}
}

func TestIsTaskUnlockedFirstAlwaysOpen(t *testing.T) {
	tasks := intakeUnlockTasks()
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs:  []string{"truckArrive", "register", "weightCheck"},
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
	order := []string{"truckArrive", "register", "weightCheck"}

	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "register",
		JobInstanceID: "T1", Submissions: nil, Tasks: tasks,
	}) {
		t.Fatal("register locked before truckArrive Submission")
	}

	afterArrive := []Submission{sub("intake", "truckArrive", "T1")}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "register",
		JobInstanceID: "T1", Submissions: afterArrive, Tasks: tasks,
	}) {
		t.Fatal("register should unlock after truckArrive on same instance")
	}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: "T1", Submissions: afterArrive, Tasks: tasks,
	}) {
		t.Fatal("weightCheck must stay locked until register Submission")
	}

	afterRegister := []Submission{
		sub("intake", "truckArrive", "T1"),
		sub("intake", "register", "T1"),
	}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: "T1", Submissions: afterRegister, Tasks: tasks,
	}) {
		t.Fatal("weightCheck unlocks after register")
	}
}

func TestIsTaskUnlockedIgnoresOtherJobInstance(t *testing.T) {
	tasks := intakeUnlockTasks()
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs:    []string{"truckArrive", "register"},
		JobID:         "intake",
		TaskID:        "register",
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
	lpVal, ok := cfg.Jobs.Get("lot_processing")
	if !ok {
		t.Fatal("lot_processing job missing")
	}
	lp := lpVal.(Job)
	if lp.StartsWhen != nil {
		t.Fatalf("fixture must not author lot_processing startsWhen, got %#v", lp.StartsWhen)
	}

	intakeVal := cfg.Jobs.Values["intake"].(Job)
	order := OrderedTaskIDs(intakeVal.Tasks)
	subs := []Submission{sub("lot_processing", "startLot", "CO-261029-1")}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "register",
		JobInstanceID: "T1", Submissions: subs, Tasks: intakeVal.Tasks,
	}) {
		t.Fatal("Unlock must not treat other Job Submissions as in-Job predecessors")
	}
}

func TestIsTaskUnlockedSubmissionGatesNext(t *testing.T) {
	tasks := []TaskConfig{
		{
			"id": "markProgress",
			"dynamicProperties": map[string]any{
				"progress": map[string]any{"read": "rows"},
			},
		},
		{"id": "next"},
	}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: []string{"markProgress", "next"}, JobID: "lot_processing", TaskID: "next",
		JobInstanceID: "lot1", Submissions: nil, Tasks: tasks,
	}) {
		t.Fatal("next must stay locked without predecessor Submission")
	}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: []string{"markProgress", "next"}, JobID: "lot_processing", TaskID: "next",
		JobInstanceID: "lot1", Submissions: []Submission{sub("lot_processing", "markProgress", "lot1")},
		Tasks: tasks,
	}) {
		t.Fatal("predecessor Submission must unlock next")
	}
}

func TestIsTaskUnlockedFanOutFromSharedPredecessor(t *testing.T) {
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

func intakeUnlockTasks() []TaskConfig {
	return []TaskConfig{
		{"id": "truckArrive"},
		{"id": "register"},
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
