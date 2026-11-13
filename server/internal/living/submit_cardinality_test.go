package living

import (
	"strings"
	"testing"
	"time"
)

// Synthetic Job: once bookend + many leaf + once close — no plant Task ids.
func submitCardinalityConfig() OrgConfig {
	start := TaskConfig{
		"id":    "start",
		"roles": []any{"operator"},
		"schema": map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		"onSubmit": map[string]any{"transform": []any{}},
	}
	repeat := TaskConfig{
		"id":     "repeat",
		"submit": "many",
		"roles":  []any{"operator"},
		"schema": map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		"onSubmit": map[string]any{"transform": []any{}},
		"onCommit": map[string]any{"navigation": "stay"},
	}
	finish := TaskConfig{
		"id":    "finish",
		"roles": []any{"operator"},
		"schema": map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		"onSubmit": map[string]any{"transform": []any{}},
	}
	return OrgConfig{
		ID:  "submit-cardinality",
		Org: "cardorg",
		Jobs: OrderedObject{
			Keys: []string{"session"},
			Values: map[string]any{
				"session": Job{
					Repeatable: true,
					StartLabel: "Start",
					InstanceID: "'S1'",
					Label:      "'Session S1'",
					Tasks:      []TaskConfig{start, repeat, finish},
				},
			},
		},
	}
}

func submitCardinalityCommitter(t *testing.T, auth Authorizer) *Committer {
	t.Helper()
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	cfg := submitCardinalityConfig()
	if err := configs.Put(t.Context(), "cardorg", "v1", cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), "cardorg", "v1"); err != nil {
		t.Fatal(err)
	}
	if auth == nil {
		auth = readyWhenGateAuthorizer{}
	}
	return &Committer{
		Domains:    domains,
		Configs:    configs,
		Authorizer: auth,
		Now:        func() time.Time { return time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC) },
	}
}

func TestValidateTaskSubmitCardinality(t *testing.T) {
	if err := validateTaskConfigs([]TaskConfig{
		{"id": "a"},
		{"id": "b", "submit": "many"},
		{"id": "c", "submit": "once"},
	}); err != nil {
		t.Fatalf("valid submit = %v", err)
	}

	if err := validateTaskConfigs([]TaskConfig{
		{"id": "bad", "submit": "sometimes"},
	}); err == nil || !strings.Contains(err.Error(), "submit must be") {
		t.Fatalf("invalid submit err = %v", err)
	}

	if err := validateTaskConfigs([]TaskConfig{
		{"id": "bad", "submit": 1},
	}); err == nil || !strings.Contains(err.Error(), "submit must be") {
		t.Fatalf("non-string submit err = %v", err)
	}

	if err := validateTaskConfigs([]TaskConfig{
		{
			"id":     "g",
			"kind":   "group",
			"submit": "many",
			"tasks":  []TaskConfig{{"id": "child"}},
		},
	}); err == nil || !strings.Contains(err.Error(), "group must not declare submit") {
		t.Fatalf("group submit err = %v", err)
	}
}

func TestSubmitManyStaysActionableAcrossTwoCommits(t *testing.T) {
	c := submitCardinalityCommitter(t, nil)
	op := OperatorIdentity{UserID: "ada", Org: "cardorg", Roles: []string{"operator"}}

	start, err := c.Commit(t.Context(), "cardorg", op, CommitRequest{
		TaskPath: "session/start",
		Value:    map[string]any{},
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := start.JobInstanceID

	first, err := c.Commit(t.Context(), "cardorg", op, CommitRequest{
		TaskPath:      "session/repeat",
		JobInstanceID: sid,
		Value:         map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := first.Worklist.Jobs[0].Instances[0].Tasks["repeat"]; got != TaskStatusActionable {
		t.Fatalf("after first many Commit status = %q, want actionable", got)
	}
	if got := first.Worklist.Jobs[0].Instances[0].Tasks["start"]; got != TaskStatusDone {
		t.Fatalf("once start status = %q, want done", got)
	}
	subsAfterFirst := len(first.Domain.Submissions)

	second, err := c.Commit(t.Context(), "cardorg", op, CommitRequest{
		TaskPath:      "session/repeat",
		JobInstanceID: sid,
		Value:         map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := second.Worklist.Jobs[0].Instances[0].Tasks["repeat"]; got != TaskStatusActionable {
		t.Fatalf("after second many Commit status = %q, want actionable", got)
	}
	if len(second.Domain.Submissions) != subsAfterFirst+1 {
		t.Fatalf("submissions = %d, want %d (append)", len(second.Domain.Submissions), subsAfterFirst+1)
	}
}

func TestSubmitOnceBecomesDoneAndSecondCommitDenied(t *testing.T) {
	c := submitCardinalityCommitter(t, nil)
	op := OperatorIdentity{UserID: "ada", Org: "cardorg", Roles: []string{"operator"}}

	start, err := c.Commit(t.Context(), "cardorg", op, CommitRequest{
		TaskPath: "session/start",
		Value:    map[string]any{},
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	sid := start.JobInstanceID
	if got := start.Worklist.Jobs[0].Instances[0].Tasks["start"]; got != TaskStatusDone {
		t.Fatalf("once start status = %q, want done", got)
	}

	_, err = c.Commit(t.Context(), "cardorg", op, CommitRequest{
		TaskPath:      "session/start",
		JobInstanceID: sid,
		Value:         map[string]any{},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("second once Commit must deny: %v", err)
	}
}

func TestProjectWorklistSubmissionCounts(t *testing.T) {
	cfg := submitCardinalityConfig()
	op := OperatorIdentity{UserID: "ada", Org: "cardorg", Roles: []string{"operator"}}
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "session", ID: "S1", Label: "Session S1"}}

	wl := ProjectWorklist(cfg, domain, op)
	inst := wl.Jobs[0].Instances[0]
	if len(inst.SubmissionCounts) != 0 {
		t.Fatalf("0 submissions: counts = %#v, want empty/nil", inst.SubmissionCounts)
	}

	domain.Submissions = []Submission{
		{JobID: "session", TaskID: "start", JobInstanceID: "S1"},
	}
	wl = ProjectWorklist(cfg, domain, op)
	inst = wl.Jobs[0].Instances[0]
	if inst.SubmissionCounts["start"] != 1 {
		t.Fatalf("1 once: start count = %d, want 1 (%#v)", inst.SubmissionCounts["start"], inst.SubmissionCounts)
	}
	if _, has := inst.SubmissionCounts["repeat"]; has {
		t.Fatalf("1 once: repeat must be absent, got %#v", inst.SubmissionCounts)
	}

	domain.Submissions = append(domain.Submissions,
		Submission{JobID: "session", TaskID: "repeat", JobInstanceID: "S1"},
		Submission{JobID: "session", TaskID: "repeat", JobInstanceID: "S1"},
		Submission{JobID: "session", TaskID: "repeat", JobInstanceID: "S1"},
	)
	wl = ProjectWorklist(cfg, domain, op)
	inst = wl.Jobs[0].Instances[0]
	if inst.SubmissionCounts["start"] != 1 || inst.SubmissionCounts["repeat"] != 3 {
		t.Fatalf("N many: counts = %#v, want start=1 repeat=3", inst.SubmissionCounts)
	}

	// Nested group leaf: same taskId under group must not inflate flat counts.
	domain.Submissions = append(domain.Submissions, Submission{
		JobID: "session", TaskID: "repeat", JobInstanceID: "S1",
		GroupID: "g", GroupInstanceID: "g1",
	})
	wl = ProjectWorklist(cfg, domain, op)
	inst = wl.Jobs[0].Instances[0]
	if inst.SubmissionCounts["repeat"] != 3 {
		t.Fatalf("nested must not count on flat: %#v", inst.SubmissionCounts)
	}
}

func TestProjectWorklistGroupSubmissionCounts(t *testing.T) {
	lots := TaskConfig{
		"id":         "lots",
		"kind":       "group",
		"repeatable": true,
		"tasks": []TaskConfig{
			{"id": "register", "roles": []any{"operator"}, "schema": map[string]any{"title": "Register"}},
		},
	}
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"intake"},
			Values: map[string]any{
				"intake": Job{
					Repeatable: true,
					Tasks: []TaskConfig{
						{"id": "truckArrive", "roles": []any{"operator"}, "schema": map[string]any{"title": "Arrive"}},
						lots,
					},
				},
			},
		},
	}
	op := OperatorIdentity{UserID: "ada", Org: "test", Roles: []string{"operator"}}
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "intake", ID: "T1", Label: "truck"}}
	domain.GroupInstances = []GroupInstance{{JobID: "intake", JobInstanceID: "T1", GroupID: "lots", ID: "L1"}}
	domain.Submissions = []Submission{sub("intake", "truckArrive", "T1")}

	wl := ProjectWorklist(cfg, domain, op)
	g := wl.Jobs[0].Instances[0].Groups["lots"]
	if len(g.Instances) != 1 || len(g.Instances[0].SubmissionCounts) != 0 {
		t.Fatalf("0 group child: %#v", g.Instances)
	}

	domain.Submissions = append(domain.Submissions, Submission{
		JobID: "intake", TaskID: "register", JobInstanceID: "T1",
		GroupID: "lots", GroupInstanceID: "L1",
	})
	wl = ProjectWorklist(cfg, domain, op)
	g = wl.Jobs[0].Instances[0].Groups["lots"]
	if g.Instances[0].SubmissionCounts["register"] != 1 {
		t.Fatalf("1 group child: %#v", g.Instances[0].SubmissionCounts)
	}

	domain.Submissions = append(domain.Submissions,
		Submission{JobID: "intake", TaskID: "register", JobInstanceID: "T1", GroupID: "lots", GroupInstanceID: "L1"},
		Submission{JobID: "intake", TaskID: "register", JobInstanceID: "T1", GroupID: "lots", GroupInstanceID: "L1"},
	)
	wl = ProjectWorklist(cfg, domain, op)
	g = wl.Jobs[0].Instances[0].Groups["lots"]
	if g.Instances[0].SubmissionCounts["register"] != 3 {
		t.Fatalf("N group child: %#v", g.Instances[0].SubmissionCounts)
	}
	if flat := wl.Jobs[0].Instances[0].SubmissionCounts["register"]; flat != 0 {
		t.Fatalf("nested must not appear on flat counts: %#v", wl.Jobs[0].Instances[0].SubmissionCounts)
	}
}

func TestRemainingWorkIgnoresSubmitMany(t *testing.T) {
	cfg := submitCardinalityConfig()
	op := OperatorIdentity{UserID: "ada", Org: "cardorg", Roles: []string{"operator"}}
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "session", ID: "S1", Label: "Session S1"}}
	domain.Submissions = []Submission{
		{JobID: "session", TaskID: "start", JobInstanceID: "S1"},
		{JobID: "session", TaskID: "repeat", JobInstanceID: "S1"},
		{JobID: "session", TaskID: "finish", JobInstanceID: "S1"},
	}
	wl := ProjectWorklist(cfg, domain, op)
	inst := wl.Jobs[0].Instances[0]
	if inst.Tasks["start"] != TaskStatusDone || inst.Tasks["finish"] != TaskStatusDone {
		t.Fatalf("once statuses = %#v", inst.Tasks)
	}
	if inst.Tasks["repeat"] != TaskStatusActionable {
		t.Fatalf("many status = %q, want actionable", inst.Tasks["repeat"])
	}
	if worklistInstanceHasRemainingWork(cfg, wl, "session", "S1") {
		t.Fatal("many must not block finished-band remaining-work")
	}

	domain.Submissions = domain.Submissions[:2] // finish not submitted
	wl = ProjectWorklist(cfg, domain, op)
	if !worklistInstanceHasRemainingWork(cfg, wl, "session", "S1") {
		t.Fatal("missing once finish must still count as remaining work")
	}
}
