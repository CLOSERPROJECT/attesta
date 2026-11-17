package org

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testTaskCommitter(t *testing.T, auth Authorizer) (*TaskCommitter, *MemoryDomainStore, *DomainLoader) {
	t.Helper()
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	if auth == nil {
		auth = AllowAllAuthorizer{}
	}
	c := &TaskCommitter{
		Domains:        domains,
		Configs:        configs,
		Authorizer:     auth,
		SubmitBindings: ExprBindings{"qrEndpoint": "https://example.test/app"},
		Now:            func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) },
	}
	return c, domains, &DomainLoader{Domains: domains, Configs: configs}
}

func TestCommitTruckArrivePersistsAndLoadReflects(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
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
	wantID := "261029-1"
	if truckID != wantID {
		t.Fatalf("jobInstanceId = %q, want %q", truckID, wantID)
	}
	if result.Domain.Kind("truck")[truckID]["truckWeight"] != 1000.0 {
		t.Fatalf("truck = %#v", result.Domain.Kind("truck"))
	}
	if len(result.Domain.Submissions) != 1 {
		t.Fatalf("submissions = %#v", result.Domain.Submissions)
	}
	sub := result.Domain.Submissions[len(result.Domain.Submissions)-1]
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
	var sawTruck bool
	for _, inst := range loaded.Domain.JobInstances {
		if inst.JobID == "intake" && inst.ID == truckID {
			sawTruck = true
		}
	}
	if !sawTruck || len(loaded.Domain.JobInstances) != 1 {
		t.Fatalf("jobInstances = %#v", loaded.Domain.JobInstances)
	}
}

func TestCommitLotRegisterMintsLoadAndPallets(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID := commitTruckArrive(t, c)
	result := commitLotRegister(t, c, truckID, registerLotValue())
	loadID := firstLoadID(result.Domain)
	if loadID == "" || result.Domain.Kind("lot")[loadID]["client"] != "CO" {
		t.Fatalf("load = %#v", result.Domain.Kind("lot"))
	}
	if result.Domain.Kind("lot")[loadID]["numeroFir"] != "FIR-1" {
		t.Fatalf("numeroFir missing: %#v", result.Domain.Kind("lot")[loadID])
	}
	if result.Domain.Kind("lot")[loadID]["netWeight"] != 300.0 {
		t.Fatalf("netWeight = %#v", result.Domain.Kind("lot")[loadID]["netWeight"])
	}
	if len(result.Domain.Kind("pallet")) != 2 {
		t.Fatalf("pallets = %#v", result.Domain.Kind("pallet"))
	}
	if result.Domain.Kind("truck")[truckID]["lastScaleReading"] != 700.0 {
		t.Fatalf("truck last reading = %#v", result.Domain.Kind("truck")[truckID])
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "register" {
		t.Fatalf("submission = %#v", last)
	}
}

func TestCommitRejectsWhenAuthorizerDenies(t *testing.T) {
	c, domains, _ := testTaskCommitter(t, DenyAllAuthorizer{})
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
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	_, err := c.Commit(t.Context(), "feragame", OperatorIdentity{UserID: "ada", Org: "feragame"}, CommitRequest{
		TaskPath:      "intake/notATask",
		JobInstanceID: "261029-1",
		Value:         map[string]any{},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unknown-target" {
		t.Fatalf("err = %v", err)
	}
}

func TestCommitWeightCheckAfterRegister(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
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

func TestCommitRegisterReadyWhenAfterArrive(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID := commitTruckArrive(t, c)
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if !sequenceOkForCommit(cfg, domain, "intake", "register", truckID) {
		t.Fatal("register sequenceOk after arrive (Done/startsWhen only)")
	}
	if !sequenceOkForCommit(cfg, domain, "intake", "weightCheck", truckID) {
		t.Fatal("weightCheck sequenceOk after arrive (Unlock removed; readyWhen separate)")
	}
	wl := ProjectWorklist(cfg, domain, OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}})
	inst, ok := worklistInstance(worklistMustJob(t, wl, "intake"), truckID)
	if !ok {
		t.Fatal("missing intake instance")
	}
	if inst.Tasks["register"] != TaskStatusActionable {
		t.Fatalf("register = %q, want actionable after arrive readyWhen", inst.Tasks["register"])
	}
	if inst.Tasks["weightCheck"] != TaskStatusUnavailable {
		t.Fatalf("weightCheck = %q, want unavailable until ≥1 lot (readyWhen)", inst.Tasks["weightCheck"])
	}
}

func TestHandleCommitHappyPath(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	h := &API{
		DomainLoader:    loader,
		TaskCommitter: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}, true
		},
	}

	body := `{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"truckWeight":1000}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/org/commit", strings.NewReader(body))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var wire map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire["jobInstanceId"] != "261029-1" {
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
	truck := domain.Kind("truck")["261029-1"]
	if truck["truckWeight"] != 1000.0 {
		t.Fatalf("stored domain = %#v", domain)
	}
	if _, has := truck["workingDayId"]; has {
		t.Fatalf("truck must not stamp workingDayId: %#v", truck)
	}

	loadRec := httptest.NewRecorder()
	loadReq := httptest.NewRequest(http.MethodGet, "/api/org/domain", nil)
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
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	h := &API{
		TaskCommitter: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return OperatorIdentity{}, false
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/org/commit", strings.NewReader(`{}`))
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
