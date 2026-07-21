# Breadcrumbs Component Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `page_header_back` with a full `breadcrumbs` component (Go view, HTML partial, CSS) on stream, process, org-admin, and platform-admin page headers.

**Architecture:** Handlers assemble a `BreadcrumbsView` (`[]BreadcrumbItem`) on each page view. A shared `breadcrumbs` template renders a semantic `<nav>`/`<ol>` trail (empty `Href` = current non-link). CSS lives in `components/breadcrumbs.css`. The old Back micro-partial and `icon-back` are removed.

**Tech Stack:** Go `html/template`, Vite CSS (`web/src/styles/`), existing `go test` + `task css:lint`.

**Spec:** `docs/superpowers/specs/2026-07-21-breadcrumbs-design.md`

## Global Constraints

- Full component: `BreadcrumbItem` + `BreadcrumbsView` in `components.go`; template `breadcrumbs`; CSS `breadcrumbs.css` with prefix `breadcrumbs-*`
- Root crumb label is exactly `Streams`, href `/`
- Trail includes current page as last crumb (empty `Href`, `aria-current="page"`)
- Process current label: `Instance: ` + trimmed instance name, else `Instance: ` + process ID
- Org admin: `Streams` → `Organization admin` (`/org-admin/profile`) → section from `ActivePanel` (`Profile` / `Roles` / `Members`)
- Platform admin: `Streams` → `Platform admin` (current) only
- Do not change page `h1` copy; do not add breadcrumbs to home picker, DPP, or auth pages
- Remove `page_header_back`, `.page-header-back`, and unused `icon-back`
- Prefer minimal, localized diffs; update `docs/css.md` and attesta UI skill to match

## File map

| File | Responsibility |
|------|----------------|
| `server/cmd/server/components.go` | `BreadcrumbItem`, `BreadcrumbsView` types |
| `server/cmd/server/breadcrumbs.go` | Trail builders + label helpers |
| `server/cmd/server/breadcrumbs_test.go` | Unit tests for builders/labels |
| `server/templates/components/breadcrumbs.html` | `{{ define "breadcrumbs" }}` markup |
| `server/cmd/server/breadcrumbs_template_test.go` | Template render tests (rename/replace `page_header_test.go`) |
| `web/src/styles/components/breadcrumbs.css` | Breadcrumb styles + markup-tree header comment |
| `web/src/styles/components.css` | Import `breadcrumbs.css` |
| `web/src/styles/components/page-header.css` | Drop `.page-header-back`; comment uses `nav.breadcrumbs?` |
| `server/cmd/server/main.go` | Add `Breadcrumbs` field on views; assemble in builders |
| `server/templates/pages/stream.html` | Use `breadcrumbs` instead of `page_header_back` |
| `server/templates/pages/process.html` | Same |
| `server/templates/pages/org_admin.html` | Same |
| `server/templates/pages/platform_admin.html` | Same |
| `server/templates/components/page_header.html` | Delete (only defines `page_header_back`) |
| `server/templates/icons.html` | Remove `icon-back` define |
| `server/cmd/server/templates_test.go` | Expect `breadcrumbs`, not `page_header_back` |
| `docs/css.md` | Index + page-header / breadcrumbs docs |
| `.agents/skills/attesta-ui-components/SKILL.md` | Full-component breadcrumbs; drop back exception |
| `AGENTS.md` | Drop `page_header_back` from page-header bullet |

---

### Task 1: Breadcrumb view types and builders

**Files:**
- Modify: `server/cmd/server/components.go` (append types near other component DTOs)
- Create: `server/cmd/server/breadcrumbs.go`
- Create: `server/cmd/server/breadcrumbs_test.go`

**Interfaces:**
- Consumes: none (pure helpers)
- Produces:
  - `type BreadcrumbItem struct { Label string; Href string }`
  - `type BreadcrumbsView struct { Items []BreadcrumbItem }`
  - `func buildStreamBreadcrumbs(workflowKey, workflowName string) BreadcrumbsView`
  - `func buildProcessBreadcrumbs(workflowKey, workflowName, instanceName, processID string) BreadcrumbsView`
  - `func buildOrgAdminBreadcrumbs(activePanel string) BreadcrumbsView`
  - `func buildPlatformAdminBreadcrumbs() BreadcrumbsView`

- [ ] **Step 1: Write the failing tests**

Create `server/cmd/server/breadcrumbs_test.go`:

```go
package main

import (
	"testing"
)

func TestBuildStreamBreadcrumbs(t *testing.T) {
	got := buildStreamBreadcrumbs("wf-a", "Alpha Stream")
	if len(got.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(got.Items))
	}
	if got.Items[0].Label != "Streams" || got.Items[0].Href != "/" {
		t.Fatalf("root = %+v", got.Items[0])
	}
	if got.Items[1].Label != "Alpha Stream" || got.Items[1].Href != "" {
		t.Fatalf("current = %+v", got.Items[1])
	}
}

func TestBuildStreamBreadcrumbsFallsBackToKey(t *testing.T) {
	got := buildStreamBreadcrumbs("wf-a", "  ")
	if got.Items[1].Label != "wf-a" {
		t.Fatalf("label = %q, want workflow key", got.Items[1].Label)
	}
}

func TestBuildProcessBreadcrumbsUsesInstanceName(t *testing.T) {
	got := buildProcessBreadcrumbs("wf-a", "Alpha Stream", "Batch 1", "abc123")
	if len(got.Items) != 3 {
		t.Fatalf("len(Items) = %d, want 3", len(got.Items))
	}
	if got.Items[1].Label != "Alpha Stream" || got.Items[1].Href != "/w/wf-a/" {
		t.Fatalf("stream crumb = %+v", got.Items[1])
	}
	if got.Items[2].Label != "Instance: Batch 1" || got.Items[2].Href != "" {
		t.Fatalf("instance crumb = %+v", got.Items[2])
	}
}

func TestBuildProcessBreadcrumbsFallsBackToProcessID(t *testing.T) {
	got := buildProcessBreadcrumbs("wf-a", "Alpha Stream", "", "abc123")
	if got.Items[2].Label != "Instance: abc123" {
		t.Fatalf("label = %q", got.Items[2].Label)
	}
}

func TestBuildOrgAdminBreadcrumbs(t *testing.T) {
	got := buildOrgAdminBreadcrumbs("members")
	if len(got.Items) != 3 {
		t.Fatalf("len(Items) = %d, want 3", len(got.Items))
	}
	if got.Items[1].Label != "Organization admin" || got.Items[1].Href != "/org-admin/profile" {
		t.Fatalf("middle = %+v", got.Items[1])
	}
	if got.Items[2].Label != "Members" || got.Items[2].Href != "" {
		t.Fatalf("section = %+v", got.Items[2])
	}
}

func TestBuildOrgAdminBreadcrumbsSections(t *testing.T) {
	cases := map[string]string{
		"profile": "Profile",
		"roles":   "Roles",
		"members": "Members",
		"":        "Profile",
		"other":   "Profile",
	}
	for panel, want := range cases {
		got := buildOrgAdminBreadcrumbs(panel)
		if got.Items[2].Label != want {
			t.Fatalf("panel %q: label = %q, want %q", panel, got.Items[2].Label, want)
		}
	}
}

func TestBuildPlatformAdminBreadcrumbs(t *testing.T) {
	got := buildPlatformAdminBreadcrumbs()
	if len(got.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(got.Items))
	}
	if got.Items[1].Label != "Platform admin" || got.Items[1].Href != "" {
		t.Fatalf("current = %+v", got.Items[1])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd server && go test ./cmd/server/ -run 'TestBuild(Stream|Process|OrgAdmin|PlatformAdmin)Breadcrumbs' -count=1
```

Expected: FAIL — undefined builders / types.

- [ ] **Step 3: Add types and builders**

Append to `server/cmd/server/components.go`:

```go
// BreadcrumbItem is one crumb in templates/components/breadcrumbs.html.
type BreadcrumbItem struct {
	Label string
	Href  string // empty => current page (non-link)
}

// BreadcrumbsView is the view model for templates/components/breadcrumbs.html.
type BreadcrumbsView struct {
	Items []BreadcrumbItem
}
```

Create `server/cmd/server/breadcrumbs.go`:

```go
package main

import "strings"

func buildStreamBreadcrumbs(workflowKey, workflowName string) BreadcrumbsView {
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: streamCrumbLabel(workflowName, workflowKey), Href: ""},
	}}
}

func buildProcessBreadcrumbs(workflowKey, workflowName, instanceName, processID string) BreadcrumbsView {
	key := strings.TrimSpace(workflowKey)
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: streamCrumbLabel(workflowName, key), Href: "/w/" + key + "/"},
		{Label: processInstanceCrumbLabel(instanceName, processID), Href: ""},
	}}
}

func buildOrgAdminBreadcrumbs(activePanel string) BreadcrumbsView {
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: "Organization admin", Href: "/org-admin/profile"},
		{Label: orgAdminSectionLabel(activePanel), Href: ""},
	}}
}

func buildPlatformAdminBreadcrumbs() BreadcrumbsView {
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: "Platform admin", Href: ""},
	}}
}

func streamCrumbLabel(workflowName, workflowKey string) string {
	if name := strings.TrimSpace(workflowName); name != "" {
		return name
	}
	return strings.TrimSpace(workflowKey)
}

func processInstanceCrumbLabel(instanceName, processID string) string {
	if name := strings.TrimSpace(instanceName); name != "" {
		return "Instance: " + name
	}
	return "Instance: " + strings.TrimSpace(processID)
}

func orgAdminSectionLabel(activePanel string) string {
	switch strings.TrimSpace(activePanel) {
	case "roles":
		return "Roles"
	case "members":
		return "Members"
	default:
		return "Profile"
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
cd server && go test ./cmd/server/ -run 'TestBuild(Stream|Process|OrgAdmin|PlatformAdmin)Breadcrumbs' -count=1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/components.go server/cmd/server/breadcrumbs.go server/cmd/server/breadcrumbs_test.go
git commit -m "$(cat <<'EOF'
feat(ui): add breadcrumbs view builders

EOF
)"
```

---

### Task 2: Breadcrumbs template and CSS

**Files:**
- Create: `server/templates/components/breadcrumbs.html`
- Create: `web/src/styles/components/breadcrumbs.css`
- Modify: `web/src/styles/components.css` (add import after `page-header.css`)
- Modify: `web/src/styles/components/page-header.css` (remove `.page-header-back`; update header comment)
- Create: `server/cmd/server/breadcrumbs_template_test.go`
- Delete: `server/cmd/server/page_header_test.go` (after new tests exist)
- Modify: `server/cmd/server/templates_test.go` (swap expected define names)

**Interfaces:**
- Consumes: `BreadcrumbsView` / `BreadcrumbItem` from Task 1
- Produces: template define `breadcrumbs`; CSS classes `.breadcrumbs`, `.breadcrumbs-list`, `.breadcrumbs-item`, `.breadcrumbs-link`, `.breadcrumbs-current`

- [ ] **Step 1: Write the failing template tests**

Create `server/cmd/server/breadcrumbs_template_test.go`:

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestBreadcrumbsTemplateRendersLinksAndCurrent(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: "Alpha", Href: "/w/alpha/"},
		{Label: "Instance: Batch 1", Href: ""},
	}}
	if err := tmpl.ExecuteTemplate(&out, "breadcrumbs", view); err != nil {
		t.Fatalf("render breadcrumbs: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`aria-label="Breadcrumb"`,
		`class="breadcrumbs"`,
		`href="/"`,
		">Streams<",
		`href="/w/alpha/"`,
		">Alpha<",
		`aria-current="page"`,
		"Instance: Batch 1",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in breadcrumbs, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, `href="">`) {
		t.Fatalf("current crumb must not be an empty-href link, got:\n%s", body)
	}
}

func TestBreadcrumbsTemplateEmptyItemsRendersNothing(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "breadcrumbs", BreadcrumbsView{}); err != nil {
		t.Fatalf("render breadcrumbs: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Fatalf("expected empty output, got: %q", out.String())
	}
}
```

Update `server/cmd/server/templates_test.go` expected names: replace `"page_header_back"` with `"breadcrumbs"`.

- [ ] **Step 2: Run template tests to verify they fail**

Run:

```bash
cd server && go test ./cmd/server/ -run 'TestBreadcrumbsTemplate|TestParseTemplates' -count=1
```

Expected: FAIL — missing `breadcrumbs` define / still looking for `page_header_back`.

- [ ] **Step 3: Add template markup**

Create `server/templates/components/breadcrumbs.html`:

```html
{{ define "breadcrumbs" }}
  {{ if .Items }}
    <nav class="breadcrumbs" aria-label="Breadcrumb">
      <ol class="breadcrumbs-list">
        {{ range .Items }}
          <li class="breadcrumbs-item">
            {{ if .Href }}
              <a class="breadcrumbs-link" href="{{ .Href }}">{{ .Label }}</a>
            {{ else }}
              <span class="breadcrumbs-current" aria-current="page">{{ .Label }}</span>
            {{ end }}
          </li>
        {{ end }}
      </ol>
    </nav>
  {{ end }}
{{ end }}
```

- [ ] **Step 4: Add CSS and update page-header**

Create `web/src/styles/components/breadcrumbs.css`:

```css
/*
 * Breadcrumbs — hierarchical page trail (full component).
 *
 *   nav.breadcrumbs
 *     ol.breadcrumbs-list
 *       li.breadcrumbs-item
 *         a.breadcrumbs-link | span.breadcrumbs-current
 */

.breadcrumbs {
  width: 100%;
}

.breadcrumbs-list {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-1) var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
  font-size: var(--text-sm);
}

.breadcrumbs-item {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
  overflow-wrap: anywhere;
}

.breadcrumbs-item:not(:last-child)::after {
  content: "/";
  color: var(--muted-foreground);
  font-weight: var(--font-normal);
}

.breadcrumbs-link {
  color: var(--muted-foreground);
  font-weight: var(--font-semibold);
  text-decoration: none;
}

.breadcrumbs-link:hover {
  text-decoration: underline;
}

.breadcrumbs-current {
  color: inherit;
  font-weight: var(--font-normal);
}
```

In `web/src/styles/components.css`, after the `page-header.css` import, add:

```css
@import url("./components/breadcrumbs.css");
```

In `web/src/styles/components/page-header.css`:
- Change markup-tree comments: replace `a.page-header-back?` with `nav.breadcrumbs?` (both heading-only and with-actions trees)
- Delete the entire `.page-header-back { … }` rule block

- [ ] **Step 5: Delete old back test; run tests and CSS lint**

Delete `server/cmd/server/page_header_test.go`.

Run:

```bash
cd server && go test ./cmd/server/ -run 'TestBreadcrumbsTemplate|TestParseTemplates|TestPageHeaderBack' -count=1
task css:lint
```

Expected: PASS for breadcrumbs/parse tests; `TestPageHeaderBack` no longer exists; css lint clean.

- [ ] **Step 6: Commit**

```bash
git add server/templates/components/breadcrumbs.html web/src/styles/components/breadcrumbs.css web/src/styles/components.css web/src/styles/components/page-header.css server/cmd/server/breadcrumbs_template_test.go server/cmd/server/templates_test.go
git rm server/cmd/server/page_header_test.go
git commit -m "$(cat <<'EOF'
feat(ui): add breadcrumbs template and styles

EOF
)"
```

---

### Task 3: Wire breadcrumbs into page views and templates

**Files:**
- Modify: `server/cmd/server/main.go`
  - `HomeView`, `ProcessPageView`, `OrgAdminView`, `PlatformAdminView` — add `Breadcrumbs BreadcrumbsView`
  - Stream dashboard builder (~`HomeView{…}` near line 4813): set `Breadcrumbs: buildStreamBreadcrumbs(workflowKey, cfg.Workflow.Name)`
  - `buildProcessPageView` (~5021): set `Breadcrumbs: buildProcessBreadcrumbs(workflowKey, pageBase.WorkflowName, instanceName, processID)` (use `cfg.Workflow.Name` if preferred over `pageBase.WorkflowName` — keep consistent with stream name source)
  - `renderOrgAdminWithErrors` (both setup and full view literals ~3755 and ~3787): set `Breadcrumbs: buildOrgAdminBreadcrumbs(activePanel)`
  - `platformAdminView` (~3279): set `Breadcrumbs: buildPlatformAdminBreadcrumbs()`
- Modify: `server/templates/pages/stream.html` — replace `{{ template "page_header_back" "/" }}` with `{{ template "breadcrumbs" .Breadcrumbs }}`
- Modify: `server/templates/pages/process.html` — replace `{{ template "page_header_back" (printf "/w/%s/" .WorkflowKey) }}` with `{{ template "breadcrumbs" .Breadcrumbs }}`
- Modify: `server/templates/pages/org_admin.html` — same replacement for back
- Modify: `server/templates/pages/platform_admin.html` — same replacement for back
- Modify: existing page template tests that render these pages if they assert on `page-header-back` / `Back` (search and update)

**Interfaces:**
- Consumes: builders from Task 1; `breadcrumbs` template from Task 2
- Produces: four page headers render breadcrumb trails from backend-assembled views

- [ ] **Step 1: Write / extend a failing page smoke test**

Add to `server/cmd/server/breadcrumbs_template_test.go` (or a new `breadcrumbs_pages_test.go`):

```go
func TestProcessPageHeaderRendersBreadcrumbs(t *testing.T) {
	tmpl := parseTestTemplates(t)
	view := ProcessPageView{
		PageBase: PageBase{
			Body:         "process_body",
			WorkflowKey:  "wf-a",
			WorkflowName: "Alpha Stream",
		},
		ProcessID:    "abc123",
		InstanceName: "Batch 1",
		Breadcrumbs:  buildProcessBreadcrumbs("wf-a", "Alpha Stream", "Batch 1", "abc123"),
	}
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "process_body", view); err != nil {
		t.Fatalf("render process_body: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`class="breadcrumbs"`,
		">Streams<",
		">Alpha Stream<",
		"Instance: Batch 1",
		`aria-current="page"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in process header, got:\n%s", want, body)
		}
	}
	if strings.Contains(body, "page-header-back") || strings.Contains(body, ">Back<") {
		t.Fatalf("expected Back link removed, got:\n%s", body)
	}
}
```

(If `process_body` needs more fields to render without panic, copy the minimal fields from `process_template_test.go` and add `Breadcrumbs`.)

- [ ] **Step 2: Run smoke test to verify it fails**

Run:

```bash
cd server && go test ./cmd/server/ -run TestProcessPageHeaderRendersBreadcrumbs -count=1
```

Expected: FAIL — missing `Breadcrumbs` field and/or still rendering Back.

- [ ] **Step 3: Wire structs, builders, and templates**

1. Add `Breadcrumbs BreadcrumbsView` to `HomeView`, `ProcessPageView`, `OrgAdminView`, `PlatformAdminView` in `main.go`.
2. Set `Breadcrumbs` in each assembly site listed in **Files**.
3. In the four page templates, replace the `page_header_back` call with:

```html
{{ template "breadcrumbs" .Breadcrumbs }}
```

4. Grep for `page_header_back` / `page-header-back` in tests and update any assertions.

- [ ] **Step 4: Run focused tests**

Run:

```bash
cd server && go test ./cmd/server/ -run 'TestProcessPageHeaderRendersBreadcrumbs|TestBreadcrumbs|TestParseTemplates|StreamPage|OrgAdmin|PlatformAdmin|ProcessTemplate' -count=1
```

Expected: PASS (adjust run pattern if some suite names differ; fix any zero-value `Breadcrumbs` gaps that leave empty headers on tested pages).

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/main.go server/templates/pages/stream.html server/templates/pages/process.html server/templates/pages/org_admin.html server/templates/pages/platform_admin.html server/cmd/server/breadcrumbs_template_test.go server/cmd/server/breadcrumbs_pages_test.go
git commit -m "$(cat <<'EOF'
feat(ui): wire breadcrumbs into stream and admin pages

EOF
)"
```

(Only add files that exist / changed.)

---

### Task 4: Remove Back partial, icon-back, and update docs

**Files:**
- Delete: `server/templates/components/page_header.html`
- Modify: `server/templates/icons.html` — remove the entire `{{ define "icon-back" }}` … `{{ end }}` block
- Modify: `docs/css.md` — see Step 3 for exact edits
- Modify: `.agents/skills/attesta-ui-components/SKILL.md` — see Step 3
- Modify: `AGENTS.md` — replace `page_header_back` mention with breadcrumbs in the component-tiers bullet

**Interfaces:**
- Consumes: breadcrumbs component fully wired (Task 3)
- Produces: no remaining `page_header_back` / `icon-back` / `.page-header-back` references in code or docs

- [ ] **Step 1: Grep for remaining references**

Run:

```bash
rg -n 'page_header_back|page-header-back|icon-back' .
```

Expected before cleanup: hits in `page_header.html`, `icons.html`, docs/skill/AGENTS (and possibly stale comments). After Step 3: no hits except historical plan/spec files under `docs/superpowers/` (those may still mention the old name — that is fine).

- [ ] **Step 2: Delete template artifacts**

- Delete `server/templates/components/page_header.html`
- Remove `icon-back` define from `server/templates/icons.html`

- [ ] **Step 3: Update docs**

In `docs/css.md`:
- Components barrel list: add `breadcrumbs`
- Component modules table: add row for `components/breadcrumbs.css` → `.breadcrumbs*` / `components/breadcrumbs.html`
- Page-header row: remove micro-partial / `.page-header-back` wording; say optional `nav.breadcrumbs` from full `breadcrumbs` component
- CSS-only exceptions list: remove `page_header_back`; keep `status_tag`
- Intended trees: `nav.breadcrumbs?` instead of `a.page-header-back?`
- Template↔CSS index and class cheat sheet: drop back; document breadcrumbs

In `.agents/skills/attesta-ui-components/SKILL.md`:
- Remove `page_header_back` micro-partial exception from CSS-only / naming / Go examples
- Add `breadcrumbs` under Reference implementations (full component) with paths to html/css/`BreadcrumbsView`
- Page-header bullet: optional trail is `{{ template "breadcrumbs" .Breadcrumbs }}`, not Back

In `AGENTS.md` component-tiers bullet: cite breadcrumbs as full component example; page-header remains CSS-only without back micro-partial.

- [ ] **Step 4: Verify**

Run:

```bash
rg -n 'page_header_back|page-header-back|icon-back' --glob '!docs/superpowers/**' .
cd server && go test ./cmd/server/ -run 'TestBreadcrumbs|TestParseTemplates|TestProcessPageHeaderRendersBreadcrumbs' -count=1
task css:lint
```

Expected: no remaining production references; tests PASS; lint clean.

- [ ] **Step 5: Commit**

```bash
git add docs/css.md .agents/skills/attesta-ui-components/SKILL.md AGENTS.md server/templates/icons.html
git rm server/templates/components/page_header.html
git commit -m "$(cat <<'EOF'
chore(ui): remove page-header Back and document breadcrumbs

EOF
)"
```

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Full component (Go + html + css) | 1–2 |
| Empty Href = current + `aria-current` | 2 |
| Empty Items → render nothing | 2 |
| Streams root `/` | 1 |
| Stream / process / org / platform trails | 1, 3 |
| `Instance: name\|id` | 1 |
| Org section from `ActivePanel` | 1, 3 |
| Keep `h1`; no DPP/home/auth | 3 (non-goals honored) |
| Remove `page_header_back` + `icon-back` | 2, 4 |
| Docs/skill/AGENTS | 4 |
| Template + builder tests + css lint | 1–4 |
