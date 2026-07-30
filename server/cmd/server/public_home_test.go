package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestPublicStreamCardsForPathFiltersAndCaps(t *testing.T) {
	tempDir := t.TempDir()
	matchYAML := strings.Replace(
		minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n  subCategorySlug: procurement\n"),
		`name: "Workflow"`,
		`name: "Match Stream"`,
		1,
	)
	otherYAML := strings.Replace(
		minimalCategorizedWorkflowYAML("  categorySlug: supply-chain\n  subCategorySlug: order-fulfillment\n"),
		`name: "Workflow"`,
		`name: "Other Stream"`,
		1,
	)
	if err := os.WriteFile(filepath.Join(tempDir, "match.yaml"), []byte(matchYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "other.yaml"), []byte(otherYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{store: store, configDir: tempDir, tmpl: parseTestTemplates(t)}

	cards, err := server.publicStreamCardsForPath(t.Context(), "supply-chain", "procurement")
	if err != nil {
		t.Fatalf("publicStreamCardsForPath: %v", err)
	}
	if len(cards) != 1 || cards[0].Name != "Match Stream" {
		t.Fatalf("cards = %#v, want only Match Stream", cards)
	}

	empty, err := server.publicStreamCardsForPath(t.Context(), "", "")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty path = %#v err=%v", empty, err)
	}
}

func TestBuildPublicHomeCategoriesMarksActiveAndURLs(t *testing.T) {
	cats := []TaxonomyCategoryNode{
		{
			Slug: "supply-chain", Name: "Supply Chain", IconURL: "/static/taxonomy/batch-traceability.svg",
			SubCategories: []TaxonomySubCategoryNode{
				{Slug: "procurement", Name: "Procurement"},
				{Slug: "order-fulfillment", Name: "Order Fulfillment"},
			},
		},
		{
			Slug: "compliance-and-quality", Name: "Compliance and Quality", IconURL: "/static/taxonomy/quality-control.svg",
			SubCategories: []TaxonomySubCategoryNode{
				{Slug: "inspection", Name: "Inspection"},
			},
		},
	}
	got := buildPublicHomeCategories(cats, "supply-chain", "order-fulfillment")
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if !got[0].Expanded || got[1].Expanded {
		t.Fatalf("expanded flags: %#v", got)
	}
	if got[0].SubCategories[1].Active != true || got[0].SubCategories[0].Active {
		t.Fatalf("active flags: %#v", got[0].SubCategories)
	}
	wantPartial := "/streams/public?category=supply-chain&subCategory=order-fulfillment"
	wantPush := "/?category=supply-chain&subCategory=order-fulfillment"
	if got[0].SubCategories[1].PartialURL != wantPartial {
		t.Fatalf("PartialURL=%q", got[0].SubCategories[1].PartialURL)
	}
	if got[0].SubCategories[1].PushURL != wantPush {
		t.Fatalf("PushURL=%q", got[0].SubCategories[1].PushURL)
	}
	if got[1].Name != "Compliance and Quality" || len(got[1].SubCategories) != 1 {
		t.Fatalf("expected zero-stream category still present: %#v", got[1])
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
