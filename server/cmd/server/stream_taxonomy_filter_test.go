package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func streamTaxonomyTestCategories() []TaxonomyCategoryNode {
	return []TaxonomyCategoryNode{
		{
			Slug:    "supply-chain",
			Name:    "Supply Chain",
			IconURL: "/static/taxonomy/supply-chain.svg",
			SubCategories: []TaxonomySubCategoryNode{
				{Slug: "procurement", Name: "Procurement", IconURL: "/static/taxonomy/procurement.svg"},
				{Slug: "fulfillment", Name: "Fulfillment", IconURL: "/static/taxonomy/fulfillment.svg"},
			},
		},
		{
			Slug:    "circularity",
			Name:    "Circularity",
			IconURL: "/static/taxonomy/circularity.svg",
			SubCategories: []TaxonomySubCategoryNode{
				{Slug: "recycling", Name: "Recycling", IconURL: "/static/taxonomy/recycling.svg"},
			},
		},
	}
}

func streamTaxonomyTestCandidates() []streamTaxonomyCandidate {
	card := func(name string) ManagedPublicStreamCardView {
		return ManagedPublicStreamCardView{Key: name, Card: PublicStreamCardView{Name: name}}
	}
	return []streamTaxonomyCandidate{
		{Card: card("Procurement Stream"), CategorySlug: "supply-chain", SubCategorySlug: "procurement"},
		{Card: card("Fulfillment Stream"), CategorySlug: "supply-chain", SubCategorySlug: "fulfillment"},
		{Card: card("Uncategorized Stream")},
	}
}

func streamTaxonomyTestCatalog(category, subCategory string, includeAll bool) StreamTaxonomyCatalogView {
	return buildStreamTaxonomyCatalogView(streamTaxonomyCatalogSpec{
		ID:                      "catalog",
		FormID:                  "filter",
		Action:                  "/my",
		ResultsID:               "results",
		IncludeAllCategories:    includeAll,
		AlwaysShowUncategorized: includeAll,
		EmptyHint:               "No matching streams.",
	}, streamTaxonomyTestCategories(), streamTaxonomyTestCandidates(), category, subCategory)
}

func streamTaxonomyStreamNames(streams []ManagedPublicStreamCardView) string {
	names := make([]string, 0, len(streams))
	for _, stream := range streams {
		names = append(names, stream.Card.Name)
	}
	return strings.Join(names, ",")
}

func TestBuildStreamTaxonomyCatalogViewSelectionMatrix(t *testing.T) {
	tests := []struct {
		name                string
		category            string
		subCategory         string
		wantStreams         string
		wantSubDisabled     bool
		wantSubCategorySize int
	}{
		{
			name:            "all",
			wantStreams:     "Procurement Stream,Fulfillment Stream,Uncategorized Stream",
			wantSubDisabled: true,
		},
		{
			name:                "category",
			category:            "supply-chain",
			wantStreams:         "Procurement Stream,Fulfillment Stream",
			wantSubCategorySize: 2,
		},
		{
			name:                "subcategory",
			category:            "supply-chain",
			subCategory:         "procurement",
			wantStreams:         "Procurement Stream",
			wantSubCategorySize: 2,
		},
		{
			name:            "uncategorized ignores subcategory",
			category:        streamTaxonomyUncategorized,
			subCategory:     "procurement",
			wantStreams:     "Uncategorized Stream",
			wantSubDisabled: true,
		},
		{
			name:                "category with no matches",
			category:            "circularity",
			wantStreams:         "",
			wantSubCategorySize: 1,
		},
		{
			name:            "invalid category resets to all",
			category:        "missing",
			subCategory:     "missing",
			wantStreams:     "Procurement Stream,Fulfillment Stream,Uncategorized Stream",
			wantSubDisabled: true,
		},
		{
			name:                "invalid subcategory resets to category",
			category:            "supply-chain",
			subCategory:         "missing",
			wantStreams:         "Procurement Stream,Fulfillment Stream",
			wantSubCategorySize: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := streamTaxonomyTestCatalog(tt.category, tt.subCategory, true)
			if got := streamTaxonomyStreamNames(view.Results.Streams); got != tt.wantStreams {
				t.Fatalf("streams = %q, want %q", got, tt.wantStreams)
			}
			if view.Filter.SubCategoryDisabled != tt.wantSubDisabled {
				t.Fatalf("SubCategoryDisabled = %t, want %t", view.Filter.SubCategoryDisabled, tt.wantSubDisabled)
			}
			if got := len(view.Filter.SubCategoryOptions); got != tt.wantSubCategorySize {
				t.Fatalf("subcategory options = %d, want %d", got, tt.wantSubCategorySize)
			}
		})
	}
}

func TestBuildStreamTaxonomyCatalogViewDiscoveryOptionsUseVisibleStreams(t *testing.T) {
	view := streamTaxonomyTestCatalog("", "", false)
	if got := len(view.Filter.CategoryOptions); got != 1 {
		t.Fatalf("category options = %d, want 1", got)
	}
	if got := view.Filter.CategoryOptions[0].Value; got != "supply-chain" {
		t.Fatalf("category option = %q, want supply-chain", got)
	}
	if !view.Filter.ShowUncategorized {
		t.Fatal("expected Uncategorized when a visible stream is uncategorized")
	}
}

func TestStreamTaxonomySelectionOnlyUsesRecognizedHTMXTargets(t *testing.T) {
	full := httptest.NewRequest(http.MethodGet, "/my?category=supply-chain&subCategory=procurement", nil)
	if category, subCategory := streamTaxonomySelectionFromRequest(full, "catalog", "results"); category != "" || subCategory != "" {
		t.Fatalf("full request selection = %q/%q, want all", category, subCategory)
	}

	otherTarget := httptest.NewRequest(http.MethodGet, "/my?category=supply-chain&subCategory=procurement", nil)
	otherTarget.Header.Set("HX-Request", "true")
	otherTarget.Header.Set("HX-Target", "other")
	if category, subCategory := streamTaxonomySelectionFromRequest(otherTarget, "catalog", "results"); category != "" || subCategory != "" {
		t.Fatalf("other HTMX target selection = %q/%q, want all", category, subCategory)
	}

	filter := httptest.NewRequest(http.MethodGet, "/my?category=supply-chain&subCategory=procurement", nil)
	filter.Header.Set("HX-Request", "true")
	filter.Header.Set("HX-Target", "results")
	if category, subCategory := streamTaxonomySelectionFromRequest(filter, "catalog", "results"); category != "supply-chain" || subCategory != "procurement" {
		t.Fatalf("filter selection = %q/%q", category, subCategory)
	}
}

func TestStreamTaxonomyFilterTemplateHTMXWiring(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var all bytes.Buffer
	if err := tmpl.ExecuteTemplate(&all, "stream_taxonomy_filter", streamTaxonomyTestCatalog("", "", true).Filter); err != nil {
		t.Fatalf("render all filter: %v", err)
	}
	allBody := all.String()
	for _, want := range []string{
		`name="category"`,
		`name="subCategory"`,
		`hx-target="#catalog"`,
		`hx-target="#results"`,
		`hx-include="#filter"`,
		`hx-trigger="change"`,
		`hx-push-url="false"`,
		`<selectedcontent></selectedcontent>`,
		`class="stream-taxonomy-filter-select"`,
		`src="/static/taxonomy/supply-chain.svg"`,
		"All categories",
		"All sub-categories",
		"Uncategorized",
		"disabled",
	} {
		if !strings.Contains(allBody, want) {
			t.Fatalf("all filter missing %q in %s", want, allBody)
		}
	}

	var category bytes.Buffer
	if err := tmpl.ExecuteTemplate(&category, "stream_taxonomy_filter", streamTaxonomyTestCatalog("supply-chain", "", true).Filter); err != nil {
		t.Fatalf("render category filter: %v", err)
	}
	categoryBody := category.String()
	subCategoryStart := strings.Index(categoryBody, `name="subCategory"`)
	if subCategoryStart < 0 {
		t.Fatalf("subcategory select missing from %s", categoryBody)
	}
	subCategoryEnd := strings.Index(categoryBody[subCategoryStart:], `</select>`)
	if subCategoryEnd < 0 {
		t.Fatalf("subcategory select missing from %s", categoryBody)
	}
	if strings.Contains(categoryBody[subCategoryStart:subCategoryStart+subCategoryEnd], "disabled") {
		t.Fatalf("real category must enable subcategory select: %s", categoryBody)
	}
	if !strings.Contains(categoryBody, `src="/static/taxonomy/procurement.svg"`) {
		t.Fatalf("expected subcategory icon in enabled select, got: %s", categoryBody)
	}
	clearIdx := strings.Index(categoryBody, "Clear filters")
	if clearIdx < 0 {
		t.Fatalf("expected Clear filters when a category is active, got: %s", categoryBody)
	}
	windowStart := clearIdx - 400
	if windowStart < 0 {
		windowStart = 0
	}
	if !strings.Contains(categoryBody[windowStart:clearIdx], `M18 6 6 18`) {
		t.Fatalf("expected close icon before Clear filters, got: %s", categoryBody[windowStart:clearIdx+20])
	}
	if strings.Contains(allBody, "Clear filters") {
		t.Fatalf("did not expect Clear filters on All categories, got: %s", allBody)
	}
	got := streamTaxonomyTestCatalog("supply-chain", "procurement", true)
	if len(got.Filter.CategoryOptions) == 0 || got.Filter.CategoryOptions[0].IconURL != "/static/taxonomy/supply-chain.svg" {
		t.Fatalf("category IconURL = %#v", got.Filter.CategoryOptions)
	}
	if len(got.Filter.SubCategoryOptions) == 0 || got.Filter.SubCategoryOptions[0].IconURL != "/static/taxonomy/procurement.svg" {
		t.Fatalf("subcategory IconURL = %#v", got.Filter.SubCategoryOptions)
	}
	for _, want := range []string{"Procurement", "Fulfillment"} {
		if !strings.Contains(categoryBody, want) {
			t.Fatalf("category filter missing %q in %s", want, categoryBody)
		}
	}
}
