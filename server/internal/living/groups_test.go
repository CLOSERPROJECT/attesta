package living

import (
	"strings"
	"testing"
)

func TestSpawnGroupInstance(t *testing.T) {
	d := EmptyDomain()
	d.JobInstances = []JobInstance{{JobID: "intake", ID: "T1", Label: "truck"}}

	next, err := SpawnGroupInstance(d, "intake", "T1", "lots", "L1")
	if err != nil {
		t.Fatal(err)
	}
	if len(next.GroupInstances) != 1 {
		t.Fatalf("groupInstances = %#v", next.GroupInstances)
	}
	got := next.GroupInstances[0]
	if got.JobID != "intake" || got.JobInstanceID != "T1" || got.GroupID != "lots" || got.ID != "L1" {
		t.Fatalf("spawned = %#v", got)
	}
	if len(d.GroupInstances) != 0 {
		t.Fatal("SpawnGroupInstance must not mutate input Domain")
	}

	if _, err := SpawnGroupInstance(next, "intake", "T1", "lots", "L1"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate spawn = %v", err)
	}

	closed, err := CloseGroup(withLotsChildSub(next, "T1", "L1"), lotsGroup(), "intake", "T1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SpawnGroupInstance(closed, "intake", "T1", "lots", "L2"); err == nil || !strings.Contains(err.Error(), "already closed") {
		t.Fatalf("spawn after close = %v", err)
	}
}

func TestCloseRequiresChildSubmissionConfigOnly(t *testing.T) {
	if closeRequiresChildSubmission(nil) {
		t.Fatal("nil group must not require child submission")
	}
	if closeRequiresChildSubmission(TaskConfig{"id": "lots", "kind": "group"}) {
		t.Fatal("id alone must not require child submission")
	}
	if closeRequiresChildSubmission(TaskConfig{
		"id": "any", "kind": "group", "closeRequiresChildSubmission": false,
	}) {
		t.Fatal("authored false must not require child submission")
	}
	if !closeRequiresChildSubmission(TaskConfig{
		"id": "any", "kind": "group", "closeRequiresChildSubmission": true,
	}) {
		t.Fatal("authored true must require child submission")
	}
}

func TestCloseGroupFlagRequiresChildSubmission(t *testing.T) {
	d := EmptyDomain()
	d, err := SpawnGroupInstance(d, "intake", "T1", "lots", "L1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = CloseGroup(d, lotsGroup(), "intake", "T1")
	if err == nil || !strings.Contains(err.Error(), "zero child leaf") {
		t.Fatalf("close with zero children = %v", err)
	}
	var ge *GroupError
	if !asGroupError(err, &ge) || ge.Kind != "missing-child-submission" {
		t.Fatalf("close err kind = %v", err)
	}

	withChild := withLotsChildSub(d, "T1", "L1")
	closed, err := CloseGroup(withChild, lotsGroup(), "intake", "T1")
	if err != nil {
		t.Fatal(err)
	}
	if !IsGroupClosed(closed.ClosedGroups, "intake", "T1", "lots") {
		t.Fatalf("closedGroups = %#v", closed.ClosedGroups)
	}

	_, err = CloseGroup(closed, lotsGroup(), "intake", "T1")
	if err == nil || !asGroupError(err, &ge) || ge.Kind != "already-closed" {
		t.Fatalf("second close = %v", err)
	}
}

func TestCloseGroupAuthoredFlagWithoutLotsID(t *testing.T) {
	group := TaskConfig{
		"id":                           "weighSeries",
		"kind":                         "group",
		"closeRequiresChildSubmission": true,
		"tasks":                        []TaskConfig{{"id": "weigh"}},
	}
	d := EmptyDomain()
	d, err := SpawnGroupInstance(d, "line", "S1", "weighSeries", "W1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CloseGroup(d, group, "line", "S1"); err == nil || !strings.Contains(err.Error(), "zero child leaf") {
		t.Fatalf("flagged group close = %v", err)
	}
	d.Submissions = []Submission{{
		JobID: "line", TaskID: "weigh", JobInstanceID: "S1",
		GroupID: "weighSeries", GroupInstanceID: "W1",
		UserID: "ada", Timestamp: "2026-11-02T10:00:00Z", Facts: map[string]any{},
	}}
	if _, err := CloseGroup(d, group, "line", "S1"); err != nil {
		t.Fatal(err)
	}
}

func TestCloseGroupWithoutMinChildAllowsEmpty(t *testing.T) {
	group := TaskConfig{
		"id":   "optionalSeries",
		"kind": "group",
		"tasks": []TaskConfig{
			{"id": "step"},
		},
	}
	d := EmptyDomain()
	closed, err := CloseGroup(d, group, "line", "S1")
	if err != nil {
		t.Fatal(err)
	}
	if !IsGroupClosed(closed.ClosedGroups, "line", "S1", "optionalSeries") {
		t.Fatalf("closedGroups = %#v", closed.ClosedGroups)
	}
}

func TestEvaluateGroupLifecycleAuthoredCloseRequires(t *testing.T) {
	group := TaskConfig{
		"id":                           "weighSeries",
		"kind":                         "group",
		"repeatable":                   true,
		"closeRequiresChildSubmission": true,
		"tasks":                        []TaskConfig{{"id": "weigh"}},
	}
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys:   []string{"line"},
			Values: map[string]any{"line": Job{Tasks: []TaskConfig{group}}},
		},
	}
	d := EmptyDomain()
	d.JobInstances = []JobInstance{{JobID: "line", ID: "S1", Label: "shift"}}

	lc := EvaluateGroupLifecycle(GroupLifecycleQuery{
		Group: group, JobID: "line", JobInstanceID: "S1", Cfg: cfg, Domain: d,
	})
	if !lc.CanSpawn || lc.CanClose {
		t.Fatalf("open empty authored-min = %#v", lc)
	}

	d, err := SpawnGroupInstance(d, "line", "S1", "weighSeries", "W1")
	if err != nil {
		t.Fatal(err)
	}
	lc = EvaluateGroupLifecycle(GroupLifecycleQuery{
		Group: group, JobID: "line", JobInstanceID: "S1", Cfg: cfg, Domain: d,
	})
	if !lc.CanSpawn || lc.CanClose {
		t.Fatalf("spawned without child = %#v", lc)
	}

	d.Submissions = []Submission{{
		JobID: "line", TaskID: "weigh", JobInstanceID: "S1",
		GroupID: "weighSeries", GroupInstanceID: "W1",
		UserID: "ada", Timestamp: "2026-11-02T10:00:00Z", Facts: map[string]any{},
	}}
	lc = EvaluateGroupLifecycle(GroupLifecycleQuery{
		Group: group, JobID: "line", JobInstanceID: "S1", Cfg: cfg, Domain: d,
	})
	if !lc.CanSpawn || !lc.CanClose {
		t.Fatalf("with child leaf = %#v", lc)
	}

	closed, err := CloseGroup(d, group, "line", "S1")
	if err != nil {
		t.Fatal(err)
	}
	lc = EvaluateGroupLifecycle(GroupLifecycleQuery{
		Group: group, JobID: "line", JobInstanceID: "S1", Cfg: cfg, Domain: closed,
	})
	if lc.CanSpawn || lc.CanClose {
		t.Fatalf("after close = %#v", lc)
	}
}

func lotsGroup() TaskConfig {
	return TaskConfig{
		"id":                           "lots",
		"kind":                         "group",
		"closeRequiresChildSubmission": true,
		"tasks": []TaskConfig{
			{"id": "register"},
		},
	}
}

func withLotsChildSub(d Domain, jobInstanceID, groupInstanceID string) Domain {
	next := cloneDomain(d)
	next.Submissions = append(next.Submissions, Submission{
		JobID:           "intake",
		TaskID:          "register",
		JobInstanceID:   jobInstanceID,
		GroupID:         "lots",
		GroupInstanceID: groupInstanceID,
		Timestamp:       "2026-11-02T10:00:00Z",
		UserID:          "ada",
		Facts:           map[string]any{},
	})
	return next
}

func asGroupError(err error, target **GroupError) bool {
	if err == nil {
		return false
	}
	ge, ok := err.(*GroupError)
	if !ok {
		return false
	}
	*target = ge
	return true
}
