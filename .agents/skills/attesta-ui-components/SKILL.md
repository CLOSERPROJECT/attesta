---
name: attesta-ui-components
description: >-
  Extract or place Attesta UI components (full / CSS-only / cluster).
  Use when adding or changing server templates, view structs, or
  web/src/styles component CSS; or when deciding whether to extract a partial.
---

# Attesta UI components

Server-rendered UI in `server/templates/` + `web/src/styles/`. For CSS layers, tokens, and CSS-only markup contracts, open `docs/css.md` before writing selectors.

## 1. Pick a tier

Done when: you can name the **tier** and why the other two do not fit.

| Tier | Template | CSS | Go view struct |
|------|----------|-----|----------------|
| **Full** | `components/{name}.html` | `components/{name}.css` when ~10+ namespaced rules | Dedicated type in `components.go` |
| **CSS-only** | None (or a narrow micro-partial) — inline HTML in the page | `components/{name}.css` with markup-tree header | None |
| **Cluster** | inline | existing cluster file (`shared.css`, `forms.css`, …) | none |

Choose **full** when reused on 2+ pages or it is an HTMX/SSE swap target, with a stable field set assembled in Go.

Choose **CSS-only** when the markup tree is reused, selectors form one family, and there is no mode dispatch / swap target. Examples: `page-header`, `panel`, `dialog`, `list-row`. Markup contract lives in the CSS file header; layer/stem rules in `docs/css.md`.

Choose **cluster** for single-class primitives (`.muted`, `.pill`), domain widget groups, or one-off markup with no dedicated styles.

**Tracer** migration: **extract** or move **one** component per change.

## 2. Place files

Done when: paths match the tier, and define name / CSS prefix follow **stem** rules below.

| Layer | Shared (full / CSS-only module) | Page-specific |
|-------|----------------------------------|---------------|
| Templates | `server/templates/components/{name}.html` (full only) | `server/templates/pages/{name}.html` |
| CSS | `web/src/styles/components/{name}.css` | `web/src/styles/pages/{name}.css` |
| Go | view DTO in `components.go`; builders with real logic in peeled files (`breadcrumbs.go`, `stream_instance_detail.go`, …) | page fields on the page view / handler |

`layout.html` stays at `server/templates/layout.html`. Root leftovers (`error_banner.html`, `icons.html`, …) migrate only when the task needs them.

Reused styles → `components/`; page-only → `pages/`. One selector, one layer (`docs/css.md`).

### Stem naming

- Template define name = file **stem**: `stream_card.html` → `{{ define "stream_card" }}`
- Page wrapper: `{page}.html`; body block: `{page}_body`
- CSS class prefix: kebab-case matching the component (`stream-card-*`, `page-header-*`)

### Go DTOs (full tier)

- Shared view DTOs live in `components.go`
- Call sites use struct literals (not fluent `With*` chains)
- Page-specific builders live beside the page (`breadcrumbs.go`, `*_builder.go`), not as presets in `components.go`
- Split a type out of `components.go` only when logic or tests grow large (~80–100+ lines)
- CSS-only: title/description/meta on the **page** view; markup inlined in the page template

```go
Card: StreamCardView{
    Key:         cfg.Workflow.Key,
    Name:        cfg.Workflow.Name,
    Description: strings.TrimSpace(cfg.Workflow.Description),
},
```

```go
Title:       cfg.Workflow.Name,
Description: strings.TrimSpace(cfg.Workflow.Description),
Breadcrumbs: buildStreamBreadcrumbs(workflowKey, cfg.Workflow.Name),
```

Templates load via `parseTemplates()` / `parseTestTemplates(t)` in `templates.go` (`templates/*.html`, `pages/*.html`, `components/*.html`).

## 3. Match a tracer

Done when: the new or changed piece mirrors a same-tier neighbor (paths, define, prefix, DTO shape).

**Full tracer — `stream_card`:**

- `server/templates/components/stream_card.html`
- `web/src/styles/components/stream-card.css`
- `StreamCardView` in `components.go`
- Tests: `stream_card_test.go`

Other full neighbors (same layout pattern): `stream_instance_card`, `stream_termination_details`, `breadcrumbs` (`Current: true` on last crumb; every crumb still has `Href`), `substep_shell` / `substep_body`, `dpp_history_step`.

**CSS-only / micro-partial:** copy markup from an existing page or micro-partial (`status_tag`, `tip`, `local_datetime`); read the CSS file header for the contract — do not invent a shared view struct.

## 4. Done when

Every touched template/CSS path is accounted for, and:

```bash
cd server && go test ./cmd/server/ -count=1
cd server && go test ./cmd/server/ -run 'Breadcrumbs|StreamCard|StreamInstance|StreamTermination|DialogMarkup|ListRowMarkup|PanelMarkup' -count=1
task css:lint
```

Narrow the `-run` filter to components you actually changed when the full package test already passed.
