package living

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
	if rule.ID != "line-pallet-weigh" {
		t.Fatalf("rule id = %q", rule.ID)
	}
	if rule.Match.JobID != "line" || rule.Match.GroupID != "" ||
		rule.Match.TaskID != "weigh" || rule.Match.AllowSpawn {
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
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)
	_ = commitStartLot(t, c, lineID, loadID)
	cfg := mustActiveConfig(t, c)
	domain := mustDomain(t, c)
	op := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}
	palletID := loadID + ".1"

	got := ResolveQR(cfg, domain, op, "https://plant.example/app?entityId="+palletID)
	if got.Error != nil {
		t.Fatalf("error = %#v", got.Error)
	}
	if got.Navigate == nil {
		t.Fatal("expected navigate")
	}
	if got.Navigate.JobID != "line" || got.Navigate.JobInstanceID != lineID ||
		got.Navigate.GroupID != "" || got.Navigate.TaskID != "weigh" {
		t.Fatalf("navigate = %#v", got.Navigate)
	}
	if got.Seed["palletId"] != palletID {
		t.Fatalf("seed = %#v", got.Seed)
	}
}

func TestResolveQRIneligibleWhenPalletMissingOrWeighed(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)
	_ = commitStartLot(t, c, lineID, loadID)
	_ = commitWeigh(t, c, lineID, loadID+".1", 11)
	cfg := mustActiveConfig(t, c)
	domain := mustDomain(t, c)
	op := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}

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
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)
	_ = commitStartLot(t, c, lineID, loadID)
	cfg := mustActiveConfig(t, c)
	domain := mustDomain(t, c)
	// administration is eligible on Domain but cannot Commit weigh (logistic role).
	op := OperatorIdentity{UserID: "admin", Org: "feragame", Roles: []string{"administration"}}

	got := ResolveQR(cfg, domain, op, loadID+".1")
	if got.Error == nil || got.Error.Code != "zero" {
		t.Fatalf("got = %#v", got)
	}
	if got.Error.Message != cfg.QR.Rules[0].Messages.OnZero {
		t.Fatalf("message = %q", got.Error.Message)
	}
}

func TestResolveQRResumedShiftIsSoleCandidate(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	line1 := commitOpenTurnoWith(t, c, map[string]any{"date": "2026-10-19", "turno": "1st"})
	_ = commitStartLot(t, c, line1, loadID)
	commitCloseTurno(t, c, line1)
	line2 := commitOpenTurnoWith(t, c, map[string]any{"date": "2026-10-19", "turno": "2nd"})
	cfg := mustActiveConfig(t, c)
	domain := mustDomain(t, c)
	op := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}

	got := ResolveQR(cfg, domain, op, loadID+".1")
	if got.Error != nil || got.Navigate == nil {
		t.Fatalf("got = %#v", got)
	}
	if got.Navigate.JobInstanceID != line2 {
		t.Fatalf("navigate = %#v, want resumed shift %s", got.Navigate, line2)
	}
}

func TestResolveQRWhenSkipsEmptyEntityID(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)
	_ = commitStartLot(t, c, lineID, loadID)
	cfg := mustActiveConfig(t, c)
	domain := mustDomain(t, c)
	op := OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}
	_ = loadID

	got := ResolveQR(cfg, domain, op, "https://plant.example/app")
	if got.Error == nil || got.Error.Code != "no-rule" {
		t.Fatalf("got = %#v", got)
	}
}

func TestHandleQrResolveHTTPOneAndErrors(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	_, loadID, _ := commitIntakeThroughLotsClosed(t, c)
	lineID := commitOpenTurno(t, c)
	_ = commitStartLot(t, c, lineID, loadID)
	h := &Handler{
		Loader:    loader,
		Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "log", Org: "feragame", Roles: []string{"logistic"}}, true
		},
	}
	palletID := loadID + ".1"

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/living/qr-resolve",
		strings.NewReader(`{"raw":"https://plant.example/app?entityId=`+palletID+`"}`))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ok status = %d body=%s", rec.Code, rec.Body.String())
	}
	var okWire QrResolveResult
	if err := json.Unmarshal(rec.Body.Bytes(), &okWire); err != nil {
		t.Fatal(err)
	}
	if okWire.Error != nil || okWire.Navigate == nil || okWire.Navigate.JobInstanceID != lineID {
		t.Fatalf("ok wire = %#v", okWire)
	}
	if okWire.Seed["palletId"] != palletID {
		t.Fatalf("seed = %#v", okWire.Seed)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/living/qr-resolve",
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
