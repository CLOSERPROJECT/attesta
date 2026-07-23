# CSS style guide (main Attesta app)

Source of truth for styling the server-rendered Attesta UI: CSS architecture, theming, role palettes, template rules, and lint.

**Scope:** `web/src/styles/`, `server/templates/**/*.html`. Formata embed and Formata Builder (`formata-arch/`) are out of scope.

For extract-vs-inline decisions and Go view structs, see `.agents/skills/attesta-ui-components/SKILL.md`.

## Layer stack

Styles load in this order from `web/src/styles.css`:

| Layer | File | Contains |
|-------|------|----------|
| Breakpoints | `breakpoints.css` | Sole source of `@custom-media` aliases (Tailwind v3 widths) |
| Tokens | `tokens.css` | `:root`, `[data-theme="dark"]`, font/type tokens (Google Fonts load in `layout.html`) |
| Role palette | `role-palette.css` | `data-role-palette` and `data-stream-status` attribute maps |
| Reset | `reset.css` | `*`, `body`, `a`, `button`, heading defaults, focus rings, reduced motion |
| Utilities | `utilities.css` | `u-*` spacing/typography/layout primitives |
| Layout | `layout/index.css` | Barrel: `chrome.css`, `grids.css` (`.rail-layout`), `responsive.css` |
| Components | `components.css` | Barrel importing `components/*.css` — see that file for the live import list |
| Pages | `pages.css` | Barrel importing `pages/*.css` — see that file for the live import list |

**Placement rule:** token → utility → layout shell/grids → component → page. A selector lives in exactly one layer.

**Responsive placement:** co-locate `@media (--…)` blocks at the **bottom** of the owning module (component or page CSS). Shared chrome/grid breakpoints belong in `layout/` (`grids.css`, `responsive.css`).

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

| File | Prefix / scope | Templates / markup |
|------|----------------|--------------------|
| `components/page-header.css` | `.page-header`, `.page-header-*` | CSS-only — inline in pages; optional trail via `breadcrumbs`; optional `.page-header-actions` |
| `components/breadcrumbs.css` | `.breadcrumbs`, `.breadcrumbs-*` | `components/breadcrumbs.html` |
| `components/stream-card.css` | `.stream-card-*` | `components/stream_card.html` |
| `components/stream-instance-card.css` | `.stream-instance-card-*` | `components/stream_instance_card.html` |
| `components/stream-termination-details.css` | `.stream-termination-details*` | `components/stream_termination_details.html` |
| `components/substep-body.css` | `.substep-body-*` | `components/substep_body.html`, `attachment_carousel.html` |
| `components/stream-step-summary.css` | `.stream-step-summary-*` | `components/stream_step_summary.html` |
| `components/stream-timeline.css` | `.stream-timeline-list`, `.stream-timeline-step*` | `components/stream_timeline.html` |
| `components/substep-shell.css` | `.substep*`, accordion shell | `components/substep_shell.html` |
| `components/substep-override.css` | `.substep-override-*`, `.js-open-substep-override` | `substep_override_editor.html` |
| `components/panel.css` | `.panel`, `.panel-sticky`, `.panel-heading`, … | CSS-only — see file header |
| `components/sidebar-nav.css` | `.sidebar-nav`, `.sidebar-nav-*` | CSS-only — see file header |
| `components/dialog.css` | `.dialog`, `.dialog-card`, … | CSS-only — see file header |
| `components/button.css` | `.btn`, `.btn-*` | CSS-only — see file header |
| `components/list-row.css` | `.list-rows`, `.list-row*` | CSS-only — see file header |
| `components/tip.css` | `.tip` | Micro-partial `components/tip.html` |
| `components/stream.css` | `.process-toolbar`, `.process-control`, `.status-tag*` | Sort toolbar in `pages/stream.html`; status pills via `status_tag` |
| `components/forms.css` | form controls / auth forms | Auth pages; dialogs that host forms |
| `components/org-admin.css` | org-admin widgets / pickers | `pages/org_admin.html` |
| `components/shared.css` | `.stack`, `.muted`, `.pill`, … | Cross-cutting primitives |

Cluster files (`forms.css`, `org-admin.css`, `shared.css`, `stream.css`) hold domain groups or primitives — not full extractable components.

### CSS-only components

Reused **markup patterns** backed by namespaced CSS, with **no** full Go template partial / view struct. Templates inline the HTML; the CSS file header comment is the markup contract.

**Use when:** reused on 2+ pages; coherent selector family; data varies only in text/attributes (no mode dispatch, no HTMX swap target).

**Do not use when:** Go assembles a stable field set → full component + view struct; or the partial is an HTMX/SSE swap target → `components/{name}.html`.

**Adding one:** create `web/src/styles/components/{name}.css` with a markup-tree header; import from `components.css`; add a row above. Do **not** add `templates/components/{name}.html` unless it graduates (narrow micro-partials such as `status_tag` / `tip` are exceptions).

| Module | Primary classes | Notes |
|--------|-----------------|-------|
| `page-header.css` | `.page-header`, `.page-header-head`, `.page-header-body`, `.page-header-actions`, `.page-header-subtitle` | Markup tree in file header (below) |
| `panel.css` | `.panel`, `.panel-sticky`, `.panel-heading`, `.panel-head-actions`, `.panel-actions`, `.panel-block` | File header |
| `sidebar-nav.css` | `.sidebar-nav`, `.sidebar-nav-link`, `.sidebar-nav-title`, `.sidebar-nav-copy` | File header |
| `dialog.css` | `.dialog`, `.dialog-card`, `.dialog-head`, `.dialog-actions` | File header; page-scoped sizing stays in page CSS |
| `button.css` | `.btn`, variants, sizes, `.btn-icon` | File header |
| `list-row.css` | `.list-rows`, `.list-row`, `.list-row-main`, `.list-row-actions` | File header |
| `tip.css` | `.tip` | Micro-partial `components/tip.html` |
| — | `local_datetime` | Micro-partial only: `components/local_datetime.html` + `web/src/main.js` |

**Page header** — CSS-only. No `PageHeaderView` / `page_header` define. Optional trail: `{{ template "breadcrumbs" .Breadcrumbs }}` with `BreadcrumbsView` (`Current: true` on the last crumb ⇒ `aria-current="page"`; every crumb still has `Href`). When right actions are needed, wrap `page-header-body` + `page-header-actions` in `div.page-header-head`; omit the head wrapper when there are no actions. Process-instance ID under the title uses `.process-header-meta*` in `pages/process.css`, not this component.

```
Heading only:
section.page-header
  nav.breadcrumbs?
  div.page-header-body
    h1                              (optional span[aria-hidden] + span.page-header-subtitle)
    p?

With actions:
section.page-header
  nav.breadcrumbs?
  div.page-header-head
    div.page-header-body
      …
    div.page-header-actions
      button | form | …
```

Root templates still pending migration (`attachment_carousel.html`, `error_banner.html`, `icons.html`, `role_palette_options.html`, `substep_override_editor.html`) live under `server/templates/`. Migrate one at a time.

### Template ↔ CSS index

| Template | Primary CSS | Also uses |
|----------|-------------|-----------|
| `layout.html` | `layout/index.css` | `components/shared.css` |
| Inline page headers | `components/page-header.css` | Optional `breadcrumbs` |
| `components/breadcrumbs.html` | `components/breadcrumbs.css` | — |
| `components/stream_card.html` | `components/stream-card.css` | `components/dialog.css` |
| `components/stream_instance_card.html` | `components/stream-instance-card.css` | `components/stream.css` (via `status_tag`) |
| `components/stream_termination_details.html` | `components/stream-termination-details.css` | `components/shared.css` (`.warning`, `.muted`); `local_datetime` |
| Inline panel / dialog / sidebar-nav / list-row | matching `components/*.css` | `button.css`, page/domain modules as needed |
| Inline rail shell (stream, org_admin) | `layout/grids.css` (`.rail-layout`) | `panel.css` (`.panel-sticky`); org_admin also `sidebar-nav.css` |
| `pages/home.html` | `pages/home.css` | `stream-card`, `dialog`, `stream`, layout |
| `pages/stream.html` | `pages/home.css`, `pages/stream.css` | rail, dialog, panel, instance-card, stream, timeline, role-palette |
| `pages/process.html` | `pages/process.css` | page-header, dialog, substep-shell, timeline, substep-body, termination-details, layout, role-palette |
| `components/stream_step_summary.html` | `components/stream-step-summary.css` | `tip` |
| `components/stream_timeline.html` | `components/stream-timeline.css` | step-summary, substep-body, role-palette |
| `components/dpp_history_step.html` | `pages/dpp.css` (`.dpp-history-*`) | timeline, step-summary, substep-shell/body, role-palette |
| `components/substep_shell.html` | `components/substep-shell.css` | substep-body, tip, role-palette |
| `components/substep_body.html` | `components/substep-body.css` | dialog, forms, role-palette |
| `pages/dpp.html` | `pages/dpp.css` | dpp_history_step + timeline stack, termination-details, role-palette |
| `pages/org_admin.html` | `pages/org-admin-page.css` | rail, dialog, sidebar-nav, panel, org-admin, role-palette |
| `pages/platform_admin.html` | `pages/platform-admin.css` | dialog, shared |
| Auth pages (`login`, `signup`, `invite`, `reset_*`) | `components/forms.css` | button, shared |
| `attachment_carousel.html` | `components/substep-body.css` | — |
| `substep_override_editor.html` | `components/substep-override.css` | — |

### `data-*` contract (templates → CSS / JS)

| Attribute | Set in | Consumed by |
|-----------|--------|-------------|
| `data-theme` | `main.js` on `<html>` | `tokens.css` (`[data-theme="dark"]`) |
| `data-role-palette` | role badges, timeline substeps | `role-palette.css` |
| `data-stream-status` | `status_tag`; `stream.html` section heads | `role-palette.css` (`--stream-color`) |
| `data-progress` (via `style="--progress: …"`) | `stream_instance_card.html` | `.stream-instance-card-progress-fill` |
| `data-org-admin-nav`, `data-org-admin-panel`, `data-org-admin-default-panel`, `data-org-admin-ready` | `org_admin.html` | `pages/org-admin-page.css`, inline script |
| `data-home-nav`, `data-home-panel`, `data-home-filter-select` | `stream.html` | inline script (panel + query sync; select is `--md-down` twin of filter nav) |
| `data-process-id`, `data-workflow-key`, `data-selected-substep`, `data-substep-id` | `process.html` | `main.js` (SSE, accordion) |
| `data-formata-*`, `data-active-role-*` | `substep_body.html` | `main.js` |
| `data-override-url` | `substep_body.html` | `main.js` |
| `data-carousel-*` | `attachment_carousel.html` | `main.js`, `substep-body.css` |
| `data-copy-text`, `data-copy-label`, `data-share-url`, `data-share-label` | `dpp.html` | `main.js` |
| `data-target` | password visibility toggles | `main.js` |
| `data-value`, `data-label`, `data-selected` | org-admin role pickers | `org-admin.css`, inline script |
| `data-role-color` | `role_palette_options.html` | org-admin palette picker script |

Behavior hooks use a `js-*` class prefix alongside `data-*` where needed; do not style `js-*` classes in CSS.

## Theming

- Light/dark mode toggles `data-theme="light|dark"` on `<html>` (`web/src/main.js`).
- Use design tokens (`var(--foreground)`, `var(--card)`, `var(--primary)`, …) — do not hardcode hex/rgb in templates or new CSS.
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

- **`breakpoints.css`** is the **only** file that may define `@custom-media`.
- Modules use `@media (--sm-down) { … }` — do not repeat `@custom-media` or write `@media (width … 640px)` in component/page CSS.
- **PostCSS:** `postcss-custom-media` expands aliases at build time (`web/postcss.config.js`).

**JS sync:** match CSS with the equivalent `matchMedia`. Example: `--md-down` ↔ `matchMedia("(max-width: 767px)")` (as in `org_admin.html`).

### Font tokens

| Token | Value |
|-------|-------|
| `--font-sans` | Inter stack (body, buttons, UI copy, `h2`–`h4`) |
| `--font-display` | Space Grotesk stack (literal `h1` only) |
| `--font-mono` | JetBrains Mono stack (hashes, codes, meta ids) |

Google Fonts load from `server/templates/layout.html`. Do **not** `@import` fonts from `tokens.css`.

### Type scale

Canonical tokens in `tokens.css` (Tailwind-aligned, rem-based). Index ≈ size: `--text-sm` = `0.875rem` = 14px.

| Token | rem | px | Typical use |
|-------|-----|----|-------------|
| `--text-xs` | `0.75rem` | 12 | Captions, pills, compact tags, meta IDs |
| `--text-sm` | `0.875rem` | 14 | Labels, secondary UI, toolbar, form labels, buttons |
| `--text-base` | `1rem` | 16 | Body, inputs |
| `--text-lg` | `1.125rem` | 18 | Card/dialog titles |
| `--text-xl` | `1.25rem` | 20 | Section headings (`h2`; nested `h3`) |
| `--text-2xl` | `1.5rem` | 24 | Large titles |
| `--text-3xl` | `1.875rem` | 30 | Page titles (`h1` default) |

**Line-height:** `--leading-none` `1`, `--leading-tight` `1.25`, `--leading-normal` `1.5`, `--leading-relaxed` `1.625`.

**Weight:** `--font-normal` `400`, `--font-medium` `500`, `--font-semibold` `600`, `--font-bold` `700` (Inter faces loaded to match).

**Heading defaults** in `reset.css`: `h1` → `--text-3xl` + `--font-display`; `h2` → `--text-xl`; `h3` → `--text-lg`; `h4` → `--text-base`; all `--font-semibold` + `--leading-tight`. Prefer tokens over raw `font-size`.

**Controls:** buttons use `button.css` (default height `--btn-height` / 36px). Form labels `--text-sm`; inputs `--text-base`.

### Spacing scale

Canonical `--space-N` in `tokens.css` on a 4px grid (`--space-4` = `1rem` = 16px). Scale runs `--space-1` … `--space-12`. Prefer `u-*` utilities or component classes over raw `px` in templates.

**Intentional `px` exceptions:** layout dimensions, `border-radius`, `border-width`, scroll anchors, and composite padding with off-scale values. Use type tokens for `font-size`.

### Elevation and backdrop tokens

`--shadow`, `--shadow-elevated`, `--shadow-dropdown`, `--backdrop-modal`, `--backdrop-fullscreen`, `--inset-highlight` — see `tokens.css`. New shadow/backdrop tints get a semantic token there, not an inline `rgb()`.

## Role palette

Role badge colors resolve at runtime from **Appwrite team prefs**, not workflow YAML or inline CSS.

```
Appwrite team prefs          Backend (roleMetaIndex)          Templates + CSS
{ slug, name, palette }  →   (orgSlug, roleSlug) → key   →   data-role-palette="blue"
                                                                      ↓
                                                             role-palette.css → --role-*
```

| Layer | Responsibility |
|-------|----------------|
| Appwrite | Canonical `{ slug, name, palette }` named key |
| Backend | Org-scoped `(orgSlug, roleSlug)` → palette key; never emits `var(--role-*-*)` to templates |
| Workflow YAML | Slug/org/name for validation; `color` fields ignored by Go |
| Templates | `data-role-palette="{{ .Palette }}"` on `.role-pill` / timeline substeps |
| `tokens.css` + `role-palette.css` | Appearance map |

**Storage:** writes persist `palette` only; reads prefer `palette`, with legacy `color` CSS-var strings falling back via `rolePaletteKeyFromStyle()`. `GET /api/catalog` returns `palette` per role. Unknown/missing → `"fallback"`.

**Lookup:** step with `organization:` → `(stepOrg, roleSlug)`; else first org containing the slug; else `"fallback"`. Symbols: `roleMetaIndex`, `roleMetaFor`, `rolePaletteKeyFromStyle`.

Named palette keys live in `rolePaletteStyles` / `rolePaletteKeys` (Go) and `role-palette.css`. `TestRolePaletteKeysMatchCSS` keeps them in sync.

```html
<span class="role-pill" data-role-palette="{{ .Palette }}">{{ .Label }}</span>
{{ template "status_tag" .Status }}
```

`status_tag` renders `<span class="status-tag status-tag-compact" data-stream-status="…">` (pipeline = status string). `.status-tag` uses `var(--stream-color)`. Static presets: `.pill-accent`, `.pill-panel`.

## Utilities (`u-*`)

Generic helpers in `utilities.css`. Add a utility when the same spacing/typography pattern appears in multiple templates. Prefer component classes for domain-specific patterns.

| Class | Effect |
|-------|--------|
| `u-mx-auto` | `margin-inline: auto` |
| `u-max-w-prose` | `max-width: 65ch` |
| `u-max-w-7xl` | `max-width: 80rem` |
| `u-m-0` | `margin: 0` |
| `u-mb-2` / `u-mb-4` / `u-mb-5` | bottom margin via `--space-*` |
| `u-ml-1` | `margin-left: var(--space-1)` |
| `u-pre-line` | `white-space: pre-line` |
| `u-text-xs` / `u-text-sm` / `u-text-base` / `u-text-lg` | type scale (`u-text-sm` also semibold) |
| `u-flex` / `u-flex-center` | flex / flex + center |
| `u-gap-2` / `u-gap-4` / `u-gap-12` | gap via `--space-*` |
| `u-divider` / `u-divider-5` / `u-divider-10` | `<hr>` with vertical `--space-*` margin |
| `u-divider-flush` | `<hr>` with `margin: 0` |
| `u-text-danger` | `color: var(--destructive)` |

`.stack.u-gap-2` and `.stack.u-gap-4` override the default stack gap (`var(--space-6)`). New `u-*` utilities should include a one-line justification in the PR.

## Inline `style=` in templates

**Do not** use inline styles for static layout, spacing, or typography.

**Allowed** — runtime values from Go:

| Pattern | Example | Consumer |
|---------|---------|----------|
| Progress width | `style="--progress: {{ .Percent }}%;"` | `.stream-instance-card-progress-fill` |

All other dynamic theming uses `data-*` (`data-role-palette`, `data-stream-status`), not inline custom properties.

## Build and lint

```bash
cd web && npm run build   # → web/dist/assets/main.css (served at /static/)
task css:lint
```

`task css:lint` runs:

1. **Template inline styles** — `deployment/scripts/check-template-inline-styles.sh`
2. **Breakpoint guard** — `deployment/scripts/check-css-breakpoints.sh` (no literal `@media (width … px)` outside `breakpoints.css`)
3. **stylelint** — `web/src/styles/**/*.css` (no hex/rgb outside `tokens.css`, no new `!important`)

## Adding new UI

1. Check for an existing component class, CSS-only module, or utility.
2. If spacing/typography repeats across templates, add a `u-*` utility.
3. If domain-specific, add a component or page class.
4. Use tokens for colors; never hardcode hex in templates.
5. Run `task css:lint` before opening a PR.

## Out of scope

- **Formata embed** shadow-DOM styling in `web/src/main.js` (`!important` overrides).
- **New palette key** — update `rolePaletteStyles`, `role-palette.css`, and the org-admin palette picker (not Go string passthrough to templates).
