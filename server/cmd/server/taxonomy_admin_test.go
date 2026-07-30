package main

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTaxonomyTree(t *testing.T) {
	store := NewMemoryStore()
	err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
	}, []SubCategory{
		{CategorySlug: "supply-chain", Slug: "order-fulfillment", Name: "Order Fulfillment", Icon: "order-fulfillment", SortOrder: 2, Description: "Ship orders"},
		{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1, Description: "PO management"},
	})
	if err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}

	got, err := loadTaxonomyTree(t.Context(), store)
	if err != nil {
		t.Fatalf("loadTaxonomyTree: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(categories) = %d, want 1", len(got))
	}

	cat := got[0]
	if cat.Name != "Supply Chain" || cat.Slug != "supply-chain" || cat.Icon != "batch-traceability" {
		t.Fatalf("category = %#v", cat)
	}
	if cat.SortOrder != 1 {
		t.Fatalf("category SortOrder = %d, want 1", cat.SortOrder)
	}
	if cat.IconURL != "/static/taxonomy/batch-traceability.svg" {
		t.Fatalf("category IconURL = %q, want /static/taxonomy/batch-traceability.svg", cat.IconURL)
	}
	if len(cat.SubCategories) != 2 {
		t.Fatalf("len(subcategories) = %d, want 2", len(cat.SubCategories))
	}
	if cat.SubCategories[0].Slug != "procurement" || cat.SubCategories[1].Slug != "order-fulfillment" {
		t.Fatalf("sub order = %q, %q; want procurement then order-fulfillment",
			cat.SubCategories[0].Slug, cat.SubCategories[1].Slug)
	}
	if cat.SubCategories[0].SortOrder != 1 || cat.SubCategories[1].SortOrder != 2 {
		t.Fatalf("sub SortOrder = %d, %d; want 1, 2",
			cat.SubCategories[0].SortOrder, cat.SubCategories[1].SortOrder)
	}
	if cat.SubCategories[0].IconURL != "/static/taxonomy/procurement-workflow.svg" {
		t.Fatalf("procurement IconURL = %q", cat.SubCategories[0].IconURL)
	}
	if cat.SubCategories[1].IconURL != "/static/taxonomy/order-fulfillment.svg" {
		t.Fatalf("order-fulfillment IconURL = %q", cat.SubCategories[1].IconURL)
	}
	if cat.SubCategories[0].Description != "PO management" {
		t.Fatalf("procurement description = %q", cat.SubCategories[0].Description)
	}
}

func TestLoadTaxonomyTreeListSubCategoriesError(t *testing.T) {
	store := NewMemoryStore()
	err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
	}, []SubCategory{
		{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1},
	})
	if err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}

	listSubErr := errors.New("list subcategories failed")
	failing := &failingListSubCategoriesStore{MemoryStore: store, err: listSubErr}

	_, err = loadTaxonomyTree(context.Background(), failing)
	if !errors.Is(err, listSubErr) {
		t.Fatalf("loadTaxonomyTree err = %v, want %v", err, listSubErr)
	}
}

func TestTaxonomyIconKeys(t *testing.T) {
	keys := taxonomyIconKeys()
	if len(keys) != len(taxonomyIconAllowlist) {
		t.Fatalf("len(keys) = %d, want %d", len(keys), len(taxonomyIconAllowlist))
	}
	for i := 1; i < len(keys); i++ {
		if keys[i] < keys[i-1] {
			t.Fatalf("keys not sorted: %q before %q", keys[i-1], keys[i])
		}
	}
	if keys[0] != "approval-workflow" {
		t.Fatalf("first key = %q, want approval-workflow", keys[0])
	}
}

func TestEnrichTaxonomyEditorTreeDeleteAndMoveEligibility(t *testing.T) {
	nodes := []TaxonomyCategoryNode{
		{
			Slug: "group-a",
			SubCategories: []TaxonomySubCategoryNode{
				{Slug: "leaf-referenced"},
				{Slug: "leaf-free"},
			},
		},
		{Slug: "group-empty"},
	}

	referenced := func(categorySlug, subSlug string) bool {
		return categorySlug == "group-a" && subSlug == "leaf-referenced"
	}

	got := enrichTaxonomyEditorTree(nodes, referenced)

	groupA := got[0]
	if groupA.CanDelete {
		t.Fatal("group with subcategories should not be deletable")
	}
	if groupA.DeleteReason != "Has subcategories" {
		t.Fatalf("group DeleteReason = %q", groupA.DeleteReason)
	}
	if groupA.CanMoveUp {
		t.Fatal("first group should not move up")
	}
	if !groupA.CanMoveDown {
		t.Fatal("first group should move down")
	}

	refLeaf := groupA.SubCategories[0]
	if refLeaf.CanDelete {
		t.Fatal("referenced leaf should not be deletable")
	}
	if refLeaf.DeleteReason != "Referenced by a stream" {
		t.Fatalf("referenced leaf DeleteReason = %q", refLeaf.DeleteReason)
	}
	if refLeaf.CanMoveUp {
		t.Fatal("first leaf should not move up")
	}
	if !refLeaf.CanMoveDown {
		t.Fatal("first leaf should move down")
	}

	freeLeaf := groupA.SubCategories[1]
	if !freeLeaf.CanDelete {
		t.Fatal("unreferenced leaf should be deletable")
	}
	if freeLeaf.CanMoveUp != true || freeLeaf.CanMoveDown != false {
		t.Fatalf("second leaf move flags = up:%v down:%v", freeLeaf.CanMoveUp, freeLeaf.CanMoveDown)
	}

	groupEmpty := got[1]
	if !groupEmpty.CanDelete {
		t.Fatal("empty group should be deletable")
	}
	if groupEmpty.DeleteReason != "" {
		t.Fatalf("empty group DeleteReason = %q, want empty", groupEmpty.DeleteReason)
	}
	if !groupEmpty.CanMoveUp || groupEmpty.CanMoveDown {
		t.Fatalf("last group move flags = up:%v down:%v", groupEmpty.CanMoveUp, groupEmpty.CanMoveDown)
	}
}

func TestParseCategoriesEditorQuery(t *testing.T) {
	t.Run("closed by default", func(t *testing.T) {
		form := parseCategoriesEditorQuery(url.Values{})
		if form.Open {
			t.Fatal("expected closed form")
		}
	})

	t.Run("create group", func(t *testing.T) {
		form := parseCategoriesEditorQuery(url.Values{"new": {"group"}})
		if !form.Open || form.Mode != "create" || form.Level != "group" {
			t.Fatalf("form = %#v", form)
		}
	})

	t.Run("create leaf", func(t *testing.T) {
		form := parseCategoriesEditorQuery(url.Values{
			"new":    {"sub"},
			"parent": {"supply-chain"},
		})
		if !form.Open || form.Mode != "create" || form.Level != "leaf" || form.ParentSlug != "supply-chain" {
			t.Fatalf("form = %#v", form)
		}
	})

	t.Run("edit group", func(t *testing.T) {
		form := parseCategoriesEditorQuery(url.Values{
			"edit": {"group"},
			"slug": {"supply-chain"},
		})
		if !form.Open || form.Mode != "edit" || form.Level != "group" || form.Slug != "supply-chain" {
			t.Fatalf("form = %#v", form)
		}
	})

	t.Run("edit leaf", func(t *testing.T) {
		form := parseCategoriesEditorQuery(url.Values{
			"edit":   {"sub"},
			"parent": {"supply-chain"},
			"slug":   {"procurement"},
		})
		if !form.Open || form.Mode != "edit" || form.Level != "leaf" ||
			form.ParentSlug != "supply-chain" || form.Slug != "procurement" {
			t.Fatalf("form = %#v", form)
		}
	})
}

func TestBuildCategoriesEditorViewCountsAndCatalogReference(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "workflow.yaml")
	content := minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n  subCategorySlug: procurement\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	server := &Server{store: store, configDir: tempDir}
	view, err := server.buildCategoriesEditorView(t.Context(), url.Values{}, "", "saved")
	if err != nil {
		t.Fatalf("buildCategoriesEditorView: %v", err)
	}
	if view.GroupCount != 1 || view.LeafCount != 2 {
		t.Fatalf("counts = %d groups, %d leaves; want 1, 2", view.GroupCount, view.LeafCount)
	}
	if view.Confirmation != "saved" {
		t.Fatalf("Confirmation = %q", view.Confirmation)
	}
	if len(view.IconKeys) != len(taxonomyIconAllowlist) {
		t.Fatalf("len(IconKeys) = %d", len(view.IconKeys))
	}

	cat := view.Categories[0]
	if cat.CanDelete {
		t.Fatal("group with leaves should not be deletable")
	}
	if cat.SubCategories[0].CanDelete || cat.SubCategories[0].DeleteReason != "Referenced by a stream" {
		t.Fatalf("procurement leaf = CanDelete:%v reason:%q", cat.SubCategories[0].CanDelete, cat.SubCategories[0].DeleteReason)
	}
	if !cat.SubCategories[1].CanDelete {
		t.Fatal("order-fulfillment leaf should be deletable")
	}
}

func TestBuildCategoriesEditorViewEditFormPopulation(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "workflow.yaml")
	if err := os.WriteFile(path, []byte(minimalCategorizedWorkflowYAML("")), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	server := &Server{store: store, configDir: tempDir}

	q := url.Values{
		"edit":   {"sub"},
		"parent": {"supply-chain"},
		"slug":   {"procurement"},
	}
	view, err := server.buildCategoriesEditorView(t.Context(), q, "name required", "")
	if err != nil {
		t.Fatalf("buildCategoriesEditorView: %v", err)
	}
	form := view.Form
	if !form.Open || form.Mode != "edit" || form.Level != "leaf" {
		t.Fatalf("form = %#v", form)
	}
	if form.Name != "Procurement" || form.Icon != "procurement-workflow" || form.Description != "PO management" {
		t.Fatalf("form fields = %#v", form)
	}
	if form.Error != "name required" {
		t.Fatalf("form.Error = %q", form.Error)
	}
}
