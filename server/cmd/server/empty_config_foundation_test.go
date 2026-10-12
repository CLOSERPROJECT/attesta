package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type failingSaveFormataStore struct {
	*MemoryStore
	err error
}

func (s *failingSaveFormataStore) SaveFormataBuilderStream(_ context.Context, stream FormataBuilderStream) (FormataBuilderStream, error) {
	return FormataBuilderStream{}, s.err
}

func TestEnsureEmptyConfigFoundationSeedsTaxonomyAndFormataWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(dir, "workflow.yaml"), "Main workflow", "string")
	writeWorkflowConfig(t, filepath.Join(dir, "secondary.yaml"), "Secondary workflow", "string")
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
	if len(streams) != 2 {
		t.Fatalf("stream count = %d, want 2", len(streams))
	}
	for _, stream := range streams {
		if !stream.UpdatedAt.Equal(fixed) {
			t.Fatalf("UpdatedAt = %v, want %v", stream.UpdatedAt, fixed)
		}
		if stream.CreatedByUserID != platformAdminStreamUserID() {
			t.Fatalf("createdByUserID = %q, want %q", stream.CreatedByUserID, platformAdminStreamUserID())
		}
		if stream.UpdatedByUserID != platformAdminStreamUserID() {
			t.Fatalf("updatedByUserID = %q, want %q", stream.UpdatedByUserID, platformAdminStreamUserID())
		}
	}

	server := &Server{store: store}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(): %v", err)
	}
	if len(catalog) != 2 {
		t.Fatalf("catalog size = %d, want 2", len(catalog))
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

func TestEnsureEmptyConfigFoundationEdgeCases(t *testing.T) {
	t.Run("nil store", func(t *testing.T) {
		if err := ensureEmptyConfigFoundation(t.Context(), nil, t.TempDir(), nil); err != nil {
			t.Fatalf("ensureEmptyConfigFoundation nil store: %v", err)
		}
	})

	t.Run("missing config dir", func(t *testing.T) {
		store := NewMemoryStore()
		err := ensureEmptyConfigFoundation(t.Context(), store, filepath.Join(t.TempDir(), "missing"), nil)
		if err == nil {
			t.Fatal("expected missing config dir error")
		}
		if !strings.Contains(err.Error(), "config dir not found") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("no yaml files", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tempDir, "note.txt"), []byte("hello"), 0o644); err != nil {
			t.Fatalf("write note.txt: %v", err)
		}
		store := NewMemoryStore()
		err := ensureEmptyConfigFoundation(t.Context(), store, tempDir, nil)
		if err == nil {
			t.Fatal("expected empty workflow catalog error")
		}
		if !strings.Contains(err.Error(), "catalog template pack is empty") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("save error", func(t *testing.T) {
		tempDir := t.TempDir()
		writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
		store := &failingSaveFormataStore{
			MemoryStore: NewMemoryStore(),
			err:         errors.New("save failed"),
		}
		err := ensureEmptyConfigFoundation(t.Context(), store, tempDir, nil)
		if err == nil {
			t.Fatal("expected save error")
		}
		if !strings.Contains(err.Error(), "seed formata stream") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
