# CSS style guide (main Attesta app)

This document describes how styling works for the server-rendered Attesta UI. It implements [ADR-0001](adr/0001-css-architecture-refactor.md) and [ADR-0004](adr/0004-css-polish.md).

**Scope:** `web/src/styles/`, `server/templates/*.html`. Formata embed and Formata Builder (`formata-arch/`) are out of scope.

## Layer stack

Styles load in this order from `web/src/styles.css`:

| Layer | File | Contains |
|-------|------|----------|
| Tokens | `tokens.css` | `:root`, `[data-theme="dark"]`, font import |
| Role palette | `role-palette.css` | `data-role-palette` and `data-stream-status` attribute maps |
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
- Role hue and stream status tokens use CSS `light-dark()` in `:root` (see [ADR-0004](adr/0004-css-polish.md)); `[data-theme="dark"]` keeps only non-role overrides.
- Token names are stable; do not rename without a dedicated migration.

### Breakpoint tokens

Canonical viewport widths in `tokens.css`:

| Token | Value | Typical use |
|-------|-------|-------------|
| `--bp-phone` | `640px` | `phone.css` overrides |
| `--bp-tablet` | `900px` | Intermediate layouts |
| `--bp-desktop` | `1200px` | Sidebar grids (`desktop.css`, `tablet.css`) |

Media query conditions must use literal `px` values (CSS cannot evaluate `var()` in `@media`). Keep breakpoint files in sync with these tokens; reference the token in layout properties inside queries when helpful (e.g. `min(var(--bp-desktop), 100vw)` on dialogs).

### Documented color literal exceptions

After the Phase 7 token hygiene audit, a few one-off compositional `rgba()` values remain outside `tokens.css`. Do not copy these into new CSS — prefer tokens or `color-mix(in srgb, var(--*) …%)`.

| File | Value | Use |
|------|-------|-----|
| `pages.css` | `rgba(255, 255, 255, 0.04)` | Inset top highlight on `.platform-admin-search-field` gradient |
| `components/org-admin.css` | `rgba(0, 0, 0, 0.12)` | Elevated dropdown shadow on `.roles-picker-menu` |
| `components/org-admin.css` | `rgba(16, 26, 20, 0.48)` | Tinted backdrop on `.manage-dialog::backdrop` |
| `components/actions.css` | `rgba(16, 26, 20, 0.16)` | Elevated shadow on attachment carousel nav buttons |
| `components/timeline.css` | `rgba(0, 0, 0, 0.42)` | Fullscreen modal backdrop on `.substep-override-modal::backdrop` |

## Utilities (`u-*`)

Generic, domain-agnostic helpers in `utilities.css`. Add a new utility when the same spacing or typography pattern appears in multiple templates.

| Class | Effect |
|-------|--------|
| `u-mx-auto` | `margin-inline: auto` |
| `u-max-w-prose` | `max-width: 65ch` |
| `u-max-w-7xl` | `max-width: 80rem` |
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
| Progress width | `style="--progress: {{ .Percent }}%;"` | `.process-progress-fill` via `width: var(--progress, 0%)` |

Stream status uses `data-stream-status="{{ .Status }}"` on `.stream-status-section-head` (mapped in `role-palette.css`); no inline style.

Role pills (`process`, `action_list`, `dpp`, `org_admin`) use `data-role-palette="{{ .Palette }}"` on `.role-pill`; see [ADR-0002](adr/0002-role-color-appwrite-source.md) and [ADR-0003](adr/0003-role-palette-storage.md). Palette keys map to `--role-*-bg` in `role-palette.css` (soft-badge: text/border from bg token, 10% tint background). The org-admin palette picker sets transient `--swatch-bg` on preview swatches only (not on role pill rows).

Static pill presets use CSS classes instead: `.pill-accent`, `.pill-panel`.

## Common component classes

| Class | Use |
|-------|-----|
| `.panel`, `.stack` | Content blocks and vertical rhythm |
| `.muted` | Secondary text color |
| `.pill`, `.role-pill` | Badges; pair with `data-role-palette` for role colors |
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
