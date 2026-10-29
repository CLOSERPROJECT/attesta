package living

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLoadTruckLoopFixtureRenamesJobsAndTasks(t *testing.T) {
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		t.Fatal(err)
	}
	roles := rolesAllowedForTask(cfg, "intake", "register")
	if len(roles) != 1 || roles[0] != "administration" {
		t.Fatalf("register rolesAllowed = %#v, want [administration]", roles)
	}

	if cfg.ID != "truck-loop" {
		t.Fatalf("id = %q", cfg.ID)
	}
	if cfg.Org != "feragame" {
		t.Fatalf("org = %q", cfg.Org)
	}
	if cfg.Version != DefaultFixtureVersion {
		t.Fatalf("version = %q", cfg.Version)
	}
	if cfg.Jobs.Len() != 2 {
		t.Fatalf("jobs len = %d, keys=%v", cfg.Jobs.Len(), cfg.Jobs.Keys)
	}
	if got := cfg.Jobs.Keys; len(got) < 2 || got[0] != "intake" || got[1] != "line" {
		t.Fatalf("job order = %v", got)
	}

	intakeAny, ok := cfg.Jobs.Get("intake")
	if !ok {
		t.Fatal("missing intake")
	}
	intake, ok := intakeAny.(Job)
	if !ok {
		t.Fatalf("intake type %T", intakeAny)
	}
	if !intake.Repeatable || intake.StartLabel != "Nuovo camion" {
		t.Fatalf("intake = %#v", intake)
	}
	wantIntakeTasks := []string{"register", "codiceFir", "palletWeight", "weightCheck"}
	if strings.Join(intake.Tasks.Keys, ",") != strings.Join(wantIntakeTasks, ",") {
		t.Fatalf("intake tasks = %v", intake.Tasks.Keys)
	}

	lineAny, _ := cfg.Jobs.Get("line")
	line := lineAny.(Job)
	if line.StartsWhen == nil {
		t.Fatal("line startsWhen missing")
	}
	if line.StartsWhen.Task.JobID != "intake" || line.StartsWhen.Task.TaskID != "palletWeight" {
		t.Fatalf("startsWhen.task = %#v", line.StartsWhen.Task)
	}
	if line.StartsWhen.Quantifier != "some" {
		t.Fatalf("quantifier = %q", line.StartsWhen.Quantifier)
	}
	wantLineTasks := []string{"shift", "lineLoad", "containerReg", "loadedExtracted"}
	if strings.Join(line.Tasks.Keys, ",") != strings.Join(wantLineTasks, ",") {
		t.Fatalf("line tasks = %v", line.Tasks.Keys)
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, rejected := range []string{`"steps"`, `"substeps"`, `"stepId"`, `"substepId"`, `/api/runs`, `RunStore`, `LoadRunPort`} {
		if strings.Contains(body, rejected) {
			t.Fatalf("serialized config still contains rejected token %q", rejected)
		}
	}
	if !strings.Contains(body, `"jobs"`) || !strings.Contains(body, `"tasks"`) {
		t.Fatalf("serialized config missing jobs/tasks: %s", body[:min(200, len(body))])
	}
}

func TestSeedFixtureConfigStoreSetsActiveVersion(t *testing.T) {
	store := NewMemoryConfigStore()
	cfg, err := SeedFixtureConfigStore(store)
	if err != nil {
		t.Fatal(err)
	}
	version, err := store.ActiveVersion(t.Context(), cfg.Org)
	if err != nil {
		t.Fatal(err)
	}
	if version != DefaultFixtureVersion {
		t.Fatalf("active = %q", version)
	}
	got, err := store.Get(t.Context(), cfg.Org, version)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "truck-loop" {
		t.Fatalf("got id = %q", got.ID)
	}
}
