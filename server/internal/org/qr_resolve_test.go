package org

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFixtureIncludesLockedQrRules(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.QR == nil || len(cfg.QR.Rules) != 1 {
		t.Fatalf("qr.rules = %#v", cfg.QR)
	}
	rule := cfg.QR.Rules[0]
	if rule.ID != "lot-pallet-weigh" {
		t.Fatalf("rule id = %q", rule.ID)
	}
	if rule.Match.JobID != "lot_processing" || rule.Match.TaskID != "weigh" {
		t.Fatalf("match = %#v", rule.Match)
	}
	if rule.Seed["palletId"] == "" {
		t.Fatalf("seed = %#v", rule.Seed)
	}
	if rule.Messages.OnZero == "" || rule.Messages.OnMany == "" || rule.Messages.OnIneligible == "" {
		t.Fatalf("messages = %#v", rule.Messages)
	}
}

func TestResolveQROneCandidateSeedsPalletID(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	_ = commitStartLot(t, c, loadID)
	cfg := mustActiveConfig(t, c)
	domain := mustDomain(t, c)
	op := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}
	palletID := loadID + ".1"

	got := ResolveQR(cfg, domain, op, "https://plant.example/app?entityId="+palletID)
	if got.Error != nil {
		t.Fatalf("error = %#v", got.Error)
	}
	if got.Navigate == nil {
		t.Fatal("expected navigate")
	}
	if got.Navigate.JobID != "lot_processing" || got.Navigate.JobInstanceID != loadID ||
		got.Navigate.TaskID != "weigh" {
		t.Fatalf("navigate = %#v", got.Navigate)
	}
	if got.Navigate.StreamInstanceID != truckID {
		t.Fatalf("streamInstanceId = %q, want truck %q", got.Navigate.StreamInstanceID, truckID)
	}
	if got.Seed["palletId"] != palletID {
		t.Fatalf("seed = %#v", got.Seed)
	}
}

func TestResolveQRReturnsStreamInstanceWithinStreamAndColdScan(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	_ = commitStartLot(t, c, loadID)
	cfg := mustActiveConfig(t, c)
	domain := mustDomain(t, c)
	op := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}
	palletID := loadID + ".1"

	t.Run("within_stream_bare_id", func(t *testing.T) {
		got := ResolveQR(cfg, domain, op, palletID)
		if got.Error != nil || got.Navigate == nil {
			t.Fatalf("got = %#v", got)
		}
		if got.Navigate.StreamInstanceID != truckID {
			t.Fatalf("streamInstanceId = %q, want %q", got.Navigate.StreamInstanceID, truckID)
		}
	})

	t.Run("cold_scan_url_no_current_instance", func(t *testing.T) {
		// Server resolve has no current-instance input; cold scan just returns the candidate's truck.
		got := ResolveQR(cfg, domain, op, "https://plant.example/app?entityId="+palletID)
		if got.Error != nil || got.Navigate == nil {
			t.Fatalf("got = %#v", got)
		}
		if got.Navigate.StreamInstanceID != truckID {
			t.Fatalf("streamInstanceId = %q, want %q", got.Navigate.StreamInstanceID, truckID)
		}
	})
}

func TestResolveQRCrossTruckReturnsOtherStreamInstance(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	op := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}

	domain := EmptyDomain()
	domain.Entities = map[string]EntityMap{
		"truck": {
			"T-A": {"truckWeight": 1000.0},
			"T-B": {"truckWeight": 1100.0},
		},
		"lot": {
			"L-A1": {"truckId": "T-A", "status": "registered", "client": "A"},
			"L-B1": {"truckId": "T-B", "status": "open", "client": "B"},
		},
		"pallet": {
			"L-B1.1": {"lotId": "L-B1"},
		},
	}
	domain.JobInstances = []JobInstance{
		{JobID: "intake", ID: "T-A", Label: "T-A"},
		{JobID: "intake", ID: "T-B", Label: "T-B"},
		{JobID: "lot_processing", ID: "L-B1", Label: "Lotto: L-B1"},
	}
	domain.Submissions = []Submission{
		sub("lot_processing", "startLot", "L-B1"),
	}

	// Filtered to T-A must not surface truck B's weigh; unfiltered resolve still finds it.
	wlA := ProjectWorklist(cfg, domain, op, "T-A")
	if lp, ok := worklistJobByID(wlA, "lot_processing"); ok {
		for _, inst := range lp.Instances {
			if inst.ID == "L-B1" {
				t.Fatal("filtered T-A Worklist must not include L-B1")
			}
		}
	}

	got := ResolveQR(cfg, domain, op, "L-B1.1")
	if got.Error != nil || got.Navigate == nil {
		t.Fatalf("got = %#v", got)
	}
	if got.Navigate.JobID != "lot_processing" || got.Navigate.JobInstanceID != "L-B1" ||
		got.Navigate.TaskID != "weigh" {
		t.Fatalf("navigate = %#v", got.Navigate)
	}
	if got.Navigate.StreamInstanceID != "T-B" {
		t.Fatalf("streamInstanceId = %q, want T-B (other truck)", got.Navigate.StreamInstanceID)
	}
	if got.Seed["palletId"] != "L-B1.1" {
		t.Fatalf("seed = %#v", got.Seed)
	}
}

func TestResolveQRIneligibleWhenPalletMissingOrWeighed(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	_ = commitStartLot(t, c, loadID)
	_ = commitWeigh(t, c, loadID, loadID+".1", 11)
	cfg := mustActiveConfig(t, c)
	domain := mustDomain(t, c)
	op := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}

	missing := ResolveQR(cfg, domain, op, "missing-pallet")
	if missing.Error == nil || missing.Error.Code != "ineligible" {
		t.Fatalf("missing = %#v", missing)
	}
	if missing.Error.Message != cfg.QR.Rules[0].Messages.OnIneligible {
		t.Fatalf("missing message = %q", missing.Error.Message)
	}

	weighed := ResolveQR(cfg, domain, op, loadID+".1")
	if weighed.Error == nil || weighed.Error.Code != "ineligible" {
		t.Fatalf("weighed = %#v", weighed)
	}
}

func TestResolveQRZeroWhenViewerCannotActOnWeigh(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	_ = commitStartLot(t, c, loadID)
	cfg := mustActiveConfig(t, c)
	domain := mustDomain(t, c)
	// administration is eligible on Domain but cannot Commit weigh (line role).
	op := OperatorIdentity{UserID: "admin", Org: "feragame", Roles: []string{"administration"}}

	got := ResolveQR(cfg, domain, op, loadID+".1")
	if got.Error == nil || got.Error.Code != "zero" {
		t.Fatalf("got = %#v", got)
	}
	if got.Error.Message != cfg.QR.Rules[0].Messages.OnZero {
		t.Fatalf("message = %q", got.Error.Message)
	}
}

func TestResolveQROpenLotIsSoleCandidate(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	_, load1, load2 := plantIntakeTwoLots(t, c)
	_ = commitStartLot(t, c, load1)
	cfg := mustActiveConfig(t, c)
	domain := mustDomain(t, c)
	op := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}

	got := ResolveQR(cfg, domain, op, load1+".1")
	if got.Error != nil || got.Navigate == nil {
		t.Fatalf("got = %#v", got)
	}
	if got.Navigate.JobInstanceID != load1 {
		t.Fatalf("navigate = %#v, want open lot %s", got.Navigate, load1)
	}
	closedPallet := ResolveQR(cfg, domain, op, load2+".1")
	if closedPallet.Error == nil || closedPallet.Error.Code != "ineligible" {
		t.Fatalf("unopened lot pallet must be ineligible: %#v", closedPallet)
	}
}

func TestResolveQRWhenSkipsEmptyEntityID(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	_ = commitStartLot(t, c, loadID)
	cfg := mustActiveConfig(t, c)
	domain := mustDomain(t, c)
	op := OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}

	got := ResolveQR(cfg, domain, op, "https://plant.example/app")
	if got.Error == nil || got.Error.Code != "no-rule" {
		t.Fatalf("got = %#v", got)
	}
}

func TestHandleQrResolveHTTPOneAndErrors(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truckID, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	_ = commitStartLot(t, c, loadID)
	h := &API{
		DomainLoader:  loader,
		TaskCommitter: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "lin", Org: "feragame", Roles: []string{"line"}}, true
		},
	}
	palletID := loadID + ".1"

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/org/qr-resolve",
		strings.NewReader(`{"raw":"https://plant.example/app?entityId=`+palletID+`"}`))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ok status = %d body=%s", rec.Code, rec.Body.String())
	}
	var okWire QrResolveResult
	if err := json.Unmarshal(rec.Body.Bytes(), &okWire); err != nil {
		t.Fatal(err)
	}
	if okWire.Error != nil || okWire.Navigate == nil || okWire.Navigate.JobInstanceID != loadID {
		t.Fatalf("ok wire = %#v", okWire)
	}
	if okWire.Navigate.StreamInstanceID != truckID {
		t.Fatalf("streamInstanceId = %q, want %q", okWire.Navigate.StreamInstanceID, truckID)
	}
	if okWire.Seed["palletId"] != palletID {
		t.Fatalf("seed = %#v", okWire.Seed)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/org/qr-resolve",
		strings.NewReader(`{"raw":"missing-pallet"}`))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ineligible status = %d body=%s", rec.Code, rec.Body.String())
	}
	var errWire QrResolveResult
	if err := json.Unmarshal(rec.Body.Bytes(), &errWire); err != nil {
		t.Fatal(err)
	}
	if errWire.Error == nil || errWire.Error.Code != "ineligible" || errWire.Error.Message == "" {
		t.Fatalf("err wire = %#v", errWire)
	}
}
