package main

import "testing"

func TestTaxonomyHasPath(t *testing.T) {
	cats := []TaxonomyCategoryNode{{
		Slug: "supply-chain",
		SubCategories: []TaxonomySubCategoryNode{
			{Slug: "procurement"},
			{Slug: "order-fulfillment"},
		},
	}}
	if !taxonomyHasPath(cats, "supply-chain", "procurement") {
		t.Fatal("expected known path")
	}
	if taxonomyHasPath(cats, "supply-chain", "missing") {
		t.Fatal("expected missing sub to be false")
	}
	if taxonomyHasPath(cats, "missing", "procurement") {
		t.Fatal("expected missing cat to be false")
	}
}

func TestResolvePublicHomeSelectionDefaultsFirstFirst(t *testing.T) {
	cats := []TaxonomyCategoryNode{
		{
			Slug: "recycling-and-recovery",
			SubCategories: []TaxonomySubCategoryNode{
				{Slug: "photovoltaic-panels"},
				{Slug: "led-lighting"},
			},
		},
		{
			Slug: "supply-chain",
			SubCategories: []TaxonomySubCategoryNode{
				{Slug: "procurement"},
			},
		},
	}
	cat, sub := resolvePublicHomeSelection(cats, "", "")
	if cat != "recycling-and-recovery" || sub != "photovoltaic-panels" {
		t.Fatalf("got %q/%q, want first/first", cat, sub)
	}
	cat, sub = resolvePublicHomeSelection(cats, "supply-chain", "procurement")
	if cat != "supply-chain" || sub != "procurement" {
		t.Fatalf("got %q/%q, want query path", cat, sub)
	}
	cat, sub = resolvePublicHomeSelection(cats, "supply-chain", "nope")
	if cat != "recycling-and-recovery" || sub != "photovoltaic-panels" {
		t.Fatalf("invalid query must fall back, got %q/%q", cat, sub)
	}
}

func TestResolvePublicHomeSelectionEmptyTaxonomy(t *testing.T) {
	cat, sub := resolvePublicHomeSelection(nil, "a", "b")
	if cat != "" || sub != "" {
		t.Fatalf("got %q/%q, want empty", cat, sub)
	}
}

func TestPublicHomeCreateStreamHref(t *testing.T) {
	if got := publicHomeCreateStreamHref(false); got != "/login?next=%2Fmy%2Forganization%2Fformata-builder" {
		t.Fatalf("anonymous href = %q", got)
	}
	if got := publicHomeCreateStreamHref(true); got != "/my/organization/formata-builder" {
		t.Fatalf("signed-in href = %q", got)
	}
}
