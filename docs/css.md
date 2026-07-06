# CSS style guide (main Attesta app)

This document describes how styling works for the server-rendered Attesta UI. It implements [ADR-0001](adr/0001-css-architecture-refactor.md).

**Scope:** `web/src/styles/`, `server/templates/*.html`. Formata embed and Formata Builder (`formata-arch/`) are out of scope.

## Layer stack

Styles load in this order from `web/src/styles.css`:

| Layer | File | Contains |
|-------|------|----------|
| Tokens | `tokens.css` | `:root`, `[data-theme="dark"]`, font import |
| Reset | `reset.css` | `*`, `body`, `a`, `button`, heading defaults |
| Utilities | `utilities.css` | `u-*` spacing/typography/layout primitives |
| Layout | `layout.css` | Page chrome: topbar, nav, stack, grids, footer |
| Components | `components.css` | Reusable UI: panels, timeline, pills, forms |
| Pages | `pages.css` | Page-specific compositions (DPP, stream, admin) |
| Breakpoints | `phone.css`, `tablet.css`, `desktop.css` | Media-query overrides |

**Placement rule:** token → utility → component → page. A selector lives in exactly one layer.

## Theming

- Light/dark mode toggles `data-theme="light|dark"` on `<html>` (see `web/src/main.js`).
- Use design tokens (`var(--ink)`, `var(--panel)`, `var(--accent)`, etc.) — do not hardcode hex/rgb in templates or new CSS.
- Token names are stable; do not rename without a dedicated migration.

## Utilities (`u-*`)

Generic, domain-agnostic helpers in `utilities.css`. Add a new utility when the same spacing or typography pattern appears in multiple templates.

| Class | Effect |
|-------|--------|
| `u-m-0` | `margin: 0` |
| `u-mb-8` | `margin-bottom: 8px` |
| `u-mb-16` | `margin-bottom: 16px` |
| `u-mb-20` | `margin-bottom: 20px` |
| `u-ml-4` | `margin-left: 4px` |
| `u-pre-line` | `white-space: pre-line` |
| `u-text-sm` | `font-size: 12px` |
| `u-text-label` | `font-size: 14px; font-weight: bold` |
| `u-flex` | `display: flex` |
| `u-flex-center` | `display: flex; align-items: center` |
| `u-gap-8` | `gap: 8px` |
| `u-gap-16` | `gap: 16px` |
| `u-divider` | Horizontal rule, 24px vertical margin |
| `u-divider-flush` | `<hr>` with `margin: 0` |
| `u-divider-20` | Horizontal rule, 20px vertical margin |
| `text-danger` | `color: var(--danger)` |

**Stack gap modifiers:** `.stack.u-gap-8` and `.stack.u-gap-16` override the default 24px stack gap.

**Prefer component classes** for domain-specific patterns (e.g. `.role-pill-label`, `.workflow-card-footer`). New `u-*` utilities should include a one-line justification in the PR.

## Inline `style=` in templates

**Do not** use inline styles for static layout, spacing, or typography. Use utilities or component classes.

**Allowed** inline styles — runtime values from Go template data:

| Pattern | Example | Consumer |
|---------|---------|----------|
| Role pill colors (org-admin only) | `style="--pill-bg: {{ .RoleColor }}; --border: {{ .RoleBorder }};"` | `.role-pill`, `.pill` in org-admin |
| Stream status | `style="--stream-color: var(--stream-{{ .Status }});"` | `.stream-status-section-head` |
| Progress width | `style="width: {{ .Percent }}%;"` | Progress bar fill |

Workflow surfaces (`process`, `action_list`, `dpp`) use `data-role-palette="{{ .Palette }}"` on `.role-pill` and `.substep`; see [ADR-0002](adr/0002-role-color-appwrite-source.md). Palette keys map to `--role-*-bg` / `--role-*-border` in `components.css`.

Static pill presets use CSS classes instead: `.pill-accent`, `.pill-panel`.

## Common component classes

| Class | Use |
|-------|-----|
| `.panel`, `.stack` | Content blocks and vertical rhythm |
| `.muted` | Secondary text color |
| `.pill`, `.role-pill` | Badges; pair with dynamic `--pill-bg` when needed |
| `.is-disabled` | Disabled pagination / non-interactive controls |
| `.pagination-btn` | Chevron pagination buttons |
| `.site-footer` | Page footer (no inline styles) |

## Build

```bash
cd web && npm run build   # produces web/dist/assets/main.css
```

Go serves the bundle at `/static/`.

## Lint

```bash
task css:lint
```

Fails on disallowed `style=` attributes in `server/templates/`. Allowed patterns are listed above.

## Adding new UI

1. Check for an existing component class or utility.
2. If spacing/typography repeats across templates, add a `u-*` utility.
3. If the pattern is domain-specific, add a component or page class.
4. Use tokens for colors; never hardcode hex in templates.
5. Run `task css:lint` before opening a PR.
