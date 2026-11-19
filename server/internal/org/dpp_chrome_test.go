package org

import "testing"

func TestEnsureDPPSnapshotChromeLeavesChromeAlone(t *testing.T) {
	cfg := OrgConfig{DPP: &DPPConfig{Snapshot: DPPSnapshotConfig{Expr: "$domain.widget.*"}}}
	chrome := []ChromeRule{{ID: "other", Kind: "text"}}
	raw := map[string]any{"other": map[string]any{"kind": "text"}}

	cases := []struct {
		name    string
		cfg     OrgConfig
		payload map[string]any
	}{
		{name: "not a passport submit", cfg: cfg, payload: map[string]any{}},
		{name: "no DPP config", cfg: OrgConfig{}, payload: passportSubmitPayload()},
		{name: "empty snapshot expr", cfg: OrgConfig{DPP: &DPPConfig{}}, payload: passportSubmitPayload()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotChrome, gotRaw := ensureDPPSnapshotChrome(tc.cfg, tc.payload, chrome, raw)
			if len(gotChrome) != 1 || gotChrome[0].ID != "other" {
				t.Fatalf("chrome = %#v, want unchanged", gotChrome)
			}
			if gotRaw["other"] == nil {
				t.Fatalf("raw = %#v, want unchanged", gotRaw)
			}
		})
	}
}

func TestEnsureDPPSnapshotChromeAppendsAndReplaces(t *testing.T) {
	cfg := OrgConfig{DPP: &DPPConfig{Snapshot: DPPSnapshotConfig{Expr: "$domain.widget.*"}}}
	payload := passportSubmitPayload()

	appended, raw := ensureDPPSnapshotChrome(cfg, payload, nil, nil)
	if len(appended) != 1 || appended[0].ID != DPPSnapshotChromeID {
		t.Fatalf("appended chrome = %#v", appended)
	}
	if appended[0].Kind != "dataReport" || appended[0].Expr != cfg.DPP.Snapshot.Expr {
		t.Fatalf("appended rule = %#v", appended[0])
	}
	entry, _ := raw[DPPSnapshotChromeID].(map[string]any)
	if entry["kind"] != "dataReport" || entry["expr"] != cfg.DPP.Snapshot.Expr {
		t.Fatalf("appended raw entry = %#v", entry)
	}

	chrome := []ChromeRule{{ID: "other"}, {ID: DPPSnapshotChromeID, Kind: "dataReport", Expr: "old"}}
	raw = map[string]any{DPPSnapshotChromeID: map[string]any{"kind": "dataReport", "expr": "old"}}
	originalEntry, _ := raw[DPPSnapshotChromeID].(map[string]any)
	replaced, gotRaw := ensureDPPSnapshotChrome(cfg, payload, chrome, raw)
	if len(replaced) != 2 || replaced[1].Kind != "dataReport" || replaced[1].Expr != cfg.DPP.Snapshot.Expr {
		t.Fatalf("replaced chrome = %#v", replaced)
	}
	replacedEntry, _ := gotRaw[DPPSnapshotChromeID].(map[string]any)
	if replacedEntry["expr"] != cfg.DPP.Snapshot.Expr {
		t.Fatalf("replaced raw entry = %#v", replacedEntry)
	}
	if originalEntry["expr"] != "old" {
		t.Fatalf("original raw entry was mutated: %#v", originalEntry)
	}
	originalEntry["expr"] = "tampered"
	if gotEntry, _ := gotRaw[DPPSnapshotChromeID].(map[string]any); gotEntry["expr"] != cfg.DPP.Snapshot.Expr {
		t.Fatalf("raw entry shares the previous entry map: %#v", gotEntry)
	}

	chrome = []ChromeRule{{ID: DPPSnapshotChromeID, Kind: "text"}}
	noRawEntry, rawWithoutEntry := ensureDPPSnapshotChrome(cfg, payload, chrome, map[string]any{})
	if len(noRawEntry) != 1 || noRawEntry[0].Kind != "dataReport" {
		t.Fatalf("chrome without raw entry = %#v", noRawEntry)
	}
	if rawWithoutEntry[DPPSnapshotChromeID] == nil {
		t.Fatalf("raw entry was not created: %#v", rawWithoutEntry)
	}
}

func passportSubmitPayload() map[string]any {
	return map[string]any{"onSubmit": map[string]any{"passport": true}}
}
