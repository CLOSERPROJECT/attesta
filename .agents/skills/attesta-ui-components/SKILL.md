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
| **CSS-only component** | **None** — inline HTML in page templates | `components/{name}.css` with markup-tree header comment | **None** |
| **Cluster / primitive** | inline | `shared.css`, `forms.css`, … | none |

**Full component** — all apply: reused 2+ pages or HTMX/SSE target; namespaced CSS; stable field set assembled in Go.

**CSS-only component** — reused markup tree + related selectors; no mode dispatch; see `docs/css.md` → CSS-only components. Example: panel section headers → `panel.css` (read file header for markup).

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

- **Template define name = file stem** (basename without `.html`): `page_header.html` → `{{ define "page_header" }}`.
- Page wrappers: `{page}.html` define wrapping `layout.html` (e.g. `process.html`).
- Page body blocks: `{page}_body` define (e.g. `process_body`).
- CSS class prefix: kebab-case matching the component (`page-header-*` for `page_header`).

## Go conventions

- All shared component view DTOs in **one** `components.go` file.
- **No** fluent `With*` methods on simple DTOs; use struct literals at call sites.
- **No** page-specific preset functions in shared component code (e.g. no `streamPageHeader()`).
- Extract a constructor only when there is real logic (validation, computed fields); page-specific constructors belong in peeled files (`stream_instance_detail.go`, `substep_views_builder.go`, …) or future `page_*.go`, not `components.go`.
- Split a type out of `components.go` only when it grows non-trivial logic or large isolated tests (~80–100+ lines).

Example:

```go
Header: PageHeaderView{
    Title:       cfg.Workflow.Name,
    Description: strings.TrimSpace(cfg.Workflow.Description),
    BackHref:    "/",
},
```

## Template loading

- Production: `parseTemplates()` in `server/cmd/server/templates.go`
- Tests: `parseTestTemplates(t)` in the same file

Globs: `templates/*.html`, `templates/pages/*.html`, `templates/components/*.html`.

## Reference implementations

`page_header` — first migrated component:

- `server/templates/components/page_header.html`
- `web/src/styles/components/page-header.css`
- `PageHeaderView` in `server/cmd/server/components.go`
- Tests: `server/cmd/server/page_header_test.go`

Also see:

- `substep_shell` — accordion chrome wrapping `substep_body` (`components/substep_shell.html`, `substep-shell.css`)
- `substep_body` — inner panel with explicit `SubstepBodyView.Mode` dispatch
- `dpp_history_step` — DPP traceability rail wrapper around `stream_timeline_step`

### CSS-only components

- `panel` — card container and section headers (`web/src/styles/components/panel.css`); inline markup in `process.html`, `stream.html`, `dpp.html`, `org_admin.html`, `platform_admin.html`
- `dialog` — modal shell (`.dialog`, `.dialog-card`, `.dialog-head`, `.dialog-actions`, … in `web/src/styles/components/dialog.css`); destructive titles stack `.dialog-title u-text-danger`; inline markup in `process.html`, `stream.html`, `home.html`, `org_admin.html`, `platform_admin.html`, `components/substep_body.html`
- `button` — composable controls (`.btn` + variants/sizes/`btn-icon` in `web/src/styles/components/button.css`)

## Docs and verification

- CSS architecture and template ↔ CSS index: `docs/css.md`
- Repo layout: `AGENTS.md`

Before claiming done:

```bash
cd server && go test ./cmd/server/ -count=1
cd server && go test ./cmd/server/ -run 'PageHeader|DialogMarkup' -count=1
task css:lint
```

## Out of scope (for now)

- Further peeling of `page_*.go` from `main.go`
- `ErrorBannerView` until a handler passes it
- Renaming legacy partial defines (`action_detail_content.html`, `role-palette-options`, …) — align when each partial is migrated
