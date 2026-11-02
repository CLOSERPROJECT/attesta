package living

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoadTaskBundleProvisionalAndAvailable(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}

	provisional, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "register", "")
	if err != nil {
		t.Fatal(err)
	}
	if provisional.JobID != "intake" || provisional.TaskID != "register" || provisional.JobInstanceID != "" {
		t.Fatalf("provisional = %s", bundleAddress(provisional))
	}

	viaNew, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "register", "new")
	if err != nil {
		t.Fatal(err)
	}
	if viaNew.JobInstanceID != "" {
		t.Fatalf("new must normalize to empty instance; got %q", viaNew.JobInstanceID)
	}

	instanceID := "NIR-261019-NO-1"
	if _, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath: "intake/register", JobInstanceID: instanceID, Value: registerValue(), Mint: startingNewMint(),
	}); err != nil {
		t.Fatal(err)
	}

	codice, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "codiceFir", instanceID)
	if err != nil {
		t.Fatal(err)
	}
	if codice.TaskID != "codiceFir" || codice.JobInstanceID != instanceID {
		t.Fatalf("codice = %s", bundleAddress(codice))
	}
}

// Addressed fetch must not depend on LoadResult.TaskFormBundles (home may omit bulk).
func TestLoadTaskBundleIndependentOfHomeBulk(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	instanceID := "NIR-261019-NO-1"
	if _, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath: "intake/register", JobInstanceID: instanceID, Value: registerValue(), Mint: startingNewMint(),
	}); err != nil {
		t.Fatal(err)
	}

	bundle, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "codiceFir", instanceID)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.TaskID != "codiceFir" || bundle.JobInstanceID != instanceID {
		t.Fatalf("addressed = %s", bundleAddress(bundle))
	}

	// Simulate ADR-0006 end-state omit on a prior Load payload; remount must still work.
	loaded, err := loader.Load(t.Context(), "feragame", admin)
	if err != nil {
		t.Fatal(err)
	}
	loaded.TaskFormBundles = nil
	again, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "codiceFir", instanceID)
	if err != nil {
		t.Fatal(err)
	}
	if again.TaskID != "codiceFir" {
		t.Fatalf("remount after empty bulk = %s", bundleAddress(again))
	}
}

func TestLoadTaskBundleDeniesRoleOutsideBoardScope(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	instanceID := "NIR-261019-NO-1"
	commitThroughCodiceFir(t, c, instanceID)

	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	_, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "palletWeight", instanceID)
	if !errors.Is(err, ErrTaskBundleForbidden) {
		t.Fatalf("admin palletWeight err = %v, want forbidden", err)
	}

	logistic := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}
	bundle, err := loader.LoadTaskBundle(t.Context(), "feragame", logistic, "intake", "palletWeight", instanceID)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.TaskID != "palletWeight" {
		t.Fatalf("logistic bundle = %s", bundleAddress(bundle))
	}
}

func TestHandleTaskFormHTTP(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	h := &Handler{
		Loader: loader, Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return admin, true
		},
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-form?jobId=intake&taskId=register", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("provisional status = %d body=%s", rec.Code, rec.Body.String())
	}
	var wire TaskFormBundle
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire.TaskID != "register" || wire.JobInstanceID != "" {
		t.Fatalf("wire = %#v", wire)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-form?jobId=intake&taskId=register&jobInstanceId=new", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("new provisional = %d", rec.Code)
	}

	instanceID := "NIR-261019-NO-1"
	commitThroughCodiceFir(t, c, instanceID)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/living/task-form?jobId=intake&taskId=palletWeight&jobInstanceId="+instanceID, nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin palletWeight = %d, want 403", rec.Code)
	}

	logH := &Handler{
		Loader: loader, Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}, true
		},
	}
	rec = httptest.NewRecorder()
	logH.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/living/task-form?jobId=intake&taskId=palletWeight&jobInstanceId="+instanceID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("logistic palletWeight = %d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-form?taskId=register", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing jobId = %d", rec.Code)
	}
}

func TestCommitNextTaskFormBundleAdvanceHandoffFinished(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	logistic := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}
	instanceID := "NIR-261019-NO-1"

	// Advance: register → next codiceFir for administration.
	reg, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath: "intake/register", JobInstanceID: instanceID, Value: registerValue(), Mint: startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if reg.NextTaskFormBundle == nil || reg.NextTaskFormBundle.TaskID != "codiceFir" {
		t.Fatalf("advance next = %s, want codiceFir", bundleAddress(reg.NextTaskFormBundle))
	}
	if reg.NextTaskFormBundle.JobInstanceID != instanceID {
		t.Fatalf("advance instance = %q", reg.NextTaskFormBundle.JobInstanceID)
	}
	if _, ok := worklistJobByID(reg.Worklist, "intake"); !ok {
		t.Fatal("advance Commit must return Worklist")
	}
	raw, err := json.Marshal(reg)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	nextWire, ok := wire["nextTaskFormBundle"].(map[string]any)
	if !ok || nextWire["taskId"] != "codiceFir" {
		t.Fatalf("advance JSON next = %#v", wire["nextTaskFormBundle"])
	}

	// Handoff: codiceFir done → palletWeight available but logistic-only.
	fir, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath: "intake/codiceFir", JobInstanceID: instanceID,
		Value: map[string]any{"codiceFir": "FIR-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if fir.NextTaskFormBundle != nil {
		t.Fatalf("handoff next must be nil; got %s", bundleAddress(fir.NextTaskFormBundle))
	}
	intake, ok := worklistJobByID(fir.Worklist, "intake")
	if !ok {
		t.Fatal("handoff missing intake worklist")
	}
	inst, ok := worklistInstance(intake, instanceID)
	if !ok || inst.Tasks["palletWeight"] != TaskStatusAvailable {
		t.Fatalf("handoff palletWeight status = %#v", inst)
	}
	if _, has := mustCommitWire(t, fir)["nextTaskFormBundle"]; has {
		t.Fatal("handoff JSON must omit nextTaskFormBundle")
	}

	// Logistic completes palletWeight → handoff to administration weightCheck.
	pw, err := c.Commit(t.Context(), "feragame", logistic, CommitRequest{
		TaskPath: "intake/palletWeight", JobInstanceID: instanceID,
		Value: palletWeightValue(map[string]any{
			instanceID + ".1": weighRow(11),
			instanceID + ".2": weighRow(12),
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if pw.NextTaskFormBundle != nil {
		t.Fatalf("logistic after palletWeight must not advance; got %s", bundleAddress(pw.NextTaskFormBundle))
	}

	// Finished instance: admin completes weightCheck → no further advance.
	done, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: instanceID,
		Value: map[string]any{"checkedTruckWeight": 100.0, "checkedPalletSum": 23.0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if done.NextTaskFormBundle != nil {
		t.Fatalf("finished next must be nil; got %s", bundleAddress(done.NextTaskFormBundle))
	}
	intake, ok = worklistJobByID(done.Worklist, "intake")
	if !ok {
		t.Fatal("finished missing intake")
	}
	inst, ok = worklistInstance(intake, instanceID)
	if !ok {
		t.Fatal("finished missing instance")
	}
	for _, tid := range intake.TaskIDs {
		if inst.Tasks[tid] != TaskStatusDone {
			t.Fatalf("finished %s = %q, want done", tid, inst.Tasks[tid])
		}
	}
}

func mustCommitWire(t *testing.T, result CommitResult) map[string]any {
	t.Helper()
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	return wire
}
