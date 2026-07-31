# `/my` category sidebar anchors Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract a shared thin-full `category_sidebar` component, use it on public `/` (HTMX leaves unchanged) and on `/my` (accessible-only anchor leaves + section ids), with a CSS-only `nav-drawer` for mobile.

**Architecture:** Rename today’s public-home sidebar DTOs into `CategorySidebar*` views. One template dispatches leaf markup on `Href` vs `PartialURL`. `/my` derives the sidebar from existing catalog groups, wraps each group in `section#AnchorID`, and docks the same sidebar DOM node in a two-column grid (desktop) or popover drawer (mobile).

**Tech Stack:** Go `net/http` + `html/template`, Vite CSS modules, Popover API (no app JS), Go unit/template tests.

**Spec:** `docs/superpowers/specs/2026-07-31-my-home-category-sidebar-anchors-design.md`

## Global Constraints

- `/my` leaves are anchors only (`href="#cat-…"`); category rows stay `<details>`/`<summary>` (not links).
- `/my` sidebar lists only categories/subcategories with ≥1 accessible stream; Uncategorized = synthetic category with one leaf → `#cat-uncategorized`.
- `/my` layout: two columns inside `u-max-w-7xl` / `.my-home`; same page background — no public-home full-bleed split band.
- Desktop (`md`+): sidebar docked; hamburger hidden. Mobile: hamburger + left `nav-drawer`. **One** sidebar DOM instance.
- No scroll-spy. No new JS modules for drawer open/close.
- Public `/` HTMX filter semantics, URLs, and selection resolution unchanged.
- Follow `docs/css.md` + `.agents/skills/attesta-ui-components`.
- Work in `.worktrees/feat/my-home-catalog` on branch `feat/my-home-catalog`.

## File map

| File | Responsibility |
|------|----------------|
| `server/cmd/server/components.go` | `CategorySidebarView`, `CategorySidebarCategoryView`, `CategorySidebarLeafView`; extend `MyHomeStreamGroupView` with slugs/`AnchorID`; optionally `ShowCategoryHeader` |
| `server/templates/components/category_sidebar.html` | Shared sidebar template |
| `web/src/styles/components/category-sidebar.css` | `.category-sidebar-*` tree styles (moved from public-home) |
| `web/src/styles/components/nav-drawer.css` | CSS-only drawer shell contract |
| `web/src/styles/components.css` | Import new component CSS |
| `server/cmd/server/public_home.go` | `buildPublicHomeCategories` → returns `CategorySidebarView` (HTMX leaves) |
| `server/templates/pages/public_home.html` | Call `category_sidebar` |
| `web/src/styles/pages/public-home.css` | Keep split-band layout; drop moved tree rules; target `.category-sidebar` |
| `server/cmd/server/my_home.go` | Fill `AnchorID`/slugs on groups; `buildMyHomeCategorySidebar(groups)` |
| `server/templates/components/my_home_stream_group.html` | Wrap leaf block in `<section id="{{ .AnchorID }}">` |
| `server/templates/pages/home.html` | Grid + nav-drawer markup + sidebar |
| `web/src/styles/pages/home.css` | `/my` grid, scroll-margin, desktop popover override, transparent sidebar band |
| `server/cmd/server/main.go` | `HomeWorkflowPickerView.Sidebar`; `handleHome` fills it |
| `server/cmd/server/category_sidebar_test.go` | Template leaf-mode tests |
| `server/cmd/server/my_home_test.go` / `my_home_stream_group_test.go` / `home_handler_test.go` / `public_home_test.go` | Update assertions |

---

### Task 1: `CategorySidebar*` types + template leaf dispatch

**Files:**
- Modify: `server/cmd/server/components.go`
- Create: `server/templates/components/category_sidebar.html`
- Create: `server/cmd/server/category_sidebar_test.go`

**Interfaces:**
- Produces:
  - `type CategorySidebarLeafView struct { Slug, Name string; Active bool; Href, PartialURL, PushURL string }`
  - `type CategorySidebarCategoryView struct { Slug, Name, IconURL string; Expanded bool; SubCategories []CategorySidebarLeafView }`
  - `type CategorySidebarView struct { Title string; Categories []CategorySidebarCategoryView }`
- Consumes: none

- [ ] **Step 1: Write failing template tests**

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCategorySidebarTemplateHTMXLeaf(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := CategorySidebarView{
		Title: "Stream Categories",
		Categories: []CategorySidebarCategoryView{{
			Slug: "supply-chain", Name: "Supply Chain", IconURL: "/c.svg", Expanded: true,
			SubCategories: []CategorySidebarLeafView{{
				Slug: "procurement", Name: "Procurement", Active: true,
				PartialURL: "/streams/public?category=supply-chain&subCategory=procurement",
				PushURL:    "/?category=supply-chain&subCategory=procurement",
			}},
		}},
	}
	if err := tmpl.ExecuteTemplate(&out, "category_sidebar", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`class="category-sidebar"`,
		`class="category-sidebar-title">Stream Categories<`,
		`hx-get="/streams/public?category=supply-chain&amp;subCategory=procurement"`,
		`hx-push-url="/?category=supply-chain&amp;subCategory=procurement"`,
		`is-active`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	if strings.Contains(body, `href="#`) {
		t.Fatalf("HTMX leaf must not use href anchor, got %s", body)
	}
}

func TestCategorySidebarTemplateAnchorLeaf(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := CategorySidebarView{
		Title: "Stream Categories",
		Categories: []CategorySidebarCategoryView{{
			Slug: "supply-chain", Name: "Supply Chain", Expanded: true,
			SubCategories: []CategorySidebarLeafView{{
				Slug: "procurement", Name: "Procurement",
				Href: "#cat-supply-chain--procurement",
			}},
		}},
	}
	if err := tmpl.ExecuteTemplate(&out, "category_sidebar", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `href="#cat-supply-chain--procurement"`) {
		t.Fatalf("missing anchor href, got %s", body)
	}
	if strings.Contains(body, "hx-get=") || strings.Contains(body, "<button") {
		t.Fatalf("anchor leaf must not render HTMX button, got %s", body)
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL** (types/template missing)

```bash
cd server && go test ./cmd/server/ -run 'TestCategorySidebarTemplate' -count=1
```

Expected: FAIL (undefined types and/or missing template).

- [ ] **Step 3: Add types in `components.go`**

Replace `PublicHomeSubCategoryView` / `PublicHomeCategoryView` with:

```go
// CategorySidebarLeafView is one subcategory row in category_sidebar.
// If Href is set, render an anchor; otherwise render an HTMX button using PartialURL/PushURL.
type CategorySidebarLeafView struct {
	Slug       string
	Name       string
	Active     bool
	Href       string // /my anchors, e.g. #cat-supply-chain--procurement
	PartialURL string // public home HTMX get
	PushURL    string // public home hx-push-url
}

// CategorySidebarCategoryView is one accordion category in category_sidebar.
type CategorySidebarCategoryView struct {
	Slug          string
	Name          string
	IconURL       string
	Expanded      bool
	SubCategories []CategorySidebarLeafView
}

// CategorySidebarView is the root for templates/components/category_sidebar.html.
type CategorySidebarView struct {
	Title      string
	Categories []CategorySidebarCategoryView
}
```

Prefer a clean rename in the same commit as Task 2 if compile breaks mid-way; do not leave permanent aliases.

- [ ] **Step 4: Add `category_sidebar.html`**

```html
{{ define "category_sidebar" }}
<aside class="category-sidebar" aria-label="Stream categories" data-landing-category-sidebar>
  <div class="category-sidebar-header">
    {{ template "icon-layout-grid" . }}
    <p class="category-sidebar-title">{{ .Title }}</p>
  </div>
  <div class="category-sidebar-list">
    {{ range .Categories }}
      <details class="category-sidebar-category" {{ if .Expanded }}open{{ end }} data-category-slug="{{ .Slug }}">
        <summary class="category-sidebar-category-summary">
          <span class="category-sidebar-category-label">
            {{ if .IconURL }}
              <img class="category-sidebar-category-icon" src="{{ .IconURL }}" alt="" width="20" height="20" />
            {{ end }}
            <span>{{ .Name }}</span>
          </span>
          <span class="category-sidebar-category-chevron" aria-hidden="true">
            {{ template "icon-chevron-right" }}
          </span>
        </summary>
        <div class="category-sidebar-subcategories">
          {{ range .SubCategories }}
            {{ if .Href }}
              <a
                class="category-sidebar-subcategory{{ if .Active }} is-active{{ end }}"
                href="{{ .Href }}"
                data-subcategory-slug="{{ .Slug }}"
              >{{ .Name }}</a>
            {{ else }}
              <button
                type="button"
                class="category-sidebar-subcategory{{ if .Active }} is-active{{ end }}"
                data-subcategory-slug="{{ .Slug }}"
                hx-get="{{ .PartialURL }}"
                hx-target="#public-home-stream-results"
                hx-swap="outerHTML"
                hx-push-url="{{ .PushURL }}"
              >{{ .Name }}</button>
            {{ end }}
          {{ end }}
        </div>
      </details>
    {{ else }}
      <p class="category-sidebar-empty">No categories yet.</p>
    {{ end }}
  </div>
</aside>
{{ end }}
```

Keep `data-landing-category-sidebar` on the aside for any existing selectors that look for it.

- [ ] **Step 5: Run tests — expect PASS**

```bash
cd server && go test ./cmd/server/ -run 'TestCategorySidebarTemplate' -count=1
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add server/cmd/server/components.go server/templates/components/category_sidebar.html server/cmd/server/category_sidebar_test.go
git commit -m "$(cat <<'EOF'
feat(ui): add category_sidebar template with HTMX/anchor leaves

EOF
)"
```

---

### Task 2: Wire public home to shared sidebar

**Files:**
- Modify: `server/cmd/server/public_home.go`
- Modify: `server/cmd/server/main.go` (public home view field type)
- Modify: `server/templates/pages/public_home.html`
- Modify: `server/cmd/server/public_home_test.go`
- Modify: `server/cmd/server/home_handler_test.go` (public home assertions that mention old class names)

**Interfaces:**
- Consumes: `CategorySidebarView` / leaf types from Task 1
- Produces: `func buildPublicHomeCategories(categories []TaxonomyCategoryNode, selectedCat, selectedSub string) CategorySidebarView`

- [ ] **Step 1: Update failing public tests** to expect `CategorySidebarView` and `class="category-sidebar"`. Keep PartialURL/PushURL behavior assertions.

```go
got := buildPublicHomeCategories(cats, "supply-chain", "order-fulfillment")
if got.Title != "Stream Categories" {
	t.Fatalf("title=%q", got.Title)
}
if got.Categories[0].SubCategories[1].PartialURL != wantPartial {
	t.Fatalf("PartialURL=%q", got.Categories[0].SubCategories[1].PartialURL)
}
```

Markup wants:

```go
`class="category-sidebar"`,
`hx-get="/streams/public?category=supply-chain&amp;subCategory=procurement"`,
```

Remove asserts on `public-home-category-sidebar` class strings.

- [ ] **Step 2: Run targeted tests — expect FAIL** on type/class mismatches

```bash
cd server && go test ./cmd/server/ -run 'TestBuildPublicHomeCategories|TestHandlePublicHomeRenders|TestHandlePublicHomeQuery' -count=1
```

- [ ] **Step 3: Change builder**

```go
func buildPublicHomeCategories(categories []TaxonomyCategoryNode, selectedCat, selectedSub string) CategorySidebarView {
	out := make([]CategorySidebarCategoryView, 0, len(categories))
	for _, cat := range categories {
		subs := make([]CategorySidebarLeafView, 0, len(cat.SubCategories))
		for _, sub := range cat.SubCategories {
			query := url.Values{
				"category":    {cat.Slug},
				"subCategory": {sub.Slug},
			}
			encoded := query.Encode()
			subs = append(subs, CategorySidebarLeafView{
				Slug:       sub.Slug,
				Name:       sub.Name,
				Active:     cat.Slug == selectedCat && sub.Slug == selectedSub,
				PartialURL: "/streams/public?" + encoded,
				PushURL:    publicHomePath(cat.Slug, sub.Slug),
			})
		}
		out = append(out, CategorySidebarCategoryView{
			Slug:          cat.Slug,
			Name:          cat.Name,
			IconURL:       cat.IconURL,
			Expanded:      cat.Slug == selectedCat,
			SubCategories: subs,
		})
	}
	return CategorySidebarView{
		Title:      "Stream Categories",
		Categories: out,
	}
}
```

Update the public home page view field to `Sidebar CategorySidebarView` and set `Sidebar: buildPublicHomeCategories(...)`.

- [ ] **Step 4: Replace inline aside in `public_home.html`**

Inside `.public-home-streams-split`, replace the whole `<aside class="public-home-category-sidebar" …>…</aside>` block with:

```html
{{ template "category_sidebar" .Sidebar }}
```

- [ ] **Step 5: Run public home tests — expect PASS**

```bash
cd server && go test ./cmd/server/ -run 'TestBuildPublicHomeCategories|TestHandlePublicHome|TestHandlePublicStreamsPartial' -count=1
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add server/cmd/server/public_home.go server/cmd/server/main.go server/templates/pages/public_home.html server/cmd/server/public_home_test.go server/cmd/server/home_handler_test.go
git commit -m "$(cat <<'EOF'
refactor: use category_sidebar on public home

EOF
)"
```

---

### Task 3: Migrate category sidebar CSS

**Files:**
- Create: `web/src/styles/components/category-sidebar.css`
- Modify: `web/src/styles/components.css`
- Modify: `web/src/styles/pages/public-home.css`

**Interfaces:**
- Produces: `.category-sidebar*` selectors (visual parity with former `.public-home-category*`)
- Consumes: public-home split layout still positions the aside

- [ ] **Step 1: Create `category-sidebar.css`**

Copy rules from `public-home.css` covering `.public-home-category-sidebar` through `.public-home-subcategory.is-active::before` (approx. lines 133–281). Rename:

| Old | New |
|-----|-----|
| `.public-home-category-sidebar` | `.category-sidebar` |
| `.public-home-category-sidebar-header` | `.category-sidebar-header` |
| `.public-home-category-sidebar-title` | `.category-sidebar-title` |
| `.public-home-category-sidebar-list` | `.category-sidebar-list` |
| `.public-home-category-sidebar-empty` | `.category-sidebar-empty` |
| `.public-home-category` | `.category-sidebar-category` |
| `.public-home-category-summary` | `.category-sidebar-category-summary` |
| `.public-home-category-label` | `.category-sidebar-category-label` |
| `.public-home-category-icon` | `.category-sidebar-category-icon` |
| `.public-home-category-chevron` | `.category-sidebar-category-chevron` |
| `.public-home-subcategories` | `.category-sidebar-subcategories` |
| `.public-home-subcategory` | `.category-sidebar-subcategory` |

Add a markup-tree header matching `category_sidebar.html`. Style `a.category-sidebar-subcategory` like the button (no underline; inherit color).

- [ ] **Step 2: Import from `components.css`**

```css
@import url("./components/category-sidebar.css");
```

Place near other full components (after `public-stream-card.css` is fine).

- [ ] **Step 3: Delete moved rules from `public-home.css`**; update split-layout selectors that referenced `.public-home-category-sidebar` to `.category-sidebar` (including `@media (--md-up)` grid-column rules). Update the file header comment tree.

- [ ] **Step 4: Lint + smoke public tests**

```bash
task css:lint
cd server && go test ./cmd/server/ -run 'TestHandlePublicHomeRenders|TestCategorySidebarTemplate' -count=1
```

Expected: css lint OK; tests PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/styles/components/category-sidebar.css web/src/styles/components.css web/src/styles/pages/public-home.css
git commit -m "$(cat <<'EOF'
style: extract category-sidebar component CSS

EOF
)"
```

---

### Task 4: Catalog section anchors on `/my`

**Files:**
- Modify: `server/cmd/server/components.go` (`MyHomeStreamGroupView`)
- Modify: `server/cmd/server/my_home.go`
- Modify: `server/templates/components/my_home_stream_group.html`
- Modify: `server/cmd/server/my_home_test.go`
- Modify: `server/cmd/server/my_home_stream_group_test.go`

**Interfaces:**
- Produces:
  - `MyHomeStreamGroupView` fields: `CategorySlug`, `SubCategorySlug`, `AnchorID string`, `ShowCategoryHeader bool`
  - `func myHomeAnchorID(categorySlug, subCategorySlug string, uncategorized bool) string`
- Consumes: existing `buildMyHomeStreamGroups`

- [ ] **Step 1: Write failing tests**

```go
func TestMyHomeAnchorID(t *testing.T) {
	if got := myHomeAnchorID("supply-chain", "procurement", false); got != "cat-supply-chain--procurement" {
		t.Fatalf("got %q", got)
	}
	if got := myHomeAnchorID("", "", true); got != "cat-uncategorized" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildMyHomeStreamGroupsSetsAnchorIDs(t *testing.T) {
	// reuse taxonomy fixtures from TestBuildMyHomeStreamGroupsOrdersTaxonomyAndUncategorized
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
```

Extend `TestMyHomeStreamGroupTemplateRendersCategoryHeaders` to pass `ShowCategoryHeader: true`, `AnchorID: "cat-supply-chain--procurement"`, and assert `id="cat-supply-chain--procurement"`.

- [ ] **Step 2: Run — expect FAIL**

```bash
cd server && go test ./cmd/server/ -run 'TestMyHomeAnchorID|TestBuildMyHomeStreamGroupsSetsAnchorIDs|TestMyHomeStreamGroupTemplate' -count=1
```

- [ ] **Step 3: Implement**

```go
func myHomeAnchorID(categorySlug, subCategorySlug string, uncategorized bool) string {
	if uncategorized {
		return "cat-uncategorized"
	}
	return "cat-" + strings.TrimSpace(categorySlug) + "--" + strings.TrimSpace(subCategorySlug)
}
```

Extend `MyHomeStreamGroupView`:

```go
type MyHomeStreamGroupView struct {
	CategoryName           string
	CategoryIconURL        string
	CategorySlug           string
	SubCategorySlug        string
	SubCategoryName        string
	SubCategoryIconURL     string
	SubCategoryDescription string
	AnchorID               string
	ShowCategoryHeader     bool
	Uncategorized          bool
	Streams                []ManagedPublicStreamCardView
}
```

In `buildMyHomeStreamGroups`, for each taxonomy group always set:

```go
group := MyHomeStreamGroupView{
	CategoryName:           category.Name,
	CategoryIconURL:        category.IconURL,
	CategorySlug:           category.Slug,
	SubCategorySlug:        sub.Slug,
	SubCategoryName:        sub.Name,
	SubCategoryIconURL:     sub.IconURL,
	SubCategoryDescription: sub.Description,
	AnchorID:               myHomeAnchorID(category.Slug, sub.Slug, false),
	ShowCategoryHeader:     !categoryHeaderEmitted,
	Streams:                streams,
}
categoryHeaderEmitted = true
```

Uncategorized:

```go
groups = append(groups, MyHomeStreamGroupView{
	CategoryName:       "Uncategorized",
	AnchorID:           myHomeAnchorID("", "", true),
	ShowCategoryHeader: true,
	Uncategorized:      true,
	Streams:            uncategorized,
})
```

Template — use `ShowCategoryHeader` and wrap leaf content:

```html
{{ define "my_home_stream_group" }}
  {{ if .Uncategorized }}
  <section class="my-home-catalog-section" id="{{ .AnchorID }}">
    {{ if .ShowCategoryHeader }}
    <header class="public-home-results-category-header">
      <h2 class="public-home-results-category-name">{{ .CategoryName }}</h2>
    </header>
    {{ end }}
    {{ if .Streams }}
    <div class="public-home-stream-grid">
      {{ range .Streams }}
        {{ template "managed_public_stream_card" . }}
      {{ end }}
    </div>
    {{ end }}
  </section>
  {{ else }}
  {{ if .ShowCategoryHeader }}
  <header class="public-home-results-category-header">
    {{ if .CategoryIconURL }}
    <img class="public-home-results-category-icon" src="{{ .CategoryIconURL }}" alt="" width="24" height="24" />
    {{ end }}
    <h2 class="public-home-results-category-name">{{ .CategoryName }}</h2>
  </header>
  {{ end }}
  <section class="my-home-catalog-section" id="{{ .AnchorID }}">
    {{ if .SubCategoryName }}
    <header class="public-home-results-subcategory-header">
      …existing subcategory markup…
    </header>
    {{ end }}
    {{ if .Streams }}
    <div class="public-home-stream-grid">
      {{ range .Streams }}
        {{ template "managed_public_stream_card" . }}
      {{ end }}
    </div>
    {{ end }}
  </section>
  {{ end }}
{{ end }}
```

Update any template tests that used `CategoryName` alone to set `ShowCategoryHeader: true`.

- [ ] **Step 4: Run — expect PASS**

```bash
cd server && go test ./cmd/server/ -run 'TestMyHomeAnchorID|TestBuildMyHomeStreamGroups|TestMyHomeStreamGroupTemplate' -count=1
```

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/components.go server/cmd/server/my_home.go server/templates/components/my_home_stream_group.html server/cmd/server/my_home_test.go server/cmd/server/my_home_stream_group_test.go
git commit -m "$(cat <<'EOF'
feat(my): add stable catalog section anchor ids

EOF
)"
```

---

### Task 5: Build `/my` sidebar from groups

**Files:**
- Modify: `server/cmd/server/my_home.go`
- Modify: `server/cmd/server/my_home_test.go`
- Modify: `server/cmd/server/main.go` (`HomeWorkflowPickerView`, `handleHome`)

**Interfaces:**
- Produces: `func buildMyHomeCategorySidebar(groups []MyHomeStreamGroupView) CategorySidebarView`
- Consumes: groups with `CategorySlug`, `SubCategorySlug`, `AnchorID`, `CategoryName`, `CategoryIconURL`, `Uncategorized`

- [ ] **Step 1: Write failing test**

```go
func TestBuildMyHomeCategorySidebarFromGroups(t *testing.T) {
	groups := []MyHomeStreamGroupView{
		{CategorySlug: "supply-chain", CategoryName: "Supply Chain", CategoryIconURL: "/c.svg", SubCategorySlug: "procurement", SubCategoryName: "Procurement", AnchorID: "cat-supply-chain--procurement", ShowCategoryHeader: true},
		{CategorySlug: "supply-chain", CategoryName: "Supply Chain", CategoryIconURL: "/c.svg", SubCategorySlug: "order-fulfillment", SubCategoryName: "Order Fulfillment", AnchorID: "cat-supply-chain--order-fulfillment"},
		{Uncategorized: true, CategoryName: "Uncategorized", AnchorID: "cat-uncategorized", ShowCategoryHeader: true},
	}
	got := buildMyHomeCategorySidebar(groups)
	if got.Title != "Stream Categories" || len(got.Categories) != 2 {
		t.Fatalf("got=%+v", got)
	}
	if !got.Categories[0].Expanded || got.Categories[0].Name != "Supply Chain" || len(got.Categories[0].SubCategories) != 2 {
		t.Fatalf("cat0=%+v", got.Categories[0])
	}
	if got.Categories[0].SubCategories[0].Href != "#cat-supply-chain--procurement" {
		t.Fatalf("href0=%q", got.Categories[0].SubCategories[0].Href)
	}
	if got.Categories[1].Name != "Uncategorized" || got.Categories[1].SubCategories[0].Href != "#cat-uncategorized" {
		t.Fatalf("uncat=%+v", got.Categories[1])
	}
	if got.Categories[0].SubCategories[0].PartialURL != "" {
		t.Fatalf("my leaves must not set PartialURL")
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

```bash
cd server && go test ./cmd/server/ -run TestBuildMyHomeCategorySidebarFromGroups -count=1
```

- [ ] **Step 3: Implement builder**

```go
func buildMyHomeCategorySidebar(groups []MyHomeStreamGroupView) CategorySidebarView {
	var cats []CategorySidebarCategoryView
	var current *CategorySidebarCategoryView

	flush := func() {
		if current != nil {
			cats = append(cats, *current)
			current = nil
		}
	}

	for _, g := range groups {
		if g.Uncategorized {
			flush()
			cats = append(cats, CategorySidebarCategoryView{
				Slug:     "uncategorized",
				Name:     "Uncategorized",
				Expanded: true,
				SubCategories: []CategorySidebarLeafView{{
					Slug: "uncategorized",
					Name: "Uncategorized",
					Href: "#" + g.AnchorID,
				}},
			})
			continue
		}
		if current == nil || current.Slug != g.CategorySlug {
			flush()
			current = &CategorySidebarCategoryView{
				Slug:     g.CategorySlug,
				Name:     g.CategoryName,
				IconURL:  g.CategoryIconURL,
				Expanded: true,
			}
		}
		current.SubCategories = append(current.SubCategories, CategorySidebarLeafView{
			Slug: g.SubCategorySlug,
			Name: g.SubCategoryName,
			Href: "#" + g.AnchorID,
		})
	}
	flush()
	return CategorySidebarView{Title: "Stream Categories", Categories: cats}
}
```

- [ ] **Step 4: Wire `handleHome`**

```go
type HomeWorkflowPickerView struct {
	PageBase
	Groups           []MyHomeStreamGroupView
	Sidebar          CategorySidebarView
	ShowCreateStream bool
	Error            string
	Confirmation     string
}
```

```go
view := HomeWorkflowPickerView{
	PageBase:         s.pageBaseForUser(user, "home_picker_body", "", ""),
	Groups:           groups,
	Sidebar:          buildMyHomeCategorySidebar(groups),
	ShowCreateStream: showCreateStream && authErr == nil,
	Error:            homePickerMessage(r, "error"),
	Confirmation:     homePickerMessage(r, "confirmation"),
}
```

- [ ] **Step 5: Run — expect PASS**

```bash
cd server && go test ./cmd/server/ -run 'TestBuildMyHomeCategorySidebarFromGroups|TestBuildMyHomeStreamGroups' -count=1
```

- [ ] **Step 6: Commit**

```bash
git add server/cmd/server/my_home.go server/cmd/server/my_home_test.go server/cmd/server/main.go
git commit -m "$(cat <<'EOF'
feat(my): build category sidebar from accessible catalog groups

EOF
)"
```

---

### Task 6: `nav-drawer` + `/my` page layout

**Files:**
- Create: `web/src/styles/components/nav-drawer.css`
- Modify: `web/src/styles/components.css`
- Modify: `web/src/styles/pages/home.css`
- Modify: `server/templates/pages/home.html`
- Modify: `server/cmd/server/my_home_stream_group_test.go`
- Modify: `server/cmd/server/home_handler_test.go`

**Interfaces:**
- Produces: CSS-only `nav-drawer` markup contract; `/my` two-column layout
- Consumes: `CategorySidebarView` on home picker view; `category_sidebar` template

- [ ] **Step 1: Write failing page markup tests**

```go
func TestHomePickerBodyRendersSidebarAndDrawerTrigger(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := HomeWorkflowPickerView{
		PageBase: PageBase{Body: "home_picker_body"},
		Groups: []MyHomeStreamGroupView{{
			CategoryName: "Supply Chain", SubCategoryName: "Procurement",
			AnchorID: "cat-supply-chain--procurement", ShowCategoryHeader: true,
			Streams: []ManagedPublicStreamCardView{{Card: PublicStreamCardView{Name: "A"}}},
		}},
		Sidebar: CategorySidebarView{
			Title: "Stream Categories",
			Categories: []CategorySidebarCategoryView{{
				Name: "Supply Chain", Expanded: true,
				SubCategories: []CategorySidebarLeafView{{Name: "Procurement", Href: "#cat-supply-chain--procurement"}},
			}},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "home_picker_body", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`class="nav-drawer-trigger"`,
		`popovertarget="my-home-category-sidebar"`,
		`id="my-home-category-sidebar"`,
		`class="category-sidebar"`,
		`href="#cat-supply-chain--procurement"`,
		`id="cat-supply-chain--procurement"`,
		`class="my-home-catalog"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}
```

Update `TestHomePickerBodyTemplateRendersEmptyState` to assert absence of `nav-drawer-trigger` and `category-sidebar`.

- [ ] **Step 2: Run — expect FAIL**

```bash
cd server && go test ./cmd/server/ -run 'TestHomePickerBodyRendersSidebarAndDrawerTrigger|TestHomePickerBodyTemplateRendersEmptyState' -count=1
```

- [ ] **Step 3: Add `nav-drawer.css`**

```css
/*
 * Nav drawer — left slide-in shell (CSS-only).
 *
 * button.nav-drawer-trigger[popovertarget="…"]
 * div.nav-drawer-panel[popover][id="…"]   (hosts page content, e.g. category_sidebar)
 *
 * Desktop: page CSS forces .nav-drawer-panel into normal flow and hides the trigger.
 * Do not use :target to open — hashes are reserved for in-page section anchors.
 */

.nav-drawer-panel {
  margin: 0;
  border: 0;
  padding: 0;
  background: var(--background);
  color: var(--foreground);
  width: min(18rem, 92vw);
  height: 100dvh;
  max-height: 100dvh;
  overflow: auto;
}

.nav-drawer-panel:popover-open {
  position: fixed;
  inset: 0 auto 0 0;
}
```

Import in `components.css`. Keep YAGNI — UA popover backdrop is enough.

- [ ] **Step 4: Rewrite `home_picker_body` layout**

```html
{{ define "home_picker_body" }}
<section class="stack u-max-w-7xl u-mx-auto my-home">
  <section class="page-header">
    <div class="page-header-head">
      <div class="page-header-body">
        <h1>Choose a stream</h1>
        <p>Select a stream to start or continue process tracking</p>
      </div>
      {{ if or .Groups .ShowCreateStream }}
      <div class="page-header-actions">
        {{ if .Groups }}
        <button
          type="button"
          class="btn btn-ghost btn-icon nav-drawer-trigger"
          popovertarget="my-home-category-sidebar"
          aria-label="Open stream categories"
          title="Stream categories"
        >
          {{ template "icon-layout-grid" . }}
        </button>
        {{ end }}
        {{ if .ShowCreateStream }}
        <a class="btn btn-primary" href="/my/organization/formata-builder">
          {{ template "icon-plus" . }} Create a stream
        </a>
        {{ end }}
      </div>
      {{ end }}
    </div>
  </section>
  {{ if .Confirmation }}
    <p class="confirmation">{{ .Confirmation }}</p>
  {{ end }}
  {{ template "error_banner.html" . }}
  {{ if .Groups }}
    <div class="my-home-body">
      <div
        id="my-home-category-sidebar"
        class="nav-drawer-panel my-home-sidebar"
        popover
      >
        {{ template "category_sidebar" .Sidebar }}
      </div>
      <div class="my-home-catalog">
        {{ range .Groups }}
          {{ template "my_home_stream_group" . }}
        {{ end }}
      </div>
    </div>
  {{ else }}
    <div class="empty-state">
      {{ template "icon-layers-2" . }}
      <p class="empty-state-title">No streams available</p>
      <p class="empty-state-hint">Streams for your organization and roles will appear here.</p>
    </div>
  {{ end }}
</section>
{{ end }}
```

- [ ] **Step 5: Update `home.css`**

```css
.my-home-catalog {
  display: flex;
  flex-direction: column;
  gap: var(--space-8);
  min-width: 0;
}

.my-home-catalog-section {
  scroll-margin-top: 44px;
}

.my-home-body {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.my-home .nav-drawer-trigger {
  display: inline-flex;
}

@media (--md-up) {
  .my-home-body {
    display: grid;
    grid-template-columns: 18rem minmax(0, 1fr);
    gap: var(--space-8);
    align-items: start;
  }

  .my-home .nav-drawer-trigger {
    display: none;
  }

  .my-home .nav-drawer-panel.my-home-sidebar {
    display: block;
    position: static;
    inset: auto;
    width: auto;
    height: auto;
    max-height: none;
    overflow: visible;
    background: transparent;
  }

  .my-home .category-sidebar {
    background: transparent;
    border-bottom: 0;
    padding: 0;
  }
}
```

If forcing a closed popover into flow fails in a target browser, fall back to a checkbox `nav-drawer` with the same class names (no app JS).

- [ ] **Step 6: Run tests + css lint**

```bash
cd server && go test ./cmd/server/ -run 'TestHomePickerBody|TestMyHomeStreamGroup|TestBuildMyHome|TestCategorySidebar|TestHandleHome' -count=1
task css:lint
```

Expected: PASS

- [ ] **Step 7: Manual check** on worktree host-dev: `/my` desktop two-column same bg; mobile hamburger opens left panel; leaf jumps to section; empty catalog has no trigger.

- [ ] **Step 8: Commit**

```bash
git add web/src/styles/components/nav-drawer.css web/src/styles/components.css web/src/styles/pages/home.css server/templates/pages/home.html server/cmd/server/my_home_stream_group_test.go server/cmd/server/home_handler_test.go
git commit -m "$(cat <<'EOF'
feat(my): dock category sidebar with mobile nav-drawer

EOF
)"
```

---

### Task 7: Handler regression sweep

**Files:**
- Modify: `server/cmd/server/home_handler_test.go`
- Any remaining stale references under `server/` / `web/`

**Interfaces:**
- Consumes: all prior tasks

- [ ] **Step 1: Grep for stale names**

```bash
rg -n 'PublicHomeCategoryView|PublicHomeSubCategoryView|public-home-category-sidebar|public-home-subcategory"' server web --glob '*.{go,html,css}'
```

Expected: no stale Go types; aside uses `.category-sidebar*`; results headers (`.public-home-results-*`) remain.

- [ ] **Step 2: Add handler assertions** on an existing `/my` catalog test:

```go
if !strings.Contains(body, `href="#cat-`) {
	t.Fatalf("expected sidebar anchor hrefs, got %s", body)
}
if !strings.Contains(body, `id="cat-`) {
	t.Fatalf("expected catalog section ids, got %s", body)
}
```

- [ ] **Step 3: Run broader package slice**

```bash
cd server && go test ./cmd/server/ -run 'TestHandleHome|TestHandlePublicHome|TestCategorySidebar|TestBuildMyHome|TestHomePicker|TestMyHome' -count=1
task css:lint
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add server/cmd/server
git commit -m "$(cat <<'EOF'
test: cover /my category sidebar anchors and public rename

EOF
)"
```

---

## Spec coverage self-check

| Spec requirement | Task |
|------------------|------|
| Shared thin-full `category_sidebar` | 1–3 |
| HTMX vs Href leaf dispatch | 1 |
| Public `/` migration, behavior unchanged | 2–3 |
| `/my` accessible-only sidebar | 5 |
| Uncategorized synthetic category + leaf | 5 |
| Section ids `cat-…` / `cat-uncategorized` | 4 |
| `ShowCategoryHeader` + always-populated category name for sidebar | 4 |
| Two-col inside max-width, same bg | 6 |
| Desktop docked / mobile hamburger drawer | 6 |
| One DOM instance | 6 |
| No scroll-spy / no app JS | 6 (Popover) |
| Empty: no sidebar/trigger | 6 |
| Tests listed in spec | 1, 4–7 |
