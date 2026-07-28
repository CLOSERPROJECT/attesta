package main

import (
	"context"
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
	applyEffectiveStreamCategorization(t.Context(), &workflow, func(_ context.Context, categorySlug, subCategorySlug string) (bool, error) {
		return categorySlug == "supply-chain" && subCategorySlug == "procurement", nil
	})
	if !workflow.IsCategorized() {
		t.Fatal("expected categorized Stream for known taxonomy path")
	}
	if workflow.CategorySlug != "supply-chain" || workflow.SubCategorySlug != "procurement" {
		t.Fatalf("got %q / %q after trim", workflow.CategorySlug, workflow.SubCategorySlug)
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

func TestWorkflowCatalogUnknownCategoryPathLoadsUncategorized(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "workflow.yaml")
	content := minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n  subCategorySlug: does-not-exist\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	store := NewMemoryStore()
	if err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
	}, []SubCategory{
		{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1},
	}); err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}

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

func TestWorkflowCatalogKnownCategoryPathLoadsCategorized(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "workflow.yaml")
	content := minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n  subCategorySlug: procurement\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	store := NewMemoryStore()
	if err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
	}, []SubCategory{
		{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1},
	}); err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}

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
