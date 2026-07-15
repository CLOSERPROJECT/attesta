# CSS-Only Components (Panel) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce **CSS-only components** as a documented pattern (CSS module + inline HTML markup, no Go template partial) and consolidate all `.panel*` layout rules into `panel.css` with a markup-tree comment as the contract.

**Architecture:** Panel section headers stay as inline HTML in page templates (already true after reverting `panel_head`). Styles move from `shared.css`, `org-admin.css`, and `stream.css` into a single `web/src/styles/components/panel.css`. `docs/css.md` documents the tier; `.agents/skills/attesta-ui-components/SKILL.md` points agents at CSS-only modules instead of extracting template partials for stable markup trees.

**Tech Stack:** Go `html/template`, Vite/PostCSS CSS modules, stylelint, existing `task css:lint` and `go test ./cmd/server/`.

## Global Constraints

- Do **not** reintroduce `panel_head.html`, the `include` template func, or template slot defines.
- A selector lives in **exactly one** CSS module (see `docs/css.md` placement rule).
- Use design tokens (`var(--space-*)`, `var(--muted-foreground)`, …) — no new hex/rgb literals.
- Responsive rules use `@media (--md-down)` / `@media (--sm-down)` aliases from `breakpoints.css` — no literal `px` widths in `@media`.
- Markup tree comment at top of `panel.css` is the **canonical contract**; `docs/css.md` holds the index only (no duplicated full trees).
- Run `task css:lint` and `cd server && go test ./cmd/server/ -count=1` before claiming done.

## File map (this plan)

| File | Responsibility |
|------|----------------|
| `web/src/styles/components/panel.css` | **Create** — all `.panel*` rules + markup-tree header comment |
| `web/src/styles/components.css` | Import `panel.css` |
| `web/src/styles/components/shared.css` | **Remove** `.panel`, `.panel-heading`, `.panel-block*`, responsive `.panel` padding |
| `web/src/styles/components/org-admin.css` | **Remove** `.panel-head-actions*` rules and responsive block at file bottom |
| `web/src/styles/components/stream.css` | **Remove** `.panel-actions` (moves to `panel.css`) |
| `docs/css.md` | New **CSS-only components** section + index row for `panel.css` |
| `.agents/skills/attesta-ui-components/SKILL.md` | Document CSS-only tier; panel example |
| `AGENTS.md` | One-line pointer to CSS-only components in templates section |
| `server/cmd/server/panel_markup_test.go` | **Create** — regression test for panel markup on `process_downloads` |

**Templates:** No changes expected — call sites already use inline markup after revert:

- `server/templates/pages/process.html` — `process_downloads`, `process_dpp`
- `server/templates/pages/stream.html` — stream instances header
- `server/templates/pages/dpp.html` — overview section
- `server/templates/pages/platform_admin.html` — organizations section
- `server/templates/pages/org_admin.html` — roles/users sections

**Leave unchanged:** `.panel.login` override in `web/src/styles/layout/chrome.css` (login shell, not the panel component family).

---

### Task 1: Document CSS-only components in `docs/css.md`

**Files:**
- Modify: `docs/css.md` (after line 50, inside **Component modules**; also update **Adding new UI** and **Common component classes**)

**Interfaces:**
- Produces: documented tier name **CSS-only components** and index row for `panel.css`

- [ ] **Step 1: Add CSS-only components section**

Insert after the **Component modules (`components/`)** table (before "Other partials…"):

```markdown
### CSS-only components

Reused **markup patterns** backed by namespaced CSS, with **no** Go template partial in `server/templates/components/`. Templates inline the HTML; the CSS file header comment is the markup contract.

**Use when:**

- The pattern is reused on 2+ pages
- Selectors form a coherent family (3+ related rules)
- Data varies only in text/attributes at the call site (no mode dispatch, no HTMX partial target)

**Do not use when:**

- Go assembles a stable field set → full template component + view struct (e.g. `page_header`)
- The partial is an HTMX/SSE swap target → extract to `components/{name}.html`

**Adding one:**

1. Create `web/src/styles/components/{name}.css` with a markup-tree comment at the top.
2. Import it from `web/src/styles/components.css`.
3. Add a row to the table below.
4. Do **not** create a matching `templates/components/{name}.html` unless it graduates.

| Module | Primary classes | Markup contract |
|--------|-----------------|-----------------|
| `panel.css` | `.panel`, `.panel-heading`, `.panel-head-actions`, `.panel-actions`, `.panel-block` | See file header in `web/src/styles/components/panel.css` |
```

- [ ] **Step 2: Add `panel.css` row to Component modules table**

In the existing **Component modules** table, add:

```markdown
| `components/panel.css` | `.panel`, `.panel-heading`, `.panel-head-actions`, `.panel-block` | Inline markup per `panel.css` header (CSS-only) |
```

- [ ] **Step 3: Update Template ↔ CSS index**

Add row:

```markdown
| Inline panel sections (process, stream, dpp, org_admin, platform_admin) | `components/panel.css` | `components/shared.css` (buttons, `.muted`) |
```

Remove any stale `panel_head` references if present (none after revert).

- [ ] **Step 4: Update Common component classes**

Replace the `.panel`, `.stack` bullet under **Common component classes** with:

```markdown
| `.panel`, `.panel-heading`, `.panel-head-actions`, `.panel-block` | Card sections — see `panel.css` header for markup tree (CSS-only component) |
| `.stack` | Vertical rhythm |
```

- [ ] **Step 5: Update Adding new UI**

Change step 1 to:

```markdown
1. Check for an existing component class, CSS-only module (`docs/css.md` → CSS-only components), or utility.
```

- [ ] **Step 6: Commit**

```bash
git add docs/css.md
git commit -m "docs: add CSS-only components tier and panel index"
```

---

### Task 2: Create `panel.css` and wire the barrel

**Files:**
- Create: `web/src/styles/components/panel.css`
- Modify: `web/src/styles/components.css`
- Modify: `web/src/styles/components/shared.css`
- Modify: `web/src/styles/components/org-admin.css`
- Modify: `web/src/styles/components/stream.css`

**Interfaces:**
- Consumes: CSS-only section from Task 1 (for cross-reference only)
- Produces: `panel.css` imported before `shared.css` in barrel; no duplicate `.panel*` selectors elsewhere

- [ ] **Step 1: Create `web/src/styles/components/panel.css`**

```css
/*
 * Panel — card container and section headers (CSS-only component).
 *
 * With actions (button, form, or action group):
 *   section.panel
 *     div.panel-head-actions
 *       div.panel-heading
 *         h2
 *         p?                         (description)
 *       button | form | div.panel-actions
 *         button…
 *     …content…
 *
 * Heading only (no actions):
 *   section.panel | div
 *     div.panel-heading
 *       h2
 *       p?
 *     …content…
 *
 * Inner block:
 *   div.panel-block[.panel-block-spaced]
 */

.panel {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: var(--space-5);
  box-shadow: 0 10px 26px var(--shadow);
  min-width: 0;
}

.panel-heading {
  margin-bottom: var(--space-6);

  h1,
  h2,
  h3 {
    margin: 0;
  }

  p {
    color: var(--muted-foreground);
    margin: 3px 0 0;
  }
}

.panel-head-actions {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
  margin-bottom: var(--space-5);

  button {
    flex-shrink: 0;
  }

  form {
    flex-shrink: 0;
  }
}

.panel-head-actions .panel-heading {
  margin-bottom: 0;
}

.panel-actions {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.panel-block {
  margin-top: var(--space-5);
}

.panel-block-spaced {
  margin-bottom: var(--space-3);
}

@media (--sm-down) {
  .panel {
    padding: var(--space-4);
  }
}

@media (--md-down) {
  .panel-head-actions {
    flex-direction: column;

    button {
      width: 100%;
    }

    form {
      width: 100%;
    }

    .panel-actions {
      width: 100%;
    }
  }
}
```

- [ ] **Step 2: Import in `web/src/styles/components.css`**

Add after the page-header import (panel is a foundational primitive):

```css
@import url("./components/panel.css");
```

- [ ] **Step 3: Remove panel rules from `shared.css`**

Delete these blocks from `web/src/styles/components/shared.css`:

- `.panel { … }` (lines ~49–56)
- `.panel-heading { … }` (lines ~58–71)
- `.panel-block { … }` and `.panel-block-spaced { … }` (lines ~473–479)
- `@media (--sm-down) { .panel { padding: … } }` (lines ~512–516)

Do **not** remove `.muted`, button styles, or other unrelated rules.

- [ ] **Step 4: Remove panel-head-actions from `org-admin.css`**

Delete:

- `.panel-head-actions { … }` block (~lines 184–198)
- `.panel-head-actions .panel-heading { … }` (~lines 200–202)
- The entire `@media (--md-down) { .panel-head-actions { … } }` block at file bottom (~lines 471–487)

- [ ] **Step 5: Remove `.panel-actions` from `stream.css`**

Delete the `.panel-actions { … }` block (~lines 141–147).

- [ ] **Step 6: Run CSS lint**

Run: `task css:lint`

Expected: PASS (no duplicate-selector errors; no literal breakpoint px)

- [ ] **Step 7: Rebuild CSS bundle (optional but recommended for local visual check)**

Run: `cd web && npm run build`

Expected: `web/dist/assets/main.css` contains `.panel-head-actions`

- [ ] **Step 8: Commit**

```bash
git add web/src/styles/components/panel.css web/src/styles/components.css \
  web/src/styles/components/shared.css web/src/styles/components/org-admin.css \
  web/src/styles/components/stream.css
git commit -m "refactor: extract panel CSS-only component module"
```

---

### Task 3: Update agent docs (skill + AGENTS.md)

**Files:**
- Modify: `.agents/skills/attesta-ui-components/SKILL.md`
- Modify: `AGENTS.md`

**Interfaces:**
- Consumes: `docs/css.md` CSS-only components section (Task 1)
- Produces: agent-facing rules that defer panel headers to `panel.css`, not template extraction

- [ ] **Step 1: Update skill — add CSS-only tier**

In `.agents/skills/attesta-ui-components/SKILL.md`, replace the **Cluster instead** bullet list under "When to extract a component" with:

```markdown
**Three tiers:**

| Tier | Template | CSS | Go view struct |
|------|----------|-----|----------------|
| **Full component** | `components/{name}.html` | `components/{name}.css` when ~10+ namespaced rules | Dedicated type in `components.go` |
| **CSS-only component** | **None** — inline HTML in page templates | `components/{name}.css` with markup-tree header comment | **None** |
| **Cluster / primitive** | inline | `shared.css`, `forms.css`, … | none |

**Full component** — all apply: reused 2+ pages or HTMX/SSE target; namespaced CSS; stable field set assembled in Go.

**CSS-only component** — reused markup tree + related selectors; no mode dispatch; see `docs/css.md` → CSS-only components. Example: panel section headers → `panel.css` (read file header for markup).

**Cluster instead:**

- Single-class primitives (`.muted`, `.pill`) → `shared.css`
- Domain groups (forms, org-admin widgets) → existing cluster files
- One-off markup with no dedicated styles → inline in page template
```

- [ ] **Step 2: Add reference under Reference implementations**

Add after the `page_header` block:

```markdown
### CSS-only components

- `panel` — card container and section headers (`web/src/styles/components/panel.css`); inline markup in `process.html`, `stream.html`, `dpp.html`, `org_admin.html`, `platform_admin.html`
```

- [ ] **Step 3: Update AGENTS.md**

In the **Templates and static assets** bullet about component tiers, replace any `panel_head` / template-slot wording (if present) with:

```markdown
- **CSS-only components** (see `docs/css.md`): reused markup patterns with a dedicated CSS module and inline HTML — no template partial. Example: panel section headers (`panel.css`; markup contract in file header).
```

- [ ] **Step 4: Commit**

```bash
git add .agents/skills/attesta-ui-components/SKILL.md AGENTS.md
git commit -m "docs: point agents at CSS-only panel component"
```

---

### Task 4: Regression test for panel markup convention

**Files:**
- Create: `server/cmd/server/panel_markup_test.go`

**Interfaces:**
- Consumes: `parseTestTemplates(t)` from `server/cmd/server/templates.go`
- Produces: `TestProcessDownloadsPanelMarkup` guarding the CSS-only markup tree on a real partial

- [ ] **Step 1: Write the test**

Create `server/cmd/server/panel_markup_test.go`:

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestProcessDownloadsPanelMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := ProcessPageView{
		PageBase:  PageBase{WorkflowPath: "/w/demo"},
		ProcessID: "proc-1",
	}
	if err := tmpl.ExecuteTemplate(&out, "process_downloads", view); err != nil {
		t.Fatalf("render process_downloads: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`id="process-downloads"`,
		`class="panel"`,
		`class="panel-head-actions"`,
		`class="panel-heading"`,
		"<h2>Downloads</h2>",
		"Export attachments and notarized data for this stream.",
		`class="secondary js-download-link"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in process_downloads markup, got:\n%s", want, body)
		}
	}

	// Actions sit as siblings of .panel-heading inside .panel-head-actions (not a separate template partial).
	headIdx := strings.Index(body, `class="panel-head-actions"`)
	headingIdx := strings.Index(body, `class="panel-heading"`)
	btnIdx := strings.Index(body, `class="secondary js-download-link"`)
	if headIdx == -1 || headingIdx == -1 || btnIdx == -1 {
		t.Fatal("expected panel-head-actions, panel-heading, and action button")
	}
	if !(headIdx < headingIdx && headingIdx < btnIdx) {
		t.Fatalf("expected panel-heading before action button inside panel-head-actions block")
	}
}
```

- [ ] **Step 2: Run test**

Run: `cd server && go test ./cmd/server/ -run TestProcessDownloadsPanelMarkup -count=1 -v`

Expected: PASS (templates already use correct inline markup after revert)

- [ ] **Step 3: Run full server test suite**

Run: `cd server && go test ./cmd/server/ -count=1`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add server/cmd/server/panel_markup_test.go
git commit -m "test: guard process_downloads panel CSS-only markup"
```

---

### Task 5: Final verification

**Files:** none (verification only)

- [ ] **Step 1: CSS lint**

Run: `task css:lint`

Expected: PASS

- [ ] **Step 2: Confirm no duplicate selectors**

Run: `rg '\.panel-head-actions|\.panel-heading|^\.panel ' web/src/styles/components/ --glob '*.css'`

Expected: matches only in `web/src/styles/components/panel.css` (plus `.panel.login` in `layout/chrome.css`)

- [ ] **Step 3: Confirm no panel_head / include leftovers**

Run: `rg 'panel_head|"include"' server/ .agents/ docs/css.md AGENTS.md`

Expected: no matches (or only unrelated prose like "include attachments")

- [ ] **Step 4: Optional visual smoke check**

If dev stack is running, open:

- `/w/{key}/process/{done-process-id}` — Downloads panel header + button
- `/org-admin/users` — Users panel header + Add user button
- Stream dashboard — Stream instances header with two buttons in `.panel-actions`

Expected: flex layout desktop; stacked full-width actions on narrow viewport.

---

## Self-review checklist

| Requirement | Task |
|-------------|------|
| CSS-only tier documented | Task 1 |
| `panel.css` with markup header | Task 2 |
| Dedupe shared/org-admin/stream | Task 2 |
| No template partial / include | Global constraints |
| Agent skill + AGENTS.md updated | Task 3 |
| Regression test | Task 4 |
| css:lint + go test | Tasks 2, 4, 5 |

**Placeholder scan:** none — all code blocks are complete.

**Out of scope (YAGNI):**

- Migrating other primitives (`.pill`, `.stack`) to CSS-only modules
- Visual regression / browser tests
- Changing template markup at call sites (already correct)
