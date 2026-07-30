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
