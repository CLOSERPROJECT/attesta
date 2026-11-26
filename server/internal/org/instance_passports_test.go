package org

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/untp"
)

func TestCollectInstancePassportsScopesToStreamInstance(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	passports := NewMemoryDPPStore()
	c.DPPs = passports
	c.Now = func() time.Time { return time.Date(2026, 11, 4, 14, 0, 0, 0, time.UTC) }

	loadA, _ := seedFullyProcessedLoad(t, c)
	domain, err := c.Domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	truckA, _ := domain.Kind("lot")[loadA]["truckId"].(string)
	if truckA == "" {
		t.Fatal("expected truckId on locked lot")
	}
	truckB := commitTruckArrive(t, c)
	_ = commitLotRegister(t, c, truckB, registerLotValue())

	locked, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath:         "lot_processing/lock",
		JobInstanceID:    loadA,
		Value:            map[string]any{"ack": true},
		StreamInstanceID: truckA,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(locked.Links) != 1 {
		t.Fatalf("links = %#v", locked.Links)
	}

	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	domain, err = c.Domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}

	focusA, err := TimelineInstance(truckA)
	if err != nil {
		t.Fatal(err)
	}
	gotA, err := CollectInstancePassports(t.Context(), cfg, domain, focusA, passports)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotA) != 1 || gotA[0].SubjectID != loadA {
		t.Fatalf("truck A passports = %#v, want subject %s", gotA, loadA)
	}
	wantLink := untp.URL("09506000134352", "FIR-1", loadA)
	if gotA[0].DigitalLink != wantLink {
		t.Fatalf("digitalLink = %q, want %q", gotA[0].DigitalLink, wantLink)
	}

	focusB, err := TimelineInstance(truckB)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := CollectInstancePassports(t.Context(), cfg, domain, focusB, passports)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotB) != 0 {
		t.Fatalf("truck B passports = %#v, want none", gotB)
	}

	boot, err := CollectInstancePassports(t.Context(), cfg, domain, TimelineBoot(), passports)
	if err != nil {
		t.Fatal(err)
	}
	if len(boot) != 0 {
		t.Fatalf("boot passports = %#v", boot)
	}

	cfgNoDPP := cfg
	cfgNoDPP.DPP = nil
	empty, err := CollectInstancePassports(t.Context(), cfgNoDPP, domain, focusA, passports)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("no passport block = %#v", empty)
	}
}

func TestHandlePassportsScopedToStreamInstance(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	passports := NewMemoryDPPStore()
	c.DPPs = passports
	c.Now = func() time.Time { return time.Date(2026, 11, 4, 14, 0, 0, 0, time.UTC) }

	loadA, _ := seedFullyProcessedLoad(t, c)
	domain, err := c.Domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	truckA, _ := domain.Kind("lot")[loadA]["truckId"].(string)
	if truckA == "" {
		t.Fatal("expected truckId")
	}
	truckB := commitTruckArrive(t, c)
	if _, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath:         "lot_processing/lock",
		JobInstanceID:    loadA,
		Value:            map[string]any{"ack": true},
		StreamInstanceID: truckA,
	}); err != nil {
		t.Fatal(err)
	}

	h := &API{
		DomainLoader: loader,
		DPPs:         passports,
		Gate: func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
			return adminOp(), true
		},
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/passports?stream=pv-recovery&instance="+truckA, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got InstancePassportsView
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Passports) != 1 || got.Passports[0].SubjectID != loadA {
		t.Fatalf("passports = %#v", got.Passports)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/passports?stream=pv-recovery&instance="+truckB, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("truck B status = %d body=%s", rec.Code, rec.Body.String())
	}
	var empty InstancePassportsView
	if err := json.Unmarshal(rec.Body.Bytes(), &empty); err != nil {
		t.Fatal(err)
	}
	if len(empty.Passports) != 0 {
		t.Fatalf("truck B = %#v", empty.Passports)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/passports?stream=pv-recovery", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing instance = %d, want 400", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/passports?stream=pv-recovery&instance=missing-truck", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing instance = %d, want 404", rec.Code)
	}
}

func TestHandlePassportsNilStoreReturnsEmpty(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truckA := commitTruckArrive(t, c)
	h := &API{
		DomainLoader: loader,
		Gate: func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
			return adminOp(), true
		},
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/passports?stream=pv-recovery&instance="+truckA, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got InstancePassportsView
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Passports == nil {
		t.Fatal("want empty slice, got nil")
	}
	if len(got.Passports) != 0 {
		t.Fatalf("got = %#v", got.Passports)
	}
}
