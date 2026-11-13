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
	if entities["workingDay"] == nil || entities["shift"] == nil {
		t.Fatalf("domain.entities missing workingDay/shift: %#v", entities)
	}
	if entities["truck"] == nil || entities["lot"] == nil {
		t.Fatalf("domain.entities missing truck/lot: %#v", entities)
	}
	truckFields, _ := entities["truck"].(map[string]any)
	fields, _ := truckFields["fields"].([]any)
	hasWD, hasDate, hasTurno := false, false, false
	for _, f := range fields {
		s, _ := f.(string)
		switch s {
		case "workingDayId":
			hasWD = true
		case "date":
			hasDate = true
		case "turno":
			hasTurno = true
		}
	}
	if !hasWD || hasDate || hasTurno {
		t.Fatalf("truck.fields = %#v, want workingDayId without date/turno", fields)
	}
	if cfg.Jobs.Len() != 4 {
		t.Fatalf("jobs len = %d, keys=%v", cfg.Jobs.Len(), cfg.Jobs.Keys)
	}
	if got := cfg.Jobs.Keys; len(got) < 4 || got[0] != "workingDay" || got[1] != "intake" || got[2] != "line" || got[3] != "passport" {
		t.Fatalf("job order = %v", got)
	}
	if cfg.Passport == nil || cfg.Passport.Subject.Entity != "lot" {
		t.Fatalf("passport.subject = %#v", cfg.Passport)
	}
	if strings.TrimSpace(cfg.Passport.Snapshot.Expr) == "" {
		t.Fatal("passport.snapshot.expr missing")
	}
	passportJob, ok := cfg.Jobs.Get("passport")
	if !ok {
		t.Fatal("missing passport Job")
	}
	pj := passportJob.(Job)
	if !pj.Repeatable || pj.Title != "Verifica dati e DPP" || pj.StartLabel != "Emetti DPP" || pj.InstanceID != "$.lotId" {
		t.Fatalf("passport Job = %#v", pj)
	}
	wantPassportTasks := []string{"select", "lock"}
	if got := OrderedTaskIDs(pj.Tasks); strings.Join(got, ",") != strings.Join(wantPassportTasks, ",") {
		t.Fatalf("passport tasks = %v", got)
	}
	lock, ok := TaskByID(pj.Tasks, "lock")
	if !ok || !taskOnSubmitPassport(map[string]any(lock)) {
		t.Fatalf("lock onSubmit.passport = %#v", lock)
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
	truckArrive, okArrive := TaskByID(intake.Tasks, "truckArrive")
	if !okArrive {
		t.Fatal("truckArrive missing")
	}
	if rw, _ := truckArrive["readyWhen"].(string); !strings.Contains(rw, "checkedTruckWeight") {
		t.Fatalf("truckArrive.readyWhen = %#v, want prior-truck weightCheck gate", truckArrive["readyWhen"])
	}
	wantIntakeTasks := []string{"truckArrive", "register", "weightCheck"}
	if got := OrderedTaskIDs(intake.Tasks); strings.Join(got, ",") != strings.Join(wantIntakeTasks, ",") {
		t.Fatalf("intake tasks = %v", got)
	}
	if _, ok := TaskByID(intake.Tasks, "lots"); ok {
		t.Fatal("intake must not author lots Task group")
	}
	register, ok := TaskByID(intake.Tasks, "register")
	if !ok || register.IsGroup() || !taskSubmitMany(register) {
		t.Fatalf("register leaf many = %#v", register)
	}
	if taskCommitNavigation(map[string]any(register)) != commitNavStay {
		t.Fatalf("register onCommit.navigation = %#v, want stay", register["onCommit"])
	}
	weightCheck, ok := TaskByID(intake.Tasks, "weightCheck")
	if !ok || weightCheck.IsGroup() {
		t.Fatalf("weightCheck = %#v", weightCheck)
	}
	if rw, _ := weightCheck["readyWhen"].(string); !strings.Contains(rw, "lot") {
		t.Fatalf("weightCheck.readyWhen = %#v, want ≥1 lot gate", weightCheck["readyWhen"])
	}
	if rw, _ := register["readyWhen"].(string); !strings.Contains(rw, "weightCheck") {
		t.Fatalf("register.readyWhen = %#v, want weightCheck Submission gate", register["readyWhen"])
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
	wantLineTasks := []string{"openTurno", "startLot", "weigh", "markProgress", "containerReg", "closeLot", "closeTurno"}
	if got := OrderedTaskIDs(line.Tasks); strings.Join(got, ",") != strings.Join(wantLineTasks, ",") {
		t.Fatalf("line tasks = %v", got)
	}
	for _, id := range wantLineTasks {
		task, ok := TaskByID(line.Tasks, id)
		if !ok || task.IsGroup() {
			t.Fatalf("line.%s must be a leaf, got %#v", id, task)
		}
		if _, has := task["unlockFrom"]; has {
			t.Fatalf("line.%s must omit unlockFrom, got %#v", id, task["unlockFrom"])
		}
	}
	for _, id := range []string{"startLot", "weigh", "markProgress", "containerReg", "closeLot"} {
		task, _ := TaskByID(line.Tasks, id)
		if !taskSubmitMany(task) {
			t.Fatalf("line.%s submit = %#v, want many", id, task["submit"])
		}
	}
	startLot, _ := TaskByID(line.Tasks, "startLot")
	startLotRoles, _ := startLot["roles"].([]any)
	if len(startLotRoles) != 1 || startLotRoles[0] != "logistic" {
		t.Fatalf("line/startLot roles = %#v, want [logistic]", startLot["roles"])
	}
	wdAny, ok := cfg.Jobs.Get("workingDay")
	if !ok {
		t.Fatal("missing workingDay")
	}
	wd := wdAny.(Job)
	wantWDTasks := []string{"open", "close"}
	if got := OrderedTaskIDs(wd.Tasks); strings.Join(got, ",") != strings.Join(wantWDTasks, ",") {
		t.Fatalf("workingDay tasks = %v", got)
	}
	wdOpen, _ := TaskByID(wd.Tasks, "open")
	if taskCommitNavigation(map[string]any(wdOpen)) != commitNavHome {
		t.Fatalf("workingDay/open onCommit.navigation = %#v, want home", wdOpen["onCommit"])
	}
	wdClose, _ := TaskByID(wd.Tasks, "close")
	if taskAdvanceTarget(map[string]any(wdClose)) {
		t.Fatalf("workingDay/close advanceTarget = true, want false")
	}
	if _, has := wdClose["unlockFrom"]; has {
		t.Fatalf("workingDay/close must omit unlockFrom, got %#v", wdClose["unlockFrom"])
	}
	openTurno, _ := TaskByID(line.Tasks, "openTurno")
	if taskCommitNavigation(map[string]any(openTurno)) != commitNavHome {
		t.Fatalf("line/openTurno onCommit.navigation = %#v, want home", openTurno["onCommit"])
	}
	openRoles, _ := openTurno["roles"].([]any)
	if len(openRoles) != 2 || openRoles[0] != "line" || openRoles[1] != "logistic" {
		t.Fatalf("line/openTurno roles = %#v, want [line, logistic]", openTurno["roles"])
	}
	closeTurno, _ := TaskByID(line.Tasks, "closeTurno")
	if taskAdvanceTarget(map[string]any(closeTurno)) {
		t.Fatalf("line/closeTurno advanceTarget = true, want false")
	}
	closeRoles, _ := closeTurno["roles"].([]any)
	if len(closeRoles) != 2 || closeRoles[0] != "line" || closeRoles[1] != "logistic" {
		t.Fatalf("line/closeTurno roles = %#v, want [line, logistic]", closeTurno["roles"])
	}
	markProgress, _ := TaskByID(line.Tasks, "markProgress")
	readyWhen, _ := markProgress["readyWhen"].(string)
	if !strings.Contains(readyWhen, "$exists(weight)") {
		t.Fatalf("markProgress.readyWhen must weigh-gate: %q", readyWhen)
	}
	if !strings.Contains(readyWhen, `status = "open"`) {
		t.Fatalf("markProgress.readyWhen = %q, want open-lot gate", readyWhen)
	}
	containerReg, _ := TaskByID(line.Tasks, "containerReg")
	reads, _ := containerReg["reads"].(map[string]any)
	lotsRead, _ := reads["lots"].(map[string]any)
	where, _ := lotsRead["where"].(string)
	if where != `status = "open"` {
		t.Fatalf("containerReg.reads.lots.where = %#v, want open lot", lotsRead["where"])
	}
	transform, _ := containerReg["onSubmit"].(map[string]any)
	rawOps, _ := transform["transform"].([]any)
	for _, raw := range rawOps {
		op, _ := raw.(map[string]any)
		set, _ := op["set"].(map[string]any)
		if status, ok := set["status"].(string); ok && strings.Contains(status, "fullyProcessed") {
			t.Fatal("containerReg must not write fullyProcessed")
		}
	}
	closeLot, _ := TaskByID(line.Tasks, "closeLot")
	closeRW, _ := closeLot["readyWhen"].(string)
	if !strings.Contains(closeRW, "$exists(weight)") || !strings.Contains(closeRW, `progress = "done"`) {
		t.Fatalf("closeLot.readyWhen = %q, want weighed+done gate", closeRW)
	}
	if _, has := TaskByID(line.Tasks, "loadedExtracted"); has {
		t.Fatal("loadedExtracted must not be on the live line Task list")
	}
	if _, has := TaskByID(line.Tasks, "palletWeight"); has {
		t.Fatal("palletWeight group must not remain on line")
	}
	if _, has := TaskByID(line.Tasks, "lineLoad"); has {
		t.Fatal("lineLoad group must not remain on line")
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
		for _, token := range []string{`"markProgress"`, `"feragame"`, `"openTurno"`, `"startLot"`, `"palletWeight"`} {
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
