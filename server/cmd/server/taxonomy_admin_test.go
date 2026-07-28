package main

import (
	"testing"
)

func TestBuildPlatformAdminTaxonomyTree(t *testing.T) {
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

	got, err := buildPlatformAdminTaxonomyTree(t.Context(), store)
	if err != nil {
		t.Fatalf("buildPlatformAdminTaxonomyTree: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(categories) = %d, want 1", len(got))
	}

	cat := got[0]
	if cat.Name != "Supply Chain" || cat.Slug != "supply-chain" || cat.Icon != "batch-traceability" {
		t.Fatalf("category = %#v", cat)
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
