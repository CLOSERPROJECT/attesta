# `/my` accessible streams catalog Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace authenticated `/my` stream picker with a taxonomy-grouped catalog of accessible streams using public result headers + `public_stream_card` (optional management menu), header Create CTA, and a shared empty-state component.

**Architecture:** Pure access filter + grouping builders feed a rewritten `home_picker_body`. Cards reuse `buildPublicStreamCardView` with `Href` overridden to `streamPath(key)+"/"`. Management actions live in a shell around the public card. Empty state is extracted as CSS-only `empty-state` and shared with public stream runs empty.

**Tech Stack:** Go `net/http` + `html/template`, existing taxonomy/public-home CSS classes, Vite CSS modules, Go unit tests (`go test`).

**Spec:** `docs/superpowers/specs/2026-07-31-my-home-catalog-design.md`

## Global Constraints

- Access: platform admin → all catalog streams; org admin (`userIsOrgAdmin`) → streams whose `organizations` include user’s `OrgSlug`; else org + matching workflow role for that org.
- No sidebar / no HTMX listing on `/my`; stack non-empty taxonomy groups then Uncategorized.
- Card click → `/my/streams/:key/` (`streamPath(key)+"/"`); public `/` cards stay on `/streams/:key`.
- No `publicHomeStreamCardLimit` on `/my`.
- Create stream only in `page-header-actions` when `canViewFormataBuilder`; not inside empty state.
- Empty: CSS-only `empty-state` extracted from `public-stream-runs-empty`; `/my` uses home-specific title/hint.
- Follow `docs/css.md` + `.agents/skills/attesta-ui-components`.
- Work in worktree `.worktrees/feat/my-home-catalog` on branch `feat/my-home-catalog`.

## File map

| File | Responsibility |
|------|----------------|
| `server/cmd/server/my_home_access.go` | `userCanAccessStream(user, cfg) bool` |
| `server/cmd/server/my_home_access_test.go` | Access matrix unit tests |
| `server/cmd/server/my_home.go` | Grouping + home catalog builders; wire helpers used by `handleHome` |
| `server/cmd/server/my_home_test.go` | Grouping / builder unit tests |
| `server/cmd/server/components.go` | `ManagedPublicStreamCardView`, `MyHomeStreamGroupView`; extend home page view fields if needed |
| `server/cmd/server/main.go` | Update `HomeWorkflowPickerView` / `handleHome`; optionally thin `workflowOptions` if still needed elsewhere |
| `server/cmd/server/home_handler_test.go` | Rewrite picker stubs/assertions for new `/my` markup + access |
| `server/templates/pages/home.html` | New `home_picker_body` |
| `server/templates/components/public_home_stream_group.html` | Shared group: category/subcategory headers + card grid |
| `server/templates/components/public_home_stream_results.html` | Call shared group for single leaf (keep id wrapper + empty CTA for `/`) |
| `server/templates/components/managed_public_stream_card.html` | Shell: menu + dialogs + `public_stream_card` |
| `server/templates/components/public_stream_card.html` | Unchanged core (called from shell / public home) |
| `server/templates/pages/public_stream.html` | Switch runs empty to `empty-state` classes |
| `web/src/styles/components/empty-state.css` | CSS-only empty-state contract |
| `web/src/styles/pages/public-stream.css` | Remove duplicated empty rules (or thin aliases if needed briefly) |
| `web/src/styles/components/public-stream-card.css` | Shell/menu positioning for managed cards |
| `web/src/styles/pages/home.css` | Light catalog stack spacing if needed |
| `web/src/styles/components.css` | Import `empty-state.css` |

---

### Task 1: Access filter

**Files:**
- Create: `server/cmd/server/my_home_access.go`
- Create: `server/cmd/server/my_home_access_test.go`

**Interfaces:**
- Produces: `func userCanAccessStream(user *AccountUser, cfg RuntimeConfig) bool`

- [ ] **Step 1: Write failing tests**

```go
package main

import "testing"

func TestUserCanAccessStream(t *testing.T) {
	cfg := RuntimeConfig{
		Organizations: []WorkflowOrganization{{Slug: "acme", Name: "Acme"}},
		Roles: []WorkflowRole{
			{OrgSlug: "acme", Slug: "operator", Name: "Operator"},
			{OrgSlug: "other", Slug: "reviewer", Name: "Reviewer"},
		},
	}
	otherOrgCfg := RuntimeConfig{
		Organizations: []WorkflowOrganization{{Slug: "other", Name: "Other"}},
		Roles:         []WorkflowRole{{OrgSlug: "other", Slug: "reviewer", Name: "Reviewer"}},
	}

	cases := []struct {
		name string
		user *AccountUser
		cfg  RuntimeConfig
		want bool
	}{
		{
			name: "platform admin sees all",
			user: &AccountUser{IsPlatformAdmin: true},
			cfg:  otherOrgCfg,
			want: true,
		},
		{
			name: "org admin sees org streams without workflow role",
			user: &AccountUser{OrgSlug: "acme", RoleSlugs: []string{"org-admin"}},
			cfg:  cfg,
			want: true,
		},
		{
			name: "org admin does not see other org streams",
			user: &AccountUser{OrgSlug: "acme", RoleSlugs: []string{"org-admin"}},
			cfg:  otherOrgCfg,
			want: false,
		},
		{
			name: "role member matching org+role",
			user: &AccountUser{OrgSlug: "acme", RoleSlugs: []string{"operator"}},
			cfg:  cfg,
			want: true,
		},
		{
			name: "role member wrong role",
			user: &AccountUser{OrgSlug: "acme", RoleSlugs: []string{"reviewer"}},
			cfg:  cfg,
			want: false,
		},
		{
			name: "outsider",
			user: &AccountUser{OrgSlug: "stranger", RoleSlugs: []string{"operator"}},
			cfg:  cfg,
			want: false,
		},
		{
			name: "nil user",
			user: nil,
			cfg:  cfg,
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := userCanAccessStream(tc.user, tc.cfg); got != tc.want {
				t.Fatalf("userCanAccessStream = %v, want %v", got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd server && go test ./cmd/server/ -run TestUserCanAccessStream -count=1`

Expected: FAIL (`userCanAccessStream` undefined)

- [ ] **Step 3: Implement**

```go
package main

import "strings"

func userCanAccessStream(user *AccountUser, cfg RuntimeConfig) bool {
	if user == nil {
		return false
	}
	if user.IsPlatformAdmin {
		return true
	}
	org := strings.TrimSpace(user.OrgSlug)
	if org == "" {
		return false
	}
	participates := false
	for _, o := range cfg.Organizations {
		if strings.TrimSpace(o.Slug) == org {
			participates = true
			break
		}
	}
	if !participates {
		return false
	}
	if userIsOrgAdmin(user) {
		return true
	}
	for _, role := range cfg.Roles {
		if strings.TrimSpace(role.OrgSlug) != org {
			continue
		}
		if containsRole(user.RoleSlugs, strings.TrimSpace(role.Slug)) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `cd server && go test ./cmd/server/ -run TestUserCanAccessStream -count=1`

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/my_home_access.go server/cmd/server/my_home_access_test.go
git commit -m "feat(my): add stream access filter for /my catalog"
```

---

### Task 2: Grouping builder + view types

**Files:**
- Modify: `server/cmd/server/components.go`
- Create: `server/cmd/server/my_home.go`
- Create: `server/cmd/server/my_home_test.go`

**Interfaces:**
- Consumes: `userCanAccessStream`, `TaxonomyCategoryNode`, `RuntimeConfig.Workflow.CategorySlug/SubCategorySlug`, `IsCategorized()`
- Produces:
  - `ManagedPublicStreamCardView` (embeds card + management fields)
  - `MyHomeStreamGroupView`
  - `func buildMyHomeStreamGroups(categories []TaxonomyCategoryNode, cardsByKey map[string]ManagedPublicStreamCardView, catalog map[string]RuntimeConfig, accessibleKeys []string) []MyHomeStreamGroupView`

Add to `components.go`:

```go
// ManagedPublicStreamCardView wraps a public stream card with optional /my management actions.
type ManagedPublicStreamCardView struct {
	Card              PublicStreamCardView
	Key               string
	CanClone          bool
	CanEdit           bool
	EditAction        string
	EditRequiresPurge bool
	CanDelete         bool
	DeleteAction      string
}

// MyHomeStreamGroupView is one taxonomy (or Uncategorized) block on /my.
type MyHomeStreamGroupView struct {
	CategoryName           string
	CategoryIconURL        string
	SubCategoryName        string
	SubCategoryIconURL     string
	SubCategoryDescription string
	Uncategorized          bool
	Streams                []ManagedPublicStreamCardView
}
```

Grouping algorithm in `my_home.go`:
1. Index accessible keys → cards.
2. Walk taxonomy categories/subs in order; for each leaf, collect keys where `cfg.Workflow` matches both slugs; if any, append group with taxonomy names/icons/description.
3. Collect remaining accessible keys (uncategorized or unknown path) → final group with `Uncategorized: true`, `CategoryName: "Uncategorized"`.
4. Within each group, preserve `accessibleKeys` order (caller passes `sortedWorkflowKeys` filtered).

- [ ] **Step 1: Write failing grouping tests**

```go
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
	if groups[1].SubCategoryName != "Order Fulfillment" || groups[1].Streams[0].Key != "a" {
		t.Fatalf("second group = %+v", groups[1])
	}
	if !groups[2].Uncategorized || groups[2].CategoryName != "Uncategorized" || groups[2].Streams[0].Key != "c" {
		t.Fatalf("uncategorized = %+v", groups[2])
	}
}

func TestBuildMyHomeStreamGroupsSkipsEmptyLeaves(t *testing.T) {
	categories := []TaxonomyCategoryNode{{
		Slug: "supply-chain", Name: "Supply Chain",
		SubCategories: []TaxonomySubCategoryNode{
			{Slug: "procurement", Name: "Procurement"},
			{Slug: "order-fulfillment", Name: "Order Fulfillment"},
		},
	}}
	catalog := map[string]RuntimeConfig{
		"b": {Workflow: WorkflowDef{CategorySlug: "supply-chain", SubCategorySlug: "procurement"}},
	}
	cards := map[string]ManagedPublicStreamCardView{"b": {Key: "b"}}
	groups := buildMyHomeStreamGroups(categories, cards, catalog, []string{"b"})
	if len(groups) != 1 || groups[0].SubCategoryName != "Procurement" {
		t.Fatalf("got %+v", groups)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

Run: `cd server && go test ./cmd/server/ -run TestBuildMyHomeStreamGroups -count=1`

- [ ] **Step 3: Implement types + `buildMyHomeStreamGroups`**

- [ ] **Step 4: Run — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/components.go server/cmd/server/my_home.go server/cmd/server/my_home_test.go
git commit -m "feat(my): group accessible streams by taxonomy for /my"
```

---

### Task 3: Extract CSS-only `empty-state`

**Files:**
- Create: `web/src/styles/components/empty-state.css`
- Modify: `web/src/styles/components.css`
- Modify: `web/src/styles/pages/public-stream.css` (remove `.public-stream-runs-empty*` rules)
- Modify: `server/templates/pages/public_stream.html` (markup classes)

**Interfaces:**
- Produces CSS-only contract (no Go DTO):

```
div.empty-state
  (icon svg)
  p.empty-state-title
  p.empty-state-hint
```

- [ ] **Step 1: Add `empty-state.css` with markup-tree header** (copy visual rules from `.public-stream-runs-empty*`, rename to `.empty-state*`)

```css
/*
 * Empty state — centered dashed placeholder (CSS-only).
 *
 * div.empty-state
 *   (optional .icon-svg)
 *   p.empty-state-title
 *   p.empty-state-hint
 */
```

Import in `components.css` after `toast.css` (or near other chrome primitives).

- [ ] **Step 2: Update public stream template**

Replace:

```html
<div class="public-stream-runs-empty">
  {{ template "icon-layers-2" . }}
  <p class="public-stream-runs-empty-title">No completed runs yet</p>
  <p class="public-stream-runs-empty-hint">
    Finished runs with a DPP will appear here.
  </p>
</div>
```

with:

```html
<div class="empty-state">
  {{ template "icon-layers-2" . }}
  <p class="empty-state-title">No completed runs yet</p>
  <p class="empty-state-hint">
    Finished runs with a DPP will appear here.
  </p>
</div>
```

Delete old CSS rules from `public-stream.css`; update the file’s tree comment.

- [ ] **Step 3: Grep for leftover `public-stream-runs-empty`**

Run: `rg public-stream-runs-empty web server`

Expected: no matches (or only plan/spec mentions)

- [ ] **Step 4: Commit**

```bash
git add web/src/styles/components/empty-state.css web/src/styles/components.css web/src/styles/pages/public-stream.css server/templates/pages/public_stream.html
git commit -m "refactor(ui): extract shared empty-state CSS component"
```

---

### Task 4: Managed public stream card shell

**Files:**
- Create: `server/templates/components/managed_public_stream_card.html`
- Modify: `web/src/styles/components/public-stream-card.css` (shell + menu)
- Create or modify: `server/cmd/server/public_stream_card_test.go` (add managed shell tests) — or `managed_public_stream_card_test.go`

**Interfaces:**
- Consumes: `ManagedPublicStreamCardView`
- Produces: template define `managed_public_stream_card`

Shell structure (menu/dialogs outside the card link):

```html
{{ define "managed_public_stream_card" }}
<div class="public-stream-card-shell">
  {{ if .CanClone }}
    <details class="public-stream-card-menu">
      <!-- same menu items as stream_card: Clone / Edit / Delete -->
    </details>
  {{ end }}
  {{ template "public_stream_card" .Card }}
  <!-- edit-purge + delete dialogs copied from stream_card, keyed by .Key -->
</div>
{{ end }}
```

Show the ellipsis when `CanClone || CanEdit || CanDelete` (today’s card gated menu on `CanClone`; preserve that behavior: menu when `CanClone`, disabled edit/delete items as today).

CSS: `.public-stream-card-shell { position: relative; }` absolute menu top-right; ensure `details` summary does not sit inside the `<a>`.

- [ ] **Step 1: Failing template test**

```go
func TestManagedPublicStreamCardRendersMenuAndMyHref(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := ManagedPublicStreamCardView{
		Key:      "wf-a",
		CanClone: true,
		CanEdit:  true,
		EditAction: "/my/organization/formata-builder?stream=wf-a",
		CanDelete: true,
		DeleteAction: "/my/streams/wf-a/delete",
		Card: PublicStreamCardView{
			Name: "Alpha",
			Href: "/my/streams/wf-a/",
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "managed_public_stream_card", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`href="/my/streams/wf-a/"`,
		`class="public-stream-card-shell"`,
		"Clone",
		`id="delete-workflow-wf-a"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func TestManagedPublicStreamCardHidesMenuWithoutClone(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := ManagedPublicStreamCardView{
		Key:  "wf-a",
		Card: PublicStreamCardView{Name: "Alpha", Href: "/my/streams/wf-a/"},
	}
	if err := tmpl.ExecuteTemplate(&out, "managed_public_stream_card", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out.String(), "public-stream-card-menu") {
		t.Fatalf("menu should be absent: %s", out.String())
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (undefined template)

- [ ] **Step 3: Implement template + CSS**

- [ ] **Step 4: Run — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add server/templates/components/managed_public_stream_card.html web/src/styles/components/public-stream-card.css server/cmd/server/managed_public_stream_card_test.go
git commit -m "feat(ui): add managed public stream card shell for /my"
```

---

### Task 5: Shared group partial + rewrite `/my` body

**Files:**
- Create: `server/templates/components/public_home_stream_group.html`
- Modify: `server/templates/components/public_home_stream_results.html`
- Modify: `server/templates/pages/home.html`
- Modify: `web/src/styles/pages/home.css` (optional stack gap)
- Modify: `server/cmd/server/main.go` types for home view fields (`Groups`, `ShowCreateStream`)
- Modify: `server/cmd/server/test_helpers_test.go` stubs if stub layouts break

**Interfaces:**
- `public_home_stream_group` accepts either:
  - public results fields (`PublicHomeStreamResultsView` subset), **or**
  - `MyHomeStreamGroupView` with `Streams` as `[]ManagedPublicStreamCardView`

Prefer **two thin defines** to avoid type friction:

1. `public_home_stream_group` — headers + `range .Streams` calling `public_stream_card` (for `/`)
2. `my_home_stream_group` — same headers; `range .Streams` calling `managed_public_stream_card`; if `.Uncategorized`, only category header “Uncategorized” (no subcategory header)

Refactor `public_home_stream_results` to keep `#public-home-stream-results` wrapper, empty CTA for `/`, and call `public_home_stream_group` for the headers+grid when streams present.

`home_picker_body`:

```html
{{ define "home_picker_body" }}
<section class="stack u-max-w-7xl u-mx-auto my-home">
  <section class="page-header">
    <div class="page-header-head">
      <div class="page-header-body">
        <h1>Choose a stream</h1>
        <p>Select a stream to start or continue process tracking</p>
      </div>
      {{ if .ShowCreateStream }}
      <div class="page-header-actions">
        <a class="btn btn-primary" href="/my/organization/formata-builder">
          {{ template "icon-plus" . }} Create a stream
        </a>
      </div>
      {{ end }}
    </div>
  </section>
  {{ if .Confirmation }}<p class="confirmation">{{ .Confirmation }}</p>{{ end }}
  {{ template "error_banner.html" . }}
  {{ if .Groups }}
    <div class="my-home-catalog">
      {{ range .Groups }}
        {{ template "my_home_stream_group" . }}
      {{ end }}
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

Update `HomeWorkflowPickerView` / `WorkflowPickerView`:

```go
type HomeWorkflowPickerView struct {
	PageBase
	Groups          []MyHomeStreamGroupView
	ShowCreateStream bool
	Error           string
	Confirmation    string
}
```

Remove `/my` dependency on `Workflows` / `ShowCreateStreamCard` once tests are updated (Task 6–7). Keep `WorkflowPickerView` only if still used elsewhere; otherwise delete dead fields carefully.

- [ ] **Step 1: Template unit test for `my_home_stream_group` headers + empty home body**

- [ ] **Step 2: Implement templates; keep public home tests green**

Run: `cd server && go test ./cmd/server/ -run 'TestHandlePublicHome|TestPublicHome|TestBuildPublicHome' -count=1`

- [ ] **Step 3: Commit**

```bash
git add server/templates/pages/home.html server/templates/components/public_home_stream_group.html server/templates/components/public_home_stream_results.html server/templates/components/my_home_stream_group.html web/src/styles/pages/home.css server/cmd/server/main.go
git commit -m "feat(my): render taxonomy-grouped catalog markup on /my"
```

---

### Task 6: Wire `handleHome` builder

**Files:**
- Modify: `server/cmd/server/my_home.go` (add `buildMyHomeCatalog` on `*Server`)
- Modify: `server/cmd/server/main.go` (`handleHome`)

**Interfaces:**
- Produces: `func (s *Server) buildMyHomeCatalog(ctx context.Context, user *AccountUser) ([]MyHomeStreamGroupView, error)`
- Management flags: reuse the same rules as `workflowOptions` (Formata `canViewFormataBuilder` → `CanClone`; Cerbos edit/delete; `EditAction`/`DeleteAction` paths). Prefer extracting a small `streamManagementFlags(...)` helper from `workflowOptions` to avoid drift; if extract is large, duplicate the permission block once with a comment pointing at `workflowOptions`.

Builder steps:
1. `catalog, err := s.workflowCatalog()`
2. `categories, err := loadTaxonomyTree(ctx, s.store)`
3. `logoURLs := organizationLogoURLMap(ctx, s.identity)`
4. Filter `sortedWorkflowKeys` with `userCanAccessStream`
5. For each key: `buildPublicStreamCardView` then `card.Href = streamPath(key) + "/"`; attach management flags into `ManagedPublicStreamCardView`
6. `return buildMyHomeStreamGroups(...)`

`handleHome`:

```go
groups, err := s.buildMyHomeCatalog(r.Context(), user)
// ...
showCreate, authErr := s.canViewFormataBuilder(r.Context(), user)
view := HomeWorkflowPickerView{
  PageBase: s.pageBaseForUser(user, "home_picker_body", "", ""),
  Groups: groups,
  ShowCreateStream: showCreate && authErr == nil && showCreate,
  Error: homePickerMessage(r, "error"),
  Confirmation: homePickerMessage(r, "confirmation"),
}
```

(Use the same error-log pattern as today when Cerbos check fails for create.)

- [ ] **Step 1: Failing handler test** (auth user with role sees only matching stream; href `/my/streams/.../`; create in header)

Use real `parseTestTemplates` + seeded taxonomy + categorized YAML (patterns from `public_home_test.go` / `home_handler_test.go`).

- [ ] **Step 2: Implement builder + wire handler**

- [ ] **Step 3: Run focused tests — PASS**

Run: `cd server && go test ./cmd/server/ -run 'TestUserCanAccessStream|TestBuildMyHomeStreamGroups|TestHandleHome' -count=1`

- [ ] **Step 4: Commit**

```bash
git add server/cmd/server/my_home.go server/cmd/server/main.go server/cmd/server/home_handler_test.go
git commit -m "feat(my): wire accessible catalog into handleHome"
```

---

### Task 7: Update legacy home picker tests + cleanup

**Files:**
- Modify: `server/cmd/server/home_handler_test.go` (replace `homePickerTemplates` stubs and assertions that expect `PICK N` / `stream-card` / create-card CTA)
- Modify: any other tests referencing `WorkflowPickerView.Workflows` on `/my`
- Delete unused `stream_card_create` usage from `/my` only; keep `stream_card.html` if still referenced; delete create define only if unused

Checklist of tests to rewrite (from current file):
- `TestHandleHomeRendersWorkflowPicker` → assert groups/empty via stub or real templates
- `TestHandleHomePickerCreateStreamCardVisibility` → assert header Create link / absence (`ShowCreateStream`), not `stream-card-cta`
- `TestHandleHomePickerRendersWorkflowCardsAndScopedLinks` → `/my/streams/.../` on public card href
- `TestHandleHomePickerDeleteButtonVisibility` → managed menu/delete dialog
- Count/turn-indicator tests: either drop turn indicator (not on public card) **or** keep out of scope — **do not** reintroduce turn bell unless spec asks (spec does not). Update tests to drop `HasUserTurn` expectations on `/my`.
- Update `homePickerTemplates()` stub to print groups/keys if still used for fast tests

- [ ] **Step 1: Fix failing suite under `TestHandleHome*`**

Run: `cd server && go test ./cmd/server/ -run 'TestHandleHome|TestManagedPublic|TestUserCanAccess|TestBuildMyHome' -count=1`

- [ ] **Step 2: Broader regression**

Run: `cd server && go test ./cmd/server/ -count=1`

Expected: PASS except known pre-existing OpenAPI docs 404 failures (`TestHandleDocsRoutes`, `TestServeOpenAPIFileRewritesServerToRequestOrigin`) — do not “fix” those in this plan.

- [ ] **Step 3: Commit**

```bash
git add server/cmd/server/home_handler_test.go server/templates/components/stream_card.html
git commit -m "test(my): align /my home tests with accessible catalog"
```

---

## Self-review (plan vs spec)

| Spec requirement | Task |
|------------------|------|
| Access rules (platform / org admin / role) | Task 1 + 6 |
| No sidebar; stacked category/subcategory headers | Tasks 2, 5 |
| Uncategorized group | Task 2 |
| `public_stream_card` + management menu | Task 4 |
| Href `/my/streams/:key/` | Tasks 4, 6 |
| Create in page-header-actions | Task 5–6 |
| Empty-state extraction + `/my` empty | Tasks 3, 5 |
| No 6-card cap | Task 2/6 (no limit applied) |
| Tests: access, grouping, markup | Tasks 1, 2, 4, 6, 7 |
| Public `/` unchanged aside from shared group/empty extract | Tasks 3, 5 |

No TBD placeholders. Types consistent: `ManagedPublicStreamCardView`, `MyHomeStreamGroupView`, `userCanAccessStream`, `buildMyHomeStreamGroups`, `buildMyHomeCatalog`.
