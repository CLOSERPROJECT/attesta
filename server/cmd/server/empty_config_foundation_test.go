package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEnsureEmptyConfigFoundationSeedsTaxonomyAndFormataWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(dir, "workflow.yaml"), "Main workflow", "string")
	seed := `
categories:
  - slug: supply-chain
    name: Supply Chain
    icon: batch-traceability
    sortOrder: 1
    subCategories:
      - slug: procurement
        name: Procurement
        icon: procurement-workflow
        sortOrder: 1
`
	if err := os.WriteFile(filepath.Join(dir, "categories.yaml"), []byte(strings.TrimSpace(seed)+"\n"), 0o644); err != nil {
		t.Fatalf("write categories.yaml: %v", err)
	}

	store := NewMemoryStore()
	fixed := time.Date(2026, 3, 6, 17, 0, 0, 0, time.UTC)
	if err := ensureEmptyConfigFoundation(t.Context(), store, dir, func() time.Time { return fixed }); err != nil {
		t.Fatalf("ensureEmptyConfigFoundation: %v", err)
	}

	cats, err := store.ListCategories(t.Context())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(cats) != 1 || cats[0].Slug != "supply-chain" {
		t.Fatalf("categories = %#v", cats)
	}
	subs, err := store.ListSubCategories(t.Context(), "supply-chain")
	if err != nil {
		t.Fatalf("ListSubCategories: %v", err)
	}
	if len(subs) != 1 || subs[0].Slug != "procurement" {
		t.Fatalf("sub-categories = %#v", subs)
	}

	streams, err := store.ListFormataBuilderStreams(t.Context())
	if err != nil {
		t.Fatalf("ListFormataBuilderStreams: %v", err)
	}
	if len(streams) != 1 {
		t.Fatalf("stream count = %d, want 1", len(streams))
	}
	if !streams[0].UpdatedAt.Equal(fixed) {
		t.Fatalf("UpdatedAt = %v, want %v", streams[0].UpdatedAt, fixed)
	}
}

func TestEnsureEmptyConfigFoundationNoopWhenTaxonomyAndFormataPresent(t *testing.T) {
	dir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(dir, "workflow.yaml"), "Disk workflow", "string")
	if err := os.WriteFile(filepath.Join(dir, "categories.yaml"), []byte(`
categories:
  - slug: from-file
    name: From File
    icon: weee
    sortOrder: 1
`), 0o644); err != nil {
		t.Fatalf("write categories.yaml: %v", err)
	}

	store := NewMemoryStore()
	if err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "existing", Name: "Existing", Icon: "weee", SortOrder: 1},
	}, nil); err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}
	original, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Stream:    "workflow:\n  name: \"Existing\"\n  steps:\n    - id: \"1\"\n      title: \"Step\"\n      order: 1\n      organization: \"org1\"\n      substeps:\n        - id: \"1.1\"\n          title: \"Input\"\n          order: 1\n          roles: [\"dep1\"]\n          inputKey: \"value\"\n          inputType: \"string\"\norganizations:\n  - slug: \"org1\"\n    name: \"Org\"\nroles:\n  - orgSlug: \"org1\"\n    slug: \"dep1\"\n    name: \"Dep\"\n",
		UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}

	if err := ensureEmptyConfigFoundation(t.Context(), store, dir, time.Now); err != nil {
		t.Fatalf("ensureEmptyConfigFoundation: %v", err)
	}

	cats, err := store.ListCategories(t.Context())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(cats) != 1 || cats[0].Slug != "existing" {
		t.Fatalf("taxonomy must stay untouched, got %#v", cats)
	}
	streams, err := store.ListFormataBuilderStreams(t.Context())
	if err != nil {
		t.Fatalf("ListFormataBuilderStreams: %v", err)
	}
	if len(streams) != 1 || streams[0].ID != original.ID {
		t.Fatalf("formata must stay untouched, got %#v", streams)
	}
}
