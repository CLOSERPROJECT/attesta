package living

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLoadFeragameConfigFixtureRenamesJobsAndTasks(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	roles := rolesAllowedForTask(cfg, "intake", "register")
	if len(roles) != 1 || roles[0] != "administration" {
		t.Fatalf("register rolesAllowed = %#v, want [administration]", roles)
	}

	if cfg.ID != "feragame-config" {
		t.Fatalf("id = %q", cfg.ID)
	}
	if cfg.Org != "feragame" {
		t.Fatalf("org = %q", cfg.Org)
	}
	if cfg.Version != DefaultFixtureVersion {
		t.Fatalf("version = %q", cfg.Version)
	}
	if cfg.Catalogs == nil {
		t.Fatal("catalogs missing")
	}
	for _, key := range []string{"tareKinds", "materials", "clients"} {
		if cfg.Catalogs[key] == nil {
			t.Fatalf("catalogs.%s missing", key)
		}
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
	if !intake.Repeatable || intake.StartLabel != "Nuovo camion" || intake.Title != "Camion" {
		t.Fatalf("intake = %#v", intake)
	}
	wantIntakeTasks := []string{"register", "codiceFir", "palletWeight", "weightCheck"}
	if got := OrderedTaskIDs(intake.Tasks); strings.Join(got, ",") != strings.Join(wantIntakeTasks, ",") {
		t.Fatalf("intake tasks = %v", got)
	}

	lineAny, _ := cfg.Jobs.Get("line")
	line := lineAny.(Job)
	if line.Title != "Linea" {
		t.Fatalf("line title = %q", line.Title)
	}
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
	if got := OrderedTaskIDs(line.Tasks); strings.Join(got, ",") != strings.Join(wantLineTasks, ",") {
		t.Fatalf("line tasks = %v", got)
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
	if got.ID != "feragame-config" {
		t.Fatalf("got id = %q", got.ID)
	}
}
