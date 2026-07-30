# Platform admin categories CRUD Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the browse-only `/admin/categories` panel with an HTMX-driven CRUD editor for taxonomy groups (`Category`) and leaves (`SubCategory`).

**Architecture:** Extend `Store` with create/update/reorder mutations (Memory + Mongo). Platform-admin handlers use two POST targets with small `intent` switches. GET query flags open inline create/edit forms. HTMX swaps `#platform-admin-categories`. Delete eligibility is computed when building the tree view; reorder uses ↑/↓ with `hx-sync` serialization.

**Tech Stack:** Go `net/http` + `html/template`, HTMX (layout), Vite CSS (`platform-admin.css`), existing `<dialog class="dialog">`, `canonifySlug`, taxonomy icon allowlist / `/static/taxonomy/*.svg`.

**Spec:** `docs/superpowers/specs/2026-07-30-platform-admin-categories-crud-design.md`

## Global Constraints

- Work only in git worktree `.worktrees/feat/admin-categories-crud` on branch `feat/admin-categories-crud`.
- English copy only; no drawer chrome / close (X); replace categories panel body inside existing admin console.
- Groups have no `Description`; leaves keep it. Slugs via `canonifySlug(name)` on create only (immutable after).
- Always-expanded tree (no collapse control). Create form at insertion point (list end); edit replaces row; append last on save.
- Delete: disabled + tooltip when blocked; confirm dialog when allowed. Eligibility on render (tree + catalog).
- Reorder: arrows only; serialize with HTMX `hx-sync` / disable-while-pending (no SortableJS, no optimistic debounce).
- POST shape: `POST /admin/categories` and `POST /admin/categories/{slug}/subcategories` with `intent=create|update|delete|reorder`.
- HTMX swap target `#platform-admin-categories`; non-HTMX may get full page. `requirePlatformAdmin` on all routes.
- Meta pill: `{N} groups · {M} categories` (groups=`Category`, categories=`SubCategory`).
- Read `docs/css.md` before CSS; prefer extending `web/src/styles/pages/platform-admin.css`.
- TDD: failing test → implement → pass → commit per task.
- Baseline note: full `go test ./cmd/server/` currently fails OpenAPI docs routes (`TestHandleDocsRoutes`, `TestServeOpenAPIFileRewritesServerToRequestOrigin`) — pre-existing, unrelated. Prefer taxonomy/admin test filters until those are fixed elsewhere.

---

## File map

| File | Responsibility |
|------|----------------|
| `server/cmd/server/store.go` | Add Create/Update/Reorder methods to `Store` interface |
| `server/cmd/server/taxonomy.go` | Memory + Mongo implementations; `ErrTaxonomySlugExists`; `taxonomyIconKeys()` |
| `server/cmd/server/taxonomy_mutations_test.go` | Store mutation unit tests |
| `server/cmd/server/taxonomy_admin.go` | Enrich tree nodes with editor flags; form/editor view builders |
| `server/cmd/server/taxonomy_admin_test.go` | Eligibility / counts / form-mode tests |
| `server/cmd/server/taxonomy_subcategory_delete.go` | Keep as leaf delete guard (handlers call `Server.DeleteSubCategory`) |
| `server/cmd/server/admin_categories.go` | Peel categories GET/POST + subcategory POST from `main.go` |
| `server/cmd/server/admin_categories_handler_test.go` | Extend handler/HTMX/intent tests |
| `server/cmd/server/main.go` | Register `/admin/categories/` route; thin wrappers if needed |
| `server/cmd/server/admin_console.go` | Subtitle/copy: “Manage stream discovery taxonomy” |
| `server/templates/pages/platform_admin.html` | Replace `platform_admin_categories_panel` with editor markup |
| `web/src/styles/pages/platform-admin.css` | Editor/form/icon-grid styles |
| `web/src/main.js` | Icon picker toggle + scroll-into-view for create form after swap |
| `server/cmd/server/test_helpers_test.go` | Stub templates if handler tests need new markers |
| `server/cmd/server/platform_admin_template_test.go` | Markup contracts for editor |

---

### Task 1: Store create/update for Category

**Files:**
- Modify: `server/cmd/server/store.go` (`Store` interface taxonomy section)
- Modify: `server/cmd/server/taxonomy.go`
- Create: `server/cmd/server/taxonomy_mutations_test.go`

**Interfaces:**
- Consumes: `Category`, `validateTaxonomyIcon`, `canonifySlug` (callers derive slug), `bumpTaxonomyRevision` / `taxonomyRevision++`
- Produces:
  - `var ErrTaxonomySlugExists = errors.New("taxonomy slug already exists")`
  - `CreateCategory(ctx context.Context, category Category) (Category, error)` — requires Name+Icon; Slug must be set by caller; sets `sortOrder=max+1`; rejects duplicate slug / invalid icon / empty name
  - `UpdateCategory(ctx context.Context, slug string, name, icon string) (Category, error)` — updates name/icon only; `mongo.ErrNoDocuments` if missing

- [ ] **Step 1: Write the failing tests**

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server/ -count=1 -run 'TestMemoryStoreCreateCategory|TestMemoryStoreUpdateCategory' -v`

Expected: FAIL (methods missing / undefined `ErrTaxonomySlugExists`)

- [ ] **Step 3: Implement minimal Memory + Mongo + interface**

Add to `Store` in `store.go`:

```go
CreateCategory(ctx context.Context, category Category) (Category, error)
UpdateCategory(ctx context.Context, slug string, name, icon string) (Category, error)
```

In `taxonomy.go`, add `ErrTaxonomySlugExists`, then MemoryStore:

```go
func (s *MemoryStore) CreateCategory(_ context.Context, category Category) (Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureTaxonomyMaps()
	slug := strings.TrimSpace(category.Slug)
	name := strings.TrimSpace(category.Name)
	icon := strings.TrimSpace(category.Icon)
	if slug == "" || name == "" {
		return Category{}, fmt.Errorf("category slug and name are required")
	}
	if err := validateTaxonomyIcon(icon); err != nil {
		return Category{}, err
	}
	if _, exists := s.categories[slug]; exists {
		return Category{}, ErrTaxonomySlugExists
	}
	maxOrder := 0
	for _, c := range s.categories {
		if c.SortOrder > maxOrder {
			maxOrder = c.SortOrder
		}
	}
	if category.ID.IsZero() {
		category.ID = primitive.NewObjectID()
	}
	category.Slug, category.Name, category.Icon = slug, name, icon
	category.SortOrder = maxOrder + 1
	s.categories[slug] = cloneCategory(category)
	s.taxonomyRevision++
	return cloneCategory(category), nil
}

func (s *MemoryStore) UpdateCategory(_ context.Context, slug, name, icon string) (Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureTaxonomyMaps()
	trimmed := strings.TrimSpace(slug)
	existing, ok := s.categories[trimmed]
	if !ok {
		return Category{}, mongo.ErrNoDocuments
	}
	name = strings.TrimSpace(name)
	icon = strings.TrimSpace(icon)
	if name == "" {
		return Category{}, fmt.Errorf("category name is required")
	}
	if err := validateTaxonomyIcon(icon); err != nil {
		return Category{}, err
	}
	existing.Name, existing.Icon = name, icon
	s.categories[trimmed] = cloneCategory(existing)
	s.taxonomyRevision++
	return cloneCategory(existing), nil
}
```

MongoStore: `InsertOne` / `UpdateOne` on `collectionCategories`; map duplicate key → `ErrTaxonomySlugExists`; call `bumpTaxonomyRevision`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd server && go test ./cmd/server/ -count=1 -run 'TestMemoryStoreCreateCategory|TestMemoryStoreUpdateCategory' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/store.go server/cmd/server/taxonomy.go server/cmd/server/taxonomy_mutations_test.go
git commit -m "feat(taxonomy): add Category create and update store APIs"
```

---

### Task 2: Store create/update for SubCategory

**Files:**
- Modify: `server/cmd/server/store.go`
- Modify: `server/cmd/server/taxonomy.go`
- Modify: `server/cmd/server/taxonomy_mutations_test.go`

**Interfaces:**
- Consumes: Task 1 patterns; parent must exist for create
- Produces:
  - `CreateSubCategory(ctx context.Context, sub SubCategory) (SubCategory, error)` — append sortOrder within parent; duplicate `(categorySlug,slug)` → `ErrTaxonomySlugExists`; missing parent → `mongo.ErrNoDocuments` (or explicit error — use `mongo.ErrNoDocuments` for parent)
  - `UpdateSubCategory(ctx context.Context, categorySlug, slug, name, icon, description string) (SubCategory, error)`

- [ ] **Step 1: Write the failing tests**

```go
func TestMemoryStoreCreateSubCategoryAppendsWithinParent(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.CreateCategory(t.Context(), Category{
		Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability",
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.CreateSubCategory(t.Context(), SubCategory{
		CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement",
		Icon: "procurement-workflow", Description: "PO",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateSubCategory(t.Context(), SubCategory{
		CategorySlug: "supply-chain", Slug: "order-fulfillment", Name: "Fulfillment",
		Icon: "order-fulfillment",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.SortOrder != 1 || second.SortOrder != 2 {
		t.Fatalf("orders %d %d", first.SortOrder, second.SortOrder)
	}
}

func TestMemoryStoreCreateSubCategoryRequiresParent(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.CreateSubCategory(t.Context(), SubCategory{
		CategorySlug: "missing", Slug: "leaf", Name: "Leaf", Icon: "weee",
	})
	if !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("err=%v", err)
	}
}

func TestMemoryStoreUpdateSubCategory(t *testing.T) {
	store := NewMemoryStore()
	_, _ = store.CreateCategory(t.Context(), Category{
		Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability",
	})
	_, err := store.CreateSubCategory(t.Context(), SubCategory{
		CategorySlug: "supply-chain", Slug: "procurement", Name: "Old",
		Icon: "procurement-workflow", Description: "d1",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.UpdateSubCategory(t.Context(), "supply-chain", "procurement", "New", "order-fulfillment", "d2")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "New" || got.Icon != "order-fulfillment" || got.Description != "d2" || got.Slug != "procurement" {
		t.Fatalf("%+v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server/ -count=1 -run 'TestMemoryStoreCreateSubCategory|TestMemoryStoreUpdateSubCategory' -v`

Expected: FAIL (methods missing)

- [ ] **Step 3: Implement Memory + Mongo + interface**

Mirror Task 1; max `sortOrder` among subs with same `categorySlug`; validate icon; bump revision.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd server && go test ./cmd/server/ -count=1 -run 'TestMemoryStoreCreateSubCategory|TestMemoryStoreUpdateSubCategory' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/store.go server/cmd/server/taxonomy.go server/cmd/server/taxonomy_mutations_test.go
git commit -m "feat(taxonomy): add SubCategory create and update store APIs"
```

---

### Task 3: Store reorder (swap with neighbor)

**Files:**
- Modify: `server/cmd/server/store.go`
- Modify: `server/cmd/server/taxonomy.go`
- Modify: `server/cmd/server/taxonomy_mutations_test.go`

**Interfaces:**
- Produces:
  - `ReorderCategory(ctx context.Context, slug, direction string) error` — `direction` is `up` or `down`; no-op or error at ends — use `fmt.Errorf("cannot move further")` sentinel `ErrTaxonomyReorderBoundary`
  - `ReorderSubCategory(ctx context.Context, categorySlug, slug, direction string) error`

- [ ] **Step 1: Write the failing tests**

```go
func TestMemoryStoreReorderCategoryUpDown(t *testing.T) {
	store := NewMemoryStore()
	_, _ = store.CreateCategory(t.Context(), Category{Slug: "a", Name: "A", Icon: "weee"})
	_, _ = store.CreateCategory(t.Context(), Category{Slug: "b", Name: "B", Icon: "batch-traceability"})
	if err := store.ReorderCategory(t.Context(), "b", "up"); err != nil {
		t.Fatal(err)
	}
	list, _ := store.ListCategories(t.Context())
	if list[0].Slug != "b" || list[1].Slug != "a" {
		t.Fatalf("order=%v %v", list[0].Slug, list[1].Slug)
	}
	if err := store.ReorderCategory(t.Context(), "b", "up"); !errors.Is(err, ErrTaxonomyReorderBoundary) {
		t.Fatalf("err=%v", err)
	}
}

func TestMemoryStoreReorderSubCategoryWithinParent(t *testing.T) {
	store := NewMemoryStore()
	_, _ = store.CreateCategory(t.Context(), Category{Slug: "g", Name: "G", Icon: "weee"})
	_, _ = store.CreateSubCategory(t.Context(), SubCategory{CategorySlug: "g", Slug: "s1", Name: "S1", Icon: "weee"})
	_, _ = store.CreateSubCategory(t.Context(), SubCategory{CategorySlug: "g", Slug: "s2", Name: "S2", Icon: "batch-traceability"})
	if err := store.ReorderSubCategory(t.Context(), "g", "s2", "up"); err != nil {
		t.Fatal(err)
	}
	subs, _ := store.ListSubCategories(t.Context(), "g")
	if subs[0].Slug != "s2" {
		t.Fatalf("got %s", subs[0].Slug)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./cmd/server/ -count=1 -run 'TestMemoryStoreReorder' -v`

Expected: FAIL

- [ ] **Step 3: Implement**

List siblings sorted by `sortOrder`, find index, swap `sortOrder` with neighbor, persist both, bump revision. Invalid direction → error.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd server && go test ./cmd/server/ -count=1 -run 'TestMemoryStoreReorder' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/store.go server/cmd/server/taxonomy.go server/cmd/server/taxonomy_mutations_test.go
git commit -m "feat(taxonomy): add sibling reorder store APIs"
```

---

### Task 4: Editor view model + delete eligibility

**Files:**
- Modify: `server/cmd/server/taxonomy_admin.go`
- Modify: `server/cmd/server/taxonomy_admin_test.go` (create if missing beyond current URL tests)
- Modify: `server/cmd/server/main.go` (`PlatformAdminView` fields) **or** keep editor fields only on a dedicated `CategoriesEditorView` embedded in panel

**Interfaces:**
- Consumes: `loadTaxonomyTree`, `catalogReferencesSubCategoryPath`, `Server.workflowCatalog`
- Produces enriched nodes + form state:

```go
type TaxonomyCategoryNode struct {
	// existing fields…
	CanDelete  bool
	CanMoveUp  bool
	CanMoveDown bool
	DeleteReason string // tooltip when !CanDelete
	SubCategories []TaxonomySubCategoryNode
}

type TaxonomySubCategoryNode struct {
	// existing fields…
	CanDelete    bool
	CanMoveUp    bool
	CanMoveDown  bool
	DeleteReason string
}

type CategoriesEditorForm struct {
	Open        bool
	Level       string // "group" | "leaf"
	Mode        string // "create" | "edit"
	ParentSlug  string
	Slug        string // edit only
	Name        string
	Icon        string
	Description string
	Error       string
}

type CategoriesEditorView struct {
	Categories []TaxonomyCategoryNode
	GroupCount int
	LeafCount  int
	Form       CategoriesEditorForm
	IconKeys   []string // allowlist for picker grid
	Confirmation string
}
```

- `func taxonomyIconKeys() []string` — sorted keys from `taxonomyIconAllowlist`
- `func enrichTaxonomyEditorTree(nodes []TaxonomyCategoryNode, referenced func(categorySlug, subSlug string) bool) []TaxonomyCategoryNode`
- `func parseCategoriesEditorQuery(q url.Values) CategoriesEditorForm`
- `func (s *Server) buildCategoriesEditorView(ctx, q, formErr, confirmation) (CategoriesEditorView, error)`

Eligibility rules:
- Group `CanDelete` iff `len(SubCategories)==0`; else reason `"Has subcategories"`
- Leaf `CanDelete` iff `!referenced(parent, slug)`; else `"Referenced by a stream"`
- `CanMoveUp` / `CanMoveDown` from sibling index

- [ ] **Step 1: Write failing unit tests** for enrich + query parse + counts (no HTTP)

- [ ] **Step 2: Run to verify fail**

Run: `cd server && go test ./cmd/server/ -count=1 -run 'TestEnrichTaxonomyEditor|TestParseCategoriesEditorQuery|TestTaxonomyIconKeys' -v`

- [ ] **Step 3: Implement helpers in `taxonomy_admin.go`**

- [ ] **Step 4: Run to verify pass**

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/taxonomy_admin.go server/cmd/server/taxonomy_admin_test.go server/cmd/server/main.go
git commit -m "feat(admin): categories editor view model and delete eligibility"
```

---

### Task 5: GET `/admin/categories` renders editor panel + HTMX partial

**Files:**
- Create: `server/cmd/server/admin_categories.go` (move/expand `handleAdminCategories`)
- Modify: `server/cmd/server/main.go` (call into new file; keep register)
- Modify: `server/templates/pages/platform_admin.html`
- Modify: `server/cmd/server/admin_console.go` (subtitle)
- Modify: `server/cmd/server/admin_categories_handler_test.go`
- Modify: `server/cmd/server/admin_console_test.go` (subtitle expectation)
- Modify: `server/cmd/server/platform_admin_template_test.go`

**Interfaces:**
- Consumes: Task 4 builders
- Produces:
  - `wantsCategoriesPanelPartial(r *http.Request) bool` — HTMX and `HX-Target` is `platform-admin-categories`
  - Panel wrapper: `<div id="platform-admin-categories">…</div>`
  - GET supports query modes; still supports `wantsAdminConsolePartial` for sidebar soft-nav

- [ ] **Step 1: Write failing handler/template tests**

Assert body contains `id="platform-admin-categories"`, meta pill `1 groups · 2 categories` (with seed), `Manage stream discovery taxonomy`, and with `?new=group` contains form markers (`name="intent" value="create"`, no description textarea for group). HTMX with `HX-Target: platform-admin-categories` returns panel without full layout chrome markers as appropriate.

- [ ] **Step 2: Run to verify fail**

Run: `cd server && go test ./cmd/server/ -count=1 -run 'TestHandleAdminCategories|TestPlatformAdminCategories' -v`

- [ ] **Step 3: Implement GET path + minimal list markup** (actions can be stubs linking to query modes; forms can be partial)

Update `platform_admin_categories_panel` to wrap editor list (always show all groups expanded with leaves). Wire `handleAdminCategories` GET through `buildCategoriesEditorView`.

- [ ] **Step 4: Run to verify pass** (update old “Browse…” assertions)

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/admin_categories.go server/cmd/server/main.go server/cmd/server/admin_console.go server/templates/pages/platform_admin.html server/cmd/server/admin_categories_handler_test.go server/cmd/server/admin_console_test.go server/cmd/server/platform_admin_template_test.go
git commit -m "feat(admin): render categories editor panel with HTMX target"
```

---

### Task 6: POST group intents on `/admin/categories`

**Files:**
- Modify: `server/cmd/server/admin_categories.go`
- Modify: `server/cmd/server/admin_categories_handler_test.go`

**Interfaces:**
- `handleAdminCategories` accepts POST when path is exactly `/admin/categories`
- Form fields: `intent`, `slug` (update/delete/reorder), `name`, `icon`, `direction` (`up`/`down`)
- On success: re-render panel (HTMX) or full page with confirmation
- On validation error: re-render with `Form` open and `Form.Error`

Slug on create: `canonifySlug(name)` before `CreateCategory`.

Delete group: `store.DeleteCategory` (already refuses with children).

- [ ] **Step 1: Write failing tests** for create / update / delete / reorder / duplicate slug / non-admin

Example create:

```go
func TestHandleAdminCategoriesCreateGroup(t *testing.T) {
	// platform admin cookie, MemoryStore, POST intent=create&name=New+Group&icon=weee
	// assert 200, body contains "New Group", store has slug "new-group"
}
```

- [ ] **Step 2: Run to verify fail**

Run: `cd server && go test ./cmd/server/ -count=1 -run 'TestHandleAdminCategoriesCreate|TestHandleAdminCategoriesUpdate|TestHandleAdminCategoriesDelete|TestHandleAdminCategoriesReorder' -v`

- [ ] **Step 3: Implement POST switch**

```go
switch intent {
case "create": // …
case "update": // …
case "delete": // …
case "reorder": // direction
default: // 400
}
```

After mutate, call shared `renderCategoriesEditor(w,r,admin, confirmation, formErr)`.

- [ ] **Step 4: Run to verify pass**

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/admin_categories.go server/cmd/server/admin_categories_handler_test.go
git commit -m "feat(admin): POST intents for category groups"
```

---

### Task 7: POST leaf intents on `/admin/categories/{slug}/subcategories`

**Files:**
- Modify: `server/cmd/server/admin_categories.go`
- Modify: `server/cmd/server/main.go` — `mux.HandleFunc("/admin/categories/", s.handleAdminCategoriesPath)`
- Modify: `server/cmd/server/admin_categories_handler_test.go`

**Interfaces:**
- Parse path: trim `/admin/categories/`, expect `{slug}/subcategories`
- Same intents; create uses parent from URL; delete via `s.DeleteSubCategory` (catalog guard)
- Fields: `name`, `icon`, `description`, `slug` (edit/delete/reorder), `direction`

- [ ] **Step 1: Write failing tests** including delete blocked when catalog references path

Seed taxonomy + put a `RuntimeConfig` in server catalog with matching `CategorySlug`/`SubCategorySlug` (follow existing catalog test helpers in `stream_categorization_test.go` / config tests). Expect delete disabled in GET and POST delete returns error path.

- [ ] **Step 2: Run to verify fail**

Run: `cd server && go test ./cmd/server/ -count=1 -run 'TestHandleAdminSubcategories' -v`

- [ ] **Step 3: Implement path handler + intents**

- [ ] **Step 4: Run to verify pass**

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/admin_categories.go server/cmd/server/main.go server/cmd/server/admin_categories_handler_test.go
git commit -m "feat(admin): POST intents for taxonomy subcategories"
```

---

### Task 8: Full editor UI — actions, forms, dialogs, reorder sync

**Files:**
- Modify: `server/templates/pages/platform_admin.html`
- Modify: `web/src/styles/pages/platform-admin.css`
- Modify: `web/src/main.js`
- Modify: `server/cmd/server/platform_admin_template_test.go`

**Interfaces:**
- Markup contracts (assert in template tests):
  - Top add: `hx-get="/admin/categories?new=group"` `hx-target="#platform-admin-categories"` `hx-swap="outerHTML"`
  - Per group add-sub: `?new=sub&parent={{.Slug}}`
  - Edit links: `?edit=group&slug=` / `?edit=sub&parent=&slug=`
  - Reorder buttons: `hx-post` + `hx-sync="closest form:abort"` or `hx-sync="#platform-admin-categories:queue all"` — pick one pattern and use consistently; also `hx-disabled-elt` on the reorder control group
  - Delete: `<dialog>` only when `CanDelete`; else `<button disabled title="{{.DeleteReason}}">`
  - Form: icon grid from `.IconKeys`; description textarea only when `eq .Form.Level "leaf"`; Cancel `hx-get="/admin/categories"`
  - Create form hosts `id="categories-editor-form"` for scroll hook

- [ ] **Step 1: Write failing template tests** for key selectors / hx attributes / no description on group form / dialog present when CanDelete

- [ ] **Step 2: Run to verify fail**

- [ ] **Step 3: Implement markup + CSS + minimal JS**

`main.js` (minimal):

```js
document.body.addEventListener("htmx:afterSwap", (e) => {
  if (e.detail?.target?.id !== "platform-admin-categories") return;
  const form = document.getElementById("categories-editor-form");
  form?.scrollIntoView({ block: "nearest", behavior: "smooth" });
});
// icon picker: toggle .is-open on [data-taxonomy-icon-picker]; set hidden input[name=icon] on choose
```

Follow `docs/css.md`; extend `.platform-admin-taxonomy-*` rather than inventing a dark drawer theme.

- [ ] **Step 4: Run template + handler smoke tests**

Run: `cd server && go test ./cmd/server/ -count=1 -run 'TestPlatformAdminCategories|TestHandleAdminCategories|TestHandleAdminSubcategories' -v`

Also: `cd web && npm run build` if assets change.

- [ ] **Step 5: Commit**

```bash
git add server/templates/pages/platform_admin.html web/src/styles/pages/platform-admin.css web/src/main.js server/cmd/server/platform_admin_template_test.go
git commit -m "feat(ui): categories editor forms, dialogs, and reorder HTMX"
```

---

### Task 9: End-to-end hardening + copy polish

**Files:**
- Modify: any remaining assertion drift (`breadcrumbs_test`, `platform_admin_htmx_test`, console copy)
- Modify: `server/cmd/server/admin_console.go` nav Copy strings to “Manage stream discovery taxonomy”

- [ ] **Step 1: Run focused regression suite**

```bash
cd server && go test ./cmd/server/ -count=1 -run 'AdminCategor|Taxonomy|PlatformAdminCategor|DeleteSubCategory|TaxonomyMutation|Reorder'
```

Expected: PASS

- [ ] **Step 2: Fix any failures** from renamed copy/markup

- [ ] **Step 3: Manual checklist** (optional in worktree with `task dev`): create group, add leaf, reorder with rapid clicks (second click waits), blocked delete tooltip, referenced leaf cannot delete

- [ ] **Step 4: Commit**

```bash
git add -u
git commit -m "test(admin): harden categories CRUD editor coverage"
```

---

## Spec coverage checklist

| Spec decision | Task |
|---------------|------|
| Replace `/admin/categories` browse panel | 5, 8 |
| Full CRUD both levels | 1–3, 6–7 |
| Always expanded / no collapse | 5, 8 |
| Inline form; create at end; edit in place; append last | 1–2, 4–5, 8 |
| No group description | 4, 8 |
| Immutable slug from `canonifySlug` | 6–7 |
| Add subcategory per group; top + for group | 8 |
| Icon allowlist grid | 4, 8 |
| Delete disabled+tooltip; confirm when allowed | 4, 8 |
| Arrows + hx-sync serialize | 3, 8 |
| Two POST targets + intent | 6–7 |
| GET query modes | 4–5 |
| HTMX `#platform-admin-categories` | 5, 8 |
| Platform admin auth | 6–7 |
| English + meta pill wording | 5, 8 |
| Worktree | Global constraint |
| Out of scope (DnD, slug edit, seed replace, public home) | — not tasked |
