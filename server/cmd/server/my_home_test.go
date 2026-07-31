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
	if groups[0].CategoryName != "Supply Chain" || groups[0].CategoryIconURL != "/c.svg" || !groups[0].ShowCategoryHeader {
		t.Fatalf("first group should show category header = %+v", groups[0])
	}
	if groups[1].SubCategoryName != "Order Fulfillment" || groups[1].Streams[0].Key != "a" {
		t.Fatalf("second group = %+v", groups[1])
	}
	if groups[1].ShowCategoryHeader {
		t.Fatalf("second group under same category must omit category header = %+v", groups[1])
	}
	if groups[1].CategoryName != "Supply Chain" || groups[1].CategoryIconURL != "/c.svg" {
		t.Fatalf("second group must retain category metadata for sidebar builder = %+v", groups[1])
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
	if groups[0].CategoryName != "Supply Chain" || groups[0].CategoryIconURL != "/c.svg" || !groups[0].ShowCategoryHeader {
		t.Fatalf("first non-empty leaf under category should show category header = %+v", groups[0])
	}
}

func TestMyHomeAnchorID(t *testing.T) {
	if got := myHomeAnchorID("supply-chain", "procurement", false); got != "cat-supply-chain--procurement" {
		t.Fatalf("got %q", got)
	}
	if got := myHomeAnchorID("", "", true); got != "cat-uncategorized" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildMyHomeStreamGroupsSetsAnchorIDs(t *testing.T) {
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
	if groups[0].AnchorID != "cat-supply-chain--procurement" || groups[0].CategorySlug != "supply-chain" {
		t.Fatalf("first=%+v", groups[0])
	}
	if !groups[0].ShowCategoryHeader || groups[0].CategoryName != "Supply Chain" {
		t.Fatalf("first header=%+v", groups[0])
	}
	if groups[1].AnchorID != "cat-supply-chain--order-fulfillment" || groups[1].ShowCategoryHeader {
		t.Fatalf("second=%+v", groups[1])
	}
	if groups[1].CategoryName != "Supply Chain" {
		t.Fatalf("second must retain CategoryName for sidebar builder, got %+v", groups[1])
	}
	if groups[2].AnchorID != "cat-uncategorized" || !groups[2].Uncategorized {
		t.Fatalf("uncat=%+v", groups[2])
	}
}

func TestBuildMyHomeCategorySidebarFromGroups(t *testing.T) {
	groups := []MyHomeStreamGroupView{
		{CategorySlug: "supply-chain", CategoryName: "Supply Chain", CategoryIconURL: "/c.svg", SubCategorySlug: "procurement", SubCategoryName: "Procurement", SubCategoryIconURL: "/s.svg", AnchorID: "cat-supply-chain--procurement", ShowCategoryHeader: true},
		{CategorySlug: "supply-chain", CategoryName: "Supply Chain", CategoryIconURL: "/c.svg", SubCategorySlug: "order-fulfillment", SubCategoryName: "Order Fulfillment", SubCategoryIconURL: "/o.svg", AnchorID: "cat-supply-chain--order-fulfillment"},
		{Uncategorized: true, CategoryName: "Uncategorized", AnchorID: "cat-uncategorized", ShowCategoryHeader: true},
	}
	got := buildMyHomeCategorySidebar(groups)
	if got.Title != "Stream Categories" || len(got.Categories) != 2 {
		t.Fatalf("got=%+v", got)
	}
	if got.CloseTargetID != "my-home-category-sidebar" {
		t.Fatalf("CloseTargetID=%q", got.CloseTargetID)
	}
	if !got.Categories[0].Expanded || got.Categories[0].Name != "Supply Chain" || len(got.Categories[0].SubCategories) != 2 {
		t.Fatalf("cat0=%+v", got.Categories[0])
	}
	if got.Categories[0].SubCategories[0].Href != "#cat-supply-chain--procurement" {
		t.Fatalf("href0=%q", got.Categories[0].SubCategories[0].Href)
	}
	if got.Categories[0].SubCategories[0].IconURL != "/s.svg" {
		t.Fatalf("IconURL0=%q", got.Categories[0].SubCategories[0].IconURL)
	}
	if got.Categories[1].Name != "Uncategorized" || got.Categories[1].SubCategories[0].Href != "#cat-uncategorized" {
		t.Fatalf("uncat=%+v", got.Categories[1])
	}
	if got.Categories[1].Expanded {
		t.Fatalf("only first category should start expanded, uncat=%+v", got.Categories[1])
	}
	if got.Categories[0].SubCategories[0].PartialURL != "" {
		t.Fatalf("my leaves must not set PartialURL")
	}
}
