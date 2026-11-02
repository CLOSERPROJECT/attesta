package living

import (
	"strings"
	"testing"
)

func testContainerMaterials() []materialProp {
	return []materialProp{
		{Key: "polvereDiVetro", Label: "Polvere di vetro"},
		{Key: "plastica", Label: "Plastica"},
		{Key: "alluminio", Label: "Alluminio"},
	}
}

func TestFlattenContainerRegFactsSkipsEmptyAndLabelsMaterials(t *testing.T) {
	mats := testContainerMaterials()
	_, err := flattenContainerRegFacts(map[string]any{
		"polvereDiVetro": []any{map[string]any{"weight": 10.0}},
	}, mats)
	if err == nil || !strings.Contains(err.Error(), "sourceLoadId") {
		t.Fatalf("err = %v", err)
	}

	facts, err := flattenContainerRegFacts(map[string]any{
		"sourceLoadId":   "load-1",
		"polvereDiVetro": []any{map[string]any{"weight": 10.0}, map[string]any{"weight": nil}, map[string]any{}},
		"plastica":       []any{map[string]any{"weight": 3.5}},
		"alluminio":      []any{},
	}, mats)
	if err != nil {
		t.Fatal(err)
	}
	if facts.SourceLoadID != "load-1" || len(facts.Rows) != 2 {
		t.Fatalf("facts = %#v", facts)
	}
	if facts.Rows[0].Material != "Polvere di vetro" || facts.Rows[0].Weight != 10 {
		t.Fatalf("row0 = %#v", facts.Rows[0])
	}
	if facts.Rows[1].Material != "Plastica" || facts.Rows[1].Weight != 3.5 {
		t.Fatalf("row1 = %#v", facts.Rows[1])
	}
}

func TestFlattenContainerRegDualSources(t *testing.T) {
	mats := testContainerMaterials()
	_, err := flattenContainerRegFacts(map[string]any{
		"sourceLoadId":   "load-1",
		"polvereDiVetro": []any{map[string]any{"weight": 10.0, "dualSources": true}},
	}, mats)
	if err == nil || !strings.Contains(err.Error(), "secondarySourceLoadId is required") {
		t.Fatalf("err = %v", err)
	}

	_, err = flattenContainerRegFacts(map[string]any{
		"sourceLoadId":          "load-1",
		"secondarySourceLoadId": "load-1",
		"polvereDiVetro":        []any{map[string]any{"weight": 10.0, "dualSources": true}},
	}, mats)
	if err == nil || !strings.Contains(err.Error(), "must differ") {
		t.Fatalf("err = %v", err)
	}

	facts, err := flattenContainerRegFacts(map[string]any{
		"sourceLoadId":          "load-1",
		"secondarySourceLoadId": "load-2",
		"polvereDiVetro":        []any{map[string]any{"weight": 10.0, "dualSources": true}},
		"plastica":              []any{map[string]any{"weight": 8.0, "dualSources": false}},
	}, mats)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.Rows) != 2 {
		t.Fatalf("rows = %#v", facts.Rows)
	}
	if facts.Rows[0].SecondarySourceLoadID != "load-2" {
		t.Fatalf("dual row = %#v", facts.Rows[0])
	}
	if facts.Rows[1].SecondarySourceLoadID != "" {
		t.Fatalf("single-source must omit secondary: %#v", facts.Rows[1])
	}
}

func TestContainerRegMaterialsFromSchema(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	jobVal, _ := cfg.Jobs.Get("line")
	job, _ := AsJob(jobVal)
	task, ok := TaskByID(job.Tasks, "containerReg")
	if !ok {
		t.Fatal("missing containerReg")
	}
	mats, ok := containerRegMaterials(map[string]any(task))
	if !ok || len(mats) != 8 {
		t.Fatalf("materials = %#v", mats)
	}
	if mats[0].Key != "polvereDiVetro" || mats[0].Label != "Polvere di vetro" {
		t.Fatalf("first = %#v", mats[0])
	}
}
