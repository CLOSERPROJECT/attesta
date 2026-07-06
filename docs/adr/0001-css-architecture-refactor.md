# ADR-0001: CSS Architecture Refactor for the Main Attesta App

- **Status:** Proposed
- **Date:** 2026-07-06
- **Scope:** Main Attesta application only — `web/src/styles/`, `web/src/styles.css`, `web/src/main.js` (theme handling only), and `server/templates/*.html`
- **Out of scope:** `formata-form` shadow-DOM overrides, `server/cmd/server/formata_builder.go`, and the entire `server/cmd/server/formata-arch/` sub-application

## Context

The main Attesta UI is server-rendered Go templates styled by a Vite-built CSS bundle. Over time, styling has accumulated in a single large stylesheet with repeated patterns pushed into inline `style` attributes in templates.

### Current layout

```
web/src/styles.css          # entry: four @imports
├── base.css                # ~2,950 lines — tokens, components, pages
├── phone.css               # ~160 lines — max-width breakpoints
├── tablet.css              # ~30 lines
└── desktop.css             # ~20 lines
```

Templates in `server/templates/` reference semantic class names (`.panel`, `.timeline-step`, `.role-pill`, etc.) but also rely heavily on inline styles for spacing, typography, and layout tweaks.

### Quantified pain (main app only)

| Signal | Count | Notes |
|--------|------:|-------|
| Lines in `base.css` | ~2,950 | Monolithic; no module boundaries |
| Class selectors in `base.css` | ~425 | Mixed naming; no documented taxonomy |
| `!important` in main CSS | 1 | `.hero button:hover` overrides global `button` |
| Inline `style=` in templates | 77 | Across 11 of 18 template files |
| Dynamic inline custom props | 16 | `--pill-bg`, `--border`, `--dept-color`, `--stream-*` |
| Hardcoded color in inline | 1 | `signup.html` uses `#a91919` instead of a token |
| Design-token blocks | 2 | `:root` + `[data-theme="dark"]`, each ~75 lines |

### Categories of inline styles

1. **Dynamic theming (keep, but formalize)** — role pills and stream status badges set CSS custom properties from Go template data:
   ```html
   style="--pill-bg: {{ .Color }}; --border: {{ .Border }};"
   ```
   Components (`.role-pill`, `.pill`) already consume these variables.

2. **Repeated static utilities (remove)** — margins, flex layout, `white-space: pre-line`, font sizes. These appear dozens of times with no shared class.

3. **Component bypass (remove)** — e.g. `layout.html` footer has `class="site-footer"` but all visual rules (padding, border, typography) are inline; `.site-footer` only sets `margin-top: auto`.

4. **Template-conditional inline (refactor)** — pagination buttons in `stream.html` / `platform_admin.html` embed Go conditionals inside `style=""` for disabled state.

### What works today

- **CSS custom properties** for light/dark theming via `[data-theme="dark"]` on `<html>`, toggled from `web/src/main.js`.
- **Semantic component classes** — `.panel`, `.stack`, `.timeline-step`, `.action-form`, etc. are used consistently across templates.
- **Responsive split** — `phone.css`, `tablet.css`, `desktop.css` hold breakpoint overrides; the pattern is sound but underused.
- **Role palette** — 22 named palette slots (`--role-red-bg`, etc.) power admin UI and badges; backend exposes values via `template.CSS` and `cssValue()`.

### What does not work

- **No single place to find "how do I add spacing?"** — developers copy inline `style` from sibling templates.
- **`base.css` is unnavigable** — tokens, layout, components, and page-specific rules are interleaved.
- **Utility gaps** — `.stack` has a fixed `gap: 24px`; templates override with inline `gap: 8px` / `gap: 16px` instead of modifier classes.
- **Token duplication** — every role color exists twice (light + dark `:root` blocks), ~88 custom properties for role badges alone.
- **No documented conventions** — nesting is used (`.topbar-pa { .brand { … } }`) without a stated standard; no BEM, no utility layer, no component documentation.

## Decision

Refactor the main Attesta CSS into a **layered, token-first architecture** with **small utility classes** for repeated layout/spacing patterns, while keeping **server-rendered templates** and the **existing Vite CSS pipeline** (no Tailwind, no CSS-in-JS, no new build tools).

### 1. Split `base.css` into layers

Replace the monolith with imported modules under `web/src/styles/`:

```
web/src/styles.css
├── tokens.css        # :root + [data-theme="dark"] custom properties only
├── reset.css         # *, body, a, button, heading defaults
├── utilities.css     # single-purpose layout/spacing/typography helpers
├── layout.css        # .page, .topbar, .stack, grids, .site-footer
├── components.css    # .panel, .pill, .timeline-*, .action-*, .role-*, forms
├── pages.css         # page-scoped compositions (home, process, org-admin, dpp)
├── phone.css         # unchanged role; may shrink as utilities absorb rules
├── tablet.css
└── desktop.css
```

**Rule:** A selector lives in exactly one layer. Tokens never appear outside `tokens.css`. Utilities never encode component semantics.

### 2. Introduce a minimal utility vocabulary

Add a small, documented set of utility classes in `utilities.css`. Prefix with `u-` to distinguish from semantic components.

| Utility | Purpose | Replaces (examples) |
|---------|---------|---------------------|
| `u-m-0` | `margin: 0` | 6+ inline instances |
| `u-mb-8` | `margin-bottom: 8px` | footer paragraphs |
| `u-mb-16` | `margin-bottom: 16px` | section spacing |
| `u-pre-line` | `white-space: pre-line` | action_list, dpp |
| `u-text-sm` | `font-size: 12px` | muted captions |
| `u-text-label` | `font-size: 14px; font-weight: bold` | attachment headers |
| `u-flex` / `u-flex-center` | flex + align | process, dpp layouts |
| `u-gap-8` / `u-gap-16` | gap modifiers | stack overrides |
| `u-divider` | styled `<hr>` | repeated `margin: 24px 0` rules |

**Rule:** Utilities are generic. If a pattern is tied to a domain concept (e.g. `.role-pill-row`), it stays a component class, not a utility.

### 3. Formalize dynamic theming via custom properties

Keep inline `style` **only** where values are computed at render time and cannot be known statically:

- Role pill colors: `style="--pill-bg: …; --border: …;"`
- Stream status colors: `style="--stream-status: …"` (refactor current `--stream-{{ .Status }}` pattern to a single property)
- Department/timeline colors: `style="--dept-color: …; --dept-border: …"`

**Do not** use inline styles for static layout, spacing, or typography.

Optional follow-up (not required for initial refactor): replace inline custom props with `data-pill-bg` / attribute selectors if we want stricter CSP or HTML linting.

### 4. Complete component classes before adding utilities

When a template uses inline styles for something a component should own, extend the component class first.

**Example:** `.site-footer` should own padding, border-top, background, and `.site-footer p` should own the muted small-text block — removing all footer inline styles from `layout.html`.

### 5. Keep the existing theming mechanism

- Light/dark toggle stays on `document.documentElement.dataset.theme`.
- Token names (`--ink`, `--panel`, `--accent`, etc.) are **not renamed** in the first pass to avoid a large blast radius.
- `web/src/main.js` theme logic is unchanged; Formata-related code in that file is explicitly out of scope and must not be moved or refactored as part of this ADR.

### 6. No new CSS tooling

- Continue using plain CSS with native nesting (already in use; Vite passes it through).
- No PostCSS plugins, Tailwind, CSS modules, or Sass unless a later ADR justifies them.
- Build output remains `web/dist/assets/main.css` served at `/static/`.

### 7. Template migration is incremental

Refactor templates file-by-file in priority order:

1. `layout.html` — footer, topbar platform-admin inline colors
2. `action_list.html` — highest density of repeated static inline patterns
3. `dpp.html`
4. `process.html`, `org_admin.html`, `platform_admin.html`, `stream.html`
5. Remaining templates (`home.html`, `signup.html`, etc.)

Each template PR should leave **zero static inline styles** except documented dynamic custom-property cases.

## Consequences

### Positive

- New UI work has clear placement rules (token → utility → component → page).
- Template diffs become smaller and more readable.
- Dark mode and role palette changes stay centralized in `tokens.css`.
- `base.css` deletion reduces merge conflicts and review fatigue.
- No runtime or build-pipeline changes; low operational risk.

### Negative / trade-offs

- Utility classes add HTML verbosity (`class="muted u-m-0 u-pre-line"` vs one inline `style`).
- A `u-*` layer can grow unchecked without discipline — utilities must be added via PR review, not ad hoc.
- Splitting `base.css` is a large one-time diff; should land as a standalone "move only" commit before behavior changes.
- Dynamic inline custom properties remain, which some linters will still flag.

### Risks and mitigations

| Risk | Mitigation |
|------|------------|
| Visual regressions across 18 templates | Migrate one template at a time; rely on existing template tests (`*_template_test.go`) and manual smoke on key pages |
| Utility sprawl | Cap initial set to ~15 classes; new utilities require justification in PR description |
| `base.css` split breaks specificity | Move-only commit first; no selector changes during split |
| Role palette token bloat | Defer palette consolidation to a follow-up ADR; out of scope for pass 1 |

## Implementation plan

### Pass 0 — Structure (no visual changes)

1. Create `tokens.css`, `reset.css`, `utilities.css`, `layout.css`, `components.css`, `pages.css`.
2. Move selectors from `base.css` into layers verbatim.
3. Update `styles.css` imports; delete `base.css`.
4. Run `npm run build` in `web/`; confirm `web/dist` output.

### Pass 1 — Utilities + layout footer

1. Add initial `u-*` utilities listed above.
2. Migrate `layout.html` footer and topbar platform-admin inline styles.
3. Extend `.site-footer` with full styles.

### Pass 2 — High-traffic templates

1. `action_list.html` — replace static inline with utilities/component classes.
2. `dpp.html`, `process.html`.

### Pass 3 — Admin and stream templates

1. `org_admin.html`, `platform_admin.html`, `stream.html`, `home.html`.
2. Fix `signup.html` hardcoded `#a91919` → `var(--danger)` or `.text-error`.

### Pass 4 — Cleanup and guardrails

1. Add a short `docs/css.md` style guide (layer rules, when to use utilities, dynamic prop convention).
2. Add a CI or `task` check that fails on new `style=` in templates except allowlisted patterns (`--pill-bg`, `--border`, `--dept-`, `--stream-`).
3. Update `AGENTS.md` with a pointer to `docs/css.md`.

## Alternatives considered

### A. Adopt Tailwind for templates

**Rejected.** Would require rewriting all Go templates, adding PostCSS/Tailwind to `web/`, and creates a second styling paradigm next to the existing semantic classes. High cost, inconsistent with current patterns.

### B. CSS modules or scoped styles per template

**Rejected.** Go `html/template` has no bundler integration for per-template CSS. Would need a naming convention hack or inline `<style>` blocks per page — worse than current state.

### C. Single `base.css` with better comments only

**Rejected.** Does not address inline style proliferation or utility gaps. Defers the problem.

### D. Inline-everything with CSS custom properties only

**Rejected.** Already partially true and is the source of inconsistency. Static layout does not belong inline.

## Success criteria

- [ ] `base.css` removed; styles live in named layers.
- [ ] Static inline `style=` count in `server/templates/` reduced from 77 to ≤ 16 (dynamic custom props only).
- [ ] Zero hardcoded hex/rgb colors in template inline styles.
- [ ] `docs/css.md` documents layer rules and utility vocabulary.
- [ ] `npm run build` and existing Go template tests pass.
- [ ] No changes to Formata / Formata Builder styling files.

## References

- Main stylesheet entry: `web/src/styles.css`
- Theme toggle: `web/src/main.js` (`applyTheme`, `[data-theme="dark"]`)
- Template CSS helpers: `cssValue()` in `server/cmd/server/main.go`
- Role palette: `rolePaletteStyles` in `server/cmd/server/main.go`
- Prior audit: conversation 2026-07-06 (scoped to main app)
