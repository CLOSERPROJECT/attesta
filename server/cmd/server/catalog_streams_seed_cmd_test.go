package main

import (
	"strings"
	"testing"
	"time"
)

func TestApplyCatalogStreamCategoryAndNameValidation(t *testing.T) {
	if _, err := applyCatalogStreamCategoryAndName("workflow:\n  name: x\n", "", "s", "n"); err == nil {
		t.Fatal("expected missing fields")
	}
	if _, err := applyCatalogStreamCategoryAndName("workflow:\n  name: x\n", "c", "s", ""); err == nil {
		t.Fatal("expected missing name")
	}
	if _, err := applyCatalogStreamCategoryAndName("nope:\n", "c", "s", "n"); err == nil {
		t.Fatal("expected missing workflow key")
	}
	if _, err := applyCatalogStreamCategoryAndName("workflow:\n  categorySlug: c\n", "c", "s", "Name"); err == nil {
		t.Fatal("expected missing workflow name")
	}
	if _, err := applyCatalogStreamCategoryAndName("workflow:\n  name: [invalid\n", "c", "s", "Name"); err == nil {
		t.Fatal("expected invalid YAML parse error")
	}
}

func TestSeedCatalogStreamsNilArgs(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	if _, err := seedCatalogStreams(t.Context(), nil, "config", now, &fixedCatalogRNG{n: 0}); err == nil || !strings.Contains(err.Error(), "store is required") {
		t.Fatalf("err = %v, want store required", err)
	}
	if _, err := seedCatalogStreams(t.Context(), NewMemoryStore(), "config", now, nil); err == nil || !strings.Contains(err.Error(), "rng is required") {
		t.Fatalf("err = %v, want rng required", err)
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
