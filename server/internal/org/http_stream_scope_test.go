package org

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleLoadScopedToStreamInstance(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truckA := commitTruckArrive(t, c)
	truckB := commitTruckArrive(t, c)
	_ = commitLotRegister(t, c, truckA, registerLotValue())

	h := &API{
		DomainLoader:  loader,
		TaskCommitter: c,
		Gate: func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
			return adminOp(), true
		},
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/domain?stream=recycling&instance="+truckA, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got LoadResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	intake, ok := worklistJobByID(got.Worklist, "intake")
	if !ok || len(intake.Instances) != 1 || intake.Instances[0].ID != truckA {
		t.Fatalf("scoped intake = %#v (want only %s)", intake, truckA)
	}
	for _, inst := range intake.Instances {
		if inst.ID == truckB {
			t.Fatalf("truck B leaked into truck A load: %#v", intake.Instances)
		}
	}
}

func TestHandleTaskFormPlayWithoutInstance(t *testing.T) {
	_, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	h := &API{
		DomainLoader: loader,
		Gate: func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
			return adminOp(), true
		},
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/task-form?stream=recycling&jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("Play task-form = %d body=%s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/task-form?stream=recycling&jobId=intake&taskId=register", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("non-Play without instance = %d, want 400", rec.Code)
	}
}

func TestHandleCommitPlayWithoutInstance(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	h := &API{DomainLoader: loader, TaskCommitter: c, Gate: func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
		return adminOp(), true
	}}
	body := `{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"truckWeight":1000}}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/commit?stream=recycling", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("Play commit = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleQrResolveColdScanRequiresStreamOnly(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	_ = commitStartLot(t, c, loadID)
	h := &API{
		DomainLoader:  loader,
		TaskCommitter: c,
		Gate: func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}, true
		},
	}
	palletID := loadID + ".1"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/qr-resolve?stream=recycling",
		strings.NewReader(`{"raw":"https://plant.example/app?entityId=`+palletID+`"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("cold scan = %d body=%s", rec.Code, rec.Body.String())
	}
	var wire QrResolveResult
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Navigate == nil || wire.Navigate.StreamInstanceID != truckID {
		t.Fatalf("cold scan navigate = %#v, want stream instance %s", wire.Navigate, truckID)
	}
}

func TestHandleCommitRejectsNonMemberJobInstance(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truckA := commitTruckArrive(t, c)
	truckB := commitTruckArrive(t, c)
	h := &API{DomainLoader: loader, TaskCommitter: c, Gate: func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
		return adminOp(), true
	}}
	body := `{"taskPath":"intake/register","jobInstanceId":"` + truckB + `","value":` + mustJSON(t, registerLotValue()) + `}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost,
		"/api/org/commit?stream=recycling&instance="+truckA, strings.NewReader(body)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("non-member = %d body=%s, want 409", rec.Code, rec.Body.String())
	}
}

func TestHandleCommitStreamIsolation(t *testing.T) {
	configs := NewMemoryConfigStore()
	domains := NewMemoryDomainStore()
	recycling, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	other, err := LoadSecondStreamConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	other.Org = recycling.Org
	recScope, err := ScopeFromConfig(recycling)
	if err != nil {
		t.Fatal(err)
	}
	otherScope, err := ScopeFromConfig(other)
	if err != nil {
		t.Fatal(err)
	}
	if err := configs.Put(t.Context(), recScope, recycling.Version, recycling); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), recScope, recycling.Version); err != nil {
		t.Fatal(err)
	}
	if err := configs.Put(t.Context(), otherScope, other.Version, other); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), otherScope, other.Version); err != nil {
		t.Fatal(err)
	}
	if err := domains.Put(t.Context(), otherScope, EmptyDomain()); err != nil {
		t.Fatal(err)
	}

	if err := domains.Put(t.Context(), recScope, EmptyDomain()); err != nil {
		t.Fatal(err)
	}
	c := &TaskCommitter{Domains: domains, Configs: configs, Authorizer: AllowAllAuthorizer{}, Now: func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) }}
	loader := &DomainLoader{Domains: domains, Configs: configs}

	h := &API{DomainLoader: loader, TaskCommitter: c, Gate: func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
		return adminOp(), true
	}}
	body := `{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"truckWeight":1000}}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/commit?stream=recycling", strings.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("recycling commit = %d body=%s", rec.Code, rec.Body.String())
	}
	otherDomain, err := domains.Get(t.Context(), otherScope)
	if err != nil {
		t.Fatal(err)
	}
	if len(otherDomain.JobInstances) != 0 || len(otherDomain.Kind("item")) != 0 || len(otherDomain.Kind("truck")) != 0 {
		t.Fatalf("other stream domain touched: %#v", otherDomain)
	}
	recDomain, err := domains.Get(t.Context(), recScope)
	if err != nil {
		t.Fatal(err)
	}
	if len(recDomain.JobInstances) == 0 {
		t.Fatal("recycling domain missing commit")
	}
}

func TestHandleStreamsListsOrgStreams(t *testing.T) {
	configs := NewMemoryConfigStore()
	domains := NewMemoryDomainStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	other, err := LoadSecondStreamConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	other.Org = "feragame"
	otherScope, err := ScopeFromConfig(other)
	if err != nil {
		t.Fatal(err)
	}
	if err := configs.Put(t.Context(), otherScope, other.Version, other); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), otherScope, other.Version); err != nil {
		t.Fatal(err)
	}
	h := &API{
		DomainLoader: &DomainLoader{Domains: domains, Configs: configs},
		Gate: func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
			return adminOp(), true
		},
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/streams", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got []StreamListItem
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("streams = %#v", got)
	}
	byKey := map[string]string{}
	for _, s := range got {
		byKey[s.Key] = s.Title
	}
	if byKey["recycling"] == "" || byKey["other"] != "Other stream" {
		t.Fatalf("streams = %#v", got)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
