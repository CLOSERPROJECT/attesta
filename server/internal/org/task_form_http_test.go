package org

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoadTaskBundleProvisionalAndAvailable(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}

	provisional, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "truckArrive", "")
	if err != nil {
		t.Fatal(err)
	}
	if provisional.JobID != "intake" || provisional.TaskID != "truckArrive" || provisional.JobInstanceID != "" {
		t.Fatalf("provisional = %s", bundleAddress(provisional))
	}

	viaNew, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "truckArrive", "new")
	if err != nil {
		t.Fatal(err)
	}
	if viaNew.JobInstanceID != "" {
		t.Fatalf("new must normalize to empty instance; got %q", viaNew.JobInstanceID)
	}

	truckID := commitTruckArrive(t, c)
	spawn, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "register", truckID)
	if err != nil {
		t.Fatal(err)
	}
	if spawn.TaskID != "register" || spawn.JobInstanceID != truckID {
		t.Fatalf("register = %#v", spawn)
	}
}

// Addressed fetch must not depend on LoadResult.TaskFormBundles (home may omit bulk).
func TestLoadTaskBundleIndependentOfHomeBulk(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	truckID := commitTruckArrive(t, c)

	bundle, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "register", truckID)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.TaskID != "register" || bundle.JobInstanceID != truckID {
		t.Fatalf("addressed = %s", bundleAddress(bundle))
	}

	loaded, err := loader.Load(t.Context(), "feragame", admin)
	if err != nil {
		t.Fatal(err)
	}
	loaded.TaskFormBundles = nil
	again, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "register", truckID)
	if err != nil {
		t.Fatal(err)
	}
	if again.TaskID != "register" {
		t.Fatalf("remount after empty bulk = %s", bundleAddress(again))
	}
}

func TestLoadTaskBundleDeniesRoleOutsideBoardScope(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	_ = commitStartLot(t, c, loadID)

	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	_, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "lot_processing", "weigh", loadID)
	if !errors.Is(err, ErrTaskBundleForbidden) {
		t.Fatalf("admin weigh err = %v, want forbidden", err)
	}

	lineOp := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}
	bundle, err := loader.LoadTaskBundle(t.Context(), "feragame", lineOp, "lot_processing", "weigh", loadID)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.TaskID != "weigh" {
		t.Fatalf("line bundle = %s", bundleAddress(bundle))
	}
}

func TestHandleTaskFormHTTP(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	h := &API{
		DomainLoader: loader, TaskCommitter: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return admin, true
		},
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/task-form?jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("provisional status = %d body=%s", rec.Code, rec.Body.String())
	}
	var wire TaskFormBundle
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire.TaskID != "truckArrive" || wire.JobInstanceID != "" {
		t.Fatalf("wire = %#v", wire)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/task-form?jobId=intake&taskId=truckArrive&jobInstanceId=new", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("new provisional = %d", rec.Code)
	}

	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	_ = commitStartLot(t, c, loadID)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/task-form?jobId=lot_processing&taskId=weigh&jobInstanceId="+loadID, nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin weigh = %d, want 403", rec.Code)
	}

	lineH := &API{
		DomainLoader: loader, TaskCommitter: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}, true
		},
	}
	rec = httptest.NewRecorder()
	lineH.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/task-form?jobId=lot_processing&taskId=weigh&jobInstanceId="+loadID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("line weigh = %d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/task-form?taskId=truckArrive", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing jobId = %d", rec.Code)
	}
}

func TestCommitNextTaskFormBundleAdvanceHandoffFinished(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}

	arrive, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath: "intake/truckArrive", Value: truckArriveValue(), Mint: startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	truckID := arrive.JobInstanceID
	// Open group first leaf with downloads → stay/home law; without FakeHttp links → home.
	if arrive.Navigation != CommitNextHome && arrive.Navigation != CommitNextTask {
		t.Fatalf("after arrive Navigation = %q", arrive.Navigation)
	}

	lot, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: truckID,
		Value:         registerLotValue(),
	})
	if err != nil {
		t.Fatal(err)
	}
	loadID := firstLoadID(lot.Domain)
	if loadID == "" {
		t.Fatal("expected lot")
	}
	if lot.Navigation != CommitNextStay {
		t.Fatalf("after lot Navigation = %q, want stay", lot.Navigation)
	}
	if lot.ContinueNavigation != CommitNextTask {
		t.Fatalf("after lot ContinueNavigation = %q, want task (weightCheck)", lot.ContinueNavigation)
	}

	done, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: truckID,
		Value: map[string]any{"checkedTruckWeight": 1000.0, "checkedLotSum": 300.0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if done.Navigation != CommitNextHome {
		t.Fatalf("finished Navigation = %q, want home", done.Navigation)
	}

	_ = commitStartLot(t, c, loadID)
	lineOp := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}
	pw, err := c.Commit(t.Context(), "feragame", lineOp, CommitRequest{
		TaskPath:      "lot_processing/weigh",
		JobInstanceID: loadID,
		Value:         weighValue(loadID+".1", 11),
	})
	if err != nil {
		t.Fatal(err)
	}
	if pw.Navigation != CommitNextStay && pw.Navigation != CommitNextHome && pw.Navigation != CommitNextTask {
		t.Fatalf("line Navigation = %q", pw.Navigation)
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
