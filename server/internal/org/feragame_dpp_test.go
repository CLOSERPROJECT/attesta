package org

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/untp"
)

// seedFullyProcessedLoad runs intake → startLot → weigh → markProgress →
// containerReg → closeLot so the lot is fullyProcessed and ready for lock.
//
// Active Config refresh after feragame YAML changes needs:
//
//	task seed -- --replace
//
// (human-gated wipe — do not run from agents).
func seedFullyProcessedLoad(t *testing.T, c *TaskCommitter) (loadID, lotID string) {
	t.Helper()
	loadID, lotID = seedLotReadyAfterWeigh(t, c)
	commitMarkProgressAllDone(t, c, lotID, loadID)
	_ = commitContainerReg(t, c, lotID, containerRegValue(loadID, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 100.0}},
	}))
	_ = commitCloseLot(t, c, lotID)
	stored, err := c.Domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	if stored.Kind("lot")[loadID]["status"] != "fullyProcessed" {
		t.Fatalf("lot after closeLot = %#v, want fullyProcessed", stored.Kind("lot")[loadID])
	}
	if stored.Kind("lot")[loadID]["locked"] == true {
		t.Fatal("lot must not be locked before lot_processing/lock")
	}
	return loadID, lotID
}

func TestFeragamePassportFixtureShape(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDPPConfig(cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.DPP.Subject.Entity != "lot" {
		t.Fatalf("subject.entity = %q", cfg.DPP.Subject.Entity)
	}
	if _, ok := cfg.Jobs.Get("passport"); ok {
		t.Fatal("passport Job must be folded into lot_processing")
	}
	job := cfg.Jobs.Values["lot_processing"].(Job)
	lock, ok := TaskByID(job.Tasks, "lock")
	if !ok {
		t.Fatal("lot_processing/lock missing")
	}
	lockRW, _ := lock["readyWhen"].(string)
	if strings.Contains(lockRW, "select") || strings.Contains(lockRW, "submissions") {
		t.Fatalf("lock readyWhen = %q, must not gate on select Submission", lockRW)
	}
	if !strings.Contains(lockRW, "fullyProcessed") || !strings.Contains(lockRW, "locked") {
		t.Fatalf("lock readyWhen = %q, want Domain eligibility", lockRW)
	}
	onSubmit, _ := lock["onSubmit"].(map[string]any)
	if _, ok := onSubmit["passport"]; !ok {
		t.Fatal("lock missing onSubmit.passport")
	}
	transform, _ := onSubmit["transform"].([]any)
	body, _ := json.Marshal(transform)
	if !strings.Contains(string(body), `"locked"`) || !strings.Contains(string(body), `"issued"`) {
		t.Fatalf("lock transform must set locked/issued Domain facts: %s", body)
	}
}

func TestFeragamePassportLockJobFlow(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	passports := NewMemoryDPPStore()
	c.DPPs = passports
	c.Now = func() time.Time { return time.Date(2026, 11, 4, 14, 0, 0, 0, time.UTC) }
	op := adminOp()

	emptyWL, err := loader.Load(t.Context(), FeragameScope(), op)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := worklistJobByID(emptyWL.Worklist, "passport"); ok {
		t.Fatal("passport Job must not appear on Worklist")
	}

	loadID, _ := seedFullyProcessedLoad(t, c)

	wl, err := loader.Load(t.Context(), FeragameScope(), op)
	if err != nil {
		t.Fatal(err)
	}
	lp, ok := worklistJobByID(wl.Worklist, "lot_processing")
	if !ok {
		t.Fatal("missing lot_processing worklist job")
	}
	var lockStatus TaskStatus
	for _, inst := range lp.Instances {
		if inst.ID == loadID {
			lockStatus = inst.Tasks["lock"]
			break
		}
	}
	if lockStatus != TaskStatusActionable {
		t.Fatalf("lot_processing/lock after closeLot = %q, want actionable; job=%#v", lockStatus, lp)
	}

	lockBundle, err := loader.LoadTaskBundle(t.Context(), FeragameScope(), op, "lot_processing", "lock", loadID)
	if err != nil {
		t.Fatal(err)
	}
	var foundChrome bool
	for _, ch := range lockBundle.Chrome {
		if ch.ID == DPPSnapshotChromeID {
			foundChrome = true
			break
		}
	}
	if !foundChrome {
		t.Fatal("Lock Load missing passportSnapshot chrome")
	}
	display, _ := lockBundle.Bindings["chromeDisplay"].(map[string]any)
	loadSnap, _ := display[DPPSnapshotChromeID].(map[string]any)
	if loadSnap == nil || loadSnap["subjectId"] != loadID {
		t.Fatalf("live snapshot = %#v", loadSnap)
	}
	if loadSnap["title"] != "DPP carico" {
		t.Fatalf("snapshot title = %#v", loadSnap["title"])
	}
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	domain, err := c.Domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	wantSnap, err := EvalDPPSnapshot(cfg.DPP.Snapshot.Expr, domain, loadID, cfg.Catalogs, nil)
	if err != nil {
		t.Fatal(err)
	}
	gotJSON, _ := json.Marshal(loadSnap)
	wantJSON, _ := json.Marshal(wantSnap)
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("Load chromeDisplay != EvalDPPSnapshot\ngot  %s\nwant %s", gotJSON, wantJSON)
	}

	locked, err := c.Commit(t.Context(), FeragameScope(), op, CommitRequest{
		TaskPath:      "lot_processing/lock",
		JobInstanceID: loadID,
		Value:         map[string]any{"ack": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	facts := locked.Domain.Kind("lot")[loadID]
	if facts["locked"] != true || facts["issued"] != true {
		t.Fatalf("transform locked/issued missing: %#v", facts)
	}
	if facts["status"] != "fullyProcessed" {
		t.Fatalf("fullyProcessed must stay register closure: %#v", facts)
	}
	if len(locked.Links) != 1 {
		t.Fatalf("Commit links = %#v", locked.Links)
	}
	wantLink := untp.URL("09506000134352", "FIR-1", loadID)
	if locked.Links[0].URL != wantLink || locked.Links[0].Action != HttpActionLink {
		t.Fatalf("link = %#v, want %q", locked.Links[0], wantLink)
	}

	rec, err := passports.Get(t.Context(), "feragame", loadID)
	if err != nil {
		t.Fatal(err)
	}
	if rec.DigitalLink != wantLink || rec.Snapshot["subjectId"] != loadID {
		t.Fatalf("passport record = %#v", rec)
	}
	freezeRows, _ := json.Marshal(rec.Snapshot["rows"])
	loadRows, _ := json.Marshal(loadSnap["rows"])
	if string(freezeRows) != string(loadRows) {
		t.Fatalf("freeze rows != Load rows\nfreeze %s\nload   %s", freezeRows, loadRows)
	}

	_, err = c.Commit(t.Context(), FeragameScope(), op, CommitRequest{
		TaskPath:      "lot_processing/lock",
		JobInstanceID: loadID,
		Value:         map[string]any{"ack": true},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "passport-exists" {
		t.Fatalf("second lock err = %v (want passport-exists)", err)
	}

	afterLock, err := loader.Load(t.Context(), FeragameScope(), op)
	if err != nil {
		t.Fatal(err)
	}
	lpAfter, ok := worklistJobByID(afterLock.Worklist, "lot_processing")
	if !ok {
		t.Fatal("missing lot_processing after lock")
	}
	for _, inst := range lpAfter.Instances {
		if inst.ID != loadID {
			continue
		}
		if status := inst.Tasks["lock"]; status == TaskStatusActionable {
			t.Fatalf("lot_processing/lock after issue = %q, want not actionable", status)
		}
	}
}
