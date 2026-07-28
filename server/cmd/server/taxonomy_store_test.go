package main

import (
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
)

func TestMemoryStoreListCategoriesOrderedBySortOrder(t *testing.T) {
	store := NewMemoryStore()
	err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 2},
		{Slug: "recycling-and-recovery", Name: "Recycling", Icon: "weee", SortOrder: 1},
	}, nil)
	if err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}

	got, err := store.ListCategories(t.Context())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Slug != "recycling-and-recovery" || got[1].Slug != "supply-chain" {
		t.Fatalf("order = %q, %q; want recycling then supply", got[0].Slug, got[1].Slug)
	}
	if got[0].ID.IsZero() || got[1].ID.IsZero() {
		t.Fatal("expected ObjectId ids on listed categories")
	}
}

func TestMemoryStoreGetCategoryBySlug(t *testing.T) {
	store := NewMemoryStore()
	if err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
	}, nil); err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}

	got, err := store.GetCategoryBySlug(t.Context(), "supply-chain")
	if err != nil {
		t.Fatalf("GetCategoryBySlug: %v", err)
	}
	if got.Name != "Supply Chain" || got.Icon != "batch-traceability" {
		t.Fatalf("got %#v", got)
	}

	if _, err := store.GetCategoryBySlug(t.Context(), "missing"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("missing err = %v, want ErrNoDocuments", err)
	}
}

func TestMemoryStoreDeleteCategoryRefusedWhileSubCategoriesExist(t *testing.T) {
	store := NewMemoryStore()
	if err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
	}, []SubCategory{
		{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1},
	}); err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}

	if err := store.DeleteCategory(t.Context(), "supply-chain"); !errors.Is(err, ErrCategoryHasSubCategories) {
		t.Fatalf("DeleteCategory err = %v, want ErrCategoryHasSubCategories", err)
	}

	if err := store.DeleteSubCategory(t.Context(), "supply-chain", "procurement"); err != nil {
		t.Fatalf("DeleteSubCategory: %v", err)
	}
	if err := store.DeleteCategory(t.Context(), "supply-chain"); err != nil {
		t.Fatalf("DeleteCategory after children removed: %v", err)
	}
	if _, err := store.GetCategoryBySlug(t.Context(), "supply-chain"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("expected category gone, err = %v", err)
	}
}

func TestMemoryStoreListAndGetSubCategories(t *testing.T) {
	store := NewMemoryStore()
	if err := store.ReplaceTaxonomy(t.Context(), []Category{
		{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
		{Slug: "compliance-and-quality", Name: "Compliance", Icon: "quality-control", SortOrder: 2},
	}, []SubCategory{
		{CategorySlug: "supply-chain", Slug: "order-fulfillment", Name: "Order Fulfillment", Icon: "order-fulfillment", SortOrder: 2},
		{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1, Description: "PO management"},
		{CategorySlug: "compliance-and-quality", Slug: "audit-workflow", Name: "Audit", Icon: "audit-workflow", SortOrder: 1},
	}); err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}

	subs, err := store.ListSubCategories(t.Context(), "supply-chain")
	if err != nil {
		t.Fatalf("ListSubCategories: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("len = %d, want 2", len(subs))
	}
	if subs[0].Slug != "procurement" || subs[1].Slug != "order-fulfillment" {
		t.Fatalf("order = %q, %q", subs[0].Slug, subs[1].Slug)
	}
	if subs[0].Description != "PO management" {
		t.Fatalf("description = %q", subs[0].Description)
	}
	if subs[0].ID.IsZero() {
		t.Fatal("expected ObjectId on sub-category")
	}

	got, err := store.GetSubCategoryBySlug(t.Context(), "supply-chain", "procurement")
	if err != nil {
		t.Fatalf("GetSubCategoryBySlug: %v", err)
	}
	if got.Name != "Procurement" {
		t.Fatalf("name = %q", got.Name)
	}
	if _, err := store.GetSubCategoryBySlug(t.Context(), "supply-chain", "missing"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("missing err = %v", err)
	}
}

func TestMemoryStoreDeleteSubCategoryMissing(t *testing.T) {
	store := NewMemoryStore()
	if err := store.DeleteSubCategory(t.Context(), "supply-chain", "missing"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("err = %v, want ErrNoDocuments", err)
	}
}

func TestMemoryStoreEnsureTaxonomyIndexesNoop(t *testing.T) {
	store := NewMemoryStore()
	if err := store.EnsureTaxonomyIndexes(t.Context()); err != nil {
		t.Fatalf("EnsureTaxonomyIndexes: %v", err)
	}
}
