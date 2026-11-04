package living

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadFeragameConfigFixtureRenamesJobsAndTasks(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	roles := rolesAllowedForTask(cfg, "intake", "truckArrive")
	if len(roles) != 1 || roles[0] != "administration" {
		t.Fatalf("truckArrive rolesAllowed = %#v, want [administration]", roles)
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
	for _, key := range []string{"tareKinds", "materials", "clients", "turni"} {
		if cfg.Catalogs[key] == nil {
			t.Fatalf("catalogs.%s missing", key)
		}
	}
	entities, _ := cfg.Domain["entities"].(map[string]any)
	if entities["truck"] == nil || entities["load"] == nil {
		t.Fatalf("domain.entities missing truck/load: %#v", entities)
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
	if !intake.Repeatable || intake.Title != "Ingresso" || intake.StartLabel != "Nuovo camion" || intake.Color != "#ea580c" {
		t.Fatalf("intake = %#v", intake)
	}
	wantIntakeTasks := []string{"truckArrive", "lots", "weightCheck"}
	if got := OrderedTaskIDs(intake.Tasks); strings.Join(got, ",") != strings.Join(wantIntakeTasks, ",") {
		t.Fatalf("intake tasks = %v", got)
	}
	lots, ok := TaskByID(intake.Tasks, "lots")
	if !ok || !lots.IsGroup() {
		t.Fatalf("lots group = %#v", lots)
	}

	lineAny, _ := cfg.Jobs.Get("line")
	line := lineAny.(Job)
	if line.Title != "Lavorazione" {
		t.Fatalf("line.Title = %q, want Lavorazione", line.Title)
	}
	if line.Color != "#0284c7" {
		t.Fatalf("line.Color = %q, want #0284c7", line.Color)
	}
	if line.StartsWhen != nil {
		t.Fatalf("line must not have startsWhen, got %#v", line.StartsWhen)
	}
	wantLineTasks := []string{"openTurno", "palletWeight", "lineLoad", "containerReg"}
	if got := OrderedTaskIDs(line.Tasks); strings.Join(got, ",") != strings.Join(wantLineTasks, ",") {
		t.Fatalf("line tasks = %v", got)
	}
	pw, ok := TaskByID(line.Tasks, "palletWeight")
	if !ok || !pw.IsGroup() {
		t.Fatalf("palletWeight group = %#v", pw)
	}
	lineLoad, ok := TaskByID(line.Tasks, "lineLoad")
	if !ok {
		t.Fatal("missing lineLoad")
	}
	if got, _ := lineLoad["unlockFrom"].(string); got != "openTurno" {
		t.Fatalf("lineLoad.unlockFrom = %#v, want openTurno", lineLoad["unlockFrom"])
	}
	readyWhen, _ := lineLoad["readyWhen"].(string)
	if !strings.Contains(readyWhen, "$exists(weight)") || !strings.Contains(readyWhen, "progress = 'done'") {
		t.Fatalf("lineLoad.readyWhen = %q, want locked Domain leftover expr", readyWhen)
	}
	if _, has := pw["unlockFrom"]; has {
		t.Fatalf("palletWeight must omit unlockFrom (linear), got %#v", pw["unlockFrom"])
	}
	containerReg, ok := TaskByID(line.Tasks, "containerReg")
	if !ok || !containerReg.IsGroup() {
		t.Fatalf("containerReg group = %#v", containerReg)
	}
	if got, _ := containerReg["unlockFrom"].(string); got != "openTurno" {
		t.Fatalf("containerReg.unlockFrom = %#v, want openTurno", containerReg["unlockFrom"])
	}
	if _, has := containerReg["readyWhen"]; has {
		t.Fatalf("containerReg must omit readyWhen, got %#v", containerReg["readyWhen"])
	}
	if containerReg["instanceLabelEntity"] != "load" {
		t.Fatalf("containerReg.instanceLabelEntity = %#v, want load", containerReg["instanceLabelEntity"])
	}
	if containerReg["closeRequiresChildSubmission"] != true {
		t.Fatalf("containerReg.closeRequiresChildSubmission = %#v", containerReg["closeRequiresChildSubmission"])
	}
	reg, ok := TaskByID(containerReg.ChildTasks(), "register")
	if !ok || reg.IsGroup() {
		t.Fatalf("containerReg.register leaf = %#v", reg)
	}
	reads, _ := reg["reads"].(map[string]any)
	loadsRead, _ := reads["loads"].(map[string]any)
	where, _ := loadsRead["where"].(string)
	if !strings.Contains(where, "fullyProcessed") {
		t.Fatalf("register.reads.loads.where = %#v, want fullyProcessed filter", loadsRead["where"])
	}
	if _, has := TaskByID(line.Tasks, "loadedExtracted"); has {
		t.Fatal("loadedExtracted must not be on the live line Task list")
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

// Fail closed: Unlock / readyWhen / Worklist engines must stay generic — no plant Task id branches.
func TestEngineSourcesOmitPlantTaskIDBranches(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	for _, name := range []string{"unlock.go", "ready_when.go", "worklist.go", "tasks.go"} {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		src := string(body)
		for _, token := range []string{`"lineLoad"`, `"feragame"`, `"openTurno"`, `"palletWeight"`} {
			if strings.Contains(src, token) {
				t.Fatalf("%s must not hard-code plant token %s", name, token)
			}
		}
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
