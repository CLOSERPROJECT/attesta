package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func minimalCategorizedWorkflowYAML(categoryLines string) string {
	return `
workflow:
  name: "Workflow"
` + categoryLines + `  steps:
    - id: "1"
      title: "Step 1"
      order: 1
      organization: "org1"
      substeps:
        - id: "1.1"
          title: "Input"
          order: 1
          roles: ["dep1"]
          inputKey: "value"
          inputType: "formata"
          schema:
            type: object
organizations:
  - slug: "org1"
    name: "Organization 1"
roles:
  - orgSlug: "org1"
    slug: "dep1"
    name: "Department 1"
`
}

func TestParseRuntimeConfigRejectsPartialCategoryPair(t *testing.T) {
	t.Run("categorySlug only", func(t *testing.T) {
		_, err := parseRuntimeConfigData("partial-cat.yaml", []byte(minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n")))
		if err == nil {
			t.Fatal("expected error when only categorySlug is set")
		}
		if !strings.Contains(err.Error(), "categorySlug") || !strings.Contains(err.Error(), "subCategorySlug") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("subCategorySlug only", func(t *testing.T) {
		_, err := parseRuntimeConfigData("partial-sub.yaml", []byte(minimalCategorizedWorkflowYAML("  subCategorySlug: procurement\n")))
		if err == nil {
			t.Fatal("expected error when only subCategorySlug is set")
		}
		if !strings.Contains(err.Error(), "categorySlug") || !strings.Contains(err.Error(), "subCategorySlug") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestParseRuntimeConfigAcceptsOmittedCategoryPair(t *testing.T) {
	cfg, err := parseRuntimeConfigData("omit.yaml", []byte(minimalCategorizedWorkflowYAML("")))
	if err != nil {
		t.Fatalf("parseRuntimeConfigData: %v", err)
	}
	if cfg.Workflow.CategorySlug != "" || cfg.Workflow.SubCategorySlug != "" {
		t.Fatalf("expected empty category pair, got %q / %q", cfg.Workflow.CategorySlug, cfg.Workflow.SubCategorySlug)
	}
	if cfg.Workflow.IsCategorized() {
		t.Fatal("expected uncategorized Stream when category pair omitted")
	}
}

func TestParseRuntimeConfigAcceptsCompleteCategoryPair(t *testing.T) {
	cfg, err := parseRuntimeConfigData("pair.yaml", []byte(minimalCategorizedWorkflowYAML(
		"  categorySlug: supply-chain\n  subCategorySlug: procurement\n",
	)))
	if err != nil {
		t.Fatalf("parseRuntimeConfigData: %v", err)
	}
	if cfg.Workflow.CategorySlug != "supply-chain" || cfg.Workflow.SubCategorySlug != "procurement" {
		t.Fatalf("got %q / %q", cfg.Workflow.CategorySlug, cfg.Workflow.SubCategorySlug)
	}
}

func TestApplyEffectiveStreamCategorizationUnknownPathClearsPair(t *testing.T) {
	workflow := WorkflowDef{
		Name:            "Workflow",
		CategorySlug:    "supply-chain",
		SubCategorySlug: "missing-leaf",
	}
	applyEffectiveStreamCategorization(t.Context(), &workflow, func(_ context.Context, categorySlug, subCategorySlug string) (bool, error) {
		if categorySlug != "supply-chain" || subCategorySlug != "missing-leaf" {
			t.Fatalf("lookup = %q / %q", categorySlug, subCategorySlug)
		}
		return false, nil
	})
	if workflow.IsCategorized() {
		t.Fatalf("expected uncategorized, got %q / %q", workflow.CategorySlug, workflow.SubCategorySlug)
	}
}

func TestApplyEffectiveStreamCategorizationKnownPathKeepsPair(t *testing.T) {
	workflow := WorkflowDef{
		Name:            "Workflow",
		CategorySlug:    " supply-chain ",
		SubCategorySlug: " procurement ",
	}
	if err := applyEffectiveStreamCategorization(t.Context(), &workflow, func(_ context.Context, categorySlug, subCategorySlug string) (bool, error) {
		return categorySlug == "supply-chain" && subCategorySlug == "procurement", nil
	}); err != nil {
		t.Fatalf("applyEffectiveStreamCategorization: %v", err)
	}
	if !workflow.IsCategorized() {
		t.Fatal("expected categorized Stream for known taxonomy path")
	}
	if workflow.CategorySlug != "supply-chain" || workflow.SubCategorySlug != "procurement" {
		t.Fatalf("got %q / %q after trim", workflow.CategorySlug, workflow.SubCategorySlug)
	}
}

func TestApplyEffectiveStreamCategorizationLookupErrorKeepsPair(t *testing.T) {
	workflow := WorkflowDef{
		Name:            "Workflow",
		CategorySlug:    "supply-chain",
		SubCategorySlug: "procurement",
	}
	err := applyEffectiveStreamCategorization(t.Context(), &workflow, func(_ context.Context, _, _ string) (bool, error) {
		return false, errors.New("taxonomy unavailable")
	})
	if err == nil {
		t.Fatal("expected lookup error")
	}
	if !workflow.IsCategorized() {
		t.Fatal("lookup errors must not clear the declared category pair")
	}
}

func TestApplyEffectiveStreamCategorizationMissingTaxonomyIsUncategorized(t *testing.T) {
	workflow := WorkflowDef{
		Name:            "Workflow",
		CategorySlug:    "supply-chain",
		SubCategorySlug: "procurement",
	}
	applyEffectiveStreamCategorization(t.Context(), &workflow, nil)
	if workflow.IsCategorized() {
		t.Fatal("expected uncategorized when taxonomy lookup is unavailable")
	}
}

func seedSupplyChainProcurementTaxonomy(t *testing.T, store Store) {
	t.Helper()
	if err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
	}, []SubCategory{
		{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1},
	}); err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}
}

func TestWorkflowCatalogUnknownCategoryPathLoadsUncategorized(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "workflow.yaml")
	content := minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n  subCategorySlug: does-not-exist\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	store := NewMemoryStore()
	seedSupplyChainProcurementTaxonomy(t, store)

	server := &Server{store: store, configDir: tempDir}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog: %v", err)
	}
	cfg := catalog["workflow"]
	if cfg.Workflow.IsCategorized() {
		t.Fatalf("expected uncategorized load, got %q / %q", cfg.Workflow.CategorySlug, cfg.Workflow.SubCategorySlug)
	}
}

func TestWorkflowCatalogFormataUnknownCategoryPathLoadsUncategorized(t *testing.T) {
	store := NewMemoryStore()
	seedSupplyChainProcurementTaxonomy(t, store)

	saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Stream: minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n  subCategorySlug: does-not-exist\n"),
	})
	if err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}

	server := &Server{store: store}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog: %v", err)
	}
	cfg, ok := catalog[saved.ID.Hex()]
	if !ok {
		t.Fatalf("catalog missing stream %s", saved.ID.Hex())
	}
	if cfg.Workflow.IsCategorized() {
		t.Fatalf("expected uncategorized Formata stream, got %q / %q", cfg.Workflow.CategorySlug, cfg.Workflow.SubCategorySlug)
	}
}

func TestWorkflowCatalogFormataKnownCategoryPathLoadsCategorized(t *testing.T) {
	store := NewMemoryStore()
	seedSupplyChainProcurementTaxonomy(t, store)

	saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Stream: minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n  subCategorySlug: procurement\n"),
	})
	if err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}

	server := &Server{store: store}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog: %v", err)
	}
	cfg, ok := catalog[saved.ID.Hex()]
	if !ok {
		t.Fatalf("catalog missing stream %s", saved.ID.Hex())
	}
	if !cfg.Workflow.IsCategorized() {
		t.Fatal("expected categorized Formata stream for known path")
	}
	if cfg.Workflow.CategorySlug != "supply-chain" || cfg.Workflow.SubCategorySlug != "procurement" {
		t.Fatalf("got %q / %q", cfg.Workflow.CategorySlug, cfg.Workflow.SubCategorySlug)
	}
}

func TestWorkflowCatalogRebuildsAfterTaxonomyReplaceWithoutStreamChange(t *testing.T) {
	store := NewMemoryStore()
	saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Stream: minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n  subCategorySlug: procurement\n"),
	})
	if err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}

	server := &Server{store: store}
	first, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(first): %v", err)
	}
	if first[saved.ID.Hex()].Workflow.IsCategorized() {
		t.Fatal("expected uncategorized before taxonomy exists")
	}

	seedSupplyChainProcurementTaxonomy(t, store)

	second, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(second): %v", err)
	}
	cfg := second[saved.ID.Hex()]
	if !cfg.Workflow.IsCategorized() {
		t.Fatal("expected categorized after taxonomy seed without stream modtime change")
	}
	if cfg.Workflow.CategorySlug != "supply-chain" || cfg.Workflow.SubCategorySlug != "procurement" {
		t.Fatalf("got %q / %q", cfg.Workflow.CategorySlug, cfg.Workflow.SubCategorySlug)
	}
}

type flakyTaxonomyLookupStore struct {
	*MemoryStore
	failRemaining int
}

func (s *flakyTaxonomyLookupStore) GetSubCategoryBySlug(ctx context.Context, categorySlug, slug string) (*SubCategory, error) {
	if s.failRemaining > 0 {
		s.failRemaining--
		return nil, errors.New("transient taxonomy lookup failure")
	}
	return s.MemoryStore.GetSubCategoryBySlug(ctx, categorySlug, slug)
}

func TestWorkflowCatalogRetriesAfterTransientTaxonomyLookupFailure(t *testing.T) {
	base := NewMemoryStore()
	seedSupplyChainProcurementTaxonomy(t, base)
	saved, err := base.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Stream: minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n  subCategorySlug: procurement\n"),
	})
	if err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}

	store := &flakyTaxonomyLookupStore{MemoryStore: base, failRemaining: 1}
	server := &Server{store: store}

	first, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(first): %v", err)
	}
	if !first[saved.ID.Hex()].Workflow.IsCategorized() {
		t.Fatal("transient lookup must keep declared category pair")
	}

	second, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(second): %v", err)
	}
	if !second[saved.ID.Hex()].Workflow.IsCategorized() {
		t.Fatal("expected categorized after transient lookup recovers")
	}
	if store.failRemaining != 0 {
		t.Fatalf("failRemaining = %d, want 0 (second call should have looked up)", store.failRemaining)
	}
}

func TestWorkflowCatalogKnownCategoryPathLoadsCategorized(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "workflow.yaml")
	content := minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n  subCategorySlug: procurement\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	store := NewMemoryStore()
	seedSupplyChainProcurementTaxonomy(t, store)

	server := &Server{store: store, configDir: tempDir}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog: %v", err)
	}
	cfg := catalog["workflow"]
	if !cfg.Workflow.IsCategorized() {
		t.Fatal("expected categorized Stream for known path")
	}
	if cfg.Workflow.CategorySlug != "supply-chain" || cfg.Workflow.SubCategorySlug != "procurement" {
		t.Fatalf("got %q / %q", cfg.Workflow.CategorySlug, cfg.Workflow.SubCategorySlug)
	}
}

func TestWorkflowCatalogSkipsTaxonomySeedFile(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "formata")
	seed := "categories:\n  - slug: supply-chain\n    name: Supply Chain\n    icon: batch-traceability\n    sortOrder: 1\n"
	if err := os.WriteFile(filepath.Join(tempDir, "categories.yaml"), []byte(seed), 0o644); err != nil {
		t.Fatalf("write categories.yaml: %v", err)
	}

	server := &Server{store: NewMemoryStore(), configDir: tempDir}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog: %v", err)
	}
	if len(catalog) != 1 {
		t.Fatalf("catalog size = %d, want 1 (categories.yaml skipped)", len(catalog))
	}
	if _, ok := catalog["categories"]; ok {
		t.Fatal("categories.yaml must not enter the workflow catalog")
	}
}

func TestIsWorkflowCatalogConfigFile(t *testing.T) {
	if !isWorkflowCatalogConfigFile("workflow.yaml") {
		t.Fatal("workflow.yaml should be included")
	}
	if isWorkflowCatalogConfigFile("categories.yaml") {
		t.Fatal("categories.yaml should be skipped")
	}
	if isWorkflowCatalogConfigFile("categories.yml") {
		t.Fatal("categories.yml should be skipped")
	}
	if isWorkflowCatalogConfigFile("readme.md") {
		t.Fatal("non-yaml should be skipped")
	}
}

func TestBootstrapFormataBuilderStreamsSkipsTaxonomySeedFile(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "formata")
	seed := "categories:\n  - slug: supply-chain\n    name: Supply Chain\n    icon: batch-traceability\n    sortOrder: 1\n"
	if err := os.WriteFile(filepath.Join(tempDir, "categories.yaml"), []byte(seed), 0o644); err != nil {
		t.Fatalf("write categories.yaml: %v", err)
	}

	store := NewMemoryStore()
	if err := bootstrapFormataBuilderStreams(t.Context(), store, tempDir, nil); err != nil {
		t.Fatalf("bootstrapFormataBuilderStreams: %v", err)
	}
	streams, err := store.ListFormataBuilderStreams(t.Context())
	if err != nil {
		t.Fatalf("ListFormataBuilderStreams: %v", err)
	}
	if len(streams) != 1 {
		t.Fatalf("seeded stream count = %d, want 1 (categories.yaml skipped)", len(streams))
	}
}
