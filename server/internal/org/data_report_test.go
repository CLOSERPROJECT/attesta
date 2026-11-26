package org

import (
	_ "embed"
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

//go:embed fixtures/parked-loaded-extracted.yaml
var parkedLoadedExtractedYAML []byte

func configWithParkedLoadedExtracted(t *testing.T) OrgConfig {
	t.Helper()
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	var task map[string]any
	if err := yaml.Unmarshal(parkedLoadedExtractedYAML, &task); err != nil {
		t.Fatal(err)
	}
	if task["id"] != "loadedExtracted" {
		t.Fatalf("parked task id = %#v", task["id"])
	}
	lpVal, _ := cfg.Jobs.Get("lot_processing")
	lp, _ := AsJob(lpVal)
	lp.Tasks = append(append([]TaskConfig{}, lp.Tasks...), TaskConfig(task))
	cfg.Jobs.Set("lot_processing", lp)
	return cfg
}

func dataReportProbeDomain() Domain {
	d := domainFromKinds(map[string]EntityMap{
		"truck": {
			"T1": {"workingDayId": "2026-10-19", "truckWeight": 1000.0, "lastScaleReading": 200.0},
		},
		"lot": {
			"L1": {"netWeight": 500.0, "truckId": "T1", "client": "NO"},
			"L2": {"netWeight": 300.0, "truckId": "T1", "client": "CO"},
		},
		"pallet": {
			"P1": {"lotId": "L1", "weight": 125.0, "progress": "done"},
			"P2": {"lotId": "L1", "weight": 100.0, "progress": "partial"},
		},
		"container": {
			"C1": {
				"sourceLotId": "L1", "material": "plastica", "weight": 50.0,
				"secondarySourceLotId": "L2",
			},
			"C2": {"sourceLotId": "L1", "material": "plastica", "weight": 20.0},
			"C3": {"sourceLotId": "L1", "material": "alluminio", "weight": 30.0},
			"C4": {"sourceLotId": "L1", "material": "mystery", "weight": 5.0},
		},
	})
	d.Submissions = []Submission{{
		JobID: "lot_processing", TaskID: "weigh", JobInstanceID: "2026-10-19-1st",
		Facts: map[string]any{
			"tareKind": "euro",
			"weights":  map[string]any{"P1": map[string]any{}, "P2": map[string]any{}},
		},
	}}
	return d
}

func TestBuildTaskFormBundleDataReportChromeDisplay(t *testing.T) {
	cfg := configWithParkedLoadedExtracted(t)
	bundle, err := BuildTaskFormBundle(cfg, dataReportProbeDomain(), ActionableTaskRef{
		JobID: "lot_processing", TaskID: "loadedExtracted", JobInstanceID: "1st",
	})
	if err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}

	chrome, _ := wire["chrome"].([]any)
	if len(chrome) != 1 {
		t.Fatalf("chrome = %#v", chrome)
	}
	rule, _ := chrome[0].(map[string]any)
	if rule["id"] != "resultsReport" || rule["kind"] != "dataReport" {
		t.Fatalf("chrome rule = %#v", rule)
	}
	if rule["title"] != "Risultati finali" {
		t.Fatalf("title = %#v", rule["title"])
	}
	expr, _ := rule["expr"].(string)
	if expr == "" {
		t.Fatal("fixture dataReport missing expr")
	}

	ui, _ := wire["uiSchema"].(map[string]any)
	fieldUI, _ := ui["resultsReport"].(map[string]any)
	comps, _ := fieldUI["ui:components"].(map[string]any)
	if comps["stringField"] != "dataReportField" {
		t.Fatalf("stringField = %#v", comps["stringField"])
	}

	bindings, _ := wire["bindings"].(map[string]any)
	chromeDisplay, _ := bindings["chromeDisplay"].(map[string]any)
	display, _ := chromeDisplay["resultsReport"].(map[string]any)
	if display == nil {
		t.Fatalf("bindings.chromeDisplay.resultsReport missing: %#v", bindings["chromeDisplay"])
	}
	if display["title"] != "Risultati finali" {
		t.Fatalf("display title = %#v", display["title"])
	}
	if display["value"] != nil || display["error"] != nil {
		t.Fatalf("value/error = %#v %#v", display["value"], display["error"])
	}
	sections, _ := display["sections"].([]any)
	if len(sections) != 1 {
		t.Fatalf("sections = %#v", sections)
	}
	section, _ := sections[0].(map[string]any)
	if section["id"] != "L1" {
		t.Fatalf("id = %#v", section["id"])
	}
	secs, _ := section["secondaryIds"].([]any)
	if len(secs) != 1 || secs[0] != "L2" {
		t.Fatalf("secondaryIds = %#v", section["secondaryIds"])
	}
	rows, _ := section["rows"].([]any)
	byLabel := map[string]map[string]any{}
	for _, rawRow := range rows {
		row, _ := rawRow.(map[string]any)
		label, _ := row["label"].(string)
		byLabel[label] = row
	}
	assertRow := func(label, value string, tone any) {
		t.Helper()
		row := byLabel[label]
		if row == nil {
			t.Fatalf("missing row %q in %#v", label, byLabel)
		}
		if row["value"] != value {
			t.Fatalf("%s value = %#v, want %q", label, row["value"], value)
		}
		if row["error"] != nil {
			t.Fatalf("%s error = %#v", label, row["error"])
		}
		if row["tone"] != tone {
			t.Fatalf("%s tone = %#v, want %#v", label, row["tone"], tone)
		}
	}
	assertRow("Netto lotto", "500", nil)
	assertRow("Massa netta pannelli", "185", nil)
	assertRow("Plastica", "70", "info")
	assertRow("Alluminio", "30", "info")
	assertRow("mystery", "5", "info")
	assertRow("Totale estratto", "105", nil)
	assertRow("Pannelli vs estratto", "80 kg (56.75675675675676%)", "danger")
}

func TestBuildTaskFormBundleDataReportEmptySections(t *testing.T) {
	cfg := configWithParkedLoadedExtracted(t)
	bundle, err := BuildTaskFormBundle(cfg, EmptyDomain(), ActionableTaskRef{
		JobID: "lot_processing", TaskID: "loadedExtracted", JobInstanceID: "1st",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(bundle.Bindings)
	if err != nil {
		t.Fatal(err)
	}
	var bindings map[string]any
	if err := json.Unmarshal(raw, &bindings); err != nil {
		t.Fatal(err)
	}
	chromeDisplay, _ := bindings["chromeDisplay"].(map[string]any)
	display, _ := chromeDisplay["resultsReport"].(map[string]any)
	if display == nil {
		t.Fatal("missing chromeDisplay.resultsReport")
	}
	sections, ok := display["sections"].([]any)
	if !ok || len(sections) != 0 {
		t.Fatalf("sections = %#v, want []", display["sections"])
	}
	if display["title"] != "Risultati finali" {
		t.Fatalf("title = %#v", display["title"])
	}
}

func TestBuildTaskFormBundleWeightCheckRegisteredData(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	// Probe: truck T1 gross 1000, tare 200 → expected net 800; lots L1+L2 = 800 → Δ 0.
	bundle, err := BuildTaskFormBundle(cfg, dataReportProbeDomain(), ActionableTaskRef{
		JobID: "intake", TaskID: "weightCheck", JobInstanceID: "T1",
	})
	if err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}

	chrome, _ := wire["chrome"].([]any)
	var registered map[string]any
	var operator map[string]any
	for _, entry := range chrome {
		rule, _ := entry.(map[string]any)
		switch rule["id"] {
		case "registeredData":
			registered = rule
		case "currentOperator":
			operator = rule
		}
	}
	if registered == nil || registered["kind"] != "dataReport" {
		t.Fatalf("registeredData chrome = %#v", registered)
	}
	if registered["title"] != "Dati registrati" {
		t.Fatalf("title = %#v", registered["title"])
	}
	expr, _ := registered["expr"].(string)
	if expr == "" {
		t.Fatal("registeredData dataReport missing expr")
	}
	if operator == nil || operator["kind"] != "readonly" {
		t.Fatalf("currentOperator must stay readonly: %#v", operator)
	}

	schemaWire, _ := wire["schema"].(map[string]any)
	if _, has := schemaWire["required"]; has {
		t.Fatalf("confirm-only weightCheck schema must omit required (Ajv rejects null); got %#v", schemaWire["required"])
	}

	ui, _ := wire["uiSchema"].(map[string]any)
	fieldUI, _ := ui["registeredData"].(map[string]any)
	comps, _ := fieldUI["ui:components"].(map[string]any)
	if comps["stringField"] != "dataReportField" {
		t.Fatalf("stringField = %#v", comps["stringField"])
	}

	bindings, _ := wire["bindings"].(map[string]any)
	chromeDisplay, _ := bindings["chromeDisplay"].(map[string]any)
	display, _ := chromeDisplay["registeredData"].(map[string]any)
	if display == nil {
		t.Fatalf("bindings.chromeDisplay.registeredData missing: %#v", bindings["chromeDisplay"])
	}
	if display["title"] != "Dati registrati" {
		t.Fatalf("display title = %#v", display["title"])
	}
	if display["value"] != nil || display["error"] != nil {
		t.Fatalf("value/error = %#v %#v", display["value"], display["error"])
	}
	if _, hasSections := display["sections"]; hasSections {
		t.Fatalf("flat weightCheck DTO must not set sections: %#v", display["sections"])
	}
	rows, _ := display["rows"].([]any)
	if len(rows) != 7 {
		t.Fatalf("rows = %#v", rows)
	}
	labels := make([]string, 0, len(rows))
	byLabel := map[string]map[string]any{}
	for _, rawRow := range rows {
		row, _ := rawRow.(map[string]any)
		label, _ := row["label"].(string)
		labels = append(labels, label)
		byLabel[label] = row
	}
	wantOrder := []string{
		"Peso lordo camion", "Lotto L1", "Lotto L2", "Tara camion",
		"Netto atteso (lordo−tara)", "Σ netti lotti", "Δ (Σ − netto atteso)",
	}
	if len(labels) != len(wantOrder) {
		t.Fatalf("row labels = %#v, want %#v", labels, wantOrder)
	}
	for i, want := range wantOrder {
		if labels[i] != want {
			t.Fatalf("row[%d] = %q, want %q (all %#v)", i, labels[i], want, labels)
		}
	}
	assertRow := func(label, value string, tone any) {
		t.Helper()
		row := byLabel[label]
		if row == nil {
			t.Fatalf("missing row %q in %#v", label, byLabel)
		}
		if row["value"] != value {
			t.Fatalf("%s value = %#v, want %q", label, row["value"], value)
		}
		if row["error"] != nil {
			t.Fatalf("%s error = %#v", label, row["error"])
		}
		if row["tone"] != tone {
			t.Fatalf("%s tone = %#v, want %#v", label, row["tone"], tone)
		}
	}
	assertRow("Peso lordo camion", "1000", nil)
	assertRow("Lotto L1", "500", nil)
	assertRow("Lotto L2", "300", nil)
	assertRow("Tara camion", "200", nil)
	assertRow("Netto atteso (lordo−tara)", "800", nil)
	assertRow("Σ netti lotti", "800", nil)
	assertRow("Δ (Σ − netto atteso)", "0 kg", "green")
}

func TestBuildTaskFormBundleDataReportBadExpr(t *testing.T) {
	cfg := configWithParkedLoadedExtracted(t)
	lpVal, _ := cfg.Jobs.Get("lot_processing")
	lp, _ := AsJob(lpVal)
	task, ok := TaskByID(lp.Tasks, "loadedExtracted")
	if !ok {
		t.Fatal("missing parked loadedExtracted")
	}
	payload := taskPayload(task)
	chrome, _ := payload["chrome"].(map[string]any)
	report, _ := chrome["resultsReport"].(map[string]any)
	report["expr"] = "$this is not (( valid"

	bundle, err := BuildTaskFormBundle(cfg, EmptyDomain(), ActionableTaskRef{
		JobID: "lot_processing", TaskID: "loadedExtracted", JobInstanceID: "1st",
	})
	if err == nil {
		t.Fatalf("expected error, got bundle chromeDisplay=%#v", bundle.Bindings["chromeDisplay"])
	}
}
