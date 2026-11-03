package living

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func mustActiveConfig(t *testing.T, c *Committer) OrgConfig {
	t.Helper()
	version, err := c.Configs.ActiveVersion(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := c.Configs.Get(t.Context(), "feragame", version)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestLoaderWorklistEmptyDomainIntakeOpenLineBlocked(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	loader := &Loader{Domains: domains, Configs: configs}
	result, err := loader.Load(t.Context(), "feragame", FixtureAllRolesOp())
	if err != nil {
		t.Fatal(err)
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

	line, ok := worklistJobByID(result.Worklist, "line")
	if !ok {
		t.Fatal("missing line worklist job")
	}
	if line.CanStart {
		t.Fatal("line canStart want false before intake palletWeight")
	}
}

func TestLoadJSONWorklistIntakeUnlockOrder(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	instanceID := "NIR-261019-NO-1"
	commitThroughCodiceFir(t, c, instanceID)

	result, err := loader.Load(t.Context(), "feragame", FixtureAllRolesOp())
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
	wlRaw, ok := wire["worklist"].(map[string]any)
	if !ok {
		t.Fatalf("Load JSON missing worklist: %s", string(raw))
	}
	_ = wlRaw

	intake, ok := worklistJobByID(result.Worklist, "intake")
	if !ok {
		t.Fatal("missing intake")
	}
	inst, ok := worklistInstance(intake, instanceID)
	if !ok {
		t.Fatalf("missing intake instance %s", instanceID)
	}
	if inst.Tasks["register"] != TaskStatusDone {
		t.Fatalf("register = %q, want done", inst.Tasks["register"])
	}
	if inst.Tasks["codiceFir"] != TaskStatusDone {
		t.Fatalf("codiceFir = %q, want done", inst.Tasks["codiceFir"])
	}
	if inst.Tasks["palletWeight"] != TaskStatusAvailable {
		t.Fatalf("palletWeight = %q, want available", inst.Tasks["palletWeight"])
	}
	if inst.Tasks["weightCheck"] != TaskStatusLocked {
		t.Fatalf("weightCheck = %q, want locked", inst.Tasks["weightCheck"])
	}

	line, ok := worklistJobByID(result.Worklist, "line")
	if !ok || line.CanStart {
		t.Fatalf("line canStart before palletWeight = %#v", line)
	}
}

func TestLoadJSONWorklistLineStartsWhenAndSubmissionDone(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	intakeID := "NIR-261019-NO-1"
	completeIntakeThroughPalletWeight(t, c, intakeID)

	loaded, err := loader.Load(t.Context(), "feragame", FixtureAllRolesOp())
	if err != nil {
		t.Fatal(err)
	}
	line, ok := worklistJobByID(loaded.Worklist, "line")
	if !ok {
		t.Fatal("missing line")
	}
	if !line.CanStart {
		t.Fatal("line canStart want true after intake palletWeight")
	}

	commitLineShift(t, c, "1st")
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: "1st",
		Value: map[string]any{
			"progress": map[string]any{
				intakeID + ".1": "done",
				intakeID + ".2": "partial",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	loaded, err = loader.Load(t.Context(), "feragame", FixtureAllRolesOp())
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
	line, ok = worklistJobByID(decoded.Worklist, "line")
	if !ok {
		t.Fatal("decoded missing line worklist")
	}
	inst, ok := worklistInstance(line, "1st")
	if !ok {
		t.Fatal("missing line instance 1st")
	}
	if inst.Tasks["shift"] != TaskStatusDone {
		t.Fatalf("shift = %q, want done", inst.Tasks["shift"])
	}
	if inst.Tasks["lineLoad"] != TaskStatusDone {
		t.Fatalf("lineLoad after Submission = %q, want done", inst.Tasks["lineLoad"])
	}
	if inst.Tasks["containerReg"] != TaskStatusAvailable {
		t.Fatalf("containerReg = %q, want available after lineLoad Submission", inst.Tasks["containerReg"])
	}
}

func TestHandleLoadJSONIncludesWorklist(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	h := &Handler{
		Loader: &Loader{Domains: domains, Configs: configs},
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "u1", Org: "feragame", Roles: []string{"administration"}}, true
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
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
	if !intake.CanStart {
		t.Fatal("intake canStart")
	}
	line, _ := worklistJobByID(got.Worklist, "line")
	if line.CanStart {
		t.Fatal("line canStart should be false")
	}
}

func TestHandleCommitJSONIncludesWorklist(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	h := &Handler{
		Loader:    loader,
		Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}, true
		},
	}

	body := `{"taskPath":"intake/register","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"client":"NO","date":"2026-10-19","wasteCode":"191212","idoneitaRiutilizzo":"NIR","truckWeight":100,"truckTare":200,"palletCount":1,"notes":""}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(body))
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
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	want := ProjectWorklist(mustActiveConfig(t, c), domain, admin)
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
	if inst.Tasks["register"] != TaskStatusDone {
		t.Fatalf("register = %q, want done", inst.Tasks["register"])
	}
	if inst.Tasks["codiceFir"] != TaskStatusAvailable {
		t.Fatalf("codiceFir = %q, want available", inst.Tasks["codiceFir"])
	}
	line, _ := worklistJobByID(got.Worklist, "line")
	if line.CanStart {
		t.Fatal("line canStart should still be false after register only")
	}
}

func TestCommitResultWorklistMatchesPostCommitDomain(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	completeIntakeThroughPalletWeight(t, c, "NIR-261019-NO-1")
	commitLineShift(t, c, "1st")
	lineOp := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	result, err := c.Commit(t.Context(), "feragame", lineOp, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: "1st",
		Value: map[string]any{
			"progress": map[string]any{
				"NIR-261019-NO-1.1": "done",
				"NIR-261019-NO-1.2": "partial",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	line, ok := worklistJobByID(result.Worklist, "line")
	if !ok {
		t.Fatal("missing line worklist on Commit")
	}
	if !line.CanStart {
		t.Fatal("line canStart want true")
	}
	inst, ok := worklistInstance(line, "1st")
	if !ok {
		t.Fatal("missing line instance 1st")
	}
	if inst.Tasks["lineLoad"] != TaskStatusDone {
		t.Fatalf("lineLoad = %q, want done after Submission", inst.Tasks["lineLoad"])
	}
	if inst.Tasks["containerReg"] != TaskStatusAvailable {
		t.Fatalf("containerReg = %q, want available after lineLoad Submission", inst.Tasks["containerReg"])
	}
	want := ProjectWorklist(mustActiveConfig(t, c), result.Domain, lineOp)
	rawGot, _ := json.Marshal(result.Worklist)
	rawWant, _ := json.Marshal(want)
	if string(rawGot) != string(rawWant) {
		t.Fatalf("Commit worklist != ProjectWorklist(domain)\ngot  %s\nwant %s", rawGot, rawWant)
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

	adminWL := ProjectWorklist(cfg, domain, admin)
	logWL := ProjectWorklist(cfg, domain, logistic)

	adminIntake, ok := worklistJobByID(adminWL, "intake")
	if !ok {
		t.Fatal("admin missing intake")
	}
	logIntake, ok := worklistJobByID(logWL, "intake")
	if !ok {
		t.Fatal("logistic missing intake")
	}
	if !adminIntake.CanStart {
		t.Fatal("administration canStart intake want true (first Task register)")
	}
	if logIntake.CanStart {
		t.Fatal("logistic canStart intake want false (first Task register is administration-only)")
	}
	if len(adminIntake.Tasks) == 0 || adminIntake.Tasks[0].ID != "register" {
		t.Fatalf("task chrome = %#v", adminIntake.Tasks)
	}
	if got := adminIntake.Tasks[0].Roles; len(got) != 1 || got[0] != "administration" {
		t.Fatalf("register roles chrome = %#v", got)
	}
	if adminIntake.Tasks[0].Title == "" {
		t.Fatal("register title chrome empty")
	}
	if adminIntake.Title != "Camion" {
		t.Fatalf("intake job title = %q", adminIntake.Title)
	}
	pallet := worklistTaskByIDOrFatal(t, adminIntake, "palletWeight")
	if len(pallet.Roles) != 1 || pallet.Roles[0] != "logistic" {
		t.Fatalf("palletWeight roles = %#v", pallet.Roles)
	}
	if pallet.Title == "" {
		t.Fatal("palletWeight title empty")
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
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	instanceID := "NIR-261019-NO-1"
	commitThroughCodiceFir(t, c, instanceID)

	cfg := mustActiveConfig(t, c)
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}

	admin := OperatorIdentity{UserID: "admin", Org: "feragame", Roles: []string{"administration"}}
	logistic := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}

	adminWL := ProjectWorklist(cfg, domain, admin)
	logWL := ProjectWorklist(cfg, domain, logistic)

	adminIntake, _ := worklistJobByID(adminWL, "intake")
	logIntake, _ := worklistJobByID(logWL, "intake")
	adminInst, ok := worklistInstance(adminIntake, instanceID)
	if !ok {
		t.Fatal("admin missing instance")
	}
	logInst, ok := worklistInstance(logIntake, instanceID)
	if !ok {
		t.Fatal("logistic missing instance")
	}
	// Unlock/Submission status stays shared (not remapped by roles).
	if adminInst.Tasks["palletWeight"] != TaskStatusAvailable || logInst.Tasks["palletWeight"] != TaskStatusAvailable {
		t.Fatalf("palletWeight status admin=%q logistic=%q, want available", adminInst.Tasks["palletWeight"], logInst.Tasks["palletWeight"])
	}
	if containsString(adminInst.ViewerAvailable, "palletWeight") {
		t.Fatalf("administration ViewerAvailable must exclude palletWeight: %#v", adminInst.ViewerAvailable)
	}
	if !containsString(logInst.ViewerAvailable, "palletWeight") {
		t.Fatalf("logistic ViewerAvailable must include palletWeight: %#v", logInst.ViewerAvailable)
	}

	adminRefs := CollectAvailableTaskRefs(cfg, adminWL)
	logRefs := CollectAvailableTaskRefs(cfg, logWL)
	if hasAvailableRef(adminRefs, "intake", "palletWeight", instanceID) {
		t.Fatalf("administration Available affordances must exclude palletWeight: %#v", adminRefs)
	}
	if !hasAvailableRef(logRefs, "intake", "palletWeight", instanceID) {
		t.Fatalf("logistic Available affordances must include palletWeight: %#v", logRefs)
	}

	adminLoaded, err := loader.Load(t.Context(), "feragame", admin)
	if err != nil {
		t.Fatal(err)
	}
	logLoaded, err := loader.Load(t.Context(), "feragame", logistic)
	if err != nil {
		t.Fatal(err)
	}
	if len(adminLoaded.TaskFormBundles) != 0 || len(logLoaded.TaskFormBundles) != 0 {
		t.Fatal("home Load must omit bulk Task form bundles for all viewers")
	}
	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "palletWeight", instanceID); err == nil {
		t.Fatal("administration addressed fetch must forbid palletWeight")
	}
	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", logistic, "intake", "palletWeight", instanceID); err != nil {
		t.Fatalf("logistic addressed fetch must allow palletWeight: %v", err)
	}
}

func hasAvailableRef(refs []AvailableTaskRef, jobID, taskID, jobInstanceID string) bool {
	for _, r := range refs {
		if r.JobID == jobID && r.TaskID == taskID && r.JobInstanceID == jobInstanceID {
			return true
		}
	}
	return false
}

func containsString(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
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

