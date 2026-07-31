package main

import "testing"

func TestBuildMyHomeStreamGroupsOrdersTaxonomyAndUncategorized(t *testing.T) {
	categories := []TaxonomyCategoryNode{
		{
			Slug: "supply-chain", Name: "Supply Chain", IconURL: "/c.svg", SortOrder: 1,
			SubCategories: []TaxonomySubCategoryNode{
				{Slug: "procurement", Name: "Procurement", Description: "PO", IconURL: "/s.svg", SortOrder: 1},
				{Slug: "order-fulfillment", Name: "Order Fulfillment", SortOrder: 2},
			},
		},
	}
	catalog := map[string]RuntimeConfig{
		"a": {Workflow: WorkflowDef{Name: "A", CategorySlug: "supply-chain", SubCategorySlug: "order-fulfillment"}},
		"b": {Workflow: WorkflowDef{Name: "B", CategorySlug: "supply-chain", SubCategorySlug: "procurement"}},
		"c": {Workflow: WorkflowDef{Name: "C"}}, // uncategorized
	}
	cards := map[string]ManagedPublicStreamCardView{
		"a": {Key: "a", Card: PublicStreamCardView{Name: "A"}},
		"b": {Key: "b", Card: PublicStreamCardView{Name: "B"}},
		"c": {Key: "c", Card: PublicStreamCardView{Name: "C"}},
	}
	groups := buildMyHomeStreamGroups(categories, cards, catalog, []string{"a", "b", "c"})
	if len(groups) != 3 {
		t.Fatalf("len(groups)=%d want 3", len(groups))
	}
	if groups[0].SubCategoryName != "Procurement" || groups[0].Streams[0].Key != "b" {
		t.Fatalf("first group = %+v", groups[0])
	}
	if groups[0].CategoryName != "Supply Chain" || groups[0].CategoryIconURL != "/c.svg" {
		t.Fatalf("first group should show category header = %+v", groups[0])
	}
	if groups[1].SubCategoryName != "Order Fulfillment" || groups[1].Streams[0].Key != "a" {
		t.Fatalf("second group = %+v", groups[1])
	}
	if groups[1].CategoryName != "" || groups[1].CategoryIconURL != "" {
		t.Fatalf("second group under same category must omit category header = %+v", groups[1])
	}
	if !groups[2].Uncategorized || groups[2].CategoryName != "Uncategorized" || groups[2].Streams[0].Key != "c" {
		t.Fatalf("uncategorized = %+v", groups[2])
	}
}

func TestBuildMyHomeStreamGroupsSkipsEmptyLeaves(t *testing.T) {
	categories := []TaxonomyCategoryNode{{
		Slug: "supply-chain", Name: "Supply Chain", IconURL: "/c.svg",
		SubCategories: []TaxonomySubCategoryNode{
			{Slug: "procurement", Name: "Procurement"},
			{Slug: "order-fulfillment", Name: "Order Fulfillment"},
		},
	}}
	catalog := map[string]RuntimeConfig{
		"b": {Workflow: WorkflowDef{CategorySlug: "supply-chain", SubCategorySlug: "order-fulfillment"}},
	}
	cards := map[string]ManagedPublicStreamCardView{"b": {Key: "b"}}
	groups := buildMyHomeStreamGroups(categories, cards, catalog, []string{"b"})
	if len(groups) != 1 || groups[0].SubCategoryName != "Order Fulfillment" {
		t.Fatalf("got %+v", groups)
	}
	if groups[0].CategoryName != "Supply Chain" || groups[0].CategoryIconURL != "/c.svg" {
		t.Fatalf("first non-empty leaf under category should show category header = %+v", groups[0])
	}
}
