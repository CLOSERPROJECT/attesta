# Font Stack Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Switch the main Attesta app to Lato (UI/body), Space Grotesk (`h1` only), and JetBrains Mono (mono), loaded from Google Fonts via `<link>` with `display=swap`.

**Architecture:** Font faces load in `layout.html` (`preconnect` + CSS stylesheet). CSS tokens in `tokens.css` define `--font-sans`, `--font-display`, `--font-mono` and remapped weight tokens for Lato’s real faces (`400` / `700` / `900`). `reset.css` applies `--font-display` on literal `h1` only. Formata stays untouched.

**Tech Stack:** Go `html/template` (`layout.html`), Vite CSS (`web/src/styles/`), Google Fonts CSS2 API, existing `go test` + `task css:lint`.

**Spec:** `docs/superpowers/specs/2026-07-17-font-stack-design.md`

## Global Constraints

- Space Grotesk on literal `h1` only — not `h2`–`h4`, not dialog/panel titles unless they are `h1`
- Lato for all non-`h1` UI that uses `--font-sans` (body, buttons, nav, labels, forms)
- JetBrains Mono via `--font-mono` token swap only (no call-site churn)
- Load Lato weights `400;700;900` only (no `500` / `600`)
- Weight tokens: `--font-medium: 400`, `--font-semibold: 700`, `--font-bold: 900`
- Google Fonts via `<link>` in `layout.html` with `display=swap`; remove `@import` from `tokens.css`
- Do not change Formata (`formata-arch/`, Formata styles in `web/src/main.js`)
- Do not change type scale sizes, line heights, or letter-spacing
- Prefer minimal, localized diffs; update `docs/css.md` to match

## File map

| File | Responsibility |
|------|----------------|
| `server/cmd/server/layout_fonts_test.go` | Assert layout HTML includes preconnect + Google Fonts stylesheet with expected families/`display=swap` |
| `server/templates/layout.html` | Emit font `preconnect` + stylesheet `<link>` for all pages (prod + Vite) |
| `web/src/styles/tokens.css` | Font family + weight tokens; no Google Fonts `@import` |
| `web/src/styles/reset.css` | `h1 { font-family: var(--font-display); }` |
| `docs/css.md` | Document display token, stacks, weight remaps, layout-based loading |

---

### Task 1: Load Google Fonts from layout

**Files:**
- Create: `server/cmd/server/layout_fonts_test.go`
- Modify: `server/templates/layout.html` (insert links after `{{ end }}` of the Vite/prod asset block, before the HTMX script — around lines 94–95)

**Interfaces:**
- Consumes: existing `parseTestTemplates(t)`, `PageBase`, `layout.html` define
- Produces: rendered layout `<head>` always includes font preconnects + stylesheet URL with `family=Lato`, `family=Space+Grotesk`, `family=JetBrains+Mono`, and `display=swap`

- [ ] **Step 1: Write the failing test**

Create `server/cmd/server/layout_fonts_test.go`:

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLayoutLoadsGoogleFontsWithSwap(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var rendered bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", PageBase{}); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := rendered.String()

	for _, want := range []string{
		`rel="preconnect" href="https://fonts.googleapis.com"`,
		`rel="preconnect" href="https://fonts.gstatic.com" crossorigin`,
		`fonts.googleapis.com/css2?`,
		`family=Lato:wght@400;700;900`,
		`family=Space+Grotesk:wght@600;700`,
		`family=JetBrains+Mono:wght@500`,
		`display=swap`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected layout to include %q, got:\n%s", want, body)
		}
	}

	if strings.Contains(body, "Source+Code+Pro") {
		t.Fatalf("expected Source Code Pro to be removed from layout fonts, got:\n%s", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd server && go test ./cmd/server/ -run TestLayoutLoadsGoogleFontsWithSwap -count=1
```

Expected: FAIL — missing preconnect / Google Fonts stylesheet substrings.

- [ ] **Step 3: Add font links to `layout.html`**

In `server/templates/layout.html`, immediately after the Vite/prod `{{ end }}` that closes the asset/favicon branch (currently just before the HTMX `<script>`), insert:

```html
      <link rel="preconnect" href="https://fonts.googleapis.com" />
      <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
      <link
        rel="stylesheet"
        href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@500&family=Lato:wght@400;700;900&family=Space+Grotesk:wght@600;700&display=swap"
      />
```

Keep these **outside** the `{{ if .ViteDevServer }}` / `{{ else }}` block so both modes load fonts.

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd server && go test ./cmd/server/ -run TestLayoutLoadsGoogleFontsWithSwap -count=1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/cmd/server/layout_fonts_test.go server/templates/layout.html
git commit -m "$(cat <<'EOF'
feat(ui): load Lato, Space Grotesk, and JetBrains Mono from layout

Move Google Fonts to layout preconnect + stylesheet links with display=swap so prod and Vite both get the new stack.
EOF
)"
```

---

### Task 2: Update font and weight tokens

**Files:**
- Modify: `web/src/styles/tokens.css` (lines 1–29: remove `@import`, replace font family + weight token values)

**Interfaces:**
- Consumes: font faces loaded in Task 1
- Produces: `--font-sans` = Lato, `--font-display` = Space Grotesk, `--font-mono` = JetBrains Mono; weight tokens remapped for Lato

- [ ] **Step 1: Confirm current token block**

Open `web/src/styles/tokens.css` and confirm it still starts with the Space Grotesk / Source Code Pro `@import` and `--font-sans` / `--font-mono` / weight tokens as in the spec baseline.

- [ ] **Step 2: Replace the font import and tokens**

Delete line 1 (`@import url("https://fonts.googleapis.com/...");`).

Replace the Fonts + Font weights sections inside `:root` with:

```css
  /* Fonts */
  --font-sans: "Lato", system-ui, sans-serif;
  --font-display: "Space Grotesk", system-ui, sans-serif;
  --font-mono: "JetBrains Mono", ui-monospace, monospace;

  /* Font sizes */
  --text-xs: 0.75rem;
  --text-sm: 0.875rem;
  --text-base: 1rem;
  --text-lg: 1.125rem;
  --text-xl: 1.25rem;
  --text-2xl: 1.5rem;
  --text-3xl: 1.875rem;

  /* Line heights */
  --leading-none: 1;
  --leading-tight: 1.25;
  --leading-normal: 1.5;
  --leading-relaxed: 1.625;

  /* Font weights — remapped to Lato’s available faces (400/700/900) */
  --font-normal: 400;
  --font-medium: 400;
  --font-semibold: 700;
  --font-bold: 900;
```

Leave all other `:root` tokens unchanged. Do not reintroduce a Google Fonts `@import`.

- [ ] **Step 3: Sanity-check the file**

Run:

```bash
rg -n '@import|Space Grotesk|Source Code Pro|--font-sans|--font-display|--font-mono|--font-medium|--font-semibold|--font-bold' web/src/styles/tokens.css
```

Expected:
- No `@import`
- No `Source Code Pro`
- `--font-sans` → Lato, `--font-display` → Space Grotesk, `--font-mono` → JetBrains Mono
- `--font-medium: 400`, `--font-semibold: 700`, `--font-bold: 900`

- [ ] **Step 4: Commit**

```bash
git add web/src/styles/tokens.css
git commit -m "$(cat <<'EOF'
feat(ui): point font tokens at Lato, Grotesk display, JetBrains mono

Drop the CSS @import and remap weight tokens to Lato’s real faces so medium/semibold/bold no longer request missing 500/600 files.
EOF
)"
```

---

### Task 3: Apply display font to `h1` only

**Files:**
- Modify: `web/src/styles/reset.css` (the existing `h1` rule ~lines 39–41)

**Interfaces:**
- Consumes: `--font-display` from Task 2
- Produces: literal `h1` uses Space Grotesk; `h2`–`h4` keep inheriting Lato from `body`

- [ ] **Step 1: Update the `h1` rule**

Change the `h1` block in `web/src/styles/reset.css` from:

```css
h1 {
  font-size: var(--text-3xl);
}
```

to:

```css
h1 {
  font-family: var(--font-display);
  font-size: var(--text-3xl);
}
```

Do **not** add `font-family` to the shared `h1, h2, h3, h4` rule or to `h2`/`h3`/`h4`.

- [ ] **Step 2: Confirm heading rules**

Run:

```bash
rg -n -A3 '^h1|^h2|^h3|^h4|font-family' web/src/styles/reset.css
```

Expected: only the `h1` rule sets `font-family: var(--font-display)`; `body` still has `font-family: var(--font-sans)`.

- [ ] **Step 3: Commit**

```bash
git add web/src/styles/reset.css
git commit -m "$(cat <<'EOF'
feat(ui): use Space Grotesk only on h1

Keep body and secondary headings on Lato via --font-sans inheritance.
EOF
)"
```

---

### Task 4: Document the type system and verify

**Files:**
- Modify: `docs/css.md` (layer table row for Tokens; “Font tokens” + “Weight tokens” sections ~lines 14, 160–198)

**Interfaces:**
- Consumes: final token names and load strategy from Tasks 1–3
- Produces: docs match runtime behavior

- [ ] **Step 1: Update the layer stack Tokens row**

In the layer stack table, change the Tokens row from:

```markdown
| Tokens | `tokens.css` | `:root`, `[data-theme="dark"]`, font import |
```

to:

```markdown
| Tokens | `tokens.css` | `:root`, `[data-theme="dark"]`, font/type tokens (Google Fonts load in `layout.html`) |
```

- [ ] **Step 2: Replace the Font tokens section**

Replace the entire `### Font tokens` subsection (through the paragraph that says Google Fonts are imported in `tokens.css`) with:

```markdown
### Font tokens

| Token | Value |
|-------|-------|
| `--font-sans` | Lato stack (body, buttons, UI copy, `h2`–`h4`) |
| `--font-display` | Space Grotesk stack (literal `h1` only) |
| `--font-mono` | JetBrains Mono stack (hashes, codes, meta ids) |

Google Fonts load from `server/templates/layout.html` (`preconnect` + stylesheet `<link>` with `display=swap`). Do **not** `@import` fonts from `tokens.css`. Use `var(--font-sans)` / `var(--font-display)` / `var(--font-mono)` in new CSS — do not repeat font family strings.
```

- [ ] **Step 3: Update the Weight tokens table**

Replace the weight tokens table with:

```markdown
| Token | Value | Loaded face |
|-------|-------|-------------|
| `--font-normal` | `400` | Lato 400 |
| `--font-medium` | `400` | Lato 400 (Lato has no 500) |
| `--font-semibold` | `700` | Lato 700 (Lato has no 600) |
| `--font-bold` | `900` | Lato 900 |
```

Also update the heading-defaults sentence if it still implies Space Grotesk weights; keep the size mapping, and note that `h1` uses `--font-display` while remaining headings inherit `--font-sans`.

Suggested heading-defaults sentence:

```markdown
**Heading defaults** are set in `reset.css` (`h1`–`h4`): `h1` → `--text-3xl` + `--font-display`, `h2` → `--text-xl`, `h3` → `--text-lg`, `h4` → `--text-base`, all with `--font-semibold` and `--leading-tight`. Prefer tokens over raw `font-size` in component CSS; remove redundant heading `font-size` overrides when they only duplicate semantics.
```

- [ ] **Step 4: Run CSS lint and the layout font test**

Run:

```bash
task css:lint
cd server && go test ./cmd/server/ -run 'TestLayoutLoadsGoogleFontsWithSwap|TestLayoutRendersFooterContent' -count=1
```

Expected: both pass (or `task css:lint` succeeds with no new errors).

- [ ] **Step 5: Spot-check in the browser (manual)**

With Vite or a built bundle:

1. Open any page with a page-header `h1` — computed `font-family` includes `Space Grotesk`.
2. Body / button / nav text — includes `Lato`.
3. A mono meta id (e.g. page-header process id) — includes `JetBrains Mono`.
4. Network: one request to `fonts.googleapis.com/css2?...display=swap`; no fonts CSS request originating from the main CSS bundle `@import`.

- [ ] **Step 6: Commit**

```bash
git add docs/css.md
git commit -m "$(cat <<'EOF'
docs(css): document Lato / Grotesk display / JetBrains mono stack

Align the style guide with layout-based Google Fonts loading and Lato weight remaps.
EOF
)"
```

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Google Fonts `<link>` + preconnect + `display=swap` in `layout.html` | Task 1 |
| Remove `@import` from `tokens.css` | Task 2 |
| `--font-sans` / `--font-display` / `--font-mono` stacks | Task 2 |
| Weight remap medium→400, semibold→700, bold→900 | Task 2 |
| `h1` only gets display font | Task 3 |
| Update `docs/css.md` | Task 4 |
| Formata untouched | All tasks (explicit non-touch) |
| Verification (lint + spot-check) | Task 4 |

## Self-review notes

- No placeholders / TBDs in steps.
- Test assertions match the exact href query string used in Task 1 Step 3.
- Weight token values in Task 2 match the Google Fonts Lato weights loaded in Task 1.
- Formata paths are never listed under Files to modify.
