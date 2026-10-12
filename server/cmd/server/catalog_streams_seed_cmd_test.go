package main

import (
	"testing"
)

func TestApplyCatalogStreamCategoryAndNameValidation(t *testing.T) {
	if _, err := applyCatalogStreamCategoryAndName("workflow:\n  name: x\n", "", "s", "n"); err == nil {
		t.Fatal("expected missing fields")
	}
	if _, err := applyCatalogStreamCategoryAndName("nope:\n", "c", "s", "n"); err == nil {
		t.Fatal("expected missing workflow key")
	}
}

func TestSeedScalarHelpers(t *testing.T) {
	if got := seedScalarDataKey(WorkflowSub{}); got != "value" {
		t.Fatalf("got %q", got)
	}
	if got := seedScalarDataKey(WorkflowSub{InputKey: " lot "}); got != "lot" {
		t.Fatalf("got %q", got)
	}
	if seedSubstepSuppliesLot(WorkflowSub{}, "") {
		t.Fatal("empty lot key")
	}
	if !seedSubstepSuppliesLot(WorkflowSub{InputKey: "lot"}, "lot") {
		t.Fatal("input key match")
	}
	if seedSubstepSuppliesLot(WorkflowSub{InputKey: "other"}, "lot") {
		t.Fatal("no schema miss")
	}
	if !seedSubstepSuppliesLot(WorkflowSub{Schema: map[string]interface{}{
		"properties": map[string]interface{}{"lot": map[string]interface{}{}},
	}}, "lot") {
		t.Fatal("schema property match")
	}
}
