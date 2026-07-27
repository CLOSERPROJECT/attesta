# Public landing: tokens + light/dark theming

Date: 2026-07-27  
Scope: `web/src/styles/tokens.css`, new `category-palette.css`, `web/src/styles/pages/public-home.css`, `server/templates/pages/public_home.html`, `docs/css.md`, `web/src/styles.css` (import order)  
Out of scope: Component extraction, Figma layout/spacing QA, live stream catalog wiring, Vite-down unstyled fallback, new category keys beyond `passport`

## Goal

Make the public homepage (`/`) follow Attesta’s shared light/dark theme tokens. Remove the parallel `--landing-*` palette. Introduce a category-color system (first key: `passport`) modeled on role palettes, ready for more keys later.

## Decisions

1. **Fidelity A:** Dark landing matches the app dark theme (green-slate), not Figma `landing_7` blue-slate. Accept hue drift from the Figma dark frame.
2. **No `--landing-*`:** Delete the block from `tokens.css`. Landing CSS uses existing semantic tokens only (plus the new category family).
3. **Theme follows `data-theme`:** Remove `.public-home { color-scheme: dark; }`. Do not force dark on the marketing page.
4. **Category colors via `data-category`:** Same shape as `data-role-palette` → `--role-chroma`. Markup sets `data-category="passport"`; CSS maps to `--category-color`.
5. **Ship only `passport` now.** Other stream categories (compliance, supply, …) get keys later without changing badge consumers.
6. **CTAs use button semantics:** Figma’s pale mint (`--landing-primary`) is not Attesta `--primary`. Filled CTAs → `--primary` / `--primary-foreground`; outline → `--border` / `--foreground` (or existing button classes where applicable).
7. **Borders never use `--background`:** Dark `--background` is a gradient; use `--border` for edges.

## Token mapping (replace `--landing-*`)

| Remove | Use instead | Notes |
|--------|-------------|--------|
| `--landing-bg` | `--background` | Page canvas |
| `--landing-surface` | `--card` | Header, cards, panels |
| `--landing-muted` | `--muted` | Inset wells / chips |
| `--landing-secondary` | `--secondary` | Secondary wells |
| `--landing-foreground` | `--foreground` | Body / headings |
| `--landing-text-muted` | `--muted-foreground` | Nav, captions |
| `--landing-text-secondary` | `--muted-foreground` | No third text tier |
| `--landing-border` | `--border` | All borders |
| `--landing-shadow` | `--shadow` | Card elevation |
| `--landing-accent` / `-soft` | `--primary` | Accent green |
| `--landing-primary` | `--primary` (+ foreground) | CTA fill — semantic remap, not 1:1 color |
| `--landing-passport` | `--category-passport` via `--category-color` | See below |

Keep shared type/spacing tokens as-is (`--text-4xl`, `--text-5xl`, fonts, `--space-*`).

## Category palette

### Tokens (`tokens.css`)

On `:root` only (like roles — theme via `light-dark()`, not `[data-theme]` overrides):

```css
--category-passport: light-dark(#1a7ab8, #48baf9);
```

- Dark leg: Figma passport `#48baf9`.
- Light leg: `#1a7ab8` (darker pair for contrast on light surfaces). Adjust only if badge contrast fails visual check.

### Map file (`category-palette.css`)

New module, loaded next to `role-palette.css` in `web/src/styles.css`:

```css
[data-category="passport"] {
  --category-color: var(--category-passport);
}

[data-category="fallback"],
.public-home-badge:not([data-category]) {
  --category-color: var(--muted-foreground);
}
```

### Markup

Replace class-driven passport color:

```html
<span class="public-home-badge" data-category="passport">Product Passport</span>
```

Drop `.public-home-badge-passport` (or leave as a no-op removed in the same change).

### Consumer CSS

Badge (and later category chips) use:

```css
background: var(--category-color);
border-color: var(--category-color);
color: var(--card);
```

Filled badge text uses `--card` (same invert pattern as today’s `--landing-surface` on the passport chip). Do not hardcode `#48baf9` in page CSS.

## Files to change

| File | Change |
|------|--------|
| `web/src/styles/tokens.css` | Delete `--landing-*`; add `--category-passport` |
| `web/src/styles/category-palette.css` | New; `data-category` → `--category-color` |
| `web/src/styles.css` | Import `category-palette.css` after `role-palette.css` |
| `web/src/styles/pages/public-home.css` | Swap all `--landing-*`; remove forced `color-scheme: dark`; badge uses `--category-color` |
| `server/templates/pages/public_home.html` | `data-category="passport"` on passport badges |
| `docs/css.md` | Document `data-category` in shared `data-*` table; note category palette layer |

## Verification

- `task css:lint` passes (no hex/rgb outside `tokens.css`).
- `npm run build` succeeds.
- Manual: toggle `data-theme` light/dark on `/` — surfaces, text, CTAs, and passport badge all flip; no remaining `--landing-*` references (`rg --landing-`).

## Explicit non-goals (later)

- Extracting landing partials / UI components
- Adding colors for compliance, supply, workflow, data, people, recycling, materials
- Matching Figma blue-slate dark canvas
- Fixing Vite-down unstyled hosting
