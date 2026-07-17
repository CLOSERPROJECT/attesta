# Sidebar nav CSS component + panel sticky

**Status:** approved  
**Date:** 2026-07-17

## Goal

Extract the shared left-rail section switcher used on org-admin and stream into a CSS-only component (`sidebar-nav`), and move sticky rail behavior onto an optional panel modifier (`panel-sticky`).

## Background

Org-admin and stream already share the same nav markup and styles under the misleading prefix `org-admin-sidebar-*`. Those rules live in `pages/org-admin-page.css` but are consumed by `stream.html`, which violates the page-module placement rule. Sticky positioning for the rail lives as `.org-admin-sidebar` in `layout/responsive.css`.

This is not a `.btn` variant: the control is a full-width selection tile (title + optional copy, `.is-active`), not a compact action button.

## Architecture

Two small, independent CSS-only changes:

1. **Panel modifier** — optional sticky behavior on any `.panel`.
2. **Sidebar nav component** — namespaced nav chrome, reused via inline HTML (no Go template partial, no view struct).

Call sites keep page-specific grids, profile chrome, and `data-*-nav` behavior hooks.

## 1. `panel-sticky`

**File:** `web/src/styles/components/panel.css`

Add modifier documented in the panel markup-tree header:

```css
@media (--xl-up) {
  .panel-sticky {
    position: sticky;
    top: 44px;
  }
}
```

**Remove** the `.org-admin-sidebar` sticky rule from `web/src/styles/layout/responsive.css`.

**Templates:** replace `org-admin-sidebar` with `panel-sticky` on the rail panel:

| Template | Before | After |
|----------|--------|-------|
| `stream.html` | `class="panel org-admin-sidebar"` | `class="panel panel-sticky"` |
| `org_admin.html` | `panel` + conditional `org-admin-sidebar` | `panel` + conditional `panel-sticky` |

## 2. `sidebar-nav` (CSS-only)

**Create:** `web/src/styles/components/sidebar-nav.css`  
**Import** from `web/src/styles/components.css`.

### Markup contract (file header)

```
nav.sidebar-nav
  button.sidebar-nav-link[.is-active]
    span.sidebar-nav-title
    span.sidebar-nav-copy?   (optional subtitle)
```

### Class rename map

| Old | New |
|-----|-----|
| `org-admin-sidebar-nav` | `sidebar-nav` |
| `org-admin-sidebar-link` | `sidebar-nav-link` |
| `org-admin-sidebar-link-title` | `sidebar-nav-title` |
| `org-admin-sidebar-link-copy` | `sidebar-nav-copy` |
| `org-admin-sidebar` | removed (use `panel-sticky`) |

### Styles

Move the existing nav rules from `pages/org-admin-page.css` into `sidebar-nav.css` with the new class names. Visual behavior stays the same (grid stack, bordered tiles, hover, `.is-active`).

Leave org-admin–only chrome (profile summary, logos, panel-main, form-actions) in `org-admin-page.css`.

### Templates

- `server/templates/pages/stream.html` — rename classes; title-only links (no copy).
- `server/templates/pages/org_admin.html` — rename classes; keep title + copy.

Behavior attributes (`data-home-nav`, `data-org-admin-nav`, `aria-*`) unchanged.

## Docs and tests

- `docs/css.md` — add `sidebar-nav` to component / CSS-only tables; note `.panel-sticky`; update template index rows for stream + org_admin.
- `.agents/skills/attesta-ui-components/SKILL.md` — reference `sidebar-nav` and `panel-sticky`.
- `server/cmd/server/panel_markup_test.go` — expect `panel panel-sticky` and `sidebar-nav` classes instead of `org-admin-sidebar*`.

## Out of scope

- Folding sidebar links into `button.css` / `.btn`
- Go template partial or view struct for the nav
- Changing grids (`home-workflow-grid`, `org-admin-grid`)
- Changing org profile summary / logo styles
- Changing panel-switch JS or `data-*-nav` contracts
- Other shared.css candidates (`workflow-card`, pills, pagination)

## Verification

```bash
task css:lint
cd server && go test ./cmd/server/ -run 'PanelMarkup|HomeBody' -count=1
```

Manual: org-admin section switcher and stream status rail look/behave as today; sticky rail still sticks at `--xl-up`.
