package org

import "testing"

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

func intakePathCfg() OrgConfig {
	return OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"intake"},
			Values: map[string]any{
				"intake": Job{Repeatable: true, Tasks: []TaskConfig{
					{"id": "truckArrive"},
					{"id": "register", "submit": "many"},
					{"id": "weightCheck"},
				}},
			},
		},
	}
}

func TestEvaluatePathAvailabilityDoneLaw(t *testing.T) {
	cfg := intakePathCfg()
	empty := EmptyDomain()

	first := EvaluatePathAvailability(cfg, empty, TaskPathRef{JobID: "intake", TaskID: "truckArrive"}, "")
	if first.Done {
		t.Fatalf("first leaf = %#v, want not done", first)
	}

	registerOpen := EvaluatePathAvailability(cfg, empty, TaskPathRef{JobID: "intake", TaskID: "register"}, "T1")
	if registerOpen.Done {
		t.Fatalf("register many before submit = %#v, want not done", registerOpen)
	}

	afterArrive := Domain{Submissions: []Submission{sub("intake", "truckArrive", "T1")}}
	arriveDone := EvaluatePathAvailability(cfg, afterArrive, TaskPathRef{JobID: "intake", TaskID: "truckArrive"}, "T1")
	if !arriveDone.Done {
		t.Fatalf("truckArrive submitted = %#v, want done", arriveDone)
	}

	registerStillOpen := EvaluatePathAvailability(cfg, afterArrive, TaskPathRef{JobID: "intake", TaskID: "register"}, "T1")
	if registerStillOpen.Done {
		t.Fatalf("register many after arrive = %#v, want not done", registerStillOpen)
	}
}

func TestSequenceOkForPathIsNotDoneAndStartsWhen(t *testing.T) {
	cfg := intakePathCfg()
	domain := Domain{Submissions: []Submission{sub("intake", "truckArrive", "T1")}}
	// No Unlock half: weightCheck is sequenceOk before register Submission.
	if !sequenceOkForPath(cfg, domain, TaskPathRef{JobID: "intake", TaskID: "weightCheck"}, "T1") {
		t.Fatal("sequenceOk before register (readyWhen is separate)")
	}
	domain.Submissions = append(domain.Submissions, sub("intake", "weightCheck", "T1"))
	if sequenceOkForPath(cfg, domain, TaskPathRef{JobID: "intake", TaskID: "weightCheck"}, "T1") {
		t.Fatal("sequenceOk after once Submission must be false (done)")
	}
}

func TestProjectTaskStatusMapsPathAvailability(t *testing.T) {
	cfg := intakePathCfg()
	domain := Domain{Submissions: []Submission{sub("intake", "truckArrive", "T1")}}

	if got := projectTaskStatus(cfg, domain, "intake", "truckArrive", "T1"); got != TaskStatusDone {
		t.Fatalf("truckArrive status = %q", got)
	}
	if got := projectTaskStatus(cfg, domain, "intake", "register", "T1"); got != TaskStatusAvailable {
		t.Fatalf("register status = %q, want available (no locked)", got)
	}
	if got := projectTaskStatus(cfg, domain, "intake", "weightCheck", "T1"); got != TaskStatusAvailable {
		t.Fatalf("weightCheck status = %q, want available (no Unlock lock)", got)
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
