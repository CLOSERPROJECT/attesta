package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunSeedCatalogStreamsCommandWithMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	_ = store.ReplaceTaxonomy(t.Context(),
		[]Category{{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1}},
		[]SubCategory{{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1}},
	)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "demo.yaml"), []byte(minimalCategorizedWorkflowYAML("")), 0o644)

	err := withSeedSession(t.Context(), nil, dir, func(context.Context) (Store, func(), error) {
		return store, func() {}, nil
	}, nil, func(sess seedSession) error {
		rng := rand.New(rand.NewSource(1))
		_, err := seedCatalogStreams(t.Context(), sess.Store, sess.ConfigDir, time.Now().UTC(), rng)
		return err
	})
	if err != nil {
		t.Fatalf("cmd: %v", err)
	}
	streams, _ := store.ListFormataBuilderStreams(t.Context())
	if len(streams) < 1 || len(streams) > 3 {
		t.Fatalf("streams=%d", len(streams))
	}
}

func TestRunSeedCatalogStreamsCommandRequiresStore(t *testing.T) {
	err := withSeedSession(t.Context(), nil, t.TempDir(), func(context.Context) (Store, func(), error) {
		return nil, nil, fmt.Errorf("boom")
	}, nil, func(seedSession) error { return nil })
	if err == nil {
		t.Fatal("expected opener error")
	}
}

func TestRunSeedCatalogStreamsCommandUsesOpener(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://127.0.0.1:1/?connectTimeoutMS=1&serverSelectionTimeoutMS=1")
	err := runSeedCatalogStreamsCommand(t.Context(), nil)
	if err == nil {
		t.Fatal("expected mongo connect/ping failure")
	}
}

func TestWithSeedSessionCatalogStreamsNilOpener(t *testing.T) {
	if err := withSeedSession(t.Context(), nil, t.TempDir(), nil, nil, func(seedSession) error { return nil }); err == nil {
		t.Fatal("expected nil opener error")
	}
}

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
