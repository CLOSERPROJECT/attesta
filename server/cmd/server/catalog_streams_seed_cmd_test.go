package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRunSeedCatalogStreamsCommandWithMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	_ = store.ReplaceTaxonomy(t.Context(),
		[]Category{{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1}},
		[]SubCategory{{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1}},
	)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "demo.yaml"), []byte(minimalCategorizedWorkflowYAML("")), 0o644)

	err := seedCatalogStreamsWithStoreOpener(t.Context(), nil, dir, func(context.Context) (Store, func(), error) {
		return store, func() {}, nil
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
	err := seedCatalogStreamsWithStoreOpener(t.Context(), nil, t.TempDir(), func(context.Context) (Store, func(), error) {
		return nil, nil, fmt.Errorf("boom")
	})
	if err == nil {
		t.Fatal("expected opener error")
	}
}
