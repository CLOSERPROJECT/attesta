package living

import (
	"encoding/json"
	"testing"
	"time"
)

func TestLockLoadRecapExposesLivePassportSnapshot(t *testing.T) {
	cfg := syntheticPassportConfig()
	domain := EmptyDomain()
	putKind(&domain, "widget", EntityMap{
		"W-7": {"lot": "LOT-7", "color": "green"},
	})
	domain.JobInstances = []JobInstance{{JobID: "issue", ID: "W-7", Label: "Widget W-7"}}

	bundle, err := BuildTaskFormBundle(cfg, domain, ActionableTaskRef{
		JobID: "issue", TaskID: "lock", JobInstanceID: "W-7",
	})
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, c := range bundle.Chrome {
		if c.ID != PassportSnapshotChromeID {
			continue
		}
		found = true
		if c.Kind != "dataReport" {
			t.Fatalf("chrome kind = %q", c.Kind)
		}
		if c.Expr != cfg.Passport.Snapshot.Expr {
			t.Fatalf("chrome expr drift from passport.snapshot")
		}
	}
	if !found {
		t.Fatal("missing passportSnapshot chrome")
	}

	ui, _ := bundle.UiSchema[PassportSnapshotChromeID].(map[string]any)
	comps, _ := ui["ui:components"].(map[string]any)
	if comps["stringField"] != "dataReportField" {
		t.Fatalf("ui stringField = %#v", comps["stringField"])
	}

	display, _ := bundle.Bindings["chromeDisplay"].(map[string]any)
	got, _ := display[PassportSnapshotChromeID].(map[string]any)
	if got == nil {
		t.Fatalf("chromeDisplay.%s missing: %#v", PassportSnapshotChromeID, display)
	}

	want, err := EvalPassportSnapshot(cfg.Passport.Snapshot.Expr, domain, "W-7", cfg.Catalogs, nil)
	if err != nil {
		t.Fatal(err)
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("Load chromeDisplay != EvalPassportSnapshot\ngot  %s\nwant %s", gotJSON, wantJSON)
	}
	if got["color"] != "green" || got["lot"] != "LOT-7" || got["subjectId"] != "W-7" {
		t.Fatalf("snapshot facts = %#v", got)
	}
	rows, _ := got["rows"].([]any)
	if len(rows) != 3 {
		t.Fatalf("rows = %#v", got["rows"])
	}
}

func TestLockLoadRecapMatchesCommitFreeze(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	passports := NewMemoryPassportStore()
	cfg := syntheticPassportConfig()
	if err := configs.Put(t.Context(), "synth", "v1", cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), "synth", "v1"); err != nil {
		t.Fatal(err)
	}
	domain := EmptyDomain()
	putKind(&domain, "widget", EntityMap{
		"W-3": {"lot": "LOT-3", "color": "red"},
	})
	domain.JobInstances = []JobInstance{{JobID: "issue", ID: "W-3", Label: "Widget W-3"}}
	if err := domains.Put(t.Context(), "synth", domain); err != nil {
		t.Fatal(err)
	}

	bundle, err := BuildTaskFormBundle(cfg, domain, ActionableTaskRef{
		JobID: "issue", TaskID: "lock", JobInstanceID: "W-3",
	})
	if err != nil {
		t.Fatal(err)
	}
	display, _ := bundle.Bindings["chromeDisplay"].(map[string]any)
	loadSnap, _ := display[PassportSnapshotChromeID].(map[string]any)
	if loadSnap == nil {
		t.Fatal("Load missing passport snapshot chromeDisplay")
	}

	c := &Committer{
		Domains:    domains,
		Configs:    configs,
		Passports:  passports,
		Authorizer: AllowAllAuthorizer{},
		Now:        func() time.Time { return time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC) },
	}
	_, err = c.Commit(t.Context(), "synth", OperatorIdentity{UserID: "ada", Org: "synth", Roles: []string{"operator"}}, CommitRequest{
		TaskPath:      "issue/lock",
		JobInstanceID: "W-3",
		Value:         map[string]any{"ack": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := passports.Get(t.Context(), "synth", "W-3")
	if err != nil {
		t.Fatal(err)
	}

	// Projection keys the operator approved (not transform-only locked) match freeze.
	for _, key := range []string{"subjectId", "color", "lot", "title"} {
		if loadSnap[key] != rec.Snapshot[key] {
			t.Fatalf("parity %s: load=%#v freeze=%#v", key, loadSnap[key], rec.Snapshot[key])
		}
	}
	loadRows, _ := json.Marshal(loadSnap["rows"])
	freezeRows, _ := json.Marshal(rec.Snapshot["rows"])
	if string(loadRows) != string(freezeRows) {
		t.Fatalf("rows parity\nload   %s\nfreeze %s", loadRows, freezeRows)
	}
}

func TestLockLoadSkipsPassportChromeWithoutOptIn(t *testing.T) {
	cfg := syntheticPassportConfig()
	lock := cfg.Jobs.Values["issue"].(Job).Tasks[0]
	onSubmit := lock["onSubmit"].(map[string]any)
	delete(onSubmit, "passport")

	domain := EmptyDomain()
	putKind(&domain, "widget", EntityMap{"W-1": {"lot": "L"}})
	bundle, err := BuildTaskFormBundle(cfg, domain, ActionableTaskRef{
		JobID: "issue", TaskID: "lock", JobInstanceID: "W-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range bundle.Chrome {
		if c.ID == PassportSnapshotChromeID {
			t.Fatal("passportSnapshot chrome without onSubmit.passport")
		}
	}
	display, _ := bundle.Bindings["chromeDisplay"].(map[string]any)
	if _, has := display[PassportSnapshotChromeID]; has {
		t.Fatal("chromeDisplay passportSnapshot without opt-in")
	}
}

func TestPrepStripsInjectedPassportSnapshotChrome(t *testing.T) {
	task := map[string]any{
		"onSubmit": map[string]any{"passport": true},
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ack": map[string]any{"type": "boolean"},
			},
		},
	}
	got, err := prepTaskSubmission(task, map[string]any{
		"ack":                     true,
		PassportSnapshotChromeID:  "ignored",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, has := got.Facts[PassportSnapshotChromeID]; has {
		t.Fatalf("facts still have chrome: %#v", got.Facts)
	}
	if got.Facts["ack"] != true {
		t.Fatalf("facts = %#v", got.Facts)
	}
}
