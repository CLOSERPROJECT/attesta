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

Use the full bundle (template + CSS + Go view struct) when **all** apply:

- Reused on 2+ pages, or is an HTMX/SSE partial target
- Has namespaced CSS (~10+ rules, e.g. `.page-header-*`)
- Has a dedicated view struct passed to `{{ template }}`

**Cluster instead:**

- Small primitives (`.panel`, `.pill`, `.muted`) → `web/src/styles/components/shared.css`
- Domain groups (forms, org-admin widgets) → existing cluster files (`forms.css`, `org-admin.css`)
- One-off markup with no dedicated styles → inline in page builder

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

## Docs and verification

- CSS architecture and template ↔ CSS index: `docs/css.md`
- Repo layout: `AGENTS.md`

Before claiming done:

```bash
cd server && go test ./cmd/server/ -count=1
cd server && go test ./cmd/server/ -run PageHeader -count=1
task css:lint
```

## Out of scope (for now)

- Further peeling of `page_*.go` from `main.go`
- `ErrorBannerView` until a handler passes it
- Renaming legacy partial defines (`action_detail_content.html`, `role-palette-options`, …) — align when each partial is migrated
