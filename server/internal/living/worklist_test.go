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

func worklistJobByID(wl Worklist, jobID string) (WorklistJob, bool) {
	for _, j := range wl.Jobs {
		if j.JobID == jobID {
			return j, true
		}
	}
	return WorklistJob{}, false
}

func worklistInstance(job WorklistJob, id string) (WorklistInstance, bool) {
	for _, inst := range job.Instances {
		if inst.ID == id {
			return inst, true
		}
	}
	return WorklistInstance{}, false
}

func TestLoaderWorklistEmptyDomainIntakeOpenLineBlocked(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	loader := &Loader{Domains: domains, Configs: configs}
	result, err := loader.Load(t.Context(), "feragame")
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

	result, err := loader.Load(t.Context(), "feragame")
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

func TestLoadJSONWorklistLineStartsWhenAndLeftoverSome(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	intakeID := "NIR-261019-NO-1"
	completeIntakeThroughPalletWeight(t, c, intakeID)

	loaded, err := loader.Load(t.Context(), "feragame")
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

	loaded, err = loader.Load(t.Context(), "feragame")
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
	if inst.Tasks["lineLoad"] != TaskStatusAvailable {
		t.Fatalf("lineLoad with leftovers = %q, want available", inst.Tasks["lineLoad"])
	}
	if inst.Tasks["containerReg"] != TaskStatusAvailable {
		t.Fatalf("containerReg = %q, want available", inst.Tasks["containerReg"])
	}

	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "line/lineLoad", JobInstanceID: "1st",
		Value: map[string]any{
			"progress": map[string]any{
				intakeID + ".2": "done",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err = loader.Load(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	line, _ = worklistJobByID(loaded.Worklist, "line")
	inst, _ = worklistInstance(line, "1st")
	if inst.Tasks["lineLoad"] != TaskStatusDone {
		t.Fatalf("lineLoad leftovers clear = %q, want done", inst.Tasks["lineLoad"])
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
	want := ProjectWorklist(mustActiveConfig(t, c), domain)
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
	result, err := c.Commit(t.Context(), "feragame", OperatorIdentity{
		UserID: "ada", Org: "feragame", Roles: []string{"line"},
	}, CommitRequest{
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
	if inst.Tasks["lineLoad"] != TaskStatusAvailable {
		t.Fatalf("lineLoad = %q, want available", inst.Tasks["lineLoad"])
	}
	if inst.Tasks["containerReg"] != TaskStatusAvailable {
		t.Fatalf("containerReg = %q, want available", inst.Tasks["containerReg"])
	}
	want := ProjectWorklist(mustActiveConfig(t, c), result.Domain)
	rawGot, _ := json.Marshal(result.Worklist)
	rawWant, _ := json.Marshal(want)
	if string(rawGot) != string(rawWant) {
		t.Fatalf("Commit worklist != ProjectWorklist(domain)\ngot  %s\nwant %s", rawGot, rawWant)
	}
}
