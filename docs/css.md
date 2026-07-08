# CSS style guide (main Attesta app)

Source of truth for styling the server-rendered Attesta UI: CSS architecture, theming, role palettes, template rules, and lint.

**Scope:** `web/src/styles/`, `server/templates/*.html`. Formata embed and Formata Builder (`formata-arch/`) are out of scope.

## Layer stack

Styles load in this order from `web/src/styles.css`:

| Layer | File | Contains |
|-------|------|----------|
| Tokens | `tokens.css` | `:root`, `[data-theme="dark"]`, font import |
| Role palette | `role-palette.css` | `data-role-palette` and `data-stream-status` attribute maps |
| Reset | `reset.css` | `*`, `body`, `a`, `button`, heading defaults, focus rings, reduced motion |
| Utilities | `utilities.css` | `u-*` spacing/typography/layout primitives |
| Layout | `layout.css` | Page chrome: topbar, nav, stack, grids, footer |
| Components | `components.css` | Barrel importing `components/*.css` (timeline, actions, forms, org-admin, stream, shared) |
| Pages | `pages.css` | Page-specific compositions (DPP, stream, admin) |
| Breakpoints | `phone.css`, `tablet.css`, `desktop.css` | Media-query overrides |

**Placement rule:** token → utility → component → page. A selector lives in exactly one layer.

## Theming

- Light/dark mode toggles `data-theme="light|dark"` on `<html>` (see `web/src/main.js`).
- Use design tokens (`var(--ink)`, `var(--panel)`, `var(--accent)`, etc.) — do not hardcode hex/rgb in templates or new CSS.
- Role hue and stream status tokens use CSS `light-dark()` in `:root`; `[data-theme="dark"]` keeps only non-role overrides.
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

A few one-off compositional `rgba()` values remain outside `tokens.css`. Do not copy these into new CSS — prefer tokens or `color-mix(in srgb, var(--*) …%)`.

| File | Value | Use |
|------|-------|-----|
| `pages.css` | `rgba(255, 255, 255, 0.04)` | Inset top highlight on `.platform-admin-search-field` gradient |
| `components/org-admin.css` | `rgba(0, 0, 0, 0.12)` | Elevated dropdown shadow on `.roles-picker-menu` |
| `components/org-admin.css` | `rgba(16, 26, 20, 0.48)` | Tinted backdrop on `.manage-dialog::backdrop` |
| `components/actions.css` | `rgba(16, 26, 20, 0.16)` | Elevated shadow on attachment carousel nav buttons |
| `components/timeline.css` | `rgba(0, 0, 0, 0.42)` | Fullscreen modal backdrop on `.substep-override-modal::backdrop` |

## Role palette

Role badge colors are resolved at runtime from **Appwrite team prefs**, not from workflow YAML or inline CSS values.

### Data flow

```
Appwrite team prefs          Backend (roleMetaIndex)          Templates + CSS
{ slug, name, palette }  →   (orgSlug, roleSlug) → key   →   data-role-palette="blue"
                                                                      ↓
                                                             role-palette.css → --role-*-bg
```

| Layer | Responsibility |
|-------|----------------|
| Appwrite | Canonical store: `{ slug, name, palette }` where `palette` is a named key (`blue`, `emerald`, …) |
| Backend | Resolves org-scoped `(orgSlug, roleSlug)` to a palette key; never emits `var(--role-*-*)` strings to workflow templates |
| Workflow YAML | Slug/org/name for validation only; `color` / `border` fields are ignored by Go |
| Templates | Set `data-role-palette="{{ .Palette }}"` on `.role-pill` and timeline substeps |
| `tokens.css` + `role-palette.css` | Single source of appearance; maps palette key → `--role-*-bg` tokens |

### Storage and API

- **Writes** persist `palette` only (no `color` / `border`).
- **Reads** use `palette` when present; legacy rows with `color` / `border` CSS var strings fall back to `rolePaletteKeyFromStyle()`.
- **`GET /api/catalog`** returns `palette` per role (no `color` / `border`).
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

17 named palette keys (`red`, `orange`, `amber`, … `rose`) are defined in `rolePaletteStyles` and mapped in `role-palette.css`.

Role pills on `process`, `action_list`, `dpp`, and `org_admin` use:

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

All other dynamic theming uses `data-*` attributes (`data-role-palette`, `data-stream-status`), not inline custom properties.

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

Runs two checks:

1. **Template inline styles** — fails on disallowed `style=` attributes in `server/templates/` (allowed patterns listed above).
2. **stylelint** — CSS rules on `web/src/styles/**/*.css` (no hex/rgb outside `tokens.css`, no new `!important`).

## Adding new UI

1. Check for an existing component class or utility.
2. If spacing/typography repeats across templates, add a `u-*` utility.
3. If the pattern is domain-specific, add a component or page class.
4. Use tokens for colors; never hardcode hex in templates.
5. Run `task css:lint` before opening a PR.

## Out of scope / known gaps

- **Formata embed** shadow-DOM styling in `web/src/main.js` (`!important` overrides) — separate effort; not covered here.
- **Adding a new palette key** requires updates to `rolePaletteStyles`, `role-palette.css`, and the org-admin palette picker — not Go string passthrough to templates.
