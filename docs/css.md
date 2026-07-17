# CSS style guide (main Attesta app)

Source of truth for styling the server-rendered Attesta UI: CSS architecture, theming, role palettes, template rules, and lint.

**Scope:** `web/src/styles/`, `server/templates/**/*.html`. Formata embed and Formata Builder (`formata-arch/`) are out of scope.

## Layer stack

Styles load in this order from `web/src/styles.css`:

| Layer | File | Contains |
|-------|------|----------|
| Breakpoints | `breakpoints.css` | Sole source of `@custom-media` aliases (Tailwind v3 widths) |
| Tokens | `tokens.css` | `:root`, `[data-theme="dark"]`, font/type tokens (Google Fonts load in `layout.html`) |
| Role palette | `role-palette.css` | `data-role-palette` and `data-stream-status` attribute maps |
| Reset | `reset.css` | `*`, `body`, `a`, `button`, heading defaults, focus rings, reduced motion |
| Utilities | `utilities.css` | `u-*` spacing/typography/layout primitives |
| Layout | `layout/index.css` | Barrel: `chrome.css` (topbar, nav, stack, footer), `grids.css` (page grids), `responsive.css` (shell breakpoint tweaks) |
| Components | `components.css` | Barrel importing `components/*.css` (panel, dialog, page-header, substep-shell, stream-timeline, forms, org-admin, stream, shared) |
| Pages | `pages.css` | Barrel importing `pages/*.css` (DPP, home, stream, process, org-admin shell, platform admin) |

**Placement rule:** token → utility → layout shell/grids → component → page. A selector lives in exactly one layer.

**Responsive placement:** co-locate `@media (--…)` blocks at the **bottom** of the owning module (component or page CSS). Shared page chrome and grid breakpoint behavior belongs in `layout/` (`grids.css`, `responsive.css`), not in separate breakpoint files.

### Page modules (`pages/`)

| File | Prefix / scope | Templates |
|------|----------------|-----------|
| `pages/process.css` | `.process-*` | `pages/process.html` |
| `pages/dpp.css` | `.dpp-*` | `pages/dpp.html` |
| `pages/home.css` | `.home-*`, `.preview-panel`, `.instances-panel` | `pages/home.html`, `pages/stream.html` (nav panels) |
| `pages/stream.css` | `.stream-*` | `pages/stream.html` |
| `pages/org-admin-page.css` | `.org-admin-*`, `.org-profile-*` (page shell) | `pages/org_admin.html` |
| `pages/platform-admin.css` | `.platform-admin-*` | `pages/platform_admin.html` |

Org-admin forms and pickers live in `components/org-admin.css`, not the page module. App-wide modal shells live in `components/dialog.css`.

### Component modules (`components/`)

| File | Prefix / scope | Templates |
|------|----------------|-----------|
| `components/page-header.css` | `.page-header-*` | `components/page_header.html` |
| `components/substep-body.css` | `.substep-body-*` | `components/substep_body.html`, `attachment_carousel.html` |
| `components/stream-step-summary.css` | `.stream-step-summary-*` | `components/stream_step_summary.html` |
| `components/stream-timeline.css` | `.stream-timeline-list`, `.stream-timeline-step*` | `components/stream_timeline.html` |
| `components/substep-shell.css` | `.substep*`, accordion shell | `components/substep_shell.html` |
| `components/substep-override.css` | `.substep-override-*`, `.js-open-substep-override` | `substep_override_editor.html` |
| `components/panel.css` | `.panel`, `.panel-sticky`, `.panel-heading`, `.panel-head-actions`, `.panel-block` | Inline markup per `panel.css` header (CSS-only) |
| `components/sidebar-nav.css` | `.sidebar-nav`, `.sidebar-nav-link`, `.sidebar-nav-title`, `.sidebar-nav-copy` | Inline markup per `sidebar-nav.css` header (CSS-only) |
| `components/dialog.css` | `.dialog`, `.dialog-card`, `.dialog-head`, `.dialog-actions`, … | Inline markup per `dialog.css` header (CSS-only) |
| `components/list-row.css` | `.list-rows`, `.list-row`, `.list-row-main`, `.list-row-actions` | Inline markup per `list-row.css` header (CSS-only) |

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
| `panel.css` | `.panel`, `.panel-sticky`, `.panel-heading`, `.panel-head-actions`, `.panel-actions`, `.panel-block` | See file header in `web/src/styles/components/panel.css` |
| `sidebar-nav.css` | `.sidebar-nav`, `.sidebar-nav-link`, `.sidebar-nav-title`, `.sidebar-nav-copy` | See file header in `web/src/styles/components/sidebar-nav.css` |
| `dialog.css` | `.dialog`, `.dialog-card`, `.dialog-head`, `.dialog-actions` | See file header in `web/src/styles/components/dialog.css` |
| `button.css` | `.btn`, `.btn-primary`, `.btn-secondary`, `.btn-ghost`, `.btn-ghost-danger`, `.btn-danger`, `.btn-outline`, `.btn-xs`, `.btn-sm`, `.btn-lg`, `.btn-icon` | See file header in `web/src/styles/components/button.css` |
| `list-row.css` | `.list-rows`, `.list-row`, `.list-row-main`, `.list-row-actions` | See file header in `web/src/styles/components/list-row.css` |

**Dialog placement:** generic shell in `dialog.css`; `#stream-preview-dialog` sizing in `pages/stream.css`; org-admin role pill wrapper in `org-admin.css`. Destructive titles use `u-text-danger` (color only) stacked on `.dialog-title`. Wide shell modifier `.dialog-wide` deferred until a second page needs it.

Other partials (`icons.html`, …) still live at `server/templates/` root until migrated one by one. Split reused styles into `components/` and page-specific styles into `pages/`.

### Template ↔ CSS index

| Template | Primary CSS | Also uses |
|----------|-------------|-----------|
| `layout.html` | `layout/index.css` | `components/shared.css` |
| `components/page_header.html` | `components/page-header.css` | — |
| Inline panel sections (process, stream, dpp, org_admin, platform_admin) | `components/panel.css` | `components/button.css`, `components/shared.css` (`.muted`); optional `.panel-sticky` |
| Inline dialog modals (process, stream, home, org_admin, platform_admin, substep_body) | `components/dialog.css` | `pages/stream.css` (#stream-preview-dialog), `components/org-admin.css` (role pill), `components/substep-body.css` (active-role spacing), `components/forms.css` |
| Inline sidebar nav (stream, org_admin) | `components/sidebar-nav.css` | `components/panel.css` (`.panel-sticky`) |
| Inline list rows (org_admin roles/users, platform_admin orgs) | `components/list-row.css` | `components/button.css`; domain: `org-admin.css` (`.user-email`, `.user-tags`), `pages/platform-admin.css` (copy/status) |
| `pages/home.html` | `pages/home.css` | `components/dialog.css`, `components/stream.css`, `layout/index.css` |
| `pages/stream.html` | `pages/home.css`, `pages/stream.css` | `components/dialog.css`, `components/sidebar-nav.css`, `components/panel.css`, `components/stream.css`, `components/stream-timeline.css`, `role-palette.css` |
| `pages/process.html` | `pages/process.css` | `components/dialog.css`, `components/substep-shell.css`, `components/stream-timeline.css`, `components/substep-body.css`, `layout/responsive.css` (`.layout-stack-separator`), `role-palette.css` |
| `components/stream_step_summary.html` | `components/stream-step-summary.css` | — |
| `components/stream_timeline.html` | `components/stream-timeline.css` | `components/stream-step-summary.css`, `components/substep-body.css`, `role-palette.css` |
| `components/dpp_history_step.html` | `pages/dpp.css` (`.dpp-history-*`) | `components/stream-timeline.css`, `components/stream-step-summary.css`, `components/substep-shell.css`, `components/substep-body.css`, `role-palette.css` |
| `components/substep_shell.html` | `components/substep-shell.css` | `components/substep-body.css`, `role-palette.css` |
| `components/substep_body.html` | `components/substep-body.css` | `components/dialog.css`, `components/forms.css`, `role-palette.css` |
| `pages/dpp.html` | `pages/dpp.css` | `components/dpp_history_step.html`, `components/stream-timeline.css`, `components/stream-step-summary.css`, `components/substep-shell.css`, `components/substep-body.css`, `role-palette.css` |
| `pages/org_admin.html` | `pages/org-admin-page.css` | `components/dialog.css`, `components/sidebar-nav.css`, `components/panel.css`, `components/org-admin.css`, `role-palette.css` |
| `pages/platform_admin.html` | `pages/platform-admin.css` | `components/dialog.css`, `components/shared.css` |
| `pages/login.html`, `pages/signup.html`, `pages/invite.html`, `pages/reset_*.html` | `components/forms.css` | `components/button.css`, `components/shared.css` |
| `attachment_carousel.html` | `components/substep-body.css` | — |
| `substep_override_editor.html` | `components/substep-override.css` | — |

### `data-*` contract (templates → CSS / JS)

| Attribute | Set in | Consumed by |
|-----------|--------|-------------|
| `data-theme` | `main.js` on `<html>` | `tokens.css` (`[data-theme="dark"]`) |
| `data-role-palette` | role badges, timeline substeps | `role-palette.css` |
| `data-stream-status` | `stream.html` section heads | `role-palette.css` |
| `data-progress` (via `style="--progress: …"`) | `stream.html` | `.process-progress-fill` in `components/stream.css` |
| `data-org-admin-nav`, `data-org-admin-panel`, `data-org-admin-default-panel`, `data-org-admin-ready` | `org_admin.html` | `pages/org-admin-page.css`, inline script in `org_admin.html` |
| `data-home-nav`, `data-home-panel` | `stream.html` | `main.js` (panel switching) |
| `data-process-id`, `data-workflow-key`, `data-selected-substep`, `data-substep-id` | `process.html` | `main.js` (SSE refresh, accordion) |
| `data-formata-*`, `data-active-role-*` | `components/substep_body.html` | `main.js` (Formata embed, role picker) |
| `data-override-url` | `components/substep_body.html` | `main.js` (substep override editor) |
| `data-carousel-*` | `attachment_carousel.html` | `main.js`, `components/substep-body.css` |
| `data-copy-text`, `data-copy-label` | `dpp.html` | `main.js` (clipboard) |
| `data-share-url`, `data-share-label` | `dpp.html` | `main.js` (share link) |
| `data-target` | password visibility toggles | `main.js` |
| `data-value`, `data-label`, `data-selected` | org-admin role pickers | `components/org-admin.css`, inline script |
| `data-role-color` | `role_palette_options.html` | org-admin palette picker script |

Behavior hooks use a `js-*` class prefix alongside `data-*` where needed; do not style `js-*` classes in CSS.

## Theming

- Light/dark mode toggles `data-theme="light|dark"` on `<html>` (see `web/src/main.js`).
- Use design tokens (`var(--foreground)`, `var(--card)`, `var(--primary)`, etc.) — do not hardcode hex/rgb in templates or new CSS.
- Role hue and stream status tokens use CSS `light-dark()` in `:root`; `[data-theme="dark"]` keeps only non-role overrides.
- Token names follow shadcn-style `{role}` + `{role}-foreground` pairs (see `tokens.css`).

### Breakpoints

**Tailwind v3 widths** (sm → 2xl) are the canonical thresholds:

| Tailwind | Width | `@custom-media` up | `@custom-media` down |
|----------|-------|--------------------|----------------------|
| sm | 640px | `--sm-up` `(width >= 640px)` | `--sm-down` `(width < 640px)` |
| md | 768px | `--md-up` | `--md-down` |
| lg | 1024px | `--lg-up` | `--lg-down` |
| xl | 1280px | `--xl-up` | `--xl-down` |
| 2xl | 1536px | `--2xl-up` | — |

- **`breakpoints.css`** is imported first in `styles.css` and is the **only** file that may define `@custom-media`.
- **Stylesheet modules** use `@media (--sm-down) { … }` (or other aliases). Do **not** repeat `@custom-media` or write `@media (width … 640px)` in component/page CSS.
- **PostCSS:** `postcss-custom-media` (see `web/postcss.config.js`) expands aliases at build time; Vite runs PostCSS on the bundled CSS.

**JS sync:** when JavaScript needs the same threshold as CSS, use the equivalent `matchMedia` query. Example: `--xl-down` `(width < 1280px)` ↔ `matchMedia("(max-width: 1279px)")` (as in `org_admin.html` for mobile panel switching). Keep JS literals aligned with the table above when adding new breakpoint checks.

### Font tokens

| Token | Value |
|-------|-------|
| `--font-sans` | Lato stack (body, buttons, UI copy, `h2`–`h4`) |
| `--font-display` | Space Grotesk stack (literal `h1` only) |
| `--font-mono` | JetBrains Mono stack (hashes, codes, meta ids) |

Google Fonts load from `server/templates/layout.html` (`preconnect` + stylesheet `<link>` with `display=swap`). Do **not** `@import` fonts from `tokens.css`. Use `var(--font-sans)` / `var(--font-display)` / `var(--font-mono)` in new CSS — do not repeat font family strings.

### Type scale

Canonical type tokens in `tokens.css` — Tailwind-aligned, rem-based. The index tracks size: `--text-sm` = `0.875rem` = 14px.

| Token | rem | px | Typical use |
|-------|-----|----|-------------|
| `--text-xs` | `0.75rem` | 12 | Captions, pills, compact tags, meta IDs |
| `--text-sm` | `0.875rem` | 14 | Labels, secondary UI, toolbar copy, form labels |
| `--text-base` | `1rem` | 16 | Body, inputs, buttons, default prose |
| `--text-lg` | `1.125rem` | 18 | Card titles, dialog titles, emphasis headings |
| `--text-xl` | `1.25rem` | 20 | Section headings (`h2` in panels; `h3` for nested/subsection titles) |
| `--text-2xl` | `1.5rem` | 24 | Large titles, emphasis page headings |
| `--text-3xl` | `1.875rem` | 30 | Page titles (`h1` default) |

**Line-height tokens**

| Token | Value | Use |
|-------|-------|-----|
| `--leading-none` | `1` | Pills, single-line badges |
| `--leading-tight` | `1.25` | Headings, compact titles |
| `--leading-normal` | `1.5` | Body, form fields, paragraphs |
| `--leading-relaxed` | `1.625` | Long-form muted copy (rare) |

**Weight tokens**

| Token | Value | Loaded face |
|-------|-------|-------------|
| `--font-normal` | `400` | Lato 400 |
| `--font-medium` | `400` | Lato 400 (Lato has no 500) |
| `--font-semibold` | `700` | Lato 700 (Lato has no 600) |
| `--font-bold` | `900` | Lato 900 |

**Heading defaults** are set in `reset.css` (`h1`–`h4`): `h1` → `--text-3xl` + `--font-display`, `h2` → `--text-xl`, `h3` → `--text-lg`, `h4` → `--text-base`, all with `--font-semibold` and `--leading-tight`. Prefer tokens over raw `font-size` in component CSS; remove redundant heading `font-size` overrides when they only duplicate semantics.

**Control sizing pattern:** buttons use the `btn` system in `button.css` (default height `--btn-height` / 36px). Form labels use `--text-sm` (`forms.css`); inputs use `--text-base`.

### Spacing scale

Canonical spacing tokens in `tokens.css` — a Tailwind-aligned scale on a 4px grid, expressed in `rem` (migrate incrementally; legacy `px` literals remain valid). The index tracks size: `--space-N` ≈ `N × 4px` (`--space-4` = `1rem` = 16px):

| Token | Value | px | Typical use |
|-------|-------|----|-------------|
| `--space-1` | `0.25rem` | 4 | Tight inline gaps |
| `--space-2` | `0.5rem` | 8 | Default small gap, compact lists |
| `--space-3` | `0.75rem` | 12 | Card padding, compact control padding |
| `--space-4` | `1rem` | 16 | Grid gaps, card/list item padding |
| `--space-5` | `1.25rem` | 20 | Panel padding, section margins/gaps |
| `--space-6` | `1.5rem` | 24 | Stack default gap |
| `--space-7` | `1.75rem` | 28 | Large section rhythm |

The scale is monotonic: `--space-N` is always larger than `--space-(N-1)`. Do not reintroduce off-grid values (6/10/14/18px) or non-monotonic indices.

Prefer `u-*` utilities or component classes over raw `px` in templates. New page CSS should use spacing tokens where practical.

**Intentional `px` exceptions** (not on the spacing scale): layout dimensions (`width`/`height`), `border-radius`, `border-width`, scroll anchors (`44px`, `140px`), and composite padding with off-scale values (e.g. `32px` horizontal chrome). Use type tokens for `font-size` — not raw `px`.

### Elevation and backdrop tokens

| Token | Use |
|-------|-----|
| `--shadow` | Default panel shadow |
| `--shadow-elevated` | Raised controls (carousel nav) |
| `--shadow-dropdown` | Floating menus |
| `--backdrop-modal` | Dialog `::backdrop` tint |
| `--backdrop-fullscreen` | Fullscreen modal backdrop |
| `--inset-highlight` | Inset top edge on gradient fields |

### Documented color literal exceptions

No compositional color literals remain outside `tokens.css`. If you need a new shadow or backdrop tint, add a semantic token in `tokens.css` rather than an inline `rgb()`.

## Role palette

Role badge colors are resolved at runtime from **Appwrite team prefs**, not from workflow YAML or inline CSS values.

### Data flow

```
Appwrite team prefs          Backend (roleMetaIndex)          Templates + CSS
{ slug, name, palette }  →   (orgSlug, roleSlug) → key   →   data-role-palette="blue"
                                                                      ↓
                                                             role-palette.css → --role-*
```

| Layer | Responsibility |
|-------|----------------|
| Appwrite | Canonical store: `{ slug, name, palette }` where `palette` is a named key (`blue`, `emerald`, …) |
| Backend | Resolves org-scoped `(orgSlug, roleSlug)` to a palette key; never emits `var(--role-*-*)` strings to workflow templates |
| Workflow YAML | Slug/org/name for validation only; `color` fields are ignored by Go |
| Templates | Set `data-role-palette="{{ .Palette }}"` on `.role-pill` and timeline substeps |
| `tokens.css` + `role-palette.css` | Single source of appearance; maps palette key → `--role-*` tokens |

### Storage and API

- **Writes** persist `palette` only (no `color`).
- **Reads** use `palette` when present; legacy rows with `color` CSS var strings fall back to `rolePaletteKeyFromStyle()`.
- **`GET /api/catalog`** returns `palette` per role (no `color`).
- **Unknown or missing role** → palette key `"fallback"` (not YAML-embedded colors).

### Lookup rules

Resolution is org-scoped via `(orgSlug, roleSlug)`:

| Caller context | Lookup key |
|----------------|------------|
| Substep on a step with `organization: <org>` | `(stepOrg, roleSlug)` |
| Substep with no step org; role unique in workflow | first org from config or identity catalog containing the slug |
| Unknown org or missing role in Appwrite | `"fallback"`; label from slug |

Key backend symbols: `roleMetaIndex`, `roleMetaFor`, `rolePaletteKeyFromStyle` in `server/cmd/server/`.

### Frontend styling

17 named palette keys (`red`, `orange`, `amber`, … `rose`) are defined in `rolePaletteStyles` / `rolePaletteKeys` (Go) and mapped in `role-palette.css`. `TestRolePaletteKeysMatchCSS` keeps Go and CSS keys in sync.

Role pills on `process`, `substep_body`, `dpp`, and `org_admin` use:

```html
<span class="role-pill" data-role-palette="{{ .Palette }}">{{ .Label }}</span>
```

Timeline substeps use the same attribute on `.substep`. Appearance is a soft-badge: text/border from the bg token with a 10% tint background (`color-mix`). The org-admin palette picker sets transient `--swatch-bg` on preview swatches only (not on role pill rows).

Stream status uses `data-stream-status="{{ .Status }}"` on `.stream-status-section-head` (mapped in `role-palette.css`).

Static pill presets use CSS classes instead: `.pill-accent`, `.pill-panel`.

## Utilities (`u-*`)

Generic, domain-agnostic helpers in `utilities.css`. Add a new utility when the same spacing or typography pattern appears in multiple templates.

| Class | Effect |
|-------|--------|
| `u-mx-auto` | `margin-inline: auto` |
| `u-max-w-prose` | `max-width: 65ch` |
| `u-max-w-7xl` | `max-width: 80rem` |
| `u-m-0` | `margin: 0` |
| `u-mb-2` | `margin-bottom: var(--space-2)` (8px) |
| `u-mb-4` | `margin-bottom: var(--space-4)` (16px) |
| `u-mb-5` | `margin-bottom: var(--space-5)` (20px) |
| `u-ml-1` | `margin-left: var(--space-1)` (4px) |
| `u-pre-line` | `white-space: pre-line` |
| `u-text-xs` | `font-size: var(--text-xs)` |
| `u-text-sm` | `font-size: var(--text-sm); font-weight: var(--font-semibold)` |
| `u-text-base` | `font-size: var(--text-base)` |
| `u-text-lg` | `font-size: var(--text-lg)` |
| `u-flex` | `display: flex` |
| `u-flex-center` | `display: flex; align-items: center` |
| `u-gap-2` | `gap: var(--space-2)` (8px) |
| `u-gap-4` | `gap: var(--space-4)` (16px) |
| `u-divider` | Horizontal rule, `var(--space-6)` vertical margin |
| `u-divider-flush` | `<hr>` with `margin: 0` |
| `u-divider-5` | Horizontal rule, `var(--space-5)` vertical margin |
| `u-text-danger` | `color: var(--destructive)` |

**Stack gap modifiers:** `.stack.u-gap-2` and `.stack.u-gap-4` override the default `var(--space-6)` stack gap.

**Prefer component classes** for domain-specific patterns (e.g. `.role-pill-label`, `.workflow-card-footer`). New `u-*` utilities should include a one-line justification in the PR.

## Inline `style=` in templates

**Do not** use inline styles for static layout, spacing, or typography. Use utilities or component classes.

**Allowed** inline styles — runtime values from Go template data:

| Pattern | Example | Consumer |
|---------|---------|----------|
| Progress width | `style="--progress: {{ .Percent }}%;"` | `.process-progress-fill` via `width: var(--progress, 0%)` |

All other dynamic theming uses `data-*` attributes (`data-role-palette`, `data-stream-status`), not inline custom properties.

## Common component classes

| Class | Use |
|-------|-----|
| `.panel`, `.panel-heading`, `.panel-head-actions`, `.panel-block` | Card sections — see `panel.css` header for markup tree (CSS-only component) |
| `.panel-sticky` | Optional sticky rail modifier on `.panel` (active at `--xl-up`) |
| `.sidebar-nav`, `.sidebar-nav-link`, `.sidebar-nav-title`, `.sidebar-nav-copy` | Section switcher tiles — see `sidebar-nav.css` header |
| `.dialog`, `.dialog-card`, `.dialog-head`, `.dialog-actions`, … | Modal shells — see `dialog.css` header (CSS-only component) |
| `.list-rows`, `.list-row`, `.list-row-main`, `.list-row-actions` | Admin list rows with actions — see `list-row.css` header |
| `.stack` | Vertical rhythm |
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

Runs three checks:

1. **Template inline styles** — `deployment/scripts/check-template-inline-styles.sh` fails on disallowed `style=` attributes in `server/templates/` (allowed patterns listed above).
2. **Breakpoint guard** — `deployment/scripts/check-css-breakpoints.sh` fails when any file under `web/src/styles/` (except `breakpoints.css`) contains `@media (width …)` with literal `px` values; use `@media (--sm-down)` etc. instead.
3. **stylelint** — CSS rules on `web/src/styles/**/*.css` (no hex/rgb outside `tokens.css`, no new `!important`).

## Adding new UI

1. Check for an existing component class, CSS-only module (**CSS-only components** above), or utility.
2. If spacing/typography repeats across templates, add a `u-*` utility.
3. If the pattern is domain-specific, add a component or page class (see **Page modules**).
4. Use tokens for colors; never hardcode hex in templates.
5. Run `task css:lint` before opening a PR.

## Out of scope / known gaps

- **Formata embed** shadow-DOM styling in `web/src/main.js` (`!important` overrides) — separate effort; not covered here.
- **Adding a new palette key** requires updates to `rolePaletteStyles`, `role-palette.css`, and the org-admin palette picker — not Go string passthrough to templates.
