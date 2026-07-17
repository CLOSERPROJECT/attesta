# Sidebar Nav + Panel Sticky Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract the shared left-rail section switcher into CSS-only `sidebar-nav`, and move sticky rail behavior onto optional `.panel-sticky`.

**Architecture:** `.panel-sticky` is a panel modifier in `panel.css` (sticky only at `--xl-up`). Nav chrome moves from `pages/org-admin-page.css` into `components/sidebar-nav.css` with renamed classes. Templates keep inline markup; no Go template partial or view struct. Behavior hooks (`data-home-nav`, `data-org-admin-nav`) stay unchanged.

**Tech Stack:** Go `html/template`, Vite/PostCSS, stylelint via `task css:lint`, Go markup tests in `server/cmd/server/*_test.go`.

**Spec:** `docs/superpowers/specs/2026-07-17-sidebar-nav-css-design.md`

## Global Constraints

- A selector lives in **exactly one** CSS module (`docs/css.md` placement rule).
- Do **not** create `server/templates/components/sidebar_nav.html` — CSS-only tier stays inline HTML.
- Do **not** compose sidebar links with `.btn` — different job (selection tiles vs action buttons).
- Sticky breakpoint stays **`--xl-up`** with `top: 44px` (existing behavior).
- Class rename map is locked: `org-admin-sidebar-nav` → `sidebar-nav`, `org-admin-sidebar-link` → `sidebar-nav-link`, `org-admin-sidebar-link-title` → `sidebar-nav-title`, `org-admin-sidebar-link-copy` → `sidebar-nav-copy`, `org-admin-sidebar` → `panel-sticky`.
- Prefer minimal diffs; do not change grids, org profile chrome, or panel-switch JS.
- Commit after each task.

---

## File structure

| File | Responsibility |
|------|----------------|
| `web/src/styles/components/panel.css` | **Modify** — add `.panel-sticky` + document in header |
| `web/src/styles/layout/responsive.css` | **Modify** — delete `.org-admin-sidebar` sticky block |
| `web/src/styles/components/sidebar-nav.css` | **Create** — nav rules + markup-tree header |
| `web/src/styles/components.css` | Import `sidebar-nav.css` |
| `web/src/styles/pages/org-admin-page.css` | **Remove** nav rules (keep profile/panel-main chrome) |
| `server/templates/pages/stream.html` | `panel-sticky` + `sidebar-nav*` classes |
| `server/templates/pages/org_admin.html` | `panel-sticky` + `sidebar-nav*` classes |
| `server/cmd/server/panel_markup_test.go` | Assert `panel panel-sticky` + `sidebar-nav` |
| `server/cmd/server/org_admin_template_sidebar_test.go` | Assert `panel-sticky` + `sidebar-nav*` |
| `docs/css.md` | Index rows for `sidebar-nav` + `.panel-sticky` |
| `.agents/skills/attesta-ui-components/SKILL.md` | Reference both |

---

### Task 1: Add `.panel-sticky` and drop `org-admin-sidebar`

**Files:**
- Modify: `web/src/styles/components/panel.css`
- Modify: `web/src/styles/layout/responsive.css`
- Modify: `server/templates/pages/stream.html`
- Modify: `server/templates/pages/org_admin.html`
- Test: `server/cmd/server/panel_markup_test.go`
- Test: `server/cmd/server/org_admin_template_sidebar_test.go`

**Interfaces:**
- Consumes: existing `.panel` card chrome
- Produces: optional modifier class `panel-sticky` (sticky at `--xl-up`, `top: 44px`)

- [ ] **Step 1: Update failing tests for `panel-sticky`**

In `panel_markup_test.go` (`TestStreamHomeBodyPanelMarkup`), replace both occurrences of:

```go
`class="panel org-admin-sidebar"`
```

with:

```go
`class="panel panel-sticky"`
```

In `org_admin_template_sidebar_test.go`, add these markers to the `for _, marker := range []string{…}` slice:

```go
`class="panel panel-sticky"`,
`class="sidebar-nav"`, // will still fail until Task 2 — omit this line in Task 1
```

For Task 1 only, add:

```go
`class="panel panel-sticky"`,
```

Do **not** assert `sidebar-nav` yet (still `org-admin-sidebar-nav` until Task 2).

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd server && go test ./cmd/server/ -run 'TestStreamHomeBodyPanelMarkup|TestOrgAdminTemplateRendersSidebarPanels' -count=1
```

Expected: FAIL — templates still emit `org-admin-sidebar`.

- [ ] **Step 3: Add `.panel-sticky` to `panel.css`**

Append to the markup-tree header comment (after the Inner block section):

```
 * Modifiers:
 *   section.panel.panel-sticky   (sticky rail at --xl-up)
```

Append before the existing `@media (--sm-down)` block:

```css
@media (--xl-up) {
  .panel-sticky {
    position: sticky;
    top: 44px;
  }
}
```

- [ ] **Step 4: Remove sticky from `responsive.css`**

Delete this entire block from `web/src/styles/layout/responsive.css`:

```css
@media (--xl-up) {
  .org-admin-sidebar {
    position: sticky;
    top: 44px;
  }
}
```

Leave the `--md-down` and `--xl-down` blocks intact.

- [ ] **Step 5: Update templates**

In `server/templates/pages/stream.html`, change:

```html
<section class="panel org-admin-sidebar">
```

to:

```html
<section class="panel panel-sticky">
```

In `server/templates/pages/org_admin.html`, change the conditional class from `org-admin-sidebar` to `panel-sticky`:

```html
<section
  class="panel{{ if not .NeedsOrganizationSetup }}
    panel-sticky
  {{ end }}"
>
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd server && go test ./cmd/server/ -run 'TestStreamHomeBodyPanelMarkup|TestOrgAdminTemplateRendersSidebarPanels' -count=1
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add web/src/styles/components/panel.css \
  web/src/styles/layout/responsive.css \
  server/templates/pages/stream.html \
  server/templates/pages/org_admin.html \
  server/cmd/server/panel_markup_test.go \
  server/cmd/server/org_admin_template_sidebar_test.go
git commit -m "$(cat <<'EOF'
refactor(ui): move rail sticky onto panel-sticky modifier

Replace org-admin-sidebar sticky layout class with an optional
panel modifier shared by stream and org-admin rails.
EOF
)"
```

---

### Task 2: Extract `sidebar-nav` CSS-only component

**Files:**
- Create: `web/src/styles/components/sidebar-nav.css`
- Modify: `web/src/styles/components.css`
- Modify: `web/src/styles/pages/org-admin-page.css`
- Modify: `server/templates/pages/stream.html`
- Modify: `server/templates/pages/org_admin.html`
- Test: `server/cmd/server/panel_markup_test.go`
- Test: `server/cmd/server/org_admin_template_sidebar_test.go`

**Interfaces:**
- Consumes: none (standalone CSS-only)
- Produces classes: `sidebar-nav`, `sidebar-nav-link`, `sidebar-nav-title`, `sidebar-nav-copy`, `.is-active` on link

- [ ] **Step 1: Update failing tests for `sidebar-nav` classes**

In `TestStreamHomeBodyPanelMarkup` (`panel_markup_test.go`), add to the `want` slice (alongside `panel panel-sticky`):

```go
`class="sidebar-nav"`,
`class="sidebar-nav-link is-active"`,
`class="sidebar-nav-title"`,
```

Ensure no assertion still expects `org-admin-sidebar-nav` / `org-admin-sidebar-link`.

In `TestOrgAdminTemplateRendersSidebarPanels`, add markers:

```go
`class="sidebar-nav"`,
`class="sidebar-nav-link"`,
`class="sidebar-nav-title"`,
`class="sidebar-nav-copy"`,
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd server && go test ./cmd/server/ -run 'TestStreamHomeBodyPanelMarkup|TestOrgAdminTemplateRendersSidebarPanels' -count=1
```

Expected: FAIL — templates still use `org-admin-sidebar-*`.

- [ ] **Step 3: Create `sidebar-nav.css`**

Create `web/src/styles/components/sidebar-nav.css` with exactly:

```css
/*
 * Sidebar nav — section switcher tiles (CSS-only component).
 *
 * nav.sidebar-nav
 *   button.sidebar-nav-link[.is-active]
 *     span.sidebar-nav-title
 *     span.sidebar-nav-copy?   (optional subtitle)
 */

.sidebar-nav {
  display: grid;
  gap: var(--space-2);
  margin-top: var(--space-5);
}

.sidebar-nav-link {
  width: 100%;
  display: grid;
  gap: 2px;
  padding: var(--space-3) var(--space-4);
  text-align: left;
  border-radius: 4px;
  border: 1px solid var(--border);
  background: var(--muted);
  color: inherit;
  font: inherit;
  cursor: pointer;
  transition:
    border-color 0.18s ease,
    background-color 0.18s ease,
    color 0.18s ease,
    transform 0.18s ease;
}

.sidebar-nav-link:hover {
  border-color: var(--primary);
  transform: translateY(-1px);
}

.sidebar-nav-link.is-active {
  border-color: var(--primary);
  background: color-mix(in srgb, var(--primary) 14%, var(--muted));
}

.sidebar-nav-title {
  font-weight: 600;
}

.sidebar-nav-copy {
  color: var(--muted-foreground);
  font-size: var(--text-sm);
  line-height: var(--leading-normal);
}
```

- [ ] **Step 4: Wire the barrel and delete old page rules**

In `web/src/styles/components.css`, add after the `panel.css` import:

```css
@import url("./components/sidebar-nav.css");
```

In `web/src/styles/pages/org-admin-page.css`, delete the entire block from `.org-admin-sidebar-nav` through `.org-admin-sidebar-link-copy` (lines that currently define those five rule sets). Keep `.org-profile-summary`, `.org-admin-panel-main`, logos, etc.

- [ ] **Step 5: Rename classes in templates**

In `server/templates/pages/stream.html` nav:

| Old | New |
|-----|-----|
| `class="org-admin-sidebar-nav"` | `class="sidebar-nav"` |
| `class="org-admin-sidebar-link is-active"` | `class="sidebar-nav-link is-active"` |
| `class="org-admin-sidebar-link"` | `class="sidebar-nav-link"` |
| `class="org-admin-sidebar-link-title"` | `class="sidebar-nav-title"` |

Leave `data-home-nav`, `aria-*`, and titles unchanged.

In `server/templates/pages/org_admin.html` nav:

| Old | New |
|-----|-----|
| `class="org-admin-sidebar-nav"` | `class="sidebar-nav"` |
| `class="org-admin-sidebar-link"` | `class="sidebar-nav-link"` |
| `class="org-admin-sidebar-link-title"` | `class="sidebar-nav-title"` |
| `class="org-admin-sidebar-link-copy"` | `class="sidebar-nav-copy"` |

Leave `data-org-admin-nav` and copy text unchanged.

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd server && go test ./cmd/server/ -run 'TestStreamHomeBodyPanelMarkup|TestOrgAdminTemplateRendersSidebarPanels|TestOrgAdminRolesPanelMarkup' -count=1
```

Expected: PASS

Also confirm no leftover old class names:

```bash
rg 'org-admin-sidebar' server/templates web/src/styles
```

Expected: no matches.

- [ ] **Step 7: Commit**

```bash
git add web/src/styles/components/sidebar-nav.css \
  web/src/styles/components.css \
  web/src/styles/pages/org-admin-page.css \
  server/templates/pages/stream.html \
  server/templates/pages/org_admin.html \
  server/cmd/server/panel_markup_test.go \
  server/cmd/server/org_admin_template_sidebar_test.go
git commit -m "$(cat <<'EOF'
refactor(ui): extract sidebar-nav CSS-only component

Move shared section-switcher tiles out of org-admin page CSS and
rename classes for reuse on stream and org-admin rails.
EOF
)"
```

---

### Task 3: Docs and skill index

**Files:**
- Modify: `docs/css.md`
- Modify: `.agents/skills/attesta-ui-components/SKILL.md`

**Interfaces:**
- Consumes: classes from Tasks 1–2
- Produces: documented index entries only

- [ ] **Step 1: Update `docs/css.md`**

In **Component modules** table, update the panel row to include `.panel-sticky`, and add:

```markdown
| `components/sidebar-nav.css` | `.sidebar-nav`, `.sidebar-nav-link`, `.sidebar-nav-title`, `.sidebar-nav-copy` | Inline markup per `sidebar-nav.css` header (CSS-only) |
```

In **CSS-only components** table:

- Update `panel.css` primary classes to include `.panel-sticky`
- Add:

```markdown
| `sidebar-nav.css` | `.sidebar-nav`, `.sidebar-nav-link`, `.sidebar-nav-title`, `.sidebar-nav-copy` | See file header in `web/src/styles/components/sidebar-nav.css` |
```

In **Template ↔ CSS index**:

- Inline panel row: note `.panel-sticky` optional modifier
- `pages/stream.html` Also uses: add `components/sidebar-nav.css`, `components/panel.css` (sticky)
- `pages/org_admin.html` Also uses: add `components/sidebar-nav.css`, `components/panel.css`

In **Common component classes** table, add:

```markdown
| `.panel-sticky` | Optional sticky rail modifier on `.panel` (active at `--xl-up`) |
| `.sidebar-nav`, `.sidebar-nav-link`, `.sidebar-nav-title`, `.sidebar-nav-copy` | Section switcher tiles — see `sidebar-nav.css` header |
```

- [ ] **Step 2: Update attesta-ui-components skill**

In the CSS-only components list, update panel bullet to mention `.panel-sticky`, and add:

```markdown
- `sidebar-nav` — section switcher tiles (`.sidebar-nav`, `.sidebar-nav-link`, … in `web/src/styles/components/sidebar-nav.css`); inline markup in `stream.html`, `org_admin.html`
```

- [ ] **Step 3: Lint and full verification**

```bash
task css:lint
cd server && go test ./cmd/server/ -run 'PanelMarkup|OrgAdminTemplateRendersSidebar' -count=1
```

Expected: css:lint exit 0; Go tests PASS.

- [ ] **Step 4: Commit**

```bash
git add docs/css.md .agents/skills/attesta-ui-components/SKILL.md
git commit -m "$(cat <<'EOF'
docs: index sidebar-nav and panel-sticky CSS components

Update css.md and the UI components skill for the new CSS-only
sidebar-nav module and panel-sticky modifier.
EOF
)"
```

---

## Self-review checklist (plan author)

1. **Spec coverage:** panel-sticky ✓ (Task 1); sidebar-nav module + rename ✓ (Task 2); docs/skill/tests ✓ (Tasks 1–3); out-of-scope items not planned ✓
2. **Placeholders:** none
3. **Name consistency:** `sidebar-nav*` and `panel-sticky` match the approved spec rename map throughout
