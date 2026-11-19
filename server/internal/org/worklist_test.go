package org

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func mustActiveConfig(t *testing.T, c *TaskCommitter) OrgConfig {
	t.Helper()
	version, err := c.Configs.ActiveVersion(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := c.Configs.Get(t.Context(), FeragameScope(), version)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestLoaderWorklistEmptyDomainIntakeOpenLotProcessingBlocked(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	loader := &DomainLoader{Domains: domains, Configs: configs}
	result, err := loader.Load(t.Context(), FeragameScope(), FixtureAllRolesOp())
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := worklistJobByID(result.Worklist, "workingDay"); ok {
		t.Fatal("workingDay Job must be absent from worklist")
	}

	intake, ok := worklistJobByID(result.Worklist, "intake")
	if !ok {
		t.Fatal("missing intake worklist job")
	}
	if !intake.CanStart {
		t.Fatal("intake canStart want true on empty domain")
	}
	if len(intake.Instances) != 0 {
		t.Fatalf("intake instances = %#v", intake.Instances)
	}

	lp, ok := worklistJobByID(result.Worklist, "lot_processing")
	if !ok {
		t.Fatal("missing lot_processing worklist job")
	}
	if lp.CanStart {
		t.Fatal("lot_processing canStart want false on empty domain (no eligible lots)")
	}

	if _, ok := worklistJobByID(result.Worklist, "passport"); ok {
		t.Fatal("passport Job must not appear on Worklist")
	}
}

func TestLoadJSONWorklistIntakeStatuses(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID, _, _ := commitIntakeThroughLotsClosed(t, c)

	result, err := loader.Load(t.Context(), FeragameScope(), FixtureAllRolesOp())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if _, ok := wire["worklist"].(map[string]any); !ok {
		t.Fatalf("Load JSON missing worklist: %s", string(raw))
	}

	intake, ok := worklistJobByID(result.Worklist, "intake")
	if !ok {
		t.Fatal("missing intake")
	}
	inst, ok := worklistInstance(intake, truckID)
	if !ok {
		t.Fatalf("missing intake instance %s", truckID)
	}
	if inst.Tasks["truckArrive"] != TaskStatusDone {
		t.Fatalf("truckArrive = %q, want done", inst.Tasks["truckArrive"])
	}
	if inst.Tasks["register"] != TaskStatusActionable {
		t.Fatalf("register = %q, want actionable (many)", inst.Tasks["register"])
	}
	if inst.Tasks["weightCheck"] != TaskStatusActionable {
		t.Fatalf("weightCheck = %q, want actionable", inst.Tasks["weightCheck"])
	}

	scoped, err := loader.LoadForStreamInstance(t.Context(), FeragameScope(), FixtureAllRolesOp(), truckID)
	if err != nil {
		t.Fatal(err)
	}
	lp, ok := worklistJobByID(scoped.Worklist, "lot_processing")
	if !ok || !lp.CanStart {
		t.Fatalf("lot_processing canStart on truck = %#v, want true", lp)
	}
}

func TestLoadJSONWorklistLotProcessingSubmissionDone(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	loadID, lotID := seedLotReadyAfterWeigh(t, c)

	loaded, err := loader.Load(t.Context(), FeragameScope(), FixtureAllRolesOp())
	if err != nil {
		t.Fatal(err)
	}
	lp, ok := worklistJobByID(loaded.Worklist, "lot_processing")
	if !ok {
		t.Fatal("missing lot_processing")
	}
	if lp.CanStart {
		t.Fatal("lot_processing canStart want false while seeded lot is still open")
	}

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	_, err = c.Commit(t.Context(), FeragameScope(), op, CommitRequest{
		TaskPath:      "lot_processing/markProgress",
		JobInstanceID: lotID,
		Value: map[string]any{
			"progress": map[string]any{
				loadID + ".1": "done",
				loadID + ".2": "partial",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	loaded, err = loader.Load(t.Context(), FeragameScope(), FixtureAllRolesOp())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(raw) {
		t.Fatal("Load JSON invalid")
	}
	var decoded LoadResult
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	lp, ok = worklistJobByID(decoded.Worklist, "lot_processing")
	if !ok {
		t.Fatal("decoded missing lot_processing worklist")
	}
	inst, ok := worklistInstance(lp, lotID)
	if !ok {
		t.Fatalf("missing lot_processing instance %s", lotID)
	}
	if inst.Tasks["startLot"] != TaskStatusDone {
		t.Fatalf("startLot = %q, want done", inst.Tasks["startLot"])
	}
	if inst.Tasks["markProgress"] != TaskStatusActionable {
		t.Fatalf("markProgress = %q, want actionable (many + leftover)", inst.Tasks["markProgress"])
	}
	if inst.Tasks["containerReg"] != TaskStatusActionable {
		t.Fatalf("containerReg = %q, want actionable (open lot; FixtureAllRoles includes logistic)", inst.Tasks["containerReg"])
	}
}

func TestHandleLoadJSONIncludesWorklist(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "intake", ID: "T-load", Label: "T-load"}}
	if err := domains.Put(t.Context(), FeragameScope(), domain); err != nil {
		t.Fatal(err)
	}
	h := &API{
		DomainLoader: &DomainLoader{Domains: domains, Configs: configs},
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "u1", Org: "feragame", Roles: []string{"administration"}}, true
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/org/domain?stream=recycling&instance=T-load", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got LoadResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Worklist.Jobs) != 2 {
		t.Fatalf("worklist jobs = %#v", got.Worklist.Jobs)
	}
	intake, _ := worklistJobByID(got.Worklist, "intake")
	if len(intake.Instances) != 1 || intake.Instances[0].ID != "T-load" {
		t.Fatalf("scoped intake instances = %#v", intake.Instances)
	}
	lp, _ := worklistJobByID(got.Worklist, "lot_processing")
	if lp.CanStart {
		t.Fatal("administration viewer must not canStart lot_processing (startLot is logistic-only)")
	}
	if _, ok := worklistJobByID(got.Worklist, "passport"); ok {
		t.Fatal("passport Job must not appear on Worklist")
	}
}

func TestHandleCommitJSONIncludesWorklist(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	h := &API{
		DomainLoader:  loader,
		TaskCommitter: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}, true
		},
	}

	body := `{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"truckWeight":1000}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/org/commit?stream=recycling", strings.NewReader(body))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var wire map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if _, ok := wire["worklist"].(map[string]any); !ok {
		t.Fatalf("Commit JSON missing worklist: %s", rec.Body.String())
	}

	var got CommitResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	domain, err := c.Domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	want := ProjectWorklist(mustActiveConfig(t, c), domain, admin, got.JobInstanceID)
	if len(got.Worklist.Jobs) != len(want.Jobs) {
		t.Fatalf("worklist jobs = %#v, want %#v", got.Worklist.Jobs, want.Jobs)
	}
	intake, ok := worklistJobByID(got.Worklist, "intake")
	if !ok {
		t.Fatal("missing intake worklist on Commit")
	}
	inst, ok := worklistInstance(intake, got.JobInstanceID)
	if !ok {
		t.Fatalf("missing intake instance %q", got.JobInstanceID)
	}
	if inst.Tasks["truckArrive"] != TaskStatusDone {
		t.Fatalf("truckArrive = %q, want done", inst.Tasks["truckArrive"])
	}
	if inst.Tasks["register"] != TaskStatusActionable {
		t.Fatalf("register = %q, want actionable", inst.Tasks["register"])
	}
	lp, _ := worklistJobByID(got.Worklist, "lot_processing")
	if lp.CanStart {
		t.Fatal("administration viewer must not canStart lot_processing after truckArrive (no lots yet / startLot logistic)")
	}
}

func TestCommitResultWorklistMatchesPostCommitDomain(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	loadID, lotID := seedLotReadyAfterWeigh(t, c)
	lineOp := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	result, err := c.Commit(t.Context(), FeragameScope(), lineOp, CommitRequest{
		TaskPath:      "lot_processing/markProgress",
		JobInstanceID: lotID,
		Value: map[string]any{
			"progress": map[string]any{
				loadID + ".1": "done",
				loadID + ".2": "partial",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	lp, ok := worklistJobByID(result.Worklist, "lot_processing")
	if !ok {
		t.Fatal("missing lot_processing worklist on Commit")
	}
	if lp.CanStart {
		t.Fatal("lot_processing canStart want false while lot is still open")
	}
	inst, ok := worklistInstance(lp, lotID)
	if !ok {
		t.Fatalf("missing lot_processing instance %s", lotID)
	}
	if inst.Tasks["markProgress"] != TaskStatusActionable {
		t.Fatalf("markProgress = %q, want actionable after partial mark (many)", inst.Tasks["markProgress"])
	}
	if inst.Tasks["containerReg"] != TaskStatusAvailable {
		t.Fatalf("containerReg = %q, want available for line viewer (open lot)", inst.Tasks["containerReg"])
	}
	want := projectWorklistUnfiltered(mustActiveConfig(t, c), result.Domain, lineOp)
	rawGot, _ := json.Marshal(result.Worklist)
	rawWant, _ := json.Marshal(want)
	if string(rawGot) != string(rawWant) {
		t.Fatalf("Commit worklist != projectWorklistUnfiltered(domain)\ngot  %s\nwant %s", rawGot, rawWant)
	}
}

func TestProjectWorklistViewerRoleSplitCanStartAndChrome(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	domain := EmptyDomain()

	admin := OperatorIdentity{UserID: "admin", Org: "feragame", Roles: []string{"administration"}}
	logistic := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}

	adminWL := projectWorklistUnfiltered(cfg, domain, admin)
	logWL := projectWorklistUnfiltered(cfg, domain, logistic)

	adminIntake, ok := worklistJobByID(adminWL, "intake")
	if !ok {
		t.Fatal("admin missing intake")
	}
	logIntake, ok := worklistJobByID(logWL, "intake")
	if !ok {
		t.Fatal("logistic missing intake")
	}
	if !adminIntake.CanStart {
		t.Fatal("administration canStart intake want true on empty domain")
	}
	if logIntake.CanStart {
		t.Fatal("logistic canStart intake want false (first Task truckArrive is administration-only)")
	}
	if _, ok := worklistJobByID(adminWL, "workingDay"); ok {
		t.Fatal("workingDay Job must be absent")
	}
	if len(adminIntake.Tasks) == 0 || adminIntake.Tasks[0].ID != "truckArrive" {
		t.Fatalf("task chrome = %#v", adminIntake.Tasks)
	}
	if got := adminIntake.Tasks[0].Roles; len(got) != 1 || got[0] != "administration" {
		t.Fatalf("truckArrive roles chrome = %#v", got)
	}
	if adminIntake.Tasks[0].Title == "" {
		t.Fatal("truckArrive title chrome empty")
	}
	register := worklistTaskByIDOrFatal(t, adminIntake, "register")
	if register.ID != "register" {
		t.Fatalf("register chrome = %#v", register)
	}
	if register.Submit != submitMany {
		t.Fatalf("register submit chrome = %q, want many", register.Submit)
	}
	arrive := worklistTaskByIDOrFatal(t, adminIntake, "truckArrive")
	if arrive.Submit != submitOnce {
		t.Fatalf("truckArrive submit chrome = %q, want once", arrive.Submit)
	}
	weight := worklistTaskByIDOrFatal(t, adminIntake, "weightCheck")
	if len(weight.Roles) != 1 || weight.Roles[0] != "administration" {
		t.Fatalf("weightCheck roles = %#v", weight.Roles)
	}
	if weight.Submit != submitOnce {
		t.Fatalf("weightCheck submit chrome = %q, want once", weight.Submit)
	}
}

func worklistTaskByIDOrFatal(t *testing.T, job WorklistJob, taskID string) WorklistTask {
	t.Helper()
	chrome, ok := worklistTaskByID(job, taskID)
	if !ok {
		t.Fatalf("missing task chrome %q", taskID)
	}
	return chrome
}

func TestProjectWorklistViewerRoleSplitAvailableAffordances(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	_ = commitStartLot(t, c, loadID)
	cfg := mustActiveConfig(t, c)
	domain, err := c.Domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}

	admin := OperatorIdentity{UserID: "admin", Org: "feragame", Roles: []string{"administration"}}
	lineOp := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}
	logistic := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}

	adminWL := projectWorklistUnfiltered(cfg, domain, admin)
	lineWL := projectWorklistUnfiltered(cfg, domain, lineOp)
	logWL := projectWorklistUnfiltered(cfg, domain, logistic)

	adminLP, _ := worklistJobByID(adminWL, "lot_processing")
	lineLP, _ := worklistJobByID(lineWL, "lot_processing")
	logLP, _ := worklistJobByID(logWL, "lot_processing")
	adminInst, ok := worklistInstance(adminLP, loadID)
	if !ok {
		t.Fatal("admin missing lot_processing instance")
	}
	lineInst, ok := worklistInstance(lineLP, loadID)
	if !ok {
		t.Fatal("line missing lot_processing instance")
	}
	logInst, ok := worklistInstance(logLP, loadID)
	if !ok {
		t.Fatal("logistic missing lot_processing instance")
	}
	if adminInst.Tasks["weigh"] != TaskStatusAvailable {
		t.Fatalf("admin weigh = %q, want available (wrong role)", adminInst.Tasks["weigh"])
	}
	if lineInst.Tasks["weigh"] != TaskStatusActionable {
		t.Fatalf("line weigh = %q, want actionable (open lot)", lineInst.Tasks["weigh"])
	}
	if logInst.Tasks["weigh"] != TaskStatusAvailable {
		t.Fatalf("logistic weigh = %q, want available (wrong role)", logInst.Tasks["weigh"])
	}
	if logInst.Tasks["containerReg"] != TaskStatusActionable {
		t.Fatalf("logistic containerReg = %q, want actionable", logInst.Tasks["containerReg"])
	}

	adminIntake, _ := worklistJobByID(adminWL, "intake")
	logIntake, _ := worklistJobByID(logWL, "intake")
	adminTruck, ok := worklistInstance(adminIntake, truckID)
	if !ok {
		t.Fatal("admin missing intake instance")
	}
	logTruck, ok := worklistInstance(logIntake, truckID)
	if !ok {
		t.Fatal("logistic missing intake instance")
	}
	if adminTruck.Tasks["weightCheck"] != TaskStatusActionable {
		t.Fatalf("admin weightCheck = %q, want actionable", adminTruck.Tasks["weightCheck"])
	}
	if logTruck.Tasks["weightCheck"] != TaskStatusAvailable {
		t.Fatalf("logistic weightCheck = %q, want available (handoff)", logTruck.Tasks["weightCheck"])
	}

	adminRefs := CollectActionableTaskRefs(cfg, adminWL)
	lineRefs := CollectActionableTaskRefs(cfg, lineWL)
	if hasActionableRef(adminRefs, "lot_processing", "weigh", loadID) {
		t.Fatalf("administration Available affordances must exclude weigh: %#v", adminRefs)
	}
	if !hasActionableRef(lineRefs, "lot_processing", "weigh", loadID) {
		t.Fatalf("line Available affordances must include weigh: %#v", lineRefs)
	}

	adminLoaded, err := loader.Load(t.Context(), FeragameScope(), admin)
	if err != nil {
		t.Fatal(err)
	}
	lineLoaded, err := loader.Load(t.Context(), FeragameScope(), lineOp)
	if err != nil {
		t.Fatal(err)
	}
	if len(adminLoaded.TaskFormBundles) != 0 || len(lineLoaded.TaskFormBundles) != 0 {
		t.Fatal("home Load must omit bulk Task form bundles for all viewers")
	}
	if _, err := loader.LoadTaskBundle(t.Context(), FeragameScope(), admin, "lot_processing", "weigh", loadID); err == nil {
		t.Fatal("administration addressed fetch must forbid weigh")
	}
	if _, err := loader.LoadTaskBundle(t.Context(), FeragameScope(), lineOp, "lot_processing", "weigh", loadID); err != nil {
		t.Fatalf("line addressed fetch must allow weigh: %v", err)
	}
}

func hasActionableRef(refs []ActionableTaskRef, jobID, taskID, jobInstanceID string) bool {
	for _, r := range refs {
		if r.JobID == jobID && r.TaskID == taskID && r.JobInstanceID == jobInstanceID {
			return true
		}
	}
	return false
}

func bundleHasTask(bundles []*TaskFormBundle, taskID string) bool {
	for _, b := range bundles {
		if b != nil && b.TaskID == taskID {
			return true
		}
	}
	return false
}

func TestProjectWorklistFlatRegisterMintsLotWithoutGroup(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID := commitTruckArrive(t, c)
	result := commitLotRegister(t, c, truckID, registerLotValue())
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	wl := projectWorklistUnfiltered(mustActiveConfig(t, c), result.Domain, op)
	inst, ok := worklistInstance(worklistMustJob(t, wl, "intake"), truckID)
	if !ok {
		t.Fatal("missing truck instance")
	}
	if inst.Tasks["register"] != TaskStatusActionable {
		t.Fatalf("register = %q, want actionable", inst.Tasks["register"])
	}
	loadID := firstLoadID(result.Domain)
	if loadID == "" {
		t.Fatal("expected minted lot")
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "register" || !containsString(last.IDsTouched, loadID) {
		t.Fatalf("register submission must touch lot id %q: %#v", loadID, last)
	}
}

func TestProjectWorklistFlatWeighTouchesPallet(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	_ = commitStartLot(t, c, loadID)
	palletID := loadID + ".1"
	result := commitWeigh(t, c, loadID, palletID, 400)
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"logistic", "line", "administration"}}
	wl := projectWorklistUnfiltered(mustActiveConfig(t, c), result.Domain, op)
	inst, ok := worklistInstance(worklistMustJob(t, wl, "lot_processing"), loadID)
	if !ok {
		t.Fatal("missing lot_processing instance")
	}
	if inst.Tasks["weigh"] != TaskStatusActionable {
		t.Fatalf("weigh = %q, want actionable (many + leftover pallet)", inst.Tasks["weigh"])
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "weigh" || !containsString(last.IDsTouched, palletID) {
		t.Fatalf("weigh submission must touch pallet %q: %#v", palletID, last)
	}
	if _, ok := result.Domain.Kind("pallet")[palletID]; !ok {
		t.Fatalf("pallet %q missing under Domain: %#v", palletID, result.Domain.Kind("pallet"))
	}
}

func worklistMustJob(t *testing.T, wl Worklist, jobID string) WorklistJob {
	t.Helper()
	job, ok := worklistJobByID(wl, jobID)
	if !ok {
		t.Fatalf("missing worklist job %q", jobID)
	}
	return job
}

func TestProjectWorklistReadyWhenFold(t *testing.T) {
	op := OperatorIdentity{UserID: "u", Org: "test", Roles: []string{"ops"}}
	other := OperatorIdentity{UserID: "v", Org: "test", Roles: []string{"other"}}

	leaf := func(id string, readyWhen any) TaskConfig {
		tc := TaskConfig{
			"id": id, "roles": []any{"ops"},
			"schema": map[string]any{"title": id},
		}
		if readyWhen != nil {
			tc["readyWhen"] = readyWhen
		}
		return tc
	}

	cfgFor := func(gated TaskConfig) OrgConfig {
		return OrgConfig{
			Jobs: OrderedObject{
				Keys: []string{"gateJob"},
				Values: map[string]any{
					"gateJob": Job{
						Tasks: []TaskConfig{
							leaf("opener", nil),
							gated,
						},
					},
				},
			},
		}
	}

	domainUnlocked := EmptyDomain()
	domainUnlocked.JobInstances = []JobInstance{{JobID: "gateJob", ID: "I1", Label: "inst"}}
	domainUnlocked.Submissions = []Submission{sub("gateJob", "opener", "I1")}

	t.Run("omit_readyWhen_role_split", func(t *testing.T) {
		cfg := cfgFor(leaf("gated", nil))
		wl := projectWorklistUnfiltered(cfg, domainUnlocked, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusActionable {
			t.Fatalf("omit + eligible = %q, want actionable", inst.Tasks["gated"])
		}
		wlOther := projectWorklistUnfiltered(cfg, domainUnlocked, other)
		instOther, _ := worklistInstance(worklistMustJob(t, wlOther, "gateJob"), "I1")
		if instOther.Tasks["gated"] != TaskStatusAvailable {
			t.Fatalf("omit + ineligible = %q, want available", instOther.Tasks["gated"])
		}
	})

	t.Run("literal_true", func(t *testing.T) {
		cfg := cfgFor(leaf("gated", "true"))
		wl := projectWorklistUnfiltered(cfg, domainUnlocked, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusActionable {
			t.Fatalf("true = %q, want actionable", inst.Tasks["gated"])
		}
	})

	t.Run("literal_false_skips_role_split", func(t *testing.T) {
		cfg := cfgFor(leaf("gated", "false"))
		wl := projectWorklistUnfiltered(cfg, domainUnlocked, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusUnavailable {
			t.Fatalf("false + eligible = %q, want unavailable", inst.Tasks["gated"])
		}
		wlOther := projectWorklistUnfiltered(cfg, domainUnlocked, other)
		instOther, _ := worklistInstance(worklistMustJob(t, wlOther, "gateJob"), "I1")
		if instOther.Tasks["gated"] != TaskStatusUnavailable {
			t.Fatalf("false + ineligible = %q, want unavailable (not available)", instOther.Tasks["gated"])
		}
		if refs := CollectActionableTaskRefs(cfg, wl); hasActionableRef(refs, "gateJob", "gated", "I1") {
			t.Fatalf("unavailable must not be CollectActionable: %#v", refs)
		}
		if ViewerMayMountTask(cfg, wl, ActionableTaskRef{JobID: "gateJob", TaskID: "gated", JobInstanceID: "I1"}) {
			t.Fatal("unavailable must not mount")
		}
	})

	t.Run("blank_string_omit_semantics", func(t *testing.T) {
		cfg := cfgFor(leaf("gated", "   "))
		wl := projectWorklistUnfiltered(cfg, domainUnlocked, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusActionable {
			t.Fatalf("blank = %q, want actionable", inst.Tasks["gated"])
		}
	})

	t.Run("invalid_fail_closed", func(t *testing.T) {
		cfg := cfgFor(leaf("gated", 1))
		wl := projectWorklistUnfiltered(cfg, domainUnlocked, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusUnavailable {
			t.Fatalf("invalid = %q, want unavailable", inst.Tasks["gated"])
		}
	})

	t.Run("domain_expr", func(t *testing.T) {
		cfg := cfgFor(leaf("gated", "$count($domain.widget.*) >= 1"))
		wl := projectWorklistUnfiltered(cfg, domainUnlocked, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusUnavailable {
			t.Fatalf("no widget = %q, want unavailable", inst.Tasks["gated"])
		}
		withWidget := domainUnlocked
		putKind(&withWidget, "widget", EntityMap{"W1": {"n": 1}})
		wl = projectWorklistUnfiltered(cfg, withWidget, op)
		inst, _ = worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusActionable {
			t.Fatalf("with widget = %q, want actionable", inst.Tasks["gated"])
		}
	})

	t.Run("not_done_folds_readyWhen", func(t *testing.T) {
		cfg := cfgFor(leaf("gated", "false"))
		empty := EmptyDomain()
		empty.JobInstances = []JobInstance{{JobID: "gateJob", ID: "I1", Label: "inst"}}
		wl := projectWorklistUnfiltered(cfg, empty, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusUnavailable {
			t.Fatalf("readyWhen false = %q, want unavailable (no locked)", inst.Tasks["gated"])
		}
	})

	t.Run("canStart_folds_first_task_readyWhen", func(t *testing.T) {
		cfg := OrgConfig{
			Jobs: OrderedObject{
				Keys: []string{"startJob"},
				Values: map[string]any{
					"startJob": Job{
						Repeatable: true,
						Tasks: []TaskConfig{
							leaf("starter", "$count($domain.widget.*) >= 1"),
						},
					},
				},
			},
		}
		wl := projectWorklistUnfiltered(cfg, EmptyDomain(), op)
		job := worklistMustJob(t, wl, "startJob")
		if job.CanStart {
			t.Fatal("canStart want false when first-Task readyWhen fails")
		}
		withWidget := EmptyDomain()
		putKind(&withWidget, "widget", EntityMap{"W1": {"n": 1}})
		wl = projectWorklistUnfiltered(cfg, withWidget, op)
		job = worklistMustJob(t, wl, "startJob")
		if !job.CanStart {
			t.Fatal("canStart want true when first-Task readyWhen holds")
		}
		omitCfg := OrgConfig{
			Jobs: OrderedObject{
				Keys: []string{"startJob"},
				Values: map[string]any{
					"startJob": Job{
						Repeatable: true,
						Tasks:      []TaskConfig{leaf("starter", nil)},
					},
				},
			},
		}
		wl = projectWorklistUnfiltered(omitCfg, EmptyDomain(), op)
		if !worklistMustJob(t, wl, "startJob").CanStart {
			t.Fatal("omit readyWhen ⇒ canStart true")
		}
	})
}
