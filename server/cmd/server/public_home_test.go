package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
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
	if cards[0].Href != "/streams/match" {
		t.Fatalf("Href = %q, want /streams/match", cards[0].Href)
	}

	empty, err := server.publicStreamCardsForPath(t.Context(), "", "")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty path = %#v err=%v", empty, err)
	}
}

func TestBuildPublicHomeStreamResultsViewFillsSelection(t *testing.T) {
	cats := []TaxonomyCategoryNode{
		{
			Slug: "supply-chain", Name: "Supply Chain", IconURL: "/static/taxonomy/batch-traceability.svg",
			SubCategories: []TaxonomySubCategoryNode{
				{Slug: "procurement", Name: "Procurement", IconURL: "/static/taxonomy/procurement-workflow.svg", Description: "PO management"},
			},
		},
	}
	streams := []PublicStreamCardView{{Name: "Example Stream"}}
	got := buildPublicHomeStreamResultsView(cats, "supply-chain", "procurement", streams, "/login?next=builder")
	if got.CategoryName != "Supply Chain" || got.CategoryIconURL != "/static/taxonomy/batch-traceability.svg" {
		t.Fatalf("category fields = %#v", got)
	}
	if got.SubCategoryName != "Procurement" || got.SubCategoryIconURL != "/static/taxonomy/procurement-workflow.svg" {
		t.Fatalf("subcategory fields = %#v", got)
	}
	if got.SubCategoryDescription != "PO management" {
		t.Fatalf("SubCategoryDescription = %q", got.SubCategoryDescription)
	}
	if len(got.Streams) != 1 || got.CreateStreamHref != "/login?next=builder" {
		t.Fatalf("streams/create href = %#v", got)
	}
}

func TestBuildPublicHomeCategoriesMarksActiveAndURLs(t *testing.T) {
	cats := []TaxonomyCategoryNode{
		{
			Slug: "supply-chain", Name: "Supply Chain", IconURL: "/static/taxonomy/batch-traceability.svg",
			SubCategories: []TaxonomySubCategoryNode{
				{Slug: "procurement", Name: "Procurement", IconURL: "/static/taxonomy/procurement-workflow.svg"},
				{Slug: "order-fulfillment", Name: "Order Fulfillment", IconURL: "/static/taxonomy/order-fulfillment.svg"},
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
	if got.Title != "Stream Categories" {
		t.Fatalf("title=%q", got.Title)
	}
	if len(got.Categories) != 2 {
		t.Fatalf("len=%d", len(got.Categories))
	}
	if !got.Categories[0].Expanded || got.Categories[1].Expanded {
		t.Fatalf("expanded flags: %#v", got.Categories)
	}
	if got.Categories[0].SubCategories[1].Active != true || got.Categories[0].SubCategories[0].Active {
		t.Fatalf("active flags: %#v", got.Categories[0].SubCategories)
	}
	if got.Categories[0].SubCategories[1].IconURL != "/static/taxonomy/order-fulfillment.svg" {
		t.Fatalf("IconURL=%q", got.Categories[0].SubCategories[1].IconURL)
	}
	wantPartial := "/streams/public?category=supply-chain&subCategory=order-fulfillment"
	wantPush := "/?category=supply-chain&subCategory=order-fulfillment"
	if got.Categories[0].SubCategories[1].PartialURL != wantPartial {
		t.Fatalf("PartialURL=%q", got.Categories[0].SubCategories[1].PartialURL)
	}
	if got.Categories[0].SubCategories[1].PushURL != wantPush {
		t.Fatalf("PushURL=%q", got.Categories[0].SubCategories[1].PushURL)
	}
	if got.Categories[1].Name != "Compliance and Quality" || len(got.Categories[1].SubCategories) != 1 {
		t.Fatalf("expected zero-stream category still present: %#v", got.Categories[1])
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

func TestHandlePublicStreamsPartialFilters(t *testing.T) {
	tempDir := t.TempDir()
	writePublicHomeWorkflowConfig(t, filepath.Join(tempDir, "match.yaml"), "Match Stream", "string", "Filtered stream")

	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{store: store, configDir: tempDir, tmpl: parseTestTemplates(t)}

	req := httptest.NewRequest(http.MethodGet, "/streams/public?category=supply-chain&subCategory=procurement", nil)
	rec := httptest.NewRecorder()
	server.handlePublicStreamsPartial(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Match Stream") {
		t.Fatalf("missing stream name: %s", body)
	}
	if !strings.Contains(body, `class="public-home-stream-results"`) {
		t.Fatalf("missing results wrapper: %s", body)
	}
	if !strings.Contains(body, `class="public-home-results-category-header"`) {
		t.Fatalf("missing results category header: %s", body)
	}
	if !strings.Contains(body, `class="public-home-results-category-name">Supply Chain<`) {
		t.Fatalf("missing category name in results: %s", body)
	}
	if !strings.Contains(body, `class="public-home-results-subcategory-header"`) {
		t.Fatalf("missing results subcategory header: %s", body)
	}
	if !strings.Contains(body, `class="public-home-results-subcategory-name"`) {
		t.Fatalf("missing subcategory name element in results: %s", body)
	}
	if !strings.Contains(body, "Procurement") {
		t.Fatalf("missing subcategory name in results: %s", body)
	}
	if !strings.Contains(body, `class="public-home-results-subcategory-description"`) {
		t.Fatalf("missing subcategory description element in results: %s", body)
	}
	if !strings.Contains(body, "PO management") {
		t.Fatalf("missing subcategory description in results: %s", body)
	}
	if strings.Contains(body, `class="public-home-header"`) {
		t.Fatalf("partial must not include landing header: %s", body)
	}
	if strings.Contains(body, "Stream Categories") {
		t.Fatalf("partial must not include sidebar header: %s", body)
	}
}

func TestHandlePublicStreamsPartialUnknownPath404(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{store: store, configDir: t.TempDir(), tmpl: parseTestTemplates(t)}
	req := httptest.NewRequest(http.MethodGet, "/streams/public?category=nope&subCategory=nope", nil)
	rec := httptest.NewRecorder()
	server.handlePublicStreamsPartial(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHandlePublicHomeRendersSubcategoryHeaderFromTaxonomySeed(t *testing.T) {
	store := NewMemoryStore()
	yamlPath := filepath.Join("..", "..", "config", "categories.yaml")
	if err := seedTaxonomyFromFile(t.Context(), store, yamlPath); err != nil {
		t.Fatalf("seedTaxonomyFromFile: %v", err)
	}
	server := &Server{store: store, configDir: t.TempDir(), tmpl: parseTestTemplates(t)}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`class="public-home-results-subcategory-header"`,
		`class="public-home-results-subcategory-name"`,
		"Photovoltaic Panels",
		`class="public-home-results-subcategory-description"`,
		"Tracking disassembly and material recovery from photovoltaic panels",
		`/static/taxonomy/photovoltaic-module.svg`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func TestHandlePublicStreamsPartialMissingQuery404(t *testing.T) {
	server := &Server{store: NewMemoryStore(), configDir: t.TempDir(), tmpl: parseTestTemplates(t)}
	req := httptest.NewRequest(http.MethodGet, "/streams/public", nil)
	rec := httptest.NewRecorder()
	server.handlePublicStreamsPartial(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestHandlePublicStreamsPartialEmptyShowsCreateCTA(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{store: store, configDir: t.TempDir(), tmpl: parseTestTemplates(t)}
	req := httptest.NewRequest(http.MethodGet, "/streams/public?category=supply-chain&subCategory=procurement", nil)
	rec := httptest.NewRecorder()
	server.handlePublicStreamsPartial(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "No public streams in this category yet") {
		t.Fatalf("missing empty copy: %s", body)
	}
	if !strings.Contains(body, `href="/login?next=%2Fmy%2Forganization%2Fformata-builder"`) {
		t.Fatalf("missing create CTA: %s", body)
	}
	if !strings.Contains(body, "Create a stream") {
		t.Fatalf("missing CTA label: %s", body)
	}
}

func TestHandlePublicStreamsPartialMethodNotAllowed(t *testing.T) {
	server := &Server{store: NewMemoryStore(), tmpl: parseTestTemplates(t)}
	rec := httptest.NewRecorder()
	server.handlePublicStreamsPartial(rec, httptest.NewRequest(http.MethodPost, "/streams/public", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandlePublicStreamsPartialLoadErrors(t *testing.T) {
	store := &failingListCategoriesStore{MemoryStore: NewMemoryStore(), err: errors.New("taxonomy down")}
	server := &Server{store: store, tmpl: parseTestTemplates(t)}
	rec := httptest.NewRecorder()
	server.handlePublicStreamsPartial(rec, httptest.NewRequest(http.MethodGet, "/streams/public?category=a&subCategory=b", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}
