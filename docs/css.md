# CSS style guide (main Attesta app)

Rules for styling the server-rendered Attesta UI.

**Scope:** `web/src/styles/`, `server/templates/**/*.html`. Formata embed and Formata Builder (`formata-arch/`) are out of scope.

**Extract / place components:** `.agents/skills/attesta-ui-components/SKILL.md`.  
**Token values:** open `web/src/styles/tokens.css` (and `utilities.css` for `u-*`) — this doc does not mirror those catalogs.

## Layer stack

Load order from `web/src/styles.css`:

1. `breakpoints.css` — sole `@custom-media` definitions  
2. `tokens.css` — `:root`, `[data-theme="dark"]`, type/spacing tokens  
3. `role-palette.css` — `data-role-palette` / `data-stream-status` maps  
4. `reset.css`  
5. `utilities.css` — `u-*`  
6. `layout/index.css` — chrome, grids (`.rail-layout`), responsive shell  
7. `components.css` — barrel; live import list is the file itself  
8. `pages.css` — barrel; live import list is the file itself  

**Placement:** token → utility → layout → component → page. A selector lives in exactly one layer.

**Responsive:** co-locate `@media (--…)` at the **bottom** of the owning module. Shared chrome/grid breakpoints belong in `layout/` (`grids.css`, `responsive.css`).

## Stem naming

Primary CSS for a template is the same **stem** under `components/` or `pages/` (underscore → kebab):

- `components/stream_card.html` → `components/stream-card.css` (classes `.stream-card-*`)
- `pages/process.html` → `pages/process.css` (classes `.process-*`)

If there is no matching CSS file, the markup is **CSS-only** (inline in the page; contract in the CSS file header) or a **cluster** (`shared.css`, `forms.css`, `org-admin.css`, `stream.css`).

Import new component/page modules from the matching barrel (`components.css` / `pages.css`).

### Exceptions (not inventable from the stem)

- `pages/home.css` also styles stream dashboard nav panels used by `pages/stream.html`
- `pages/org-admin-page.css` ↔ `pages/org_admin.html` (page shell); widgets/pickers stay in `components/org-admin.css`
- `components/stream.css` is a cluster (sort toolbar, `.status-tag*` via `status_tag`) — not paired 1:1 with a `stream_*.html` component
- `components/dpp_history_step.html` styles live under `pages/dpp.css` (`.dpp-history-*`)
- `attachment_carousel.html` (templates root) → `components/substep-body.css`
- `substep_override_editor.html` (templates root) → `components/substep-override.css`
- Auth pages (`login`, `signup`, `invite`, `reset_*`) → `components/forms.css` (no per-page CSS module)

Root templates still pending migration (`error_banner.html`, `icons.html`, `role_palette_options.html`, …) live under `server/templates/`. Migrate one at a time when a task needs them.

## CSS-only components

Reused markup + namespaced CSS, **no** full Go template partial / view struct. Inline HTML in pages; the CSS file header is the markup contract.

**Use when:** 2+ pages; coherent selector family; no mode dispatch / HTMX swap target.  
**Otherwise:** full component + view struct, or a cluster file.

**Add one:** create `components/{name}.css` with a markup-tree header; import from `components.css`. Do not add `templates/components/{name}.html` unless it graduates (`status_tag`, `tip`, `local_datetime` are the narrow micro-partial exceptions).

**Page header** (worked example): CSS-only — no `PageHeaderView`. Optional trail via full `breadcrumbs` (`Current: true` on last crumb; every crumb still has `Href`). With actions, wrap body + actions in `.page-header-head`; omit the head when there are no actions. Process-instance ID under the title is `.process-header-meta*` in `pages/process.css`, not page-header.

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

Other CSS-only modules (`panel`, `dialog`, `sidebar-nav`, `button`, `list-row`, …): read the matching file header. Page-scoped dialog sizing stays in page CSS (e.g. `#stream-preview-dialog` in `pages/stream.css`). Destructive dialog titles stack `.dialog-title u-text-danger`.

## Theming

- Light/dark: `data-theme="light|dark"` on `<html>` (`web/src/main.js`).
- Use tokens (`var(--foreground)`, `var(--card)`, `var(--primary)`, …) — no hardcoded hex/rgb in templates or new CSS.
- Role / stream hues use `light-dark()` in `:root`; `[data-theme="dark"]` keeps only non-role overrides.
- Token names follow shadcn-style `{role}` + `{role}-foreground` pairs.

### Breakpoints

Tailwind v3 widths; aliases only in `breakpoints.css`:

| Tailwind | Width | up | down |
|----------|-------|----|------|
| sm | 640px | `--sm-up` | `--sm-down` |
| md | 768px | `--md-up` | `--md-down` |
| lg | 1024px | `--lg-up` | `--lg-down` |
| xl | 1280px | `--xl-up` | `--xl-down` |
| 2xl | 1536px | `--2xl-up` | — |

Modules use `@media (--sm-down) { … }`. Never redefine `@custom-media` or write `@media (width … 640px)` in component/page CSS. PostCSS (`postcss-custom-media`) expands aliases at build time.

**JS sync:** `--md-down` ↔ `matchMedia("(max-width: 767px)")` (as in `org_admin.html`). Keep new JS literals aligned with the table.

### Fonts and type

| Token | Use |
|-------|-----|
| `--font-sans` | Body, buttons, UI copy, `h2`–`h4` (Inter) |
| `--font-display` | Literal `h1` only (Space Grotesk) |
| `--font-mono` | Hashes, codes, meta ids (JetBrains Mono) |

Google Fonts load from `layout.html` — do not `@import` fonts from `tokens.css`.

Type / weight / leading tokens live in `tokens.css`. Heading defaults in `reset.css`: `h1` → `--text-3xl` + `--font-display`; `h2` → `--text-xl`; `h3` → `--text-lg`; `h4` → `--text-base`; all `--font-semibold` + `--leading-tight`. Prefer tokens over raw `font-size`.

Buttons: `button.css` (default height `--btn-height`). Form labels `--text-sm`; inputs `--text-base`.

### Spacing and elevation

`--space-1` … `--space-12` on a 4px grid in `tokens.css`. Prefer `u-*` or component classes over raw `px` in templates.

**`px` exceptions:** layout dimensions, `border-radius`, `border-width`, scroll anchors, composite off-scale padding. Use type tokens for `font-size`.

Shadows / backdrops (`--shadow*`, `--backdrop-*`, `--inset-highlight`): define new tints as tokens in `tokens.css`, not inline `rgb()`.

## Role palette

Runtime colors come from **Appwrite team prefs**, not workflow YAML or inline CSS.

```
Appwrite { slug, name, palette }  →  backend (orgSlug, roleSlug)  →  data-role-palette="blue"
                                                                         ↓
                                                                role-palette.css → --role-*
```

| Layer | Responsibility |
|-------|----------------|
| Appwrite | Canonical named `palette` key |
| Backend | Resolve `(orgSlug, roleSlug)` → key; never emit `var(--role-*-*)` to templates |
| Workflow YAML | Slug/org/name only; `color` ignored by Go |
| Templates | `data-role-palette="{{ .Palette }}"` on `.role-pill` / timeline substeps |
| `tokens.css` + `role-palette.css` | Appearance |

Writes persist `palette` only; legacy `color` CSS-var strings fall back via `rolePaletteKeyFromStyle()`. `GET /api/catalog` returns `palette`. Unknown → `"fallback"`. Lookup: step `organization:` → `(stepOrg, roleSlug)`; else first org containing the slug. Keys stay in sync via `TestRolePaletteKeysMatchCSS`.

```html
<span class="role-pill" data-role-palette="{{ .Palette }}">{{ .Label }}</span>
{{ template "status_tag" .Status }}
```

`status_tag` (pipeline = status string) sets `data-stream-status`; `.status-tag` uses `var(--stream-color)`. Static presets: `.pill-accent`, `.pill-panel`.

## Shared `data-*` (CSS theming)

| Attribute | Consumer |
|-----------|----------|
| `data-theme` | `tokens.css` |
| `data-role-palette` | `role-palette.css` |
| `data-stream-status` | `role-palette.css` → `--stream-color` |
| `style="--progress: …%"` | `.stream-instance-card-progress-fill` only |

Page-local hooks (`data-org-admin-*`, `data-home-*`, `data-process-id`, `data-formata-*`, carousel/copy/share, …) stay next to their template / `main.js` — do not catalog them here.

Behavior hooks may use a `js-*` class alongside `data-*`; **never style `js-*` in CSS**.

## Utilities (`u-*`)

Generic, domain-agnostic helpers in `utilities.css`. Add a `u-*` when the same spacing/typography pattern appears in multiple templates; prefer component classes for domain patterns. New utilities get a one-line PR justification.

`.stack.u-gap-*` overrides the default stack gap (`var(--space-6)`).

## Inline `style=` in templates

No inline styles for static layout, spacing, or typography.

**Allowed** (runtime from Go):

| Pattern | Example | Consumer |
|---------|---------|----------|
| Progress width | `style="--progress: {{ .Percent }}%;"` | `.stream-instance-card-progress-fill` |

All other dynamic theming uses `data-*`, not inline custom properties.

## Build and lint

```bash
cd web && npm run build   # → web/dist/assets/main.css (served at /static/)
task css:lint
```

`task css:lint`:

1. Template inline styles — `deployment/scripts/check-template-inline-styles.sh`
2. Breakpoint guard — no literal `@media (width … px)` outside `breakpoints.css`
3. stylelint — no hex/rgb outside `tokens.css`, no new `!important`

## Adding new UI

1. Prefer an existing component, CSS-only module, or `u-*`.
2. New domain styles → component or page module (stem rule + barrel import).
3. Tokens for color; run `task css:lint` before the PR.

## Out of scope

- Formata embed shadow-DOM styling in `web/src/main.js`
- New palette key → `rolePaletteStyles`, `role-palette.css`, and the org-admin picker (not Go string passthrough)
