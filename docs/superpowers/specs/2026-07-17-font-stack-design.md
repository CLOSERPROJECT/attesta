# Font stack redesign

Date: 2026-07-17  
Scope: Main Attesta app (`web/src/styles/`, `server/templates/layout.html`, `docs/css.md`)  
Out of scope: Formata (`formata-arch/`, Formata-injected styles in `web/src/main.js`), self-hosting fonts, type scale / size changes

## Goal

Split the type system into three clear roles:

| Role | Face | Used for |
|------|------|----------|
| Display | Space Grotesk | Literal `h1` only |
| Sans (UI / body) | Lato | Everything else that currently uses `--font-sans` (body, buttons, nav, labels, `h2`–`h4`, etc.) |
| Mono | JetBrains Mono | Existing `--font-mono` call sites (hashes, codes, meta ids) |

Load fonts with `font-display: swap` via Google Fonts `<link>` tags (not CSS `@import`).

## Decisions

1. **Display scope:** Space Grotesk on literal `h1` only (not all headings, not dialog/panel titles unless they are `h1`).
2. **UI chrome:** Buttons, nav, labels, forms inherit Lato with body — no separate “chrome” face.
3. **Lato weights:** Classic Lato exposes `100 / 300 / 400 / 700 / 900` only (no 500 or 600). Do not request missing weights.
4. **Weight token remap** (so existing `--font-medium` / `--font-semibold` / `--font-bold` usage stays valid without faux synthesis):

   | Token | Old | New |
   |-------|-----|-----|
   | `--font-normal` | 400 | 400 |
   | `--font-medium` | 500 | 400 |
   | `--font-semibold` | 600 | 700 |
   | `--font-bold` | 700 | 900 |

5. **Loading:** Google Fonts via `<link>` in `layout.html` + `preconnect`, with `display=swap`. Remove the font `@import` from `tokens.css`.
6. **Formata:** Unchanged.

## Loading

In `server/templates/layout.html` `<head>`, outside the Vite/prod asset `if`/`else` (so both modes get fonts):

1. `preconnect` to `https://fonts.googleapis.com`
2. `preconnect` to `https://fonts.gstatic.com` with `crossorigin`
3. Stylesheet link, e.g.:

```text
https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@500&family=Lato:wght@400;700;900&family=Space+Grotesk:wght@600;700&display=swap
```

Weights loaded:

- Lato: `400`, `700`, `900`
- Space Grotesk: `600`, `700`
- JetBrains Mono: `500`

## Tokens (`web/src/styles/tokens.css`)

Remove the Google Fonts `@import` line.

Set:

```css
--font-sans: "Lato", system-ui, sans-serif;
--font-display: "Space Grotesk", system-ui, sans-serif;
--font-mono: "JetBrains Mono", ui-monospace, monospace;

--font-normal: 400;
--font-medium: 400;
--font-semibold: 700;
--font-bold: 900;
```

## Application (`web/src/styles/reset.css`)

- `body` keeps `font-family: var(--font-sans)` (Lato).
- On the existing `h1` rule, add `font-family: var(--font-display)`.
- Do not set display font on `h2`–`h4`.
- Mono: no call-site changes; `--font-mono` token swap is enough.
- Buttons already use `var(--font-sans)` — they pick up Lato automatically.

## Docs (`docs/css.md`)

Update the fonts section to:

- Document `--font-display` alongside `--font-sans` / `--font-mono`
- Note Lato / Space Grotesk / JetBrains Mono stacks
- Note weight remaps for Lato’s available faces
- State that Google Fonts load from `layout.html` with `display=swap`, not via `@import` in `tokens.css`
- Adjust the tokens layer table row if it still says “font import” in `tokens.css`

## Files to change

| File | Change |
|------|--------|
| `server/templates/layout.html` | `preconnect` + Google Fonts CSS `<link>` |
| `web/src/styles/tokens.css` | Drop `@import`; new stacks + weight values |
| `web/src/styles/reset.css` | `h1` uses `--font-display` |
| `docs/css.md` | Document the new type system |

## Verification

1. Rebuild or run Vite so CSS changes apply.
2. Spot-check: page-header `h1` is Space Grotesk; body/buttons/nav are Lato; a mono meta id / code is JetBrains Mono.
3. Confirm Network tab shows one fonts CSS request with `display=swap`, and no duplicate fetch from a CSS `@import`.
4. Run `task css:lint` if that remains the usual CSS gate.

## Amendment (2026-07-17)

Sans face changed from **Lato** to **Inter**. Inter provides `400` / `500` / `600` / `700`, so weight tokens stay at their semantic values (`--font-medium: 500`, `--font-semibold: 600`, `--font-bold: 700`) with no Lato-style remap. Google Fonts load: `family=Inter:wght@400;500;600;700`.
