package org

import "testing"

func intakePathCfg() OrgConfig {
	return OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"intake"},
			Values: map[string]any{
				"intake": Job{Repeatable: true, Tasks: intakeUnlockTasks()},
			},
		},
	}
}

func TestEvaluatePathUnlockFlatLeaf(t *testing.T) {
	cfg := intakePathCfg()
	empty := EmptyDomain()

	first := EvaluatePathUnlock(cfg, empty, TaskPathRef{JobID: "intake", TaskID: "truckArrive"}, "")
	if !first.Unlocked || first.Done {
		t.Fatalf("first leaf = %#v", first)
	}

	registerLocked := EvaluatePathUnlock(cfg, empty, TaskPathRef{JobID: "intake", TaskID: "register"}, "T1")
	if registerLocked.Unlocked || registerLocked.Done {
		t.Fatalf("register before arrive = %#v", registerLocked)
	}

	afterArrive := Domain{Submissions: []Submission{sub("intake", "truckArrive", "T1")}}
	registerOpen := EvaluatePathUnlock(cfg, afterArrive, TaskPathRef{JobID: "intake", TaskID: "register"}, "T1")
	if !registerOpen.Unlocked || registerOpen.Done {
		t.Fatalf("register after arrive = %#v", registerOpen)
	}

	arriveDone := EvaluatePathUnlock(cfg, afterArrive, TaskPathRef{JobID: "intake", TaskID: "truckArrive"}, "T1")
	if !arriveDone.Unlocked || !arriveDone.Done {
		t.Fatalf("truckArrive submitted = %#v", arriveDone)
	}
}

func TestSequenceOkForPathUsesUnlockedHalf(t *testing.T) {
	cfg := intakePathCfg()
	domain := Domain{Submissions: []Submission{sub("intake", "truckArrive", "T1")}}
	if sequenceOkForPath(cfg, domain, TaskPathRef{JobID: "intake", TaskID: "weightCheck"}, "T1") {
		t.Fatal("sequenceOk before register")
	}
	domain.Submissions = append(domain.Submissions, sub("intake", "register", "T1"))
	if !sequenceOkForPath(cfg, domain, TaskPathRef{JobID: "intake", TaskID: "weightCheck"}, "T1") {
		t.Fatal("sequenceOk after register")
	}
}

func TestProjectStatusAdaptersMapPathUnlock(t *testing.T) {
	cfg := intakePathCfg()
	domain := Domain{Submissions: []Submission{sub("intake", "truckArrive", "T1")}}

	if got := projectTaskStatus(cfg, domain, "intake", "truckArrive", "T1"); got != TaskStatusDone {
		t.Fatalf("truckArrive status = %q", got)
	}
	if got := projectTaskStatus(cfg, domain, "intake", "register", "T1"); got != TaskStatusAvailable {
		t.Fatalf("register status = %q", got)
	}
}
