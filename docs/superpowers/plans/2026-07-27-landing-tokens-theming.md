# Landing tokens + light/dark Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Drop `--landing-*`, make `/` follow Attesta `data-theme` light/dark tokens, and add a `data-category` → `--category-color` palette starting with `passport`.

**Architecture:** Reuse existing semantic tokens (`--background`, `--foreground`, `--card`, `--muted`, `--primary`, `--border`, `--shadow`, …). Add `--category-passport` with `light-dark()` and a `category-palette.css` map mirroring `role-palette.css`. Landing page CSS and passport badges consume those tokens only.

**Tech Stack:** CSS custom properties, PostCSS build (`web/`), Go `html/template` (`public_home.html`), `task css:lint`.

**Spec:** `docs/superpowers/specs/2026-07-27-landing-tokens-theming-design.md`

## Global Constraints

- No `--landing-*` after completion (`rg --landing-` empty in repo).
- No hardcoded hex/rgb outside `tokens.css` (stylelint / `task css:lint`).
- Borders use `--border`, never `--background` (dark `--background` is a gradient).
- Filled CTAs use `--primary` / `--primary-foreground`; outline/ghost use `--border` / `--foreground` / `--card`.
- Ship only category key `passport` now.
- Keep `--text-4xl` / `--text-5xl` and other shared type/spacing tokens.

## File map

| File | Responsibility |
|------|----------------|
| `web/src/styles/tokens.css` | Delete `--landing-*`; add `--category-passport` |
| `web/src/styles/category-palette.css` | New: `data-category` → `--category-color` |
| `web/src/styles.css` | Import category palette after role palette |
| `web/src/styles/pages/public-home.css` | Swap tokens; drop forced dark; badge uses `--category-color` |
| `server/templates/pages/public_home.html` | `data-category="passport"` on passport badges |
| `docs/css.md` | Layer stack + `data-category` row |

---

### Task 1: Category palette foundation

**Files:**
- Modify: `web/src/styles/tokens.css`
- Create: `web/src/styles/category-palette.css`
- Modify: `web/src/styles.css`
- Modify: `docs/css.md`

**Interfaces:**
- Produces: `--category-passport`, `--category-color` (via `[data-category="passport"]`)

- [ ] **Step 1: Replace `--landing-*` block with `--category-passport` in tokens**

In `web/src/styles/tokens.css`, delete the entire “Public landing” block (`--landing-bg` … `--landing-shadow`). Keep `--text-4xl` / `--text-5xl`. After the font-size tokens (or near role/stream tokens), add:

```css
  /* Category (marketing / stream taxonomy) */
  --category-passport: light-dark(#1a7ab8, #48baf9);
```

- [ ] **Step 2: Create `category-palette.css`**

Create `web/src/styles/category-palette.css`:

```css
/* Category key → shared custom property (consumed by badges, later chips) */
[data-category="passport"] {
  --category-color: var(--category-passport);
}

[data-category="fallback"],
.public-home-badge:not([data-category]) {
  --category-color: var(--muted-foreground);
}
```

- [ ] **Step 3: Import after role-palette**

In `web/src/styles.css`, after `role-palette.css`:

```css
@import "./styles/category-palette.css";
```

- [ ] **Step 4: Document in `docs/css.md`**

Update layer stack item 3 to mention category palette (or insert as new item 4 and renumber):

```markdown
3. `role-palette.css` — `data-role-palette` / `data-stream-status` maps  
4. `category-palette.css` — `data-category` → `--category-color`  
```

(renumber subsequent layers)

Add to Shared `data-*` table:

```markdown
| `data-category` | `category-palette.css` → `--category-color` |
```

- [ ] **Step 5: Commit**

```bash
git add web/src/styles/tokens.css web/src/styles/category-palette.css web/src/styles.css docs/css.md
git commit -m "$(cat <<'EOF'
feat(css): add category palette and drop landing color tokens

Introduce data-category → --category-color (passport first) and remove the parallel --landing-* palette from tokens.
EOF
)"
```

---

### Task 2: Wire landing CSS + markup to shared tokens

**Files:**
- Modify: `web/src/styles/pages/public-home.css`
- Modify: `server/templates/pages/public_home.html`
- Test: `task css:lint`, `cd web && npm run build`, `rg --landing-`

**Interfaces:**
- Consumes: semantic tokens + `--category-color` from Task 1

- [ ] **Step 1: Update passport badges in template**

In `server/templates/pages/public_home.html`, replace every:

```html
<span class="public-home-badge public-home-badge-passport">Product Passport</span>
```

with:

```html
<span class="public-home-badge" data-category="passport">Product Passport</span>
```

(four occurrences)

- [ ] **Step 2: Retheme `public-home.css`**

Apply these substitutions throughout `web/src/styles/pages/public-home.css`:

| From | To |
|------|-----|
| `var(--landing-bg)` | `var(--background)` — **except** where used as `border-color` → use `var(--border)` |
| `var(--landing-surface)` | `var(--card)` |
| `var(--landing-muted)` | `var(--muted)` |
| `var(--landing-secondary)` | `var(--secondary)` |
| `var(--landing-foreground)` | `var(--foreground)` |
| `var(--landing-text-muted)` | `var(--muted-foreground)` |
| `var(--landing-text-secondary)` | `var(--muted-foreground)` |
| `var(--landing-border)` | `var(--border)` |
| `var(--landing-shadow)` | `var(--shadow)` |
| `var(--landing-accent)` | `var(--primary)` |
| `var(--landing-accent-soft)` | `var(--primary)` |

Remove from `.public-home`:

```css
  color-scheme: dark;
```

Replace button rules:

```css
.public-home-btn-primary {
  background: var(--primary);
  border-color: var(--primary);
  color: var(--primary-foreground);
}

.public-home-btn-primary:hover {
  background: color-mix(in srgb, var(--primary) 90%, white);
}

.public-home-btn-secondary {
  background: var(--card);
  border-color: var(--border);
  color: var(--foreground);
}

.public-home-btn-secondary:hover {
  background: color-mix(in srgb, var(--foreground) 6%, var(--card));
}

.public-home-btn-ghost {
  background: var(--card);
  border-color: var(--border);
  color: var(--foreground);
}
```

Replace passport badge block — delete `.public-home-badge-passport` and style category badges:

```css
.public-home-badge[data-category] {
  background: var(--category-color);
  border-color: var(--category-color);
  color: var(--card);
}
```

(Keep the base `.public-home-badge` rules for non-category badges; remove any rules that still reference `--landing-passport`.)

Other `--landing-primary` uses (e.g. code sample accent, avatar ring border): map to `var(--primary)`.

After edits, confirm zero matches:

```bash
rg --landing- web/src server docs
```

Expected: no matches (spec file may still mention `--landing-*` in historical mapping tables — that is OK under `docs/superpowers/`).

- [ ] **Step 3: Lint and build**

```bash
task css:lint
cd web && npm run build
```

Expected: both succeed.

- [ ] **Step 4: Handler smoke (optional but preferred)**

```bash
cd server && go test ./cmd/server/ -run 'TestHandlePublicHome'
```

Expected: PASS (markup still contains Sign In / Dashboard assertions; `data-category` does not break them).

- [ ] **Step 5: Commit**

```bash
git add web/src/styles/pages/public-home.css server/templates/pages/public_home.html
git commit -m "$(cat <<'EOF'
feat(ui): theme public home with shared light/dark tokens

Swap landing-only colors for semantic tokens, follow data-theme, and drive passport badges via data-category.
EOF
)"
```

---

## Spec coverage check

| Spec requirement | Task |
|------------------|------|
| Delete `--landing-*` | 1 |
| Add `--category-passport` + map file + import | 1 |
| Docs layer + `data-category` | 1 |
| Swap public-home CSS; drop `color-scheme: dark` | 2 |
| CTA primary/outline semantics | 2 |
| Borders not `--background` | 2 |
| Markup `data-category="passport"` | 2 |
| `css:lint` + build + no `--landing-*` in styles | 2 |
| Non-goals (components, more categories, Figma blue dark, Vite) | omitted by design |
