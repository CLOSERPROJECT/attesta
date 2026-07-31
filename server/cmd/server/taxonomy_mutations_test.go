package main

import (
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
)

func TestMemoryStoreCreateCategoryAppendsSortOrder(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.CreateCategory(t.Context(), Category{
		Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.CreateCategory(t.Context(), Category{
		Slug: "recycling-and-recovery", Name: "Recycling", Icon: "weee",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.SortOrder != 2 {
		t.Fatalf("SortOrder=%d, want 2", got.SortOrder)
	}
}

func TestMemoryStoreCreateCategoryDuplicateSlug(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.CreateCategory(t.Context(), Category{
		Slug: "supply-chain", Name: "A", Icon: "batch-traceability",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateCategory(t.Context(), Category{
		Slug: "supply-chain", Name: "B", Icon: "weee",
	})
	if !errors.Is(err, ErrTaxonomySlugExists) {
		t.Fatalf("err=%v, want ErrTaxonomySlugExists", err)
	}
}

func TestMemoryStoreUpdateCategoryDoesNotChangeSlug(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.CreateCategory(t.Context(), Category{
		Slug: "supply-chain", Name: "Old", Icon: "batch-traceability",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.UpdateCategory(t.Context(), "supply-chain", "New Name", "weee")
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "supply-chain" || got.Name != "New Name" || got.Icon != "weee" {
		t.Fatalf("got %+v", got)
	}
	_, err = store.UpdateCategory(t.Context(), "missing", "X", "weee")
	if !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("err=%v", err)
	}
}

func TestMemoryStoreCreateSubCategoryAppendsWithinParent(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.CreateCategory(t.Context(), Category{
		Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability",
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.CreateSubCategory(t.Context(), SubCategory{
		CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement",
		Icon: "procurement-workflow", Description: "PO",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateSubCategory(t.Context(), SubCategory{
		CategorySlug: "supply-chain", Slug: "order-fulfillment", Name: "Fulfillment",
		Icon: "order-fulfillment",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.SortOrder != 1 || second.SortOrder != 2 {
		t.Fatalf("orders %d %d", first.SortOrder, second.SortOrder)
	}
}

func TestMemoryStoreCreateSubCategoryRequiresParent(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.CreateSubCategory(t.Context(), SubCategory{
		CategorySlug: "missing", Slug: "leaf", Name: "Leaf", Icon: "weee",
	})
	if !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("err=%v", err)
	}
}

func TestMemoryStoreCreateSubCategoryDuplicateSlug(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.CreateCategory(t.Context(), Category{
		Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateSubCategory(t.Context(), SubCategory{
		CategorySlug: "supply-chain", Slug: "procurement", Name: "A",
		Icon: "procurement-workflow",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CreateSubCategory(t.Context(), SubCategory{
		CategorySlug: "supply-chain", Slug: "procurement", Name: "B",
		Icon: "order-fulfillment",
	})
	if !errors.Is(err, ErrTaxonomySlugExists) {
		t.Fatalf("err=%v, want ErrTaxonomySlugExists", err)
	}
}

func TestMemoryStoreUpdateSubCategory(t *testing.T) {
	store := NewMemoryStore()
	_, _ = store.CreateCategory(t.Context(), Category{
		Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability",
	})
	_, err := store.CreateSubCategory(t.Context(), SubCategory{
		CategorySlug: "supply-chain", Slug: "procurement", Name: "Old",
		Icon: "procurement-workflow", Description: "d1",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.UpdateSubCategory(t.Context(), "supply-chain", "procurement", "New", "order-fulfillment", "d2")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "New" || got.Icon != "order-fulfillment" || got.Description != "d2" || got.Slug != "procurement" {
		t.Fatalf("%+v", got)
	}
}

func TestMemoryStoreReorderCategoryUpDown(t *testing.T) {
	store := NewMemoryStore()
	_, _ = store.CreateCategory(t.Context(), Category{Slug: "a", Name: "A", Icon: "weee"})
	_, _ = store.CreateCategory(t.Context(), Category{Slug: "b", Name: "B", Icon: "batch-traceability"})
	if err := store.ReorderCategory(t.Context(), "b", "up"); err != nil {
		t.Fatal(err)
	}
	list, _ := store.ListCategories(t.Context())
	if list[0].Slug != "b" || list[1].Slug != "a" {
		t.Fatalf("order=%v %v", list[0].Slug, list[1].Slug)
	}
	if err := store.ReorderCategory(t.Context(), "b", "up"); !errors.Is(err, ErrTaxonomyReorderBoundary) {
		t.Fatalf("err=%v", err)
	}
}

func TestMemoryStoreReorderSubCategoryWithinParent(t *testing.T) {
	store := NewMemoryStore()
	_, _ = store.CreateCategory(t.Context(), Category{Slug: "g", Name: "G", Icon: "weee"})
	_, _ = store.CreateSubCategory(t.Context(), SubCategory{CategorySlug: "g", Slug: "s1", Name: "S1", Icon: "weee"})
	_, _ = store.CreateSubCategory(t.Context(), SubCategory{CategorySlug: "g", Slug: "s2", Name: "S2", Icon: "batch-traceability"})
	if err := store.ReorderSubCategory(t.Context(), "g", "s2", "up"); err != nil {
		t.Fatal(err)
	}
	subs, _ := store.ListSubCategories(t.Context(), "g")
	if subs[0].Slug != "s2" {
		t.Fatalf("got %s", subs[0].Slug)
	}
}

func TestMemoryStoreTaxonomyMutationValidationGaps(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.CreateCategory(t.Context(), Category{Slug: "g", Name: "G", Icon: "batch-traceability"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateCategory(t.Context(), "g", "G", "not-a-real-icon"); err == nil {
		t.Fatal("expected invalid icon on update")
	}
	if _, err := store.UpdateCategory(t.Context(), "missing", "G", "batch-traceability"); err == nil {
		t.Fatal("expected missing category")
	}
	if _, err := store.CreateSubCategory(t.Context(), SubCategory{CategorySlug: "g", Slug: "s", Name: "S", Icon: "not-a-real-icon"}); err == nil {
		t.Fatal("expected invalid leaf icon")
	}
	_, err = store.CreateSubCategory(t.Context(), SubCategory{CategorySlug: "g", Slug: "s", Name: "S", Icon: "procurement-workflow"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateSubCategory(t.Context(), "g", "s", "", "procurement-workflow", ""); err == nil {
		t.Fatal("expected empty name")
	}
	if _, err := store.UpdateSubCategory(t.Context(), "g", "s", "S", "not-a-real-icon", ""); err == nil {
		t.Fatal("expected invalid icon on leaf update")
	}
	if _, err := store.UpdateSubCategory(t.Context(), "g", "missing", "S", "procurement-workflow", ""); err == nil {
		t.Fatal("expected missing leaf")
	}
	if err := store.ReorderSubCategory(t.Context(), "g", "missing", "up"); err == nil {
		t.Fatal("expected missing reorder")
	}
}

func TestTaxonomyIconURLEmpty(t *testing.T) {
	if got := taxonomyIconURL(""); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := taxonomyIconURL("batch-traceability"); got == "" {
		t.Fatal("expected non-empty url")
	}
}
