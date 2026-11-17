package org

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
	for _, key := range []string{"tareKinds", "materials", "clients"} {
		if cfg.Catalogs[key] == nil {
			t.Fatalf("catalogs.%s missing", key)
		}
	}
	if cfg.Catalogs["turni"] != nil {
		t.Fatal("catalogs.turni must be dropped")
	}
	entities, _ := cfg.Domain["entities"].(map[string]any)
	if entities["workingDay"] != nil || entities["shift"] != nil {
		t.Fatalf("domain.entities must omit workingDay/shift: %#v", entities)
	}
	if entities["truck"] == nil || entities["lot"] == nil {
		t.Fatalf("domain.entities missing truck/lot: %#v", entities)
	}
	truckFields, _ := entities["truck"].(map[string]any)
	fields, _ := truckFields["fields"].([]any)
	for _, f := range fields {
		s, _ := f.(string)
		switch s {
		case "workingDayId", "date", "turno":
			t.Fatalf("truck.fields must omit calendar stamps, got %#v", fields)
		}
	}
	lotFields, _ := entities["lot"].(map[string]any)
	lotFieldList, _ := lotFields["fields"].([]any)
	hasStatus := false
	for _, f := range lotFieldList {
		s, _ := f.(string)
		if s == "status" {
			hasStatus = true
		}
		if s == "parked" {
			t.Fatalf("lot.fields must not include parked: %#v", lotFieldList)
		}
	}
	if !hasStatus {
		t.Fatalf("lot.fields = %#v, want status", lotFieldList)
	}
	if cfg.Jobs.Len() != 3 {
		t.Fatalf("jobs len = %d, keys=%v", cfg.Jobs.Len(), cfg.Jobs.Keys)
	}
	if got := cfg.Jobs.Keys; len(got) != 3 || got[0] != "intake" || got[1] != "lot_processing" || got[2] != "passport" {
		t.Fatalf("job order = %v, want [intake lot_processing passport]", got)
	}
	if _, ok := cfg.Jobs.Get("workingDay"); ok {
		t.Fatal("workingDay Job must be dropped")
	}
	if _, ok := cfg.Jobs.Get("line"); ok {
		t.Fatal("line Job must be renamed to lot_processing")
	}
	if cfg.DPP == nil || cfg.DPP.Subject.Entity != "lot" {
		t.Fatalf("passport.subject = %#v", cfg.DPP)
	}
	if strings.TrimSpace(cfg.DPP.Snapshot.Expr) == "" {
		t.Fatal("passport.snapshot.expr missing")
	}
	snap := cfg.DPP.Snapshot.Expr
	for _, rejected := range []string{"workingDay", "shift", "Giornata", "Turno"} {
		if strings.Contains(snap, rejected) {
			t.Fatalf("passport.snapshot must omit %q", rejected)
		}
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
	if !ok || !taskOnSubmitDPP(map[string]any(lock)) {
		t.Fatalf("lock onSubmit.passport = %#v", lock)
	}
	for _, id := range wantPassportTasks {
		task, _ := TaskByID(pj.Tasks, id)
		roles, _ := task["roles"].([]any)
		if len(roles) != 1 || roles[0] != "administration" {
			t.Fatalf("passport/%s roles = %#v, want [administration]", id, task["roles"])
		}
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
	if !ok || !taskSubmitMany(register) {
		t.Fatalf("register leaf many = %#v", register)
	}
	if taskCommitNavigation(map[string]any(register)) != commitNavStay {
		t.Fatalf("register onCommit.navigation = %#v, want stay", register["onCommit"])
	}
	weightCheck, ok := TaskByID(intake.Tasks, "weightCheck")
	if !ok {
		t.Fatalf("weightCheck = %#v", weightCheck)
	}
	if rw, _ := weightCheck["readyWhen"].(string); !strings.Contains(rw, "lot") {
		t.Fatalf("weightCheck.readyWhen = %#v, want ≥1 lot gate", weightCheck["readyWhen"])
	}
	if rw, _ := weightCheck["readyWhen"].(string); !strings.Contains(rw, `taskId = "truckArrive"`) {
		t.Fatalf("weightCheck.readyWhen = %#v, want truckArrive Submission gate", weightCheck["readyWhen"])
	}
	if rw, _ := register["readyWhen"].(string); !strings.Contains(rw, "weightCheck") {
		t.Fatalf("register.readyWhen = %#v, want weightCheck Submission gate", register["readyWhen"])
	}
	if rw, _ := register["readyWhen"].(string); !strings.Contains(rw, `taskId = "truckArrive"`) {
		t.Fatalf("register.readyWhen = %#v, want truckArrive Submission gate", register["readyWhen"])
	}
	for _, id := range wantIntakeTasks {
		task, _ := TaskByID(intake.Tasks, id)
		roles, _ := task["roles"].([]any)
		if len(roles) != 1 || roles[0] != "administration" {
			t.Fatalf("intake/%s roles = %#v, want [administration]", id, task["roles"])
		}
	}

	lpAny, ok := cfg.Jobs.Get("lot_processing")
	if !ok {
		t.Fatal("missing lot_processing")
	}
	lp := lpAny.(Job)
	if lp.Title != "Lavorazione" {
		t.Fatalf("lot_processing.Title = %q, want Lavorazione", lp.Title)
	}
	if lp.StartLabel != "Apri lotto" {
		t.Fatalf("lot_processing.StartLabel = %q, want Apri lotto", lp.StartLabel)
	}
	if lp.Color != "#0284c7" {
		t.Fatalf("lot_processing.Color = %q, want #0284c7", lp.Color)
	}
	if lp.InstanceID != "$.lotId" {
		t.Fatalf("lot_processing.InstanceID = %q, want $.lotId", lp.InstanceID)
	}
	if lp.StartsWhen != nil {
		t.Fatalf("lot_processing must not have startsWhen, got %#v", lp.StartsWhen)
	}
	wantLPTasks := []string{"startLot", "weigh", "markProgress", "containerReg", "closeLot"}
	if got := OrderedTaskIDs(lp.Tasks); strings.Join(got, ",") != strings.Join(wantLPTasks, ",") {
		t.Fatalf("lot_processing tasks = %v", got)
	}
	for _, id := range []string{"openTurno", "closeTurno"} {
		if _, has := TaskByID(lp.Tasks, id); has {
			t.Fatalf("lot_processing must drop %s", id)
		}
	}
	for _, id := range wantLPTasks {
		task, ok := TaskByID(lp.Tasks, id)
		if !ok {
			t.Fatalf("lot_processing.%s must be a leaf, got %#v", id, task)
		}
		if _, has := task["unlockFrom"]; has {
			t.Fatalf("lot_processing.%s must omit unlockFrom, got %#v", id, task["unlockFrom"])
		}
	}
	for _, id := range []string{"weigh", "markProgress", "containerReg", "closeLot"} {
		task, _ := TaskByID(lp.Tasks, id)
		if !taskSubmitMany(task) {
			t.Fatalf("lot_processing.%s submit = %#v, want many", id, task["submit"])
		}
	}
	startLot, _ := TaskByID(lp.Tasks, "startLot")
	startLotRoles, _ := startLot["roles"].([]any)
	if len(startLotRoles) != 1 || startLotRoles[0] != "logistic" {
		t.Fatalf("lot_processing/startLot roles = %#v, want [logistic]", startLot["roles"])
	}
	startRW, _ := startLot["readyWhen"].(string)
	if !strings.Contains(startRW, `status = "open"`) || !strings.Contains(startRW, "fullyProcessed") {
		t.Fatalf("startLot.readyWhen = %q, want one-open-lot exclusivity", startRW)
	}
	weigh, _ := TaskByID(lp.Tasks, "weigh")
	weighRoles, _ := weigh["roles"].([]any)
	if len(weighRoles) != 1 || weighRoles[0] != "line" {
		t.Fatalf("lot_processing/weigh roles = %#v, want [line]", weigh["roles"])
	}
	markProgress, _ := TaskByID(lp.Tasks, "markProgress")
	markRoles, _ := markProgress["roles"].([]any)
	if len(markRoles) != 1 || markRoles[0] != "line" {
		t.Fatalf("lot_processing/markProgress roles = %#v, want [line]", markProgress["roles"])
	}
	readyWhen, _ := markProgress["readyWhen"].(string)
	if !strings.Contains(readyWhen, "$exists(weight)") {
		t.Fatalf("markProgress.readyWhen must weigh-gate: %q", readyWhen)
	}
	if !strings.Contains(readyWhen, `status = "open"`) {
		t.Fatalf("markProgress.readyWhen = %q, want open-lot gate", readyWhen)
	}
	containerReg, _ := TaskByID(lp.Tasks, "containerReg")
	containerRoles, _ := containerReg["roles"].([]any)
	if len(containerRoles) != 1 || containerRoles[0] != "logistic" {
		t.Fatalf("lot_processing/containerReg roles = %#v, want [logistic]", containerReg["roles"])
	}
	reads, _ := containerReg["reads"].(map[string]any)
	lotsRead, _ := reads["lots"].(map[string]any)
	where, _ := lotsRead["where"].(string)
	if !strings.Contains(where, `status = "open"`) {
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
	closeLot, _ := TaskByID(lp.Tasks, "closeLot")
	closeRoles, _ := closeLot["roles"].([]any)
	if len(closeRoles) != 1 || closeRoles[0] != "logistic" {
		t.Fatalf("lot_processing/closeLot roles = %#v, want [logistic]", closeLot["roles"])
	}
	closeRW, _ := closeLot["readyWhen"].(string)
	if !strings.Contains(closeRW, "$exists(weight)") || !strings.Contains(closeRW, `progress = "done"`) {
		t.Fatalf("closeLot.readyWhen = %q, want weighed+done gate", closeRW)
	}
	if _, has := TaskByID(lp.Tasks, "loadedExtracted"); has {
		t.Fatal("loadedExtracted must not be on the live lot_processing Task list")
	}
	if _, has := TaskByID(lp.Tasks, "palletWeight"); has {
		t.Fatal("palletWeight group must not remain on lot_processing")
	}
	if _, has := TaskByID(lp.Tasks, "lineLoad"); has {
		t.Fatal("lineLoad group must not remain on lot_processing")
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

// Fail closed: path / readyWhen / Worklist engines must stay generic — no plant Task id branches.
func TestEngineSourcesOmitPlantTaskIDBranches(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	for _, name := range []string{"path_availability.go", "ready_when.go", "worklist.go", "tasks.go"} {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		src := string(body)
		for _, token := range []string{`"markProgress"`, `"feragame"`, `"startLot"`, `"lot_processing"`, `"truckArrive"`} {
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
