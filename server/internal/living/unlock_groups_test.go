package living

import (
	"testing"
)

func TestIsTaskUnlockedGroupPredecessorRequiresClose(t *testing.T) {
	tasks := intakeWithLotsTasks()
	order := OrderedTaskIDs(tasks)

	afterArrive := []Submission{sub("intake", "truckArrive", "T1")}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: "T1", Submissions: afterArrive, Tasks: tasks,
	}) {
		t.Fatal("weightCheck must stay locked before lots close")
	}

	childOnly := append(afterArrive, Submission{
		JobID: "intake", TaskID: "register", JobInstanceID: "T1",
		GroupID: "lots", GroupInstanceID: "L1",
		Timestamp: "2026-11-02T10:00:00Z", UserID: "ada", Facts: map[string]any{},
	})
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: "T1", Submissions: childOnly, Tasks: tasks,
	}) {
		t.Fatal("one child Submission must not unlock weightCheck")
	}

	closed := []ClosedGroup{{JobID: "intake", JobInstanceID: "T1", GroupID: "lots"}}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: "T1", Submissions: childOnly, ClosedGroups: closed, Tasks: tasks,
	}) {
		t.Fatal("weightCheck unlocks when lots is closed")
	}
}

func TestIsTaskUnlockedLeafToLeafUnchanged(t *testing.T) {
	tasks := []TaskConfig{
		{"id": "truckArrive"},
		{"id": "weightCheck"},
	}
	order := []string{"truckArrive", "weightCheck"}
	afterArrive := []Submission{sub("intake", "truckArrive", "T1")}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: "T1", Submissions: afterArrive, Tasks: tasks,
	}) {
		t.Fatal("flat leaf→leaf unlock must still work")
	}
}

func TestIsTaskUnlockedWithinOpenGroupInstance(t *testing.T) {
	children := []TaskConfig{
		{"id": "prepare"},
		{"id": "register"},
	}
	order := OrderedTaskIDs(children)

	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "prepare",
		JobInstanceID: "T1", GroupID: "lots", GroupInstanceID: "L1",
		Tasks: children,
	}) {
		t.Fatal("first child leaf always unlocked")
	}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "register",
		JobInstanceID: "T1", GroupID: "lots", GroupInstanceID: "L1",
		Tasks: children,
	}) {
		t.Fatal("second child locked without prior Submission on this group instance")
	}

	subs := []Submission{{
		JobID: "intake", TaskID: "prepare", JobInstanceID: "T1",
		GroupID: "lots", GroupInstanceID: "L1",
		Timestamp: "2026-11-02T10:00:00Z", UserID: "ada", Facts: map[string]any{},
	}}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "register",
		JobInstanceID: "T1", GroupID: "lots", GroupInstanceID: "L1",
		Submissions: subs, Tasks: children,
	}) {
		t.Fatal("child unlocks after predecessor Submission on same group instance")
	}

	otherInstance := []Submission{{
		JobID: "intake", TaskID: "prepare", JobInstanceID: "T1",
		GroupID: "lots", GroupInstanceID: "L2",
		Timestamp: "2026-11-02T10:00:00Z", UserID: "ada", Facts: map[string]any{},
	}}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: order, JobID: "intake", TaskID: "register",
		JobInstanceID: "T1", GroupID: "lots", GroupInstanceID: "L1",
		Submissions: otherInstance, Tasks: children,
	}) {
		t.Fatal("Submission on another group instance must not unlock")
	}
}

func intakeWithLotsTasks() []TaskConfig {
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
