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

	provisional, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "truckArrive", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if provisional.JobID != "intake" || provisional.TaskID != "truckArrive" || provisional.JobInstanceID != "" {
		t.Fatalf("provisional = %s", bundleAddress(provisional))
	}

	viaNew, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "truckArrive", "new", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if viaNew.JobInstanceID != "" {
		t.Fatalf("new must normalize to empty instance; got %q", viaNew.JobInstanceID)
	}

	truckID := commitTruckArrive(t, c)
	spawn, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "register", truckID, "lots", ProvisionalGroupInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	if spawn.TaskID != "register" || spawn.GroupID != "lots" || spawn.JobInstanceID != truckID {
		t.Fatalf("lots/register = %#v", spawn)
	}
}

// Addressed fetch must not depend on LoadResult.TaskFormBundles (home may omit bulk).
func TestLoadTaskBundleIndependentOfHomeBulk(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	truckID := commitTruckArrive(t, c)

	bundle, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "register", truckID, "lots", ProvisionalGroupInstanceID)
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
	again, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "intake", "register", truckID, "lots", ProvisionalGroupInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	if again.TaskID != "register" {
		t.Fatalf("remount after empty bulk = %s", bundleAddress(again))
	}
}

func TestLoadTaskBundleDeniesRoleOutsideBoardScope(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	_, _, _ = commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)

	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	_, err := loader.LoadTaskBundle(t.Context(), "feragame", admin, "line", "weigh", lineID, "palletWeight", ProvisionalGroupInstanceID)
	if !errors.Is(err, ErrTaskBundleForbidden) {
		t.Fatalf("admin weigh err = %v, want forbidden", err)
	}

	logistic := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}
	bundle, err := loader.LoadTaskBundle(t.Context(), "feragame", logistic, "line", "weigh", lineID, "palletWeight", ProvisionalGroupInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.TaskID != "weigh" {
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
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-form?jobId=intake&taskId=truckArrive", nil))
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
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-form?jobId=intake&taskId=truckArrive&jobInstanceId=new", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("new provisional = %d", rec.Code)
	}

	_, _, _ = commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/living/task-form?jobId=line&taskId=weigh&jobInstanceId="+lineID+"&groupId=palletWeight&groupInstanceId="+ProvisionalGroupInstanceID, nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin weigh = %d, want 403", rec.Code)
	}

	logH := &Handler{
		Loader: loader, Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}, true
		},
	}
	rec = httptest.NewRecorder()
	logH.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/living/task-form?jobId=line&taskId=weigh&jobInstanceId="+lineID+"&groupId=palletWeight&groupInstanceId="+ProvisionalGroupInstanceID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("logistic weigh = %d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-form?taskId=truckArrive", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing jobId = %d", rec.Code)
	}
}

func TestCommitNextTaskFormBundleAdvanceHandoffFinished(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	admin := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	logistic := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}

	arrive, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath: "intake/truckArrive", Value: truckArriveValue(), Mint: startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	truckID := arrive.JobInstanceID
	// Open group first leaf with downloads → stay/overview law; without FakeHttp links → overview.
	if arrive.Next != CommitNextOverview && arrive.Next != CommitNextTask {
		t.Fatalf("after arrive Next = %q", arrive.Next)
	}

	lot, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath:      GroupTaskPath("intake", "lots", ProvisionalGroupInstanceID, "register"),
		JobInstanceID: truckID,
		Value:         registerLotValue(),
		Mint:          spawningNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	loadID := firstLoadID(lot.Domain)
	if loadID == "" {
		t.Fatal("expected load")
	}
	if lot.Next != CommitNextOverview && lot.Next != CommitNextTask {
		t.Fatalf("after lot Next = %q", lot.Next)
	}

	closed, err := c.CloseGroup(t.Context(), "feragame", admin, CloseGroupRequest{
		JobID: "intake", JobInstanceID: truckID, GroupID: "lots",
	})
	if err != nil {
		t.Fatal(err)
	}
	if closed.Next != CommitNextTask {
		t.Fatalf("after close Next = %q, want task", closed.Next)
	}
	if closed.NextTaskFormBundle == nil || closed.NextTaskFormBundle.TaskID != "weightCheck" {
		t.Fatalf("after close bundle = %s, want weightCheck", bundleAddress(closed.NextTaskFormBundle))
	}

	done, err := c.Commit(t.Context(), "feragame", admin, CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: truckID,
		Value: map[string]any{"checkedTruckWeight": 1000.0, "checkedLotSum": 300.0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if done.Next != CommitNextHome && done.Next != CommitNextOverview {
		t.Fatalf("finished Next = %q", done.Next)
	}

	lineID := commitOpenTurno(t, c)
	pw, err := c.Commit(t.Context(), "feragame", logistic, CommitRequest{
		TaskPath:      GroupTaskPath("line", "palletWeight", ProvisionalGroupInstanceID, "weigh"),
		JobInstanceID: lineID,
		Value:         weighValue(loadID+".1", 11),
		Mint:          spawningNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if pw.Next != CommitNextOverview && pw.Next != CommitNextTask {
		t.Fatalf("logistic Next = %q", pw.Next)
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
