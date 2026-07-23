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

**Full component** — reused 2+ pages or HTMX/SSE target; namespaced CSS; stable field set assembled in Go.

**CSS-only component** — reused markup tree + related selectors; no mode dispatch; see `docs/css.md` → CSS-only components. Examples: `panel.css`, `page-header.css` (optional trail via full `breadcrumbs`).

**Cluster instead:**

- Single-class primitives (`.muted`, `.pill`) → `shared.css`
- Domain groups (forms, org-admin widgets) → existing cluster files
- One-off markup with no dedicated styles → inline in page template

Migrate **one component at a time**. Do not move every partial in one pass.

## File layout

| Layer | Shared components | Full pages | Still at templates root |
|-------|-------------------|------------|-------------------------|
| Templates | `server/templates/components/{name}.html` | `server/templates/pages/{name}.html` | `error_banner.html`, `icons.html`, `attachment_carousel.html`, `role_palette_options.html`, `substep_override_editor.html` |
| CSS | `web/src/styles/components/{name}.css` | `web/src/styles/pages/{name}.css` | cluster files under `components/` (`shared`, `forms`, …) |
| Go view structs | `server/cmd/server/components.go` | handlers/page structs mostly in `main.go` | peeled assembly: `stream_instance_detail.go`, `substep_views_builder.go`, `timeline_builder.go`, `breadcrumbs.go` |

`layout.html` stays at `server/templates/layout.html`.

Separate **reused** styles (`components/`) from **page-specific** styles (`pages/`). A selector lives in exactly one layer — architecture and module index: `docs/css.md`.

## Naming

- **Template define name = file stem** (basename without `.html`): `stream_card.html` → `{{ define "stream_card" }}`. Micro-partials may share a file (e.g. `status_tag.html` defines `status_tag` only).
- Page wrappers: `{page}.html` define wrapping `layout.html` (e.g. `process.html`).
- Page body blocks: `{page}_body` define (e.g. `process_body`).
- CSS class prefix: kebab-case matching the component (`page-header-*`, `stream-card-*`, `breadcrumbs-*`).

## Go conventions

- All shared component view DTOs in **one** `components.go` file.
- **No** fluent `With*` methods on simple DTOs; use struct literals at call sites.
- **No** page-specific preset functions in shared component code (e.g. no `streamPageHeader()`).
- Extract a constructor only when there is real logic; page-specific constructors belong in peeled files or future `page_*.go`, not `components.go`. Trail builders such as `buildProcessBreadcrumbs` live in `breadcrumbs.go`.
- Split a type out of `components.go` only when it grows non-trivial logic or large isolated tests (~80–100+ lines).
- **CSS-only components** have **no** shared view struct. Put title/description/meta on the page view (or inline) and render the markup tree in the page template.

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
- `StreamCardView` in `components.go`
- Tests: `stream_card_test.go`

`stream_instance_card` — stream dashboard instance list row:

- `server/templates/components/stream_instance_card.html`
- `web/src/styles/components/stream-instance-card.css`
- `StreamInstanceCard` in `components.go`
- Tests: `stream_instance_card_test.go`
- Shared status tags remain in `components/stream.css` (`.status-tag*`)

`stream_termination_details` — early-termination warning banner:

- `server/templates/components/stream_termination_details.html`
- `web/src/styles/components/stream-termination-details.css`
- `StreamTerminationDetailsView` in `components.go`
- Builder: `buildStreamTerminationDetailsView` in `stream_instance_detail.go`
- Tests: `stream_termination_details_test.go`
- Call sites: `process.html`, `dpp.html`

`breadcrumbs` — hierarchical page trail in page headers:

- `server/templates/components/breadcrumbs.html`
- `web/src/styles/components/breadcrumbs.css`
- `BreadcrumbItem` / `BreadcrumbsView` in `components.go`
- Builders: `breadcrumbs.go` (`buildStreamBreadcrumbs`, `buildProcessBreadcrumbs`, …)
- Tests: `breadcrumbs_test.go`, `breadcrumbs_template_test.go`
- Call site: `{{ template "breadcrumbs" .Breadcrumbs }}` (`Current: true` on last crumb ⇒ `aria-current="page"`; every crumb is a link)

Also see:

- `substep_shell` — accordion chrome wrapping `substep_body`
- `substep_body` — inner panel with explicit `SubstepBodyView.Mode` dispatch
- `dpp_history_step` — DPP traceability rail wrapper around `stream_timeline_step`

### CSS-only / micro-partials

- `page-header` — `page-header.css`; inline in pages; **no** `PageHeaderView`. Heading-only: `page-header-body` under `section.page-header`. With actions: `page-header-head` wraps body + actions. Optional trail: `breadcrumbs` (above).
- `status_tag` — `status_tag.html`; pipeline is status **string**; `data-stream-status` → `--stream-color` / `.status-tag` in `stream.css`. Call: `{{ template "status_tag" .Status }}`
- `panel` — `panel.css`; optional `.panel-sticky`; inline on process/stream/dpp/org_admin/platform_admin
- `sidebar-nav` — `sidebar-nav.css`; inline in `org_admin.html`. Stream status filter uses `.stream-status-filter-*` in `pages/stream.css`, not this component.
- `dialog` — `dialog.css`; destructive titles stack `.dialog-title u-text-danger`; inline on process/stream/home/org_admin/platform_admin/`substep_body`
- `button` — `.btn` + variants in `button.css`
- `list-row` — `list-row.css`; inline on org_admin / platform_admin
- `tip` — `tip.css` + `tip.html` via `{{ template "tip" (dict …) }}` (`ImgSrc`/`ImgClass` or `Icon`/`InnerClass`; optional `Class`, `AriaLabel`, `Role`). `Icon` is any template define name via the `render` func.
- `local_datetime` — `local_datetime.html` via `{{ template "local_datetime" (dict "ISO" "…" "Human" "…") }}`; client `formatLocalDateTimes` in `main.js`.

## Docs and verification

- CSS architecture and template ↔ CSS index: `docs/css.md`
- Repo layout: `AGENTS.md`

Before claiming done:

```bash
cd server && go test ./cmd/server/ -count=1
cd server && go test ./cmd/server/ -run 'Breadcrumbs|StreamCard|StreamInstance|StreamTermination|DialogMarkup|ListRowMarkup|PanelMarkup' -count=1
task css:lint
```

## Out of scope (for now)

- Further peeling of `page_*.go` from `main.go`
- `ErrorBannerView` / migrating `error_banner.html` (legacy define name) until a handler owns a proper DTO
- Renaming legacy defines such as `role-palette-options` — align when each partial is migrated
