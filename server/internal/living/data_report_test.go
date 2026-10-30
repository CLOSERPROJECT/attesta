package living

import (
	"encoding/json"
	"testing"
)

func dataReportProbeDomain() Domain {
	return Domain{
		Load: EntityMap{
			"L1": {"truckWeight": 1000.0, "truckTare": 200.0},
		},
		Pallet: EntityMap{
			"P1": {"loadId": "L1", "weight": 125.0, "progress": "done"},
			"P2": {"loadId": "L1", "weight": 100.0, "progress": "partial"},
		},
		Container: EntityMap{
			"C1": {
				"sourceLoadId": "L1", "material": "plastica", "weight": 50.0,
				"secondarySourceLoadId": "L2",
			},
			"C2": {"sourceLoadId": "L1", "material": "plastica", "weight": 20.0},
			"C3": {"sourceLoadId": "L1", "material": "alluminio", "weight": 30.0},
			"C4": {"sourceLoadId": "L1", "material": "mystery", "weight": 5.0},
		},
		Submissions: []Submission{{
			JobID: "intake", TaskID: "palletWeight", JobInstanceID: "L1",
			Facts: map[string]any{
				"tareKind": "euro",
				"weights":  map[string]any{"P1": map[string]any{}, "P2": map[string]any{}},
			},
		}},
	}
}

func TestBuildTaskFormBundleDataReportChromeDisplay(t *testing.T) {
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := BuildTaskFormBundle(cfg, dataReportProbeDomain(), AvailableTaskRef{
		JobID: "line", TaskID: "loadedExtracted", JobInstanceID: "1st",
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
	if section["loadId"] != "L1" {
		t.Fatalf("loadId = %#v", section["loadId"])
	}
	secs, _ := section["secondaryLoadIds"].([]any)
	if len(secs) != 1 || secs[0] != "L2" {
		t.Fatalf("secondaryLoadIds = %#v", section["secondaryLoadIds"])
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
	assertRow("Camion lordo", "1000", nil)
	assertRow("Massa netta pannelli", "175", nil)
	assertRow("Plastica", "70", "info")
	assertRow("Alluminio", "30", "info")
	assertRow("mystery", "5", "info")
	assertRow("Totale estratto", "105", nil)
	assertRow("Pannelli vs estratto", "70 kg (60%)", "danger")
}

func TestBuildTaskFormBundleDataReportEmptySections(t *testing.T) {
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := BuildTaskFormBundle(cfg, EmptyDomain(), AvailableTaskRef{
		JobID: "line", TaskID: "loadedExtracted", JobInstanceID: "1st",
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
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		t.Fatal(err)
	}
	// Probe: L1 truck 1000; pallets 125+100 → sum 225; match 22.5% → danger.
	bundle, err := BuildTaskFormBundle(cfg, dataReportProbeDomain(), AvailableTaskRef{
		JobID: "intake", TaskID: "weightCheck", JobInstanceID: "L1",
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
	if len(rows) != 3 {
		t.Fatalf("rows = %#v", rows)
	}
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
	assertRow("Peso camion live", "1000", nil)
	assertRow("Somma pesi bancali live", "225", nil)
	assertRow("Match %", "22.5", "danger")
}

func TestBuildTaskFormBundleDataReportBadExpr(t *testing.T) {
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		t.Fatal(err)
	}
	lineVal, _ := cfg.Jobs.Get("line")
	line, _ := AsJob(lineVal)
	task, ok := TaskByID(line.Tasks, "loadedExtracted")
	if !ok {
		t.Fatal("missing loadedExtracted")
	}
	payload := taskPayload(task)
	chrome, _ := payload["chrome"].(map[string]any)
	report, _ := chrome["resultsReport"].(map[string]any)
	report["expr"] = "$this is not (( valid"

	bundle, err := BuildTaskFormBundle(cfg, EmptyDomain(), AvailableTaskRef{
		JobID: "line", TaskID: "loadedExtracted", JobInstanceID: "1st",
	})
	if err == nil {
		t.Fatalf("expected error, got bundle chromeDisplay=%#v", bundle.Bindings["chromeDisplay"])
	}
}
