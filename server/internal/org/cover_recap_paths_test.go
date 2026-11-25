package org

import (
	"errors"
	"strings"
	"testing"
)

func TestTaskRecapParseEdges(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		task    TaskConfig
		wantNil bool
		wantErr string
	}{
		{name: "nil task", task: nil, wantNil: true},
		{name: "missing recap", task: TaskConfig{"id": "t"}, wantNil: true},
		{name: "nil recap", task: TaskConfig{"id": "t", "recap": nil}, wantNil: true},
		{name: "empty object", task: TaskConfig{"id": "t", "recap": map[string]any{}}, wantNil: true},
		{name: "not object", task: TaskConfig{"id": "t", "recap": "x"}, wantErr: "recap must be an object"},
		{name: "columns not list", task: TaskConfig{"id": "t", "recap": map[string]any{
			"entity": "lot", "columns": "x",
		}}, wantErr: "recap.columns must be a list"},
		{name: "column not object", task: TaskConfig{"id": "t", "recap": map[string]any{
			"entity": "lot", "columns": []any{"x"},
		}}, wantErr: "must be an object"},
		{name: "empty field", task: TaskConfig{"id": "t", "recap": map[string]any{
			"entity": "lot", "columns": []any{map[string]any{"field": "  "}},
		}}, wantErr: "field is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := taskRecap(tc.task)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err=%v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantNil && got != nil {
				t.Fatalf("got %#v, want nil", got)
			}
		})
	}
}

func TestValidateRecapConfigEdges(t *testing.T) {
	t.Parallel()

	if err := ValidateRecapConfig(OrgConfig{}); err != nil {
		t.Fatalf("nil jobs: %v", err)
	}

	cfg := validStreamCfg()
	cfg.Jobs.Values["intake"] = "not-a-job"
	if err := ValidateRecapConfig(cfg); err != nil {
		t.Fatalf("skip non-job: %v", err)
	}

	cfg = validStreamCfg()
	intake := cfg.Jobs.Values["intake"].(Job)
	intake.Tasks = []TaskConfig{{"id": "arrive", "recap": "bad"}}
	cfg.Jobs.Values["intake"] = intake
	if err := ValidateRecapConfig(cfg); err == nil || !strings.Contains(err.Error(), "recap must be an object") {
		t.Fatalf("malformed recap = %v", err)
	}

	cfg = validStreamCfg()
	intake = cfg.Jobs.Values["intake"].(Job)
	intake.Tasks = []TaskConfig{{
		"id": "arrive",
		"recap": map[string]any{
			"entity":  "",
			"columns": []any{map[string]any{"field": "truckId"}},
		},
	}}
	cfg.Jobs.Values["intake"] = intake
	if err := ValidateRecapConfig(cfg); err == nil || !strings.Contains(err.Error(), "recap.entity is required") {
		t.Fatalf("empty entity = %v", err)
	}

	cfg = validStreamCfg()
	cfg.Domain = map[string]any{}
	intake = cfg.Jobs.Values["intake"].(Job)
	intake.Tasks = []TaskConfig{{
		"id": "arrive",
		"recap": map[string]any{
			"entity":  "lot",
			"columns": []any{map[string]any{"field": "truckId"}},
		},
	}}
	cfg.Jobs.Values["intake"] = intake
	if err := ValidateRecapConfig(cfg); err == nil || !strings.Contains(err.Error(), "not in domain.entities") {
		t.Fatalf("kinds nil = %v", err)
	}

	cfg = validStreamCfg()
	cfg.Domain = map[string]any{
		"entities": map[string]any{
			"lot": map[string]any{"fields": "not-a-list"},
		},
	}
	if err := ValidateRecapConfig(cfg); err == nil {
		t.Fatal("want EntityFieldsFromConfig error")
	}

	streamFail := validStreamCfg()
	streamFail.Stream = nil
	if err := ValidateOrgConfig(streamFail); err == nil || !strings.Contains(err.Error(), "stream") {
		t.Fatalf("ValidateOrgConfig stream = %v", err)
	}
}

func TestProjectTaskRecapErrorEdges(t *testing.T) {
	t.Parallel()
	addr := SubmissionAddress{JobID: "j", TaskID: "t", JobInstanceID: "i"}
	domain := EmptyDomain()
	domain.Submissions = []Submission{{
		JobID: "j", TaskID: "t", JobInstanceID: "i", Timestamp: "t1", UserID: "ada",
	}}

	cases := []struct {
		name    string
		cfg     OrgConfig
		addr    SubmissionAddress
		wantErr string
	}{
		{name: "invalid addr", addr: SubmissionAddress{}, wantErr: "submission address required"},
		{
			name:    "unknown job",
			cfg:     OrgConfig{Jobs: OrderedObject{Keys: []string{}, Values: map[string]any{}}},
			addr:    addr,
			wantErr: `unknown job "j"`,
		},
		{
			name: "not a job",
			cfg: OrgConfig{Jobs: OrderedObject{
				Keys: []string{"j"}, Values: map[string]any{"j": "nope"},
			}},
			addr:    addr,
			wantErr: "is not a Job",
		},
		{
			name: "unknown task",
			cfg: OrgConfig{Jobs: OrderedObject{
				Keys:   []string{"j"},
				Values: map[string]any{"j": Job{Tasks: []TaskConfig{{"id": "other"}}}},
			}},
			addr:    addr,
			wantErr: `unknown task "t"`,
		},
		{
			name: "bad recap",
			cfg: OrgConfig{Jobs: OrderedObject{
				Keys: []string{"j"},
				Values: map[string]any{"j": Job{Tasks: []TaskConfig{{
					"id": "t", "recap": "x",
				}}}},
			}},
			addr:    addr,
			wantErr: "recap must be an object",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ProjectTaskRecap(tc.cfg, domain, tc.addr)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err=%v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestProjectTaskRecapSkipsEmptyTouchIDAndEmptyFactKey(t *testing.T) {
	t.Parallel()
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"j"},
			Values: map[string]any{
				"j": Job{Tasks: []TaskConfig{{
					"id": "t",
					"recap": map[string]any{
						"entity":  "lot",
						"columns": []any{map[string]any{"field": "client"}},
					},
				}}},
			},
		},
	}
	domain := domainFromKinds(map[string]EntityMap{
		"lot": {"L1": {"client": "CO"}},
	})
	domain.Submissions = []Submission{{
		JobID: "j", TaskID: "t", JobInstanceID: "i",
		Timestamp: "t1", UserID: "ada",
		EntitiesTouched: []EntityTouch{
			{Op: "mint", Entity: "lot", ID: "  "},
			{Op: "mint", Entity: "other", ID: "X"},
			{Op: "mint", Entity: "lot", ID: "L1"},
		},
	}}
	table, err := ProjectTaskRecap(cfg, domain, SubmissionAddress{JobID: "j", TaskID: "t", JobInstanceID: "i"})
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 1 || table.Rows[0].Key != "L1" {
		t.Fatalf("rows = %#v", table.Rows)
	}

	flat := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"j"},
			Values: map[string]any{"j": Job{Tasks: []TaskConfig{{"id": "t"}}}},
		},
	}
	domain = EmptyDomain()
	domain.Submissions = []Submission{
		{
			JobID: "j", TaskID: "t", JobInstanceID: "i",
			Timestamp: "t1", UserID: "ada",
			Facts: map[string]any{"": 1, "a": 2},
		},
		{
			JobID: "j", TaskID: "t", JobInstanceID: "i",
			Timestamp: "t2", UserID: "bob",
			Facts: map[string]any{"a": 3, "b": 4},
		},
	}
	table, err = ProjectTaskRecap(flat, domain, SubmissionAddress{JobID: "j", TaskID: "t", JobInstanceID: "i"})
	if err != nil {
		t.Fatal(err)
	}
	if table.Mode != taskRecapModeSubmission || len(table.Columns) != 2 {
		t.Fatalf("columns = %#v", table.Columns)
	}
	if table.Columns[0].Key != "a" || table.Columns[1].Key != "b" {
		t.Fatalf("union = %#v", table.Columns)
	}
}

func TestLoadTaskRecapConfigAndLinkErrors(t *testing.T) {
	c, domains, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID := commitTruckArrive(t, c)
	addr := SubmissionAddress{JobID: "intake", JobInstanceID: truckID, TaskID: "truckArrive"}

	if _, err := (&DomainLoader{
		Domains: domains,
		Configs: activeVersionErrStore{ConfigStore: loader.Configs, err: errors.New("active boom")},
	}).LoadTaskRecap(t.Context(), FeragameScope(), addr); err == nil || !strings.Contains(err.Error(), "active boom") {
		t.Fatalf("ActiveVersion = %v", err)
	}

	if _, err := (&DomainLoader{
		Domains: domains,
		Configs: getConfigErrStore{ConfigStore: loader.Configs, err: errors.New("get cfg boom")},
	}).LoadTaskRecap(t.Context(), FeragameScope(), addr); err == nil || !strings.Contains(err.Error(), "get cfg boom") {
		t.Fatalf("Configs.Get = %v", err)
	}

	if _, err := (&DomainLoader{
		Domains: domains, Configs: loader.Configs,
		LinkStore: &errLinkStore{err: errors.New("links boom")},
	}).LoadTaskRecap(t.Context(), FeragameScope(), addr); err == nil || !strings.Contains(err.Error(), "links boom") {
		t.Fatalf("LinkStore = %v", err)
	}

	// ProjectTaskRecap failure after Domains.Get: Config has matching Job key but wrong type.
	badJobs := NewMemoryConfigStore()
	version := "v-bad"
	badCfg := OrgConfig{
		Version: version,
		Jobs: OrderedObject{
			Keys:   []string{"intake"},
			Values: map[string]any{"intake": "not-job"},
		},
	}
	if err := badJobs.Put(t.Context(), FeragameScope(), version, badCfg); err != nil {
		t.Fatal(err)
	}
	if err := badJobs.SetActiveVersion(t.Context(), FeragameScope(), version); err != nil {
		t.Fatal(err)
	}
	if _, err := (&DomainLoader{Domains: domains, Configs: badJobs}).LoadTaskRecap(
		t.Context(), FeragameScope(), addr,
	); err == nil || !strings.Contains(err.Error(), "is not a Job") {
		t.Fatalf("ProjectTaskRecap = %v", err)
	}
}
