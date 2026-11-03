package living

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testCommitter(t *testing.T, auth Authorizer) (*Committer, *MemoryDomainStore, *Loader) {
	t.Helper()
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	if auth == nil {
		auth = AllowAllAuthorizer{}
	}
	c := &Committer{
		Domains:    domains,
		Configs:    configs,
		Authorizer: auth,
		Now:        func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) },
	}
	return c, domains, &Loader{Domains: domains, Configs: configs}
}

func TestCommitTruckArrivePersistsAndLoadReflects(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}

	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	truckID := result.JobInstanceID
	wantID := "261019-1st-1"
	if truckID != wantID {
		t.Fatalf("jobInstanceId = %q, want %q", truckID, wantID)
	}
	if result.Domain.Kind("truck")[truckID]["truckWeight"] != 1000.0 {
		t.Fatalf("truck = %#v", result.Domain.Kind("truck"))
	}
	if len(result.Domain.Submissions) != 1 {
		t.Fatalf("submissions = %#v", result.Domain.Submissions)
	}
	sub := result.Domain.Submissions[0]
	if sub.JobID != "intake" || sub.TaskID != "truckArrive" || sub.JobInstanceID != truckID {
		t.Fatalf("submission path keys = %#v", sub)
	}

	loaded, err := loader.Load(t.Context(), "feragame", FixtureAllRolesOp())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Domain.Kind("truck")[truckID]["lastScaleReading"] != 1000.0 {
		t.Fatalf("loaded domain = %#v", loaded.Domain)
	}
	if len(loaded.Domain.JobInstances) != 1 || loaded.Domain.JobInstances[0].ID != truckID {
		t.Fatalf("jobInstances = %#v", loaded.Domain.JobInstances)
	}
}

func TestCommitLotRegisterMintsLoadAndPallets(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	truckID := commitTruckArrive(t, c)
	result := commitLotRegister(t, c, truckID, registerLotValue())
	loadID := firstLoadID(result.Domain)
	if loadID == "" || result.Domain.Kind("load")[loadID]["client"] != "NO" {
		t.Fatalf("load = %#v", result.Domain.Kind("load"))
	}
	if result.Domain.Kind("load")[loadID]["numeroFir"] != "FIR-1" {
		t.Fatalf("numeroFir missing: %#v", result.Domain.Kind("load")[loadID])
	}
	if result.Domain.Kind("load")[loadID]["netWeight"] != 300.0 {
		t.Fatalf("netWeight = %#v", result.Domain.Kind("load")[loadID]["netWeight"])
	}
	if len(result.Domain.Kind("pallet")) != 2 {
		t.Fatalf("pallets = %#v", result.Domain.Kind("pallet"))
	}
	if result.Domain.Kind("truck")[truckID]["lastScaleReading"] != 700.0 {
		t.Fatalf("truck last reading = %#v", result.Domain.Kind("truck")[truckID])
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "register" || last.GroupID != "lots" {
		t.Fatalf("submission = %#v", last)
	}
}

func TestCommitRejectsWhenAuthorizerDenies(t *testing.T) {
	c, domains, _ := testCommitter(t, DenyAllAuthorizer{})
	_, err := c.Commit(t.Context(), "feragame", OperatorIdentity{UserID: "gio", Org: "feragame"}, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("err = %v", err)
	}
	_, getErr := domains.Get(t.Context(), "feragame")
	if getErr == nil {
		t.Fatal("domain should remain unset when Authorizer denies")
	}
}

func TestCommitRejectsUnsupportedPath(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, err := c.Commit(t.Context(), "feragame", OperatorIdentity{UserID: "ada", Org: "feragame"}, CommitRequest{
		TaskPath:      "intake/notATask",
		JobInstanceID: "261019-1st-1",
		Value:         map[string]any{},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unknown-target" {
		t.Fatalf("err = %v", err)
	}
}

func TestCommitWeightCheckAfterLotsClose(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	truckID, _, _ := commitIntakeThroughLotsClosed(t, c)
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: truckID,
		Value: map[string]any{"checkedTruckWeight": 1000.0, "checkedLotSum": 300.0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Kind("truck")[truckID]["checkedTruckWeight"] != 1000.0 {
		t.Fatalf("truck = %#v", result.Domain.Kind("truck")[truckID])
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "weightCheck" {
		t.Fatalf("submissions = %#v", result.Domain.Submissions)
	}
}

func TestCommitLotsUnlocksAfterArrive(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	truckID := commitTruckArrive(t, c)
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	intake := cfg.Jobs.Values["intake"].(Job)
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(intake.Tasks), JobID: "intake", TaskID: "lots",
		JobInstanceID: truckID, Submissions: domain.Submissions, Tasks: intake.Tasks,
	}) {
		t.Fatal("truckArrive should unlock lots group")
	}
	if IsTaskUnlocked(UnlockQuery{
		OrderedIDs: OrderedTaskIDs(intake.Tasks), JobID: "intake", TaskID: "weightCheck",
		JobInstanceID: truckID, Submissions: domain.Submissions, Tasks: intake.Tasks,
		ClosedGroups: domain.ClosedGroups,
	}) {
		t.Fatal("weightCheck must stay locked until lots close")
	}
}

func TestHandleCommitHappyPath(t *testing.T) {
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
	if wire["jobInstanceId"] != "261019-1st-1" {
		t.Fatalf("jobInstanceId = %#v", wire["jobInstanceId"])
	}
	if _, has := wire["domain"]; has {
		t.Fatalf("Commit JSON must omit domain: %#v", wire["domain"])
	}
	wl, _ := wire["worklist"].(map[string]any)
	if wl == nil {
		t.Fatalf("Commit JSON missing worklist: %#v", wire)
	}

	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if domain.Kind("truck")["261019-1st-1"]["turno"] != "1st" {
		t.Fatalf("stored domain = %#v", domain)
	}

	loadRec := httptest.NewRecorder()
	loadReq := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
	h.ServeHTTP(loadRec, loadReq)
	if loadRec.Code != http.StatusOK {
		t.Fatalf("load status = %d", loadRec.Code)
	}
	var loadWire map[string]any
	if err := json.Unmarshal(loadRec.Body.Bytes(), &loadWire); err != nil {
		t.Fatal(err)
	}
	if _, has := loadWire["domain"]; has {
		t.Fatalf("Load JSON must omit domain after Commit")
	}
	if _, ok := loadWire["worklist"].(map[string]any); !ok {
		t.Fatalf("Load JSON missing worklist: %#v", loadWire)
	}
}

func TestHandleCommitRejectsWhenGateDenies(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	h := &Handler{
		Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return OperatorIdentity{}, false
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(`{}`))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func asCommitError(err error, dest **CommitError) bool {
	if err == nil {
		return false
	}
	ce, ok := err.(*CommitError)
	if !ok {
		return false
	}
	*dest = ce
	return true
}
