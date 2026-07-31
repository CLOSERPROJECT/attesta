# Public home category filter sidebar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the public home horizontal category tabs with a taxonomy accordion sidebar and HTMX-filtered stream results (light tokens, Lucide chrome icons).

**Architecture:** Full-page `GET /` loads taxonomy + resolves `category`/`subCategory` (default first/first), SSR sidebar + results. `GET /streams/public` returns only the results partial for HTMX swap. Filter catalog streams by `Workflow.CategorySlug` + `SubCategorySlug` after the existing 6-card cap applied post-filter. Accordion expand is native `<details>`; subcategory clicks are HTMX.

**Tech Stack:** Go `net/http` + `html/template`, HTMX 1.9 (already in `layout.html`), Vite CSS (`public-home.css`), Lucide glyphs via `icons.html` (same paths as `@lucide/svelte`, `currentColor` — not `<img>` static files, which cannot inherit theme color).

**Spec:** `docs/superpowers/specs/2026-07-30-public-home-category-sidebar-design.md`

## Global Constraints

- Light shared tokens only — no new `--landing-*`, no forced dark sidebar.
- All taxonomy categories/subcategories render even with zero matching streams.
- Parent category = expand/collapse only; filter only on subcategory.
- Default selection = first category, first subcategory by `sortOrder` (ignore stream counts).
- Empty results CTA **Create a stream**: anonymous → `/login?next=/my/organization/formata-builder`; signed-in (`ShowLogout`) → `/my/organization/formata-builder`.
- Remove horizontal tabs + `initLandingTabs` + **See all streams** link.
- Partial path exactly `GET /streams/public`; unknown/missing query slugs → 404.
- Uncategorized catalog streams never appear in filtered results.
- Follow `.agents/skills/attesta-ui-components` + `docs/css.md` (results partial = **full** component because it is an HTMX swap target).

## File map

| File | Responsibility |
|------|----------------|
| `server/cmd/server/public_home.go` | Selection, sidebar views, filtered cards, create-stream href, handlers helpers |
| `server/cmd/server/public_home_test.go` | Unit tests for selection/filter/href helpers |
| `server/cmd/server/components.go` | `PublicHomeStreamResultsView`, sidebar row view types |
| `server/cmd/server/main.go` | Wire `handlePublicHome`, register `/streams/public`, thin handler stubs if needed |
| `server/templates/components/public_home_stream_results.html` | HTMX swap target fragment |
| `server/templates/pages/public_home.html` | Sidebar + results shell; remove tabs / See all |
| `server/templates/icons.html` | `icon-layout-grid`, `icon-chevron-down`, `icon-chevron-right` (Lucide) |
| `web/src/styles/pages/public-home.css` | Two-column / stack layout; sidebar states; delete tabs rules |
| `web/src/main.js` | Remove `initLandingTabs`; subcategory active-class sync |
| `server/cmd/server/home_handler_test.go` | Update catalog-card tests for taxonomy filter; add sidebar/partial/CTA cases |
| `docs/css.md` | Only if a new component CSS module is added (prefer page CSS for sidebar) |

---

### Task 1: Selection + path helpers (pure Go)

**Files:**
- Create: `server/cmd/server/public_home.go`
- Create: `server/cmd/server/public_home_test.go`

**Interfaces:**
- Produces:
  - `func taxonomyHasPath(categories []TaxonomyCategoryNode, categorySlug, subCategorySlug string) bool`
  - `func resolvePublicHomeSelection(categories []TaxonomyCategoryNode, categorySlug, subCategorySlug string) (cat, sub string)`
  - `func publicHomeCreateStreamHref(signedIn bool) string`

- [ ] **Step 1: Write failing tests**

Create `server/cmd/server/public_home_test.go`:

```go
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
	if got := publicHomeCreateStreamHref(false); got != "/login?next=/my/organization/formata-builder" {
		t.Fatalf("anonymous href = %q", got)
	}
	if got := publicHomeCreateStreamHref(true); got != "/my/organization/formata-builder" {
		t.Fatalf("signed-in href = %q", got)
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL**

Run:

```bash
cd server/cmd/server && go test -count=1 -run 'TestTaxonomyHasPath|TestResolvePublicHomeSelection|TestPublicHomeCreateStreamHref' .
```

Expected: FAIL — undefined functions.

- [ ] **Step 3: Implement helpers**

Create `server/cmd/server/public_home.go`:

```go
package main

import (
	"net/url"
	"strings"
)

func taxonomyHasPath(categories []TaxonomyCategoryNode, categorySlug, subCategorySlug string) bool {
	cat := strings.TrimSpace(categorySlug)
	sub := strings.TrimSpace(subCategorySlug)
	if cat == "" || sub == "" {
		return false
	}
	for _, category := range categories {
		if category.Slug != cat {
			continue
		}
		for _, leaf := range category.SubCategories {
			if leaf.Slug == sub {
				return true
			}
		}
		return false
	}
	return false
}

func resolvePublicHomeSelection(categories []TaxonomyCategoryNode, categorySlug, subCategorySlug string) (string, string) {
	if taxonomyHasPath(categories, categorySlug, subCategorySlug) {
		return strings.TrimSpace(categorySlug), strings.TrimSpace(subCategorySlug)
	}
	if len(categories) == 0 || len(categories[0].SubCategories) == 0 {
		return "", ""
	}
	return categories[0].Slug, categories[0].SubCategories[0].Slug
}

func publicHomeCreateStreamHref(signedIn bool) string {
	target := organizationPath("formata-builder")
	if signedIn {
		return target
	}
	return "/login?next=" + url.QueryEscape(target)
}
```

Note: `loadTaxonomyTree` already returns categories ordered by store `sortOrder` and subs ordered likewise — do not re-sort here unless tests prove otherwise.

- [ ] **Step 4: Run tests — expect PASS**

Run the same `go test` command as Step 2. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/public_home.go server/cmd/server/public_home_test.go
git commit -m "$(cat <<'EOF'
feat(home): add public home taxonomy selection helpers

Resolve category/subCategory defaults and Create a stream login-next href for the landing filter sidebar.
EOF
)"
```

---

### Task 2: Filtered public stream cards

**Files:**
- Modify: `server/cmd/server/main.go` (`publicStreamCards` / callers around lines 2066–2126)
- Modify: `server/cmd/server/components.go` (`PublicStreamCardView` — optional CategorySlug fields only if needed for debugging; prefer filter-before-append without exposing on card)
- Modify: `server/cmd/server/home_handler_test.go` (catalog card tests that currently expect uncategorized streams)
- Modify or extend: `server/cmd/server/public_home.go` / `public_home_test.go`

**Interfaces:**
- Consumes: `resolvePublicHomeSelection`, taxonomy path helpers
- Produces: `func (s *Server) publicStreamCardsForPath(ctx context.Context, categorySlug, subCategorySlug string) ([]PublicStreamCardView, error)` — when either slug empty, return `nil, nil` (empty results). Cap with `publicHomeStreamCardLimit` **after** filter. Match when `cfg.Workflow.CategorySlug` and `SubCategorySlug` equal the path (use `IsCategorized()` / trim). Refactor existing `publicStreamCards` to call `publicStreamCardsForPath` only from handlers that pass a path — **do not** keep an unfiltered public-home path.

- [ ] **Step 1: Write failing filter test**

In `public_home_test.go` (handler-level is fine in `home_handler_test.go` if preferred):

```go
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
	seedPlatformAdminTaxonomy(t, store) // keeps categorized pairs valid at catalog load
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
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
cd server/cmd/server && go test -count=1 -run TestPublicStreamCardsForPathFiltersAndCaps .
```

Expected: FAIL — method undefined.

- [ ] **Step 3: Implement `publicStreamCardsForPath`**

In `public_home.go` (or peel from `main.go`):

```go
func (s *Server) publicStreamCardsForPath(ctx context.Context, categorySlug, subCategorySlug string) ([]PublicStreamCardView, error) {
	cat := strings.TrimSpace(categorySlug)
	sub := strings.TrimSpace(subCategorySlug)
	if cat == "" || sub == "" {
		return nil, nil
	}
	catalog, err := s.workflowCatalog()
	if err != nil {
		if err.Error() == "workflow config catalog is empty" {
			return nil, nil
		}
		return nil, err
	}
	keys := sortedWorkflowKeys(catalog)
	logoURLs := organizationLogoURLMap(ctx, s.identity)
	cards := make([]PublicStreamCardView, 0, publicHomeStreamCardLimit)
	for _, key := range keys {
		cfg := catalog[key]
		if !cfg.Workflow.IsCategorized() {
			continue
		}
		if cfg.Workflow.CategorySlug != cat || cfg.Workflow.SubCategorySlug != sub {
			continue
		}
		// reuse the card-building body currently inside publicStreamCards
		// (steps, orgs, instance metrics) — extract a private
		// buildPublicStreamCardView(ctx, key, cfg, logoURLs) (*PublicStreamCardView, error)
		// to avoid duplication.
		card, buildErr := s.buildPublicStreamCardView(ctx, key, cfg, logoURLs)
		if buildErr != nil {
			return nil, buildErr
		}
		cards = append(cards, card)
		if len(cards) >= publicHomeStreamCardLimit {
			break
		}
	}
	return cards, nil
}
```

Refactor `publicStreamCards` in `main.go` to either delete it or make it call `publicStreamCardsForPath` after the handler resolves selection (preferred: delete the unfiltered helper and only use `ForPath`).

- [ ] **Step 4: Fix existing public-home catalog tests**

Any test that writes uncategorized YAML and expects cards on `/` will now see **empty** results when taxonomy is empty or default leaf has no matches. Update those tests to:

1. `seedPlatformAdminTaxonomy(t, store)` (or a two-category seed if needed).
2. Write workflows with `categorySlug: supply-chain` + `subCategorySlug: procurement` (first/first of that seed).
3. Keep assertions on card content.

Tests to revisit (non-exhaustive — `rg public-stream-card home_handler_test.go`):

- `TestHandlePublicHomeRendersCatalogStreamCards`
- `TestHandlePublicHomeRendersStreamStepPreviewFromCatalog`
- passport / metrics / org avatar / limit tests that assert card presence

`TestHandlePublicHomeEmptyCatalogRendersNoStreamCards` should still pass (empty catalog → no cards; also assert Create CTA appears once UI exists — can wait for Task 4).

- [ ] **Step 5: Run targeted tests**

```bash
cd server/cmd/server && go test -count=1 -run 'TestPublicStreamCardsForPath|TestHandlePublicHome' .
```

Expected: PASS for filter + updated home tests (template markup may still show old tabs until Task 4–5; card assertions should pass).

- [ ] **Step 6: Commit**

```bash
git add server/cmd/server/public_home.go server/cmd/server/public_home_test.go server/cmd/server/main.go server/cmd/server/home_handler_test.go
git commit -m "$(cat <<'EOF'
feat(home): filter public stream cards by taxonomy path

Cap results after subcategory match so the landing grid only shows categorized catalog streams for the selected leaf.
EOF
)"
```

---

### Task 3: Sidebar view builder + results view struct

**Files:**
- Modify: `server/cmd/server/components.go`
- Modify: `server/cmd/server/public_home.go`
- Modify: `server/cmd/server/public_home_test.go`

**Interfaces:**
- Produces:

```go
// PublicHomeSubCategoryView is one filterable leaf in the landing sidebar.
type PublicHomeSubCategoryView struct {
	Slug      string
	Name      string
	Active    bool
	PartialURL string // /streams/public?category=…&subCategory=…
	PushURL    string // /?category=…&subCategory=…
}

// PublicHomeCategoryView is one accordion category on the landing sidebar.
type PublicHomeCategoryView struct {
	Slug          string
	Name          string
	IconURL       string
	Expanded      bool
	SubCategories []PublicHomeSubCategoryView
}

// PublicHomeStreamResultsView is the HTMX swap target for landing stream cards.
type PublicHomeStreamResultsView struct {
	Streams          []PublicStreamCardView
	CreateStreamHref string
}
```

```go
func buildPublicHomeCategories(categories []TaxonomyCategoryNode, selectedCat, selectedSub string) []PublicHomeCategoryView
```

Rules: every category/sub included; `Expanded == true` only for `selectedCat` on first paint (spec allows multi-open via user toggles later — SSR only needs the selected category expanded); `Active` on matching sub; URLs use `url.Values`.

- [ ] **Step 1: Write failing builder test**

```go
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
```

- [ ] **Step 2: Run — expect FAIL**

```bash
cd server/cmd/server && go test -count=1 -run TestBuildPublicHomeCategoriesMarksActiveAndURLs .
```

- [ ] **Step 3: Implement types + builder**

Add structs to `components.go`. Implement `buildPublicHomeCategories` in `public_home.go` using `url.Values{"category": {cat}, "subCategory": {sub}}.Encode()`.

- [ ] **Step 4: Run — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/components.go server/cmd/server/public_home.go server/cmd/server/public_home_test.go
git commit -m "$(cat <<'EOF'
feat(home): build public home sidebar category views

Map taxonomy trees into accordion rows with HTMX partial and push URLs for subcategory filters.
EOF
)"
```

---

### Task 4: Results partial template + `/streams/public` handler

**Files:**
- Create: `server/templates/components/public_home_stream_results.html`
- Modify: `server/cmd/server/public_home.go` (handlers)
- Modify: `server/cmd/server/main.go` (`newMux` register + slim `handlePublicHome` wiring)
- Modify: `server/cmd/server/home_handler_test.go` or `public_home_test.go`

**Interfaces:**
- Produces: `func (s *Server) handlePublicStreamsPartial(w http.ResponseWriter, r *http.Request)`
- Template define: `public_home_stream_results`
- Mux: `mux.HandleFunc("/streams/public", s.handlePublicStreamsPartial)` registered **before** `mux.HandleFunc("/", …)` for clarity

- [ ] **Step 1: Write failing handler tests**

```go
func TestHandlePublicStreamsPartialFilters(t *testing.T) {
	// seed taxonomy + one matching categorized workflow (same as Task 2)
	// GET /streams/public?category=supply-chain&subCategory=procurement
	// expect 200, body contains stream name, does NOT contain public-home-header / Stream Categories sidebar header
	// expect class public-home-stream-results
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
	if !strings.Contains(body, `href="/login?next=/my/organization/formata-builder"`) {
		t.Fatalf("missing create CTA: %s", body)
	}
	if !strings.Contains(body, "Create a stream") {
		t.Fatalf("missing CTA label: %s", body)
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (handler/template missing)

- [ ] **Step 3: Add results template**

Create `server/templates/components/public_home_stream_results.html`:

```html
{{ define "public_home_stream_results" }}
  <div class="public-home-stream-results" id="public-home-stream-results">
    {{ if .Streams }}
      <div class="public-home-stream-grid">
        {{ range .Streams }}
          {{ template "public_stream_card" . }}
        {{ end }}
      </div>
    {{ else }}
      <div class="public-home-stream-empty">
        <p>No public streams in this category yet.</p>
        <a class="btn btn-primary btn-lg" href="{{ .CreateStreamHref }}">Create a stream</a>
      </div>
    {{ end }}
  </div>
{{ end }}
```

- [ ] **Step 4: Implement handler**

```go
func (s *Server) handlePublicStreamsPartial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cat := strings.TrimSpace(r.URL.Query().Get("category"))
	sub := strings.TrimSpace(r.URL.Query().Get("subCategory"))
	categories, err := loadTaxonomyTree(r.Context(), s.store)
	if err != nil {
		http.Error(w, "failed to load categories", http.StatusInternalServerError)
		return
	}
	if !taxonomyHasPath(categories, cat, sub) {
		http.NotFound(w, r)
		return
	}
	streams, err := s.publicStreamCardsForPath(r.Context(), cat, sub)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	signedIn := false
	if _, _, err := s.currentUser(r); err == nil {
		signedIn = true
	}
	view := PublicHomeStreamResultsView{
		Streams:          streams,
		CreateStreamHref: publicHomeCreateStreamHref(signedIn),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "public_home_stream_results", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
```

Register in `newMux()`:

```go
mux.HandleFunc("/streams/public", s.handlePublicStreamsPartial)
```

(place near other public routes, before `/`).

- [ ] **Step 5: Run tests — expect PASS**

```bash
cd server/cmd/server && go test -count=1 -run 'TestHandlePublicStreamsPartial' .
```

- [ ] **Step 6: Commit**

```bash
git add server/templates/components/public_home_stream_results.html server/cmd/server/public_home.go server/cmd/server/main.go server/cmd/server/public_home_test.go server/cmd/server/home_handler_test.go
git commit -m "$(cat <<'EOF'
feat(home): add HTMX public streams filter partial

Serve /streams/public as a results-only fragment filtered by taxonomy category and subcategory.
EOF
)"
```

---

### Task 5: Lucide chrome icons + sidebar markup on `/`

**Files:**
- Modify: `server/templates/icons.html`
- Modify: `server/templates/pages/public_home.html`
- Modify: `server/cmd/server/main.go` (`handlePublicHome` view struct)
- Modify: `server/cmd/server/home_handler_test.go`

**Interfaces:**
- `handlePublicHome` view gains:
  - `Categories []PublicHomeCategoryView`
  - `StreamResults PublicHomeStreamResultsView`
- Icons: `icon-layout-grid`, `icon-chevron-down`, `icon-chevron-right` — Lucide paths from `@lucide/svelte` v0.561 (`layout-grid` four rects; chevrons `m6 9 6 6 6-6` / `m9 18 6-6-6-6`), same SVG attrs as existing `icon-*` defines (`stroke="currentColor"`, `class="icon-svg …"`).

- [ ] **Step 1: Write / extend failing full-page tests**

```go
func TestHandlePublicHomeRendersTaxonomySidebar(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	// optional: categorized stream on procurement
	server := &Server{store: store, configDir: t.TempDir(), tmpl: parseTestTemplates(t)}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	body := rec.Body.String()
	for _, want := range []string{
		`class="public-home-category-sidebar"`,
		"Supply Chain",
		"Procurement",
		"Order Fulfillment",
		`hx-get="/streams/public?category=supply-chain&subCategory=procurement"`,
		`hx-target="#public-home-stream-results"`,
		`hx-push-url="/?category=supply-chain&subCategory=procurement"`,
		`id="public-home-stream-results"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	for _, gone := range []string{
		`data-landing-tabs`,
		`public-home-tabs`,
		"See all streams",
		`icon-cat-compliance.svg`,
	} {
		if strings.Contains(body, gone) {
			t.Fatalf("did not expect %q", gone)
		}
	}
}

func TestHandlePublicHomeQuerySelectsSubcategory(t *testing.T) {
	store := NewMemoryStore()
	seedPlatformAdminTaxonomy(t, store)
	server := &Server{store: store, configDir: t.TempDir(), tmpl: parseTestTemplates(t)}
	req := httptest.NewRequest(http.MethodGet, "/?category=supply-chain&subCategory=order-fulfillment", nil)
	rec := httptest.NewRecorder()
	server.handlePublicHome(rec, req)
	body := rec.Body.String()
	// Active class on order-fulfillment button; details open on supply-chain
	if !strings.Contains(body, `subCategory=order-fulfillment`) || !strings.Contains(body, "is-active") {
		t.Fatalf("expected active subcategory markup, got %s", body)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

- [ ] **Step 3: Add Lucide icon defines**

Append to `server/templates/icons.html` (mirror existing icon SVG wrapper):

```html
{{ define "icon-layout-grid" }}
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-svg icon-svg-md" aria-hidden="true">
    <rect width="7" height="7" x="3" y="3" rx="1" />
    <rect width="7" height="7" x="14" y="3" rx="1" />
    <rect width="7" height="7" x="14" y="14" rx="1" />
    <rect width="7" height="7" x="3" y="14" rx="1" />
  </svg>
{{ end }}

{{ define "icon-chevron-down" }}
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-svg icon-svg-sm" aria-hidden="true">
    <path d="m6 9 6 6 6-6" />
  </svg>
{{ end }}

{{ define "icon-chevron-right" }}
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-svg icon-svg-sm" aria-hidden="true">
    <path d="m9 18 6-6-6-6" />
  </svg>
{{ end }}
```

(Spec mentioned `web/public` static files; use `icons.html` instead so `currentColor` works under light tokens — same Lucide glyphs.)

- [ ] **Step 4: Rewrite `#streams` body in `public_home.html`**

Replace tabs + grid + See all with:

```html
    <section class="public-home-streams" id="streams">
      <div class="public-home-section-head public-home-section-head-center">
        <p class="public-home-label">Stream Categories</p>
        <h2>Explore Public Streams</h2>
      </div>

      <div class="public-home-streams-layout">
        <aside class="public-home-category-sidebar" aria-label="Stream categories" data-landing-category-sidebar>
          <div class="public-home-category-sidebar-header">
            {{ template "icon-layout-grid" . }}
            <p class="public-home-category-sidebar-title">Stream Categories</p>
          </div>
          <div class="public-home-category-sidebar-list">
            {{ range .Categories }}
              <details class="public-home-category" {{ if .Expanded }}open{{ end }} data-category-slug="{{ .Slug }}">
                <summary class="public-home-category-summary">
                  <span class="public-home-category-label">
                    {{ if .IconURL }}
                      <img class="public-home-category-icon" src="{{ .IconURL }}" alt="" width="20" height="20" />
                    {{ end }}
                    <span>{{ .Name }}</span>
                  </span>
                  <span class="public-home-category-chevron" aria-hidden="true">
                    <span class="public-home-category-chevron-closed">{{ template "icon-chevron-right" $ }}</span>
                    <span class="public-home-category-chevron-open">{{ template "icon-chevron-down" $ }}</span>
                  </span>
                </summary>
                <div class="public-home-subcategories">
                  {{ range .SubCategories }}
                    <button
                      type="button"
                      class="public-home-subcategory{{ if .Active }} is-active{{ end }}"
                      data-subcategory-slug="{{ .Slug }}"
                      hx-get="{{ .PartialURL }}"
                      hx-target="#public-home-stream-results"
                      hx-swap="innerHTML"
                      hx-push-url="{{ .PushURL }}"
                    >{{ .Name }}</button>
                  {{ end }}
                </div>
              </details>
            {{ else }}
              <p class="public-home-category-sidebar-empty">No categories yet.</p>
            {{ end }}
          </div>
        </aside>

        {{ template "public_home_stream_results" .StreamResults }}
      </div>
    </section>
```

Fix `$` / `.` scoping carefully inside `range` (use `{{ template "icon-chevron-right" }}` without data if icons ignore context).

- [ ] **Step 5: Wire `handlePublicHome`**

```go
func (s *Server) handlePublicHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	base := s.pageBase("public_home_body", "", "")
	signedIn := false
	if user, _, err := s.currentUser(r); err == nil {
		base = s.pageBaseForUser(user, "public_home_body", "", "")
		signedIn = true
	}
	categories, err := loadTaxonomyTree(r.Context(), s.store)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cat, sub := resolvePublicHomeSelection(categories, r.URL.Query().Get("category"), r.URL.Query().Get("subCategory"))
	streams, err := s.publicStreamCardsForPath(r.Context(), cat, sub)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	view := struct {
		PageBase
		Categories    []PublicHomeCategoryView
		StreamResults PublicHomeStreamResultsView
	}{
		PageBase:   base,
		Categories: buildPublicHomeCategories(categories, cat, sub),
		StreamResults: PublicHomeStreamResultsView{
			Streams:          streams,
			CreateStreamHref: publicHomeCreateStreamHref(signedIn),
		},
	}
	if err := s.tmpl.ExecuteTemplate(w, "public_home.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
```

- [ ] **Step 6: Run tests — expect PASS**

```bash
cd server/cmd/server && go test -count=1 -run 'TestHandlePublicHome|TestHandlePublicStreamsPartial|TestPublicStreamCardsForPath|TestBuildPublicHome|TestResolvePublicHome|TestTaxonomyHasPath|TestPublicHomeCreateStreamHref' .
```

- [ ] **Step 7: Commit**

```bash
git add server/templates/icons.html server/templates/pages/public_home.html server/cmd/server/main.go server/cmd/server/home_handler_test.go
git commit -m "$(cat <<'EOF'
feat(ui): render taxonomy filter sidebar on public home

Replace horizontal category tabs with an accordion sidebar and HTMX-wired subcategory filters.
EOF
)"
```

---

### Task 6: Sidebar + layout CSS (light tokens)

**Files:**
- Modify: `web/src/styles/pages/public-home.css`
- Optionally update markup-tree header comment at top of that file

**Interfaces:**
- Consumes: `--card`, `--muted`, `--border`, `--foreground`, `--muted-foreground`, `--secondary`, `--primary`, `--space-*`
- Produces: `.public-home-streams-layout` two-column from `--md-up`; stack below; sidebar accordion styles; active subcategory left accent via `border-inline-start` or `box-shadow` using `--primary`; remove `.public-home-tabs*` rules

- [ ] **Step 1: Replace streams layout rules**

Delete `.public-home-tabs`, `.public-home-tabs-arrow`, `.public-home-tabs-scroll`, `.public-home-tab`, `.public-home-tab.is-active`, `.public-home-tab-icon` (+ img).

Add (token-only, no hex):

```css
.public-home-streams-layout {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  width: 100%;
  max-width: 80rem;
  align-items: stretch;
}

.public-home-category-sidebar {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  overflow: hidden;
}

.public-home-category-sidebar-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-4);
  border-bottom: 1px solid var(--border);
}

.public-home-category-sidebar-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--foreground);
}

.public-home-category-sidebar-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
}

.public-home-category {
  border-radius: 8px;
  background: var(--background);
}

.public-home-category[open] {
  background: var(--muted);
  border: 1px solid var(--border);
}

.public-home-category-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  cursor: pointer;
  list-style: none;
}

.public-home-category-summary::-webkit-details-marker {
  display: none;
}

.public-home-category-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--muted-foreground);
}

.public-home-category[open] .public-home-category-label {
  font-weight: var(--font-semibold);
  color: var(--foreground);
}

.public-home-category-icon {
  width: 20px;
  height: 20px;
}

.public-home-category-chevron-open {
  display: none;
}

.public-home-category[open] .public-home-category-chevron-open {
  display: inline-flex;
}

.public-home-category[open] .public-home-category-chevron-closed {
  display: none;
}

.public-home-subcategories {
  display: flex;
  flex-direction: column;
  padding: 0 var(--space-3) var(--space-3) var(--space-5);
}

.public-home-subcategory {
  display: block;
  width: 100%;
  text-align: left;
  border: 0;
  border-radius: 6px;
  padding: var(--space-2) var(--space-3);
  background: transparent;
  color: var(--muted-foreground);
  font-size: var(--text-sm);
  cursor: pointer;
}

.public-home-subcategory.is-active {
  background: var(--secondary);
  color: var(--foreground);
  box-shadow: inset 3px 0 0 var(--primary);
}

.public-home-stream-results {
  flex: 1;
  min-width: 0;
}

.public-home-stream-empty {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-4);
  padding: var(--space-6);
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  color: var(--muted-foreground);
}

@media (--md-up) {
  .public-home-streams-layout {
    flex-direction: row;
    align-items: flex-start;
  }

  .public-home-category-sidebar {
    flex: 0 0 20rem;
    max-width: 20rem;
  }
}
```

Update `.public-home-streams` so it no longer forces `align-items: center` in a way that breaks full-width layout (keep section head centered; layout block `width: 100%`).

Update responsive blocks that referenced `.public-home-stream-grid` only — keep grid columns as today inside results.

- [ ] **Step 2: Lint CSS**

```bash
task css:lint
```

Expected: PASS.

- [ ] **Step 3: Build web assets if needed for local visual check**

```bash
cd web && npm run build
```

- [ ] **Step 4: Commit**

```bash
git add web/src/styles/pages/public-home.css
git commit -m "$(cat <<'EOF'
feat(css): style public home category sidebar with light tokens

Lay out the taxonomy accordion beside stream results and drop the horizontal tab styles.
EOF
)"
```

---

### Task 7: Client active-state sync; remove landing tabs JS

**Files:**
- Modify: `web/src/main.js` (around `initLandingTabs`, ~1452–1483)

**Interfaces:**
- Remove `initLandingTabs` and its call.
- Add `initLandingCategorySidebar` that:
  - On click of `.public-home-subcategory` (or `htmx:beforeRequest` from sidebar), removes `is-active` from all subcategory buttons in `[data-landing-category-sidebar]` and adds it to the clicked one.
  - Does not prevent HTMX default.

- [ ] **Step 1: Replace JS**

```js
const initLandingCategorySidebar = () => {
  const root = document.querySelector("[data-landing-category-sidebar]");
  if (!(root instanceof HTMLElement)) {
    return;
  }
  root.addEventListener("click", (event) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      return;
    }
    const btn = target.closest(".public-home-subcategory");
    if (!(btn instanceof HTMLElement) || !root.contains(btn)) {
      return;
    }
    for (const other of root.querySelectorAll(".public-home-subcategory")) {
      other.classList.remove("is-active");
    }
    btn.classList.add("is-active");
  });
};

initLandingCategorySidebar();
```

Delete `initLandingTabs` entirely.

- [ ] **Step 2: Smoke-check bundle**

```bash
cd web && npm run build
```

Expected: build succeeds; no references to `js-landing-tabs` / `data-landing-tabs` remain:

```bash
rg "landing-tabs|initLandingTabs|public-home-tab" web/src server/templates
```

Expected: no matches (except maybe this plan/spec).

- [ ] **Step 3: Commit**

```bash
git add web/src/main.js
git commit -m "$(cat <<'EOF'
feat(ui): sync landing subcategory active state for HTMX filters

Replace horizontal tab scroll JS with sidebar active-class handling on subcategory clicks.
EOF
)"
```

---

### Task 8: Final verification

**Files:** none new — regression only

- [ ] **Step 1: Backend tests**

```bash
cd server/cmd/server && go test -count=1 -run 'TestHandlePublicHome|TestHandlePublicStreamsPartial|TestPublicStreamCardsForPath|TestBuildPublicHome|TestResolvePublicHome|TestTaxonomyHasPath|TestPublicHomeCreateStreamHref' .
```

Expected: all PASS.

- [ ] **Step 2: Broader home tests**

```bash
cd server/cmd/server && go test -count=1 -run 'TestHandlePublicHome' .
```

Expected: PASS (including logged-in dashboard / empty catalog cases).

- [ ] **Step 3: CSS lint**

```bash
task css:lint
```

Expected: PASS.

- [ ] **Step 4: Manual checklist** (host-dev or `task dev`)

1. Open `/` — first category expanded, first subcategory active, results for that leaf (or empty + Create a stream).
2. Click another subcategory — grid swaps without full reload; URL updates to `/?category=…&subCategory=…`.
3. Refresh — same selection restored.
4. Narrow viewport — sidebar stacks above results.
5. Signed-out empty CTA → login with next to formata-builder; signed-in → direct formata-builder.

- [ ] **Step 5: Commit only if Step 4 found fixups**; otherwise done.

If docs need a one-line note in `docs/css.md` about sidebar living in `public-home.css`, add and commit:

```bash
git commit -m "docs(css): note public home category sidebar lives in public-home.css"
```

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Sidebar replaces horizontal tabs | 5, 6, 7 |
| Light tokens / no dark Figma palette | 6 |
| All categories even with zero streams | 3, 5 |
| Parent expand-only; filter on subcategory | 5 (`details` + HTMX on buttons) |
| Default first/first | 1, 5 |
| Live taxonomy store | 4, 5 |
| Mobile stack | 6 |
| Empty + Create a stream login-next | 1, 4 |
| Remove See all streams / tabs JS | 5, 7 |
| Lucide LayoutGrid + chevrons | 5 (`icons.html`) |
| `GET /streams/public` HTMX partial | 4 |
| URL push `/?category=&subCategory=` | 3, 5 |
| Filter by category+sub; cap after filter | 2 |
| Uncategorized excluded | 2 |
| Partial 404 unknown/missing | 4 |

## Notes for implementers

- `parseTestTemplates(t)` loads real templates from `server/templates` — no stub update needed for new defines once files exist under `components/`.
- `seedPlatformAdminTaxonomy` first/first is `supply-chain` / `procurement` — use that in tests unless you seed a richer tree.
- Catalog streams without a valid taxonomy path are cleared by `applyEffectiveStreamCategorization` on load — seed taxonomy **before** relying on categorized YAML in the same process, or write + reload catalog after `ReplaceTaxonomy`.
- Do not mount `@lucide/svelte` on the marketing page.
