package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTaxonomySeedFileParsesNestedYAML(t *testing.T) {
	categories, subs, err := loadTaxonomySeedFile(filepath.Join("..", "..", "config", "categories.yaml"))
	if err != nil {
		t.Fatalf("loadTaxonomySeedFile: %v", err)
	}
	if len(categories) != 4 {
		t.Fatalf("categories = %d, want 4", len(categories))
	}
	if len(subs) != 20 {
		t.Fatalf("sub-categories = %d, want 20", len(subs))
	}
	if categories[0].Slug != "recycling-and-recovery" || categories[0].Icon != "weee" {
		t.Fatalf("first category = %#v", categories[0])
	}
	found := false
	for _, sub := range subs {
		if sub.CategorySlug == "recycling-and-recovery" && sub.Slug == "photovoltaic-panels" {
			found = true
			if sub.Icon != "photovoltaic-module" {
				t.Fatalf("icon = %q", sub.Icon)
			}
			if !strings.Contains(sub.Description, "photovoltaic") {
				t.Fatalf("description = %q", sub.Description)
			}
		}
	}
	if !found {
		t.Fatal("expected photovoltaic-panels sub-category")
	}
}

func TestSeedTaxonomyFullReplaceAndRerun(t *testing.T) {
	store := NewMemoryStore()
	yamlPath := writeTempTaxonomySeed(t, `
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
        description: Purchase orders
`)

	if err := seedTaxonomyFromFile(t.Context(), store, yamlPath); err != nil {
		t.Fatalf("seedTaxonomyFromFile: %v", err)
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
		t.Fatalf("subs = %#v", subs)
	}

	replacement := writeTempTaxonomySeed(t, `
categories:
  - slug: compliance-and-quality
    name: Compliance and Quality
    icon: quality-control
    sortOrder: 1
    subCategories:
      - slug: inspection
        name: Inspection
        icon: inspection-workflow
        sortOrder: 1
`)
	if err := seedTaxonomyFromFile(t.Context(), store, replacement); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
	cats, err = store.ListCategories(t.Context())
	if err != nil {
		t.Fatalf("ListCategories after replace: %v", err)
	}
	if len(cats) != 1 || cats[0].Slug != "compliance-and-quality" {
		t.Fatalf("after replace categories = %#v", cats)
	}
	if _, err := store.GetCategoryBySlug(t.Context(), "supply-chain"); err == nil {
		t.Fatal("expected prior category wiped by full replace")
	}
	subs, err = store.ListSubCategories(t.Context(), "")
	if err != nil {
		t.Fatalf("ListSubCategories all: %v", err)
	}
	if len(subs) != 1 || subs[0].Slug != "inspection" {
		t.Fatalf("after replace subs = %#v", subs)
	}
}

func TestSeedTaxonomyRejectsUnknownIcon(t *testing.T) {
	store := NewMemoryStore()
	path := writeTempTaxonomySeed(t, `
categories:
  - slug: supply-chain
    name: Supply Chain
    icon: not-a-real-icon
    sortOrder: 1
`)
	err := seedTaxonomyFromFile(t.Context(), store, path)
	if err == nil {
		t.Fatal("expected icon allowlist error")
	}
	if !strings.Contains(err.Error(), "invalid taxonomy icon") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseTaxonomySeedYAMLValidationErrors(t *testing.T) {
	if _, _, err := parseTaxonomySeedYAML([]byte("categories: [")); err == nil {
		t.Fatal("expected yaml parse error")
	}
	if _, _, err := parseTaxonomySeedYAML([]byte(`
categories:
  - name: Missing Slug
    icon: weee
    sortOrder: 1
`)); err == nil || !strings.Contains(err.Error(), "slug is required") {
		t.Fatalf("missing slug err = %v", err)
	}
	if _, _, err := parseTaxonomySeedYAML([]byte(`
categories:
  - slug: supply-chain
    name: Supply Chain
    icon: batch-traceability
    sortOrder: 1
  - slug: supply-chain
    name: Dup
    icon: weee
    sortOrder: 2
`)); err == nil || !strings.Contains(err.Error(), "duplicate category slug") {
		t.Fatalf("dup category err = %v", err)
	}
	if _, _, err := parseTaxonomySeedYAML([]byte(`
categories:
  - slug: supply-chain
    name: Supply Chain
    icon: batch-traceability
    sortOrder: 1
    subCategories:
      - name: Missing
        icon: procurement-workflow
        sortOrder: 1
`)); err == nil || !strings.Contains(err.Error(), "sub-category slug is required") {
		t.Fatalf("missing sub slug err = %v", err)
	}
	if _, _, err := parseTaxonomySeedYAML([]byte(`
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
      - slug: procurement
        name: Dup
        icon: order-fulfillment
        sortOrder: 2
`)); err == nil || !strings.Contains(err.Error(), "duplicate sub-category slug") {
		t.Fatalf("dup sub err = %v", err)
	}
	if err := seedTaxonomyFromFile(t.Context(), nil, writeTempTaxonomySeed(t, "categories: []\n")); err == nil {
		t.Fatal("expected nil store error")
	}
	if err := seedTaxonomyFromFile(t.Context(), NewMemoryStore(), filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestSeedTaxonomyBypassesCategoryDeleteGuard(t *testing.T) {
	store := NewMemoryStore()
	first := writeTempTaxonomySeed(t, `
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
`)
	if err := seedTaxonomyFromFile(t.Context(), store, first); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := store.DeleteCategory(t.Context(), "supply-chain"); err == nil {
		t.Fatal("expected delete guard before seed wipe")
	}

	empty := writeTempTaxonomySeed(t, "categories: []\n")
	if err := seedTaxonomyFromFile(t.Context(), store, empty); err != nil {
		t.Fatalf("seed empty: %v", err)
	}
	cats, err := store.ListCategories(t.Context())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(cats) != 0 {
		t.Fatalf("expected empty after seed wipe, got %#v", cats)
	}
}

func writeTempTaxonomySeed(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "categories.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)+"\n"), 0o644); err != nil {
		t.Fatalf("write temp seed: %v", err)
	}
	return path
}

func TestBootstrapTaxonomySeedsWhenEmpty(t *testing.T) {
	dir := t.TempDir()
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
	if err := bootstrapTaxonomy(t.Context(), store, dir); err != nil {
		t.Fatalf("bootstrapTaxonomy: %v", err)
	}
	cats, err := store.ListCategories(t.Context())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(cats) != 1 || cats[0].Slug != "supply-chain" {
		t.Fatalf("categories = %#v", cats)
	}
}

func TestBootstrapTaxonomySkipsWhenTaxonomyExists(t *testing.T) {
	dir := t.TempDir()
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

	if err := bootstrapTaxonomy(t.Context(), store, dir); err != nil {
		t.Fatalf("bootstrapTaxonomy: %v", err)
	}
	cats, err := store.ListCategories(t.Context())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(cats) != 1 || cats[0].Slug != "existing" {
		t.Fatalf("bootstrap must not replace existing taxonomy, got %#v", cats)
	}
}

func TestBootstrapTaxonomyNoopWhenSeedFileMissing(t *testing.T) {
	store := NewMemoryStore()
	if err := bootstrapTaxonomy(t.Context(), store, t.TempDir()); err != nil {
		t.Fatalf("bootstrapTaxonomy missing file: %v", err)
	}
	cats, err := store.ListCategories(t.Context())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(cats) != 0 {
		t.Fatalf("expected empty taxonomy, got %#v", cats)
	}
}

func TestBootstrapTaxonomyNilStoreNoop(t *testing.T) {
	if err := bootstrapTaxonomy(t.Context(), nil, t.TempDir()); err != nil {
		t.Fatalf("bootstrapTaxonomy nil store: %v", err)
	}
}
