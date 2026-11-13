package living

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/untp"
)

// seedFullyProcessedLoad runs intake → startLot → weigh → markProgress →
// containerReg → closeLot so the lot is terminal fullyProcessed for Passport.
//
// Active Config refresh after feragame YAML changes needs:
//
//	task seed -- --replace
//
// (human-gated wipe — do not run from agents).
func seedFullyProcessedLoad(t *testing.T, c *Committer) (loadID, lineID string) {
	t.Helper()
	loadID, lineID = seedLineReadyForLineLoad(t, c)
	commitLineLoadDone(t, c, lineID, loadID)
	_ = commitContainerReg(t, c, lineID, containerRegValue(loadID, map[string][]any{
		"polvereDiVetro": {map[string]any{"weight": 100.0}},
	}))
	_ = commitCloseLot(t, c, lineID)
	stored, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Kind("lot")[loadID]["status"] != "fullyProcessed" {
		t.Fatalf("lot after closeLot = %#v, want fullyProcessed", stored.Kind("lot")[loadID])
	}
	if stored.Kind("lot")[loadID]["locked"] == true {
		t.Fatal("lot must not be passport-locked before Lock Job")
	}
	return loadID, lineID
}

func TestFeragamePassportFixtureShape(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePassportConfig(cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Passport.Subject.Entity != "lot" {
		t.Fatalf("subject.entity = %q", cfg.Passport.Subject.Entity)
	}
	job := cfg.Jobs.Values["passport"].(Job)
	selectTask, _ := TaskByID(job.Tasks, "select")
	readyWhen, _ := selectTask["readyWhen"].(string)
	if !strings.Contains(readyWhen, "fullyProcessed") || !strings.Contains(readyWhen, "locked") {
		t.Fatalf("select readyWhen = %q, want eligible-load count", readyWhen)
	}
	reads, _ := selectTask["reads"].(map[string]any)
	loads, _ := reads["lots"].(map[string]any)
	where, _ := loads["where"].(string)
	if !strings.Contains(where, "fullyProcessed") || !strings.Contains(where, "locked") {
		t.Fatalf("select eligibility where = %q", where)
	}
	lock, _ := TaskByID(job.Tasks, "lock")
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
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	passports := NewMemoryPassportStore()
	c.Passports = passports
	c.Now = func() time.Time { return time.Date(2026, 11, 4, 14, 0, 0, 0, time.UTC) }
	op := adminOp()

	emptyWL, err := loader.Load(t.Context(), "feragame", op)
	if err != nil {
		t.Fatal(err)
	}
	if passportWL, ok := worklistJobByID(emptyWL.Worklist, "passport"); !ok || passportWL.CanStart {
		t.Fatalf("passport canStart on empty Domain = %#v, want false", passportWL)
	}

	loadID, _ := seedFullyProcessedLoad(t, c)

	wl, err := loader.Load(t.Context(), "feragame", op)
	if err != nil {
		t.Fatal(err)
	}
	passportWL, ok := worklistJobByID(wl.Worklist, "passport")
	if !ok || !passportWL.CanStart {
		t.Fatalf("passport Job canStart after eligible load = %#v", passportWL)
	}

	selectBundle, err := loader.LoadTaskBundle(t.Context(), "feragame", op, "passport", "select", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(loadEnumIDs(t, selectBundle, "lotId"), loadID) {
		t.Fatalf("eligible load missing from select enum")
	}

	selected, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "passport/select",
		Value:    map[string]any{"lotId": loadID},
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if selected.JobInstanceID != loadID {
		t.Fatalf("Job instance id = %q, want lot subject %q", selected.JobInstanceID, loadID)
	}

	lockBundle, err := loader.LoadTaskBundle(t.Context(), "feragame", op, "passport", "lock", loadID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	var foundChrome bool
	for _, ch := range lockBundle.Chrome {
		if ch.ID == PassportSnapshotChromeID {
			foundChrome = true
			break
		}
	}
	if !foundChrome {
		t.Fatal("Lock Load missing passportSnapshot chrome")
	}
	display, _ := lockBundle.Bindings["chromeDisplay"].(map[string]any)
	loadSnap, _ := display[PassportSnapshotChromeID].(map[string]any)
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
	domain, err := c.Domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	wantSnap, err := EvalPassportSnapshot(cfg.Passport.Snapshot.Expr, domain, loadID, cfg.Catalogs, nil)
	if err != nil {
		t.Fatal(err)
	}
	gotJSON, _ := json.Marshal(loadSnap)
	wantJSON, _ := json.Marshal(wantSnap)
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("Load chromeDisplay != EvalPassportSnapshot\ngot  %s\nwant %s", gotJSON, wantJSON)
	}

	locked, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "passport/lock",
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

	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "passport/lock",
		JobInstanceID: loadID,
		Value:         map[string]any{"ack": true},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "passport-exists" {
		t.Fatalf("second lock err = %v (want passport-exists)", err)
	}

	// No eligible loads left ⇒ canStart false (provisional select is not mountable).
	afterLock, err := loader.Load(t.Context(), "feragame", op)
	if err != nil {
		t.Fatal(err)
	}
	if passportAfter, ok := worklistJobByID(afterLock.Worklist, "passport"); !ok || passportAfter.CanStart {
		t.Fatalf("passport canStart after last lock = %#v, want false", passportAfter)
	}
}
