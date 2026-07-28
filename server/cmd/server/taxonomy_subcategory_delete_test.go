package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
)

func seedProcurementTaxonomy(t *testing.T, store Store) {
	t.Helper()
	if err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
	}, []SubCategory{
		{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1},
		{CategorySlug: "supply-chain", Slug: "order-fulfillment", Name: "Order Fulfillment", Icon: "order-fulfillment", SortOrder: 2},
	}); err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}
}

func TestDeleteSubCategoryRefusedWhenCatalogStreamReferences(t *testing.T) {
	store := NewMemoryStore()
	seedProcurementTaxonomy(t, store)

	err := deleteSubCategory(t.Context(), store, "supply-chain", "procurement",
		func(_ context.Context, categorySlug, subCategorySlug string) (bool, error) {
			return categorySlug == "supply-chain" && subCategorySlug == "procurement", nil
		},
	)
	if !errors.Is(err, ErrSubCategoryReferencedByStream) {
		t.Fatalf("err = %v, want ErrSubCategoryReferencedByStream", err)
	}
	if _, err := store.GetSubCategoryBySlug(t.Context(), "supply-chain", "procurement"); err != nil {
		t.Fatalf("expected sub-category retained, get err = %v", err)
	}
}

func TestDeleteSubCategoryAllowedWhenNoCatalogStreamReferences(t *testing.T) {
	store := NewMemoryStore()
	seedProcurementTaxonomy(t, store)

	err := deleteSubCategory(t.Context(), store, "supply-chain", "procurement",
		func(_ context.Context, categorySlug, subCategorySlug string) (bool, error) {
			return false, nil
		},
	)
	if err != nil {
		t.Fatalf("deleteSubCategory: %v", err)
	}
	if _, err := store.GetSubCategoryBySlug(t.Context(), "supply-chain", "procurement"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("expected sub-category gone, err = %v", err)
	}
}

func TestDeleteSubCategoryScansCatalogBlueprintsForMatchingPath(t *testing.T) {
	catalog := map[string]RuntimeConfig{
		"workflow": {
			Workflow: WorkflowDef{
				CategorySlug:    "supply-chain",
				SubCategorySlug: "procurement",
			},
		},
		"other": {
			Workflow: WorkflowDef{
				CategorySlug:    "supply-chain",
				SubCategorySlug: "order-fulfillment",
			},
		},
	}
	if !catalogReferencesSubCategoryPath(catalog, "supply-chain", "procurement") {
		t.Fatal("expected catalog to reference supply-chain/procurement")
	}
	if catalogReferencesSubCategoryPath(catalog, "supply-chain", "missing") {
		t.Fatal("did not expect reference to missing path")
	}
	if catalogReferencesSubCategoryPath(map[string]RuntimeConfig{
		"uncategorized": {Workflow: WorkflowDef{}},
	}, "supply-chain", "procurement") {
		t.Fatal("uncategorized blueprint must not count as a reference")
	}
}

func TestSeedReplaceBypassesSubCategoryStreamRefRestrict(t *testing.T) {
	store := NewMemoryStore()
	seedProcurementTaxonomy(t, store)

	err := deleteSubCategory(t.Context(), store, "supply-chain", "procurement",
		func(_ context.Context, _, _ string) (bool, error) {
			return true, nil
		},
	)
	if !errors.Is(err, ErrSubCategoryReferencedByStream) {
		t.Fatalf("guarded delete err = %v, want ErrSubCategoryReferencedByStream", err)
	}

	empty := writeTempTaxonomySeed(t, "categories: []\n")
	if err := seedTaxonomyFromFile(t.Context(), store, empty); err != nil {
		t.Fatalf("seed wipe: %v", err)
	}
	subs, err := store.ListSubCategories(t.Context(), "")
	if err != nil {
		t.Fatalf("ListSubCategories: %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("expected seed full-replace to wipe sub-categories, got %#v", subs)
	}
}

func TestServerDeleteSubCategoryUsesLiveCatalogRefs(t *testing.T) {
	dir := t.TempDir()
	workflowYAML := minimalCategorizedWorkflowYAML(
		"  categorySlug: supply-chain\n  subCategorySlug: procurement\n",
	)
	if err := os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(workflowYAML), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	store := NewMemoryStore()
	seedProcurementTaxonomy(t, store)

	server := &Server{store: store, configDir: dir}
	if err := server.DeleteSubCategory(t.Context(), "supply-chain", "procurement"); !errors.Is(err, ErrSubCategoryReferencedByStream) {
		t.Fatalf("DeleteSubCategory err = %v, want ErrSubCategoryReferencedByStream", err)
	}

	if err := server.DeleteSubCategory(t.Context(), "supply-chain", "order-fulfillment"); err != nil {
		t.Fatalf("unreferenced DeleteSubCategory: %v", err)
	}
	if _, err := store.GetSubCategoryBySlug(t.Context(), "supply-chain", "order-fulfillment"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("expected order-fulfillment gone, err = %v", err)
	}
}

func TestServerDeleteSubCategoryRefusesWhenFormataStreamReferences(t *testing.T) {
	store := NewMemoryStore()
	seedProcurementTaxonomy(t, store)

	if _, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Stream: minimalCategorizedWorkflowYAML(
			"  categorySlug: supply-chain\n  subCategorySlug: procurement\n",
		),
	}); err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}

	// Empty config dir: file-catalog scan would miss the Formata reference.
	server := &Server{store: store, configDir: t.TempDir()}
	if err := server.DeleteSubCategory(t.Context(), "supply-chain", "procurement"); !errors.Is(err, ErrSubCategoryReferencedByStream) {
		t.Fatalf("DeleteSubCategory err = %v, want ErrSubCategoryReferencedByStream", err)
	}
	if _, err := store.GetSubCategoryBySlug(t.Context(), "supply-chain", "procurement"); err != nil {
		t.Fatalf("expected sub-category retained, get err = %v", err)
	}
}

func TestServerDeleteSubCategoryIgnoresStaleFileRefsWhenFormataCatalogWins(t *testing.T) {
	dir := t.TempDir()
	staleYAML := minimalCategorizedWorkflowYAML(
		"  categorySlug: supply-chain\n  subCategorySlug: procurement\n",
	)
	if err := os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(staleYAML), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	store := NewMemoryStore()
	seedProcurementTaxonomy(t, store)

	// Live catalog is Formata-only and does not reference procurement.
	if _, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Stream: minimalCategorizedWorkflowYAML(
			"  categorySlug: supply-chain\n  subCategorySlug: order-fulfillment\n",
		),
	}); err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}

	server := &Server{store: store, configDir: dir}
	if err := server.DeleteSubCategory(t.Context(), "supply-chain", "procurement"); err != nil {
		t.Fatalf("DeleteSubCategory: %v (stale file ref must not block)", err)
	}
	if _, err := store.GetSubCategoryBySlug(t.Context(), "supply-chain", "procurement"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("expected procurement deleted, err = %v", err)
	}
}
