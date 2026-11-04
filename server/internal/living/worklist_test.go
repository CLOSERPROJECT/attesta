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
	if !line.CanStart {
		t.Fatal("line canStart want true (plant line has no startsWhen)")
	}

	passport, ok := worklistJobByID(result.Worklist, "passport")
	if !ok {
		t.Fatal("missing passport worklist job")
	}
	if passport.CanStart {
		t.Fatal("passport canStart want false on empty Domain (select readyWhen)")
	}
}

func TestLoadJSONWorklistIntakeUnlockOrder(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	truckID, _, _ := commitIntakeThroughLotsClosed(t, c)

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
	if inst.Tasks["lots"] != TaskStatusDone {
		t.Fatalf("lots = %q, want done", inst.Tasks["lots"])
	}
	if inst.Tasks["weightCheck"] != TaskStatusActionable {
		t.Fatalf("weightCheck = %q, want actionable", inst.Tasks["weightCheck"])
	}

	line, ok := worklistJobByID(result.Worklist, "line")
	if !ok || !line.CanStart {
		t.Fatalf("line canStart = %#v, want true", line)
	}
}

func TestLoadJSONWorklistLineStartsWhenAndSubmissionDone(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)

	loaded, err := loader.Load(t.Context(), "feragame", FixtureAllRolesOp())
	if err != nil {
		t.Fatal(err)
	}
	line, ok := worklistJobByID(loaded.Worklist, "line")
	if !ok {
		t.Fatal("missing line")
	}
	if !line.CanStart {
		t.Fatal("line canStart want true")
	}

	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: lineID,
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
	inst, ok := worklistInstance(line, lineID)
	if !ok {
		t.Fatalf("missing line instance %s", lineID)
	}
	if inst.Tasks["openTurno"] != TaskStatusDone {
		t.Fatalf("openTurno = %q, want done", inst.Tasks["openTurno"])
	}
	if inst.Tasks["lineLoad"] != TaskStatusDone {
		t.Fatalf("lineLoad after Submission = %q, want done", inst.Tasks["lineLoad"])
	}
	// Group node chrome stays available (lots pattern); Spawn/child leaf is role-scoped.
	if inst.Tasks["containerReg"] != TaskStatusAvailable {
		t.Fatalf("containerReg = %q, want available (unlockFrom openTurno)", inst.Tasks["containerReg"])
	}
	if g := inst.Groups["containerReg"]; !g.CanSpawn {
		t.Fatalf("containerReg CanSpawn = %#v, want true for logistic-capable viewer", g)
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
	if len(got.Worklist.Jobs) != 3 {
		t.Fatalf("worklist jobs = %#v", got.Worklist.Jobs)
	}
	intake, _ := worklistJobByID(got.Worklist, "intake")
	if !intake.CanStart {
		t.Fatal("intake canStart")
	}
	line, _ := worklistJobByID(got.Worklist, "line")
	if line.CanStart {
		t.Fatal("administration viewer must not canStart line (openTurno is line-only)")
	}
	passport, _ := worklistJobByID(got.Worklist, "passport")
	if passport.CanStart || passport.StartLabel != "Blocca passaporto" {
		t.Fatalf("passport Job = %#v, want canStart false on empty Domain", passport)
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

	body := `{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"date":"2026-10-19","turno":"1st","truckWeight":1000}}`
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
	if inst.Tasks["truckArrive"] != TaskStatusDone {
		t.Fatalf("truckArrive = %q, want done", inst.Tasks["truckArrive"])
	}
	if inst.Tasks["lots"] != TaskStatusAvailable {
		t.Fatalf("lots = %q, want available", inst.Tasks["lots"])
	}
	line, _ := worklistJobByID(got.Worklist, "line")
	if line.CanStart {
		t.Fatal("administration viewer must not canStart line after truckArrive")
	}
}

func TestCommitResultWorklistMatchesPostCommitDomain(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	loadID, lineID := seedLineReadyForLineLoad(t, c)
	lineOp := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	result, err := c.Commit(t.Context(), "feragame", lineOp, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: lineID,
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
	line, ok := worklistJobByID(result.Worklist, "line")
	if !ok {
		t.Fatal("missing line worklist on Commit")
	}
	if !line.CanStart {
		t.Fatal("line canStart want true")
	}
	inst, ok := worklistInstance(line, lineID)
	if !ok {
		t.Fatalf("missing line instance %s", lineID)
	}
	if inst.Tasks["lineLoad"] != TaskStatusDone {
		t.Fatalf("lineLoad = %q, want done after Submission", inst.Tasks["lineLoad"])
	}
	if inst.Tasks["containerReg"] != TaskStatusAvailable {
		t.Fatalf("containerReg = %q, want available for line viewer (unlockFrom openTurno)", inst.Tasks["containerReg"])
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
		t.Fatal("administration canStart intake want true (first Task truckArrive)")
	}
	if logIntake.CanStart {
		t.Fatal("logistic canStart intake want false (first Task truckArrive is administration-only)")
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
	lots := worklistTaskByIDOrFatal(t, adminIntake, "lots")
	if lots.ID != "lots" {
		t.Fatalf("lots chrome = %#v", lots)
	}
	weight := worklistTaskByIDOrFatal(t, adminIntake, "weightCheck")
	if len(weight.Roles) != 1 || weight.Roles[0] != "administration" {
		t.Fatalf("weightCheck roles = %#v", weight.Roles)
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
	truckID, _, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)

	cfg := mustActiveConfig(t, c)
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}

	admin := OperatorIdentity{UserID: "admin", Org: "feragame", Roles: []string{"administration"}}
	logistic := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}

	adminWL := ProjectWorklist(cfg, domain, admin)
	logWL := ProjectWorklist(cfg, domain, logistic)

	adminLine, _ := worklistJobByID(adminWL, "line")
	logLine, _ := worklistJobByID(logWL, "line")
	adminInst, ok := worklistInstance(adminLine, lineID)
	if !ok {
		t.Fatal("admin missing line instance")
	}
	logInst, ok := worklistInstance(logLine, lineID)
	if !ok {
		t.Fatal("logistic missing line instance")
	}
	if adminInst.Tasks["palletWeight"] != TaskStatusAvailable || logInst.Tasks["palletWeight"] != TaskStatusAvailable {
		t.Fatalf("palletWeight group status admin=%q logistic=%q, want available (group unlock)", adminInst.Tasks["palletWeight"], logInst.Tasks["palletWeight"])
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
	logRefs := CollectActionableTaskRefs(cfg, logWL)
	if hasNestedActionableRef(adminRefs, "line", "palletWeight", lineID, "weigh") {
		t.Fatalf("administration Available affordances must exclude weigh: %#v", adminRefs)
	}
	if !hasNestedActionableRef(logRefs, "line", "palletWeight", lineID, "weigh") {
		t.Fatalf("logistic Available affordances must include weigh: %#v", logRefs)
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
	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "line", "weigh", lineID, "palletWeight", ProvisionalGroupInstanceID); err == nil {
		t.Fatal("administration addressed fetch must forbid weigh")
	}
	if _, err := loader.LoadTaskBundle(t.Context(), "feragame", logistic, "line", "weigh", lineID, "palletWeight", ProvisionalGroupInstanceID); err != nil {
		t.Fatalf("logistic addressed fetch must allow weigh: %v", err)
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

func TestProjectWorklistGroupChromeSpawnCloseAndChildStatuses(t *testing.T) {
	lots := TaskConfig{
		"id":                           "lots",
		"kind":                         "group",
		"repeatable":                   true,
		"startLabel":                   "Nuovo lotto",
		"closeLabel":                   "Chiudi scarico",
		"closeRequiresChildSubmission": true,
		"tasks": []TaskConfig{
			{
				"id": "register", "roles": []any{"administration"},
				"schema": map[string]any{"title": "Register"},
			},
		},
	}
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"intake"},
			Values: map[string]any{
				"intake": Job{
					Repeatable: true,
					Tasks: []TaskConfig{
						{
							"id": "truckArrive", "roles": []any{"administration"},
							"schema": map[string]any{"title": "Arrive"},
						},
						lots,
						{
							"id": "weightCheck", "roles": []any{"administration"},
							"schema": map[string]any{"title": "Weight"},
						},
					},
				},
			},
		},
	}
	op := OperatorIdentity{UserID: "ada", Org: "test", Roles: []string{"administration"}}

	domain := EmptyDomain()
	domain.JobInstances = []JobInstance{{JobID: "intake", ID: "T1", Label: "truck"}}
	domain.Submissions = []Submission{sub("intake", "truckArrive", "T1")}

	wl := ProjectWorklist(cfg, domain, op)
	intake, ok := worklistJobByID(wl, "intake")
	if !ok {
		t.Fatal("missing intake")
	}
	inst, ok := worklistInstance(intake, "T1")
	if !ok {
		t.Fatal("missing instance")
	}
	if inst.Tasks["lots"] != TaskStatusAvailable {
		t.Fatalf("lots = %q, want available", inst.Tasks["lots"])
	}
	if inst.Tasks["weightCheck"] != TaskStatusLocked {
		t.Fatalf("weightCheck = %q, want locked", inst.Tasks["weightCheck"])
	}
	g, ok := inst.Groups["lots"]
	if !ok || !g.CanSpawn || g.CanClose || g.StartLabel != "Nuovo lotto" || g.CloseLabel != "Chiudi scarico" {
		t.Fatalf("open empty lots chrome = %#v", g)
	}
	regChrome, ok := worklistTaskByID(intake, "register")
	if !ok || regChrome.Title != "Register" {
		t.Fatalf("nested child chrome on job.Tasks = %#v ok=%v", regChrome, ok)
	}
	if refs := CollectActionableTaskRefs(cfg, wl); hasActionableRef(refs, "intake", "weightCheck", "T1") {
		t.Fatalf("no flat actionable while only group is open: %#v", refs)
	}

	domain, err := SpawnGroupInstance(domain, "intake", "T1", "lots", "L1")
	if err != nil {
		t.Fatal(err)
	}
	domain = withLotsChildSub(domain, "T1", "L1")
	wl = ProjectWorklist(cfg, domain, op)
	inst, _ = worklistInstance(worklistMustJob(t, wl, "intake"), "T1")
	g = inst.Groups["lots"]
	if !g.CanSpawn || !g.CanClose || len(g.Instances) != 1 {
		t.Fatalf("after child submit lots = %#v", g)
	}
	if g.Instances[0].ID != "L1" || g.Instances[0].Tasks["register"] != TaskStatusDone {
		t.Fatalf("group instance chrome = %#v", g.Instances[0])
	}
	if inst.Tasks["weightCheck"] != TaskStatusLocked {
		t.Fatalf("weightCheck before close = %q", inst.Tasks["weightCheck"])
	}

	closed, err := CloseGroup(domain, lots, "intake", "T1")
	if err != nil {
		t.Fatal(err)
	}
	wl = ProjectWorklist(cfg, closed, op)
	inst, _ = worklistInstance(worklistMustJob(t, wl, "intake"), "T1")
	if inst.Tasks["lots"] != TaskStatusDone {
		t.Fatalf("lots after close = %q", inst.Tasks["lots"])
	}
	g = inst.Groups["lots"]
	if g.CanSpawn || g.CanClose {
		t.Fatalf("closed group affordances = %#v", g)
	}
	if inst.Tasks["weightCheck"] != TaskStatusActionable {
		t.Fatalf("weightCheck after close = %q, want actionable", inst.Tasks["weightCheck"])
	}
	if refs := CollectActionableTaskRefs(cfg, wl); !hasActionableRef(refs, "intake", "weightCheck", "T1") {
		t.Fatalf("actionable weightCheck missing from CollectActionable: %#v", refs)
	}
}

func TestProjectWorklistGroupInstanceLabelFromLoadAfterRegister(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	truckID := commitTruckArrive(t, c)
	result := commitLotRegister(t, c, truckID, registerLotValue())
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	wl := ProjectWorklist(mustActiveConfig(t, c), result.Domain, op)
	inst, ok := worklistInstance(worklistMustJob(t, wl, "intake"), truckID)
	if !ok {
		t.Fatal("missing truck instance")
	}
	g, ok := inst.Groups["lots"]
	if !ok || len(g.Instances) != 1 {
		t.Fatalf("lots group = %#v", g)
	}
	gi := g.Instances[0]
	if gi.ID != "lots-1" {
		t.Fatalf("group instance id = %q, want lots-1", gi.ID)
	}
	if gi.Label == "" {
		t.Fatal("expected group instance label from minted load")
	}
	if _, ok := result.Domain.Kind("load")[gi.Label]; !ok {
		t.Fatalf("label %q not present under Domain load: %#v", gi.Label, result.Domain.Kind("load"))
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.GroupInstanceID != "lots-1" || !containsString(last.IDsTouched, gi.Label) {
		t.Fatalf("register submission must touch load id %q: %#v", gi.Label, last)
	}
}

func TestGroupInstanceEntityLabelIgnoresNonMatchingIDsTouched(t *testing.T) {
	domain := EmptyDomain()
	putKind(&domain, "load", EntityMap{"L-real": {"client": "NO"}})
	putKind(&domain, "pallet", EntityMap{"P1": {"loadId": "L-real"}})
	domain.Submissions = []Submission{{
		JobID: "intake", TaskID: "register", JobInstanceID: "T1",
		GroupID: "lots", GroupInstanceID: "lots-1",
		IDsTouched: []string{"P1", "L-real"},
	}}
	got := groupInstanceEntityLabel(domain, "intake", "T1", "lots", "lots-1", "load")
	if got != "L-real" {
		t.Fatalf("label = %q, want L-real (first IDsTouched ∩ entity kind)", got)
	}
	if groupInstanceEntityLabel(domain, "intake", "T1", "lots", "lots-2", "load") != "" {
		t.Fatal("unrelated group instance must omit label")
	}
}

func TestGroupInstanceEntityLabelEmptyWithoutEntityKind(t *testing.T) {
	domain := EmptyDomain()
	putKind(&domain, "load", EntityMap{"L-real": {"client": "NO"}})
	domain.Submissions = []Submission{{
		JobID: "intake", TaskID: "register", JobInstanceID: "T1",
		GroupID: "lots", GroupInstanceID: "lots-1",
		IDsTouched: []string{"L-real"},
	}}
	if groupInstanceEntityLabel(domain, "intake", "T1", "lots", "lots-1", "") != "" {
		t.Fatal("empty entityKind must omit label even when matching entities exist")
	}

	lots := TaskConfig{
		"id": "lots", "kind": "group", "repeatable": true,
		"tasks": []TaskConfig{{"id": "register", "roles": []any{"administration"}}},
	}
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"intake"},
			Values: map[string]any{
				"intake": Job{
					Repeatable: true,
					Tasks: []TaskConfig{
						{"id": "truckArrive", "roles": []any{"administration"}},
						lots,
					},
				},
			},
		},
	}
	domain.JobInstances = []JobInstance{{JobID: "intake", ID: "T1"}}
	domain.GroupInstances = []GroupInstance{{
		JobID: "intake", JobInstanceID: "T1", GroupID: "lots", ID: "lots-1",
	}}
	wl := ProjectWorklist(cfg, domain, OperatorIdentity{UserID: "ada", Org: "x", Roles: []string{"administration"}})
	inst, ok := worklistInstance(worklistMustJob(t, wl, "intake"), "T1")
	if !ok {
		t.Fatal("missing instance")
	}
	g := inst.Groups["lots"]
	if len(g.Instances) != 1 || g.Instances[0].Label != "" {
		t.Fatalf("group without instanceLabelEntity must omit label: %#v", g.Instances)
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

// Synthetic Job/Task ids only — plant ids must not appear in this fold suite.
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
		wl := ProjectWorklist(cfg, domainUnlocked, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusActionable {
			t.Fatalf("omit + eligible = %q, want actionable", inst.Tasks["gated"])
		}
		wlOther := ProjectWorklist(cfg, domainUnlocked, other)
		instOther, _ := worklistInstance(worklistMustJob(t, wlOther, "gateJob"), "I1")
		if instOther.Tasks["gated"] != TaskStatusAvailable {
			t.Fatalf("omit + ineligible = %q, want available", instOther.Tasks["gated"])
		}
	})

	t.Run("literal_true", func(t *testing.T) {
		cfg := cfgFor(leaf("gated", "true"))
		wl := ProjectWorklist(cfg, domainUnlocked, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusActionable {
			t.Fatalf("true = %q, want actionable", inst.Tasks["gated"])
		}
	})

	t.Run("literal_false_skips_role_split", func(t *testing.T) {
		cfg := cfgFor(leaf("gated", "false"))
		wl := ProjectWorklist(cfg, domainUnlocked, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusUnavailable {
			t.Fatalf("false + eligible = %q, want unavailable", inst.Tasks["gated"])
		}
		wlOther := ProjectWorklist(cfg, domainUnlocked, other)
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
		wl := ProjectWorklist(cfg, domainUnlocked, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusActionable {
			t.Fatalf("blank = %q, want actionable", inst.Tasks["gated"])
		}
	})

	t.Run("invalid_fail_closed", func(t *testing.T) {
		cfg := cfgFor(leaf("gated", 1))
		wl := ProjectWorklist(cfg, domainUnlocked, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusUnavailable {
			t.Fatalf("invalid = %q, want unavailable", inst.Tasks["gated"])
		}
	})

	t.Run("domain_expr", func(t *testing.T) {
		cfg := cfgFor(leaf("gated", "$count($domain.widget.*) >= 1"))
		wl := ProjectWorklist(cfg, domainUnlocked, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusUnavailable {
			t.Fatalf("no widget = %q, want unavailable", inst.Tasks["gated"])
		}
		withWidget := domainUnlocked
		putKind(&withWidget, "widget", EntityMap{"W1": {"n": 1}})
		wl = ProjectWorklist(cfg, withWidget, op)
		inst, _ = worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusActionable {
			t.Fatalf("with widget = %q, want actionable", inst.Tasks["gated"])
		}
	})

	t.Run("locked_skips_readyWhen", func(t *testing.T) {
		cfg := cfgFor(leaf("gated", "false"))
		empty := EmptyDomain()
		empty.JobInstances = []JobInstance{{JobID: "gateJob", ID: "I1", Label: "inst"}}
		wl := ProjectWorklist(cfg, empty, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		if inst.Tasks["gated"] != TaskStatusLocked {
			t.Fatalf("locked before unlock = %q, want locked (not unavailable)", inst.Tasks["gated"])
		}
	})

	t.Run("group_child_readyWhen", func(t *testing.T) {
		cfg := OrgConfig{
			Jobs: OrderedObject{
				Keys: []string{"gateJob"},
				Values: map[string]any{
					"gateJob": Job{
						Tasks: []TaskConfig{
							leaf("opener", nil),
							{
								"id": "bundle", "kind": "group",
								"tasks": []TaskConfig{
									{
										"id": "childGate", "roles": []any{"ops"},
										"readyWhen": "false",
										"schema":    map[string]any{"title": "Child"},
									},
								},
							},
						},
					},
				},
			},
		}
		domain := EmptyDomain()
		domain.JobInstances = []JobInstance{{JobID: "gateJob", ID: "I1", Label: "inst"}}
		domain.Submissions = []Submission{sub("gateJob", "opener", "I1")}
		var err error
		domain, err = SpawnGroupInstance(domain, "gateJob", "I1", "bundle", "G1")
		if err != nil {
			t.Fatal(err)
		}
		wl := ProjectWorklist(cfg, domain, op)
		inst, _ := worklistInstance(worklistMustJob(t, wl, "gateJob"), "I1")
		g := inst.Groups["bundle"]
		if len(g.Instances) != 1 || g.Instances[0].Tasks["childGate"] != TaskStatusUnavailable {
			t.Fatalf("group child readyWhen false = %#v", g.Instances)
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
		wl := ProjectWorklist(cfg, EmptyDomain(), op)
		job := worklistMustJob(t, wl, "startJob")
		if job.CanStart {
			t.Fatal("canStart want false when first-Task readyWhen fails")
		}
		withWidget := EmptyDomain()
		putKind(&withWidget, "widget", EntityMap{"W1": {"n": 1}})
		wl = ProjectWorklist(cfg, withWidget, op)
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
		wl = ProjectWorklist(omitCfg, EmptyDomain(), op)
		if !worklistMustJob(t, wl, "startJob").CanStart {
			t.Fatal("omit readyWhen ⇒ canStart true")
		}
	})
}

