---
name: attesta-ui-components
description: >-
  Attesta server-rendered UI component conventions: when to extract a partial,
  where templates/CSS/Go structs live, and naming rules. Use when adding or
  refactoring templates, page headers, partials, view structs, or CSS in
  server/templates/ and web/src/styles/.
---

# Attesta UI components

## When to extract a component

**Three tiers:**

| Tier | Template | CSS | Go view struct |
|------|----------|-----|----------------|
| **Full component** | `components/{name}.html` | `components/{name}.css` when ~10+ namespaced rules | Dedicated type in `components.go` |
| **CSS-only component** | **None** (or a narrow micro-partial) — inline HTML in page templates | `components/{name}.css` with markup-tree header comment | **None** |
| **Cluster / primitive** | inline | `shared.css`, `forms.css`, … | none |

**Full component** — all apply: reused 2+ pages or HTMX/SSE target; namespaced CSS; stable field set assembled in Go.

**CSS-only component** — reused markup tree + related selectors; no mode dispatch; see `docs/css.md` → CSS-only components. Example: panel section headers → `panel.css` (read file header for markup). Page headers → `page-header.css` (inline in pages; optional trail via full `breadcrumbs` component).

**Cluster instead:**

- Single-class primitives (`.muted`, `.pill`) → `shared.css`
- Domain groups (forms, org-admin widgets) → existing cluster files
- One-off markup with no dedicated styles → inline in page template

Migrate **one component at a time**. Do not move every partial in one pass.

## File layout

| Layer | Shared components | Full pages | Not yet migrated |
|-------|-------------------|------------|------------------|
| Templates | `server/templates/components/{name}.html` | `server/templates/pages/{name}.html` | `server/templates/{name}.html` (root) |
| CSS | `web/src/styles/components/{name}.css` | `web/src/styles/pages/{name}.css` | cluster files at `components/` |
| Go view structs | `server/cmd/server/components.go` | handlers and page structs mostly in `main.go` | peeled view assembly: `stream_instance_detail.go`, `substep_views_builder.go`, `timeline_builder.go` |

`layout.html` stays at `server/templates/layout.html`.

Separate **reused** styles (`components/`) from **page-specific** styles (`pages/`). A selector lives in exactly one layer (see `docs/css.md`).

## Naming

- **Template define name = file stem** (basename without `.html`): `stream_card.html` → `{{ define "stream_card" }}`. Micro-partials may share a file (e.g. `status_tag.html` defines `status_tag` only).
- Page wrappers: `{page}.html` define wrapping `layout.html` (e.g. `process.html`).
- Page body blocks: `{page}_body` define (e.g. `process_body`).
- CSS class prefix: kebab-case matching the component (`page-header-*` for page header; `stream-card-*` for `stream_card`; `breadcrumbs-*` for `breadcrumbs`).

## Go conventions

- All shared component view DTOs in **one** `components.go` file.
- **No** fluent `With*` methods on simple DTOs; use struct literals at call sites.
- **No** page-specific preset functions in shared component code (e.g. no `streamPageHeader()`).
- Extract a constructor only when there is real logic (validation, computed fields); page-specific constructors belong in peeled files (`stream_instance_detail.go`, `substep_views_builder.go`, …) or future `page_*.go`, not `components.go`. Trail builders such as `buildProcessBreadcrumbs` live in `breadcrumbs.go` next to the component.
- Split a type out of `components.go` only when it grows non-trivial logic or large isolated tests (~80–100+ lines).
- **CSS-only components** (page header, panel, dialog, list-row, …) have **no** shared view struct. Put title/description/meta strings on the page view (or inline in the template) and render the markup tree in the page template.

Example (full component):

```go
Card: StreamCardView{
    Key:         cfg.Workflow.Key,
    Name:        cfg.Workflow.Name,
    Description: strings.TrimSpace(cfg.Workflow.Description),
},
```

Example (CSS-only page header + full breadcrumbs trail):

```go
// Page view fields used by inlined page-header markup (no PageHeaderView).
Title:       cfg.Workflow.Name,
Description: strings.TrimSpace(cfg.Workflow.Description),
Breadcrumbs: buildStreamBreadcrumbs(workflowKey, cfg.Workflow.Name), // {{ template "breadcrumbs" .Breadcrumbs }}
```

## Template loading

- Production: `parseTemplates()` in `server/cmd/server/templates.go`
- Tests: `parseTestTemplates(t)` in the same file

Globs: `templates/*.html`, `templates/pages/*.html`, `templates/components/*.html`.

## Reference implementations

`stream_card` — home stream picker card:

- `server/templates/components/stream_card.html`
- `web/src/styles/components/stream-card.css`
- `StreamCardView` in `server/cmd/server/components.go`
- Tests: `server/cmd/server/stream_card_test.go`

`stream_instance_card` — stream dashboard instance list row:

- `server/templates/components/stream_instance_card.html`
- `web/src/styles/components/stream-instance-card.css`
- `StreamInstanceCard` in `server/cmd/server/components.go`
- Tests: `server/cmd/server/stream_instance_card_test.go`
- Shared status tags remain in `components/stream.css` (`.status-tag*`)

`breadcrumbs` — hierarchical page trail in page headers:

- `server/templates/components/breadcrumbs.html`
- `web/src/styles/components/breadcrumbs.css`
- `BreadcrumbItem` / `BreadcrumbsView` in `server/cmd/server/components.go`
- Builders: `server/cmd/server/breadcrumbs.go` (`buildStreamBreadcrumbs`, `buildProcessBreadcrumbs`, …)
- Tests: `server/cmd/server/breadcrumbs_test.go`, `breadcrumbs_template_test.go`
- Call site: `{{ template "breadcrumbs" .Breadcrumbs }}` (`Current: true` on last crumb ⇒ `aria-current="page"`; every crumb is a link)

Also see:

- `substep_shell` — accordion chrome wrapping `substep_body` (`components/substep_shell.html`, `substep-shell.css`)
- `substep_body` — inner panel with explicit `SubstepBodyView.Mode` dispatch
- `dpp_history_step` — DPP traceability rail wrapper around `stream_timeline_step`

### CSS-only components

- `page-header` — page chrome title block (`web/src/styles/components/page-header.css`); inline markup in page templates under `server/templates/pages/`; **no** `PageHeaderView` / full `page_header` define. Heading-only: `page-header-body` directly under `section.page-header`. With actions: `page-header-head` wraps `page-header-body` + `page-header-actions`. Optional trail: `{{ template "breadcrumbs" .Breadcrumbs }}` (full component above).
- `status_tag` — stream status pill micro-partial (`server/templates/components/status_tag.html`); pipeline is status **string**; renders `<span class="status-tag status-tag-compact" data-stream-status="{{ . }}">`; colors from `data-stream-status` → `--stream-color` in `role-palette.css` / `.status-tag` in `stream.css`. Call site: `{{ template "status_tag" .Status }}`
- `panel` — card container and section headers (`web/src/styles/components/panel.css`); optional `.panel-sticky`; inline markup in `process.html`, `stream.html`, `dpp.html`, `org_admin.html`, `platform_admin.html`
- `sidebar-nav` — section switcher tiles (`.sidebar-nav`, `.sidebar-nav-link`, … in `web/src/styles/components/sidebar-nav.css`); inline markup in `org_admin.html`. Stream status filter uses bespoke `.stream-status-filter-*` in `pages/stream.css`, not this component.
- `dialog` — modal shell (`.dialog`, `.dialog-card`, `.dialog-head`, `.dialog-actions`, … in `web/src/styles/components/dialog.css`); destructive titles stack `.dialog-title u-text-danger`; inline markup in `process.html`, `stream.html`, `home.html`, `org_admin.html`, `platform_admin.html`, `components/substep_body.html`
- `button` — composable controls (`.btn` + variants/sizes/`btn-icon` in `web/src/styles/components/button.css`)
- `list-row` — bordered list item with main + actions (`.list-rows`, `.list-row`, `.list-row-main`, `.list-row-actions` in `web/src/styles/components/list-row.css`); inline markup in `org_admin.html`, `platform_admin.html`
- `tip` — inverted hover/focus label with caret (`.tip` in `web/src/styles/components/tip.css`); micro-partial `server/templates/components/tip.html` via `{{ template "tip" (dict "Tooltip" "…" …) }}` (`ImgSrc`/`ImgClass` or `Icon`/`InnerClass`; optional `Class`, `AriaLabel`, `Role`). `Icon` is any template define name, executed via the `render` template func.

## Docs and verification

- CSS architecture and template ↔ CSS index: `docs/css.md`
- Repo layout: `AGENTS.md`

Before claiming done:

```bash
cd server && go test ./cmd/server/ -count=1
cd server && go test ./cmd/server/ -run 'Breadcrumbs|StreamCard|DialogMarkup|ListRowMarkup' -count=1
task css:lint
```

## Out of scope (for now)

- Further peeling of `page_*.go` from `main.go`
- `ErrorBannerView` until a handler passes it
- Renaming legacy partial defines (`action_detail_content.html`, `role-palette-options`, …) — align when each partial is migrated
