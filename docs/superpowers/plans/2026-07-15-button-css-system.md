# Button CSS System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce a composable `button.css` system (`btn` + variants + sizes + `btn-icon`) and migrate every control button/link in Attesta templates off bare `primary` / `secondary` / `danger` / `ghost` classes.

**Architecture:** New CSS-only component `web/src/styles/components/button.css` owns base, variant, size, and icon-shape rules. Context classes (`.dialog-close`, `.workflow-card-menu-trigger`, `.pagination-btn`, `.nav-action`) keep only placement/domain behavior. Templates compose classes: `btn btn-primary`, `btn btn-ghost btn-icon`, `btn btn-ghost btn-icon btn-xs`, etc. Bare element `button { … }` primary styling is removed so every interactive control must opt into `.btn`.

**Tech Stack:** Server-rendered Go `html/template`, Vite + PostCSS CSS layers (`components.css` barrel), Stylelint via `task css:lint`, Go markup tests in `server/cmd/server/*_test.go`.

## Global Constraints

- Read `docs/css.md` and `.agents/skills/attesta-ui-components/SKILL.md` before editing styles/templates.
- Default control height is **36px** (`--btn-height: 2.25rem`).
- Size scale: **xs = 28px** (`1.75rem`), **sm = 32px** (`2rem`), default **36px**, **lg = 40px** (`2.5rem`).
- Naming is `btn-*` only — no bare `.primary` / `.secondary` / `.danger` / `.ghost` button classes after migration.
- Nav chrome uses **`btn-outline`** (bordered transparent).
- Destructive icon actions use **`btn-ghost-danger`** (ghost layout + destructive color).
- Org row actions use **`btn btn-ghost btn-icon btn-xs`** (and `btn-ghost-danger` for delete).
- Dialog close uses **`btn btn-ghost btn-icon dialog-close`** (default 36px; placement offset stays on `.dialog-close`).
- Out of scope (leave as specialized text-link buttons): `.dpp-integrity-hash-button`, `.substep-body-digest-button`, `.platform-admin-invite-button`, `.workflow-card-menu-item*`, `.account-menu-item*`, accordion chevron `<span>`s.
- Prefer minimal diffs; do not refactor unrelated CSS.
- Commit after each task.

---

## File structure

| File | Responsibility |
|------|----------------|
| `web/src/styles/tokens.css` | Add `--btn-height*` / `--btn-padding-x*` tokens |
| `web/src/styles/components/button.css` | **Create** — base `.btn`, variants, sizes, `.btn-icon`, shared disabled/focus; markup contract in header |
| `web/src/styles/components.css` | Import `button.css` before `shared.css` |
| `web/src/styles/components/shared.css` | Remove legacy button rules; slim workflow menu trigger to placement/open only |
| `web/src/styles/components/dialog.css` | Slim `.dialog-close` to placement + icon display; update header comment; fix `.dialog-actions` mobile selectors to `.btn` |
| `web/src/styles/components/org-admin.css` | Delete `.user-action-button*` block |
| `web/src/styles/layout/chrome.css` | Slim `.nav-action` to nav-specific hover/open; compose with `btn btn-outline btn-icon btn-lg` |
| `docs/css.md` | Document `button.css` in CSS-only table + template index; update control sizing note |
| `server/templates/**/*.html` | All button/CTA class migrations (exact map below) |
| `server/cmd/server/dialog_markup_test.go` | Assert new close button classes |
| `server/cmd/server/panel_markup_test.go` | Assert `btn btn-primary` / `btn btn-secondary` |
| `server/cmd/server/home_handler_test.go` | Assert new menu trigger + submit classes |
| `.agents/skills/attesta-ui-components/SKILL.md` | Mention `button` CSS-only component |

### Composition contract (locked)

```
<button|a class="btn btn-{variant} [btn-{size}] [btn-icon] [context…]">
```

| Layer | Classes (pick ≤1 variant, ≤1 size, optional icon) |
|-------|---------------------------------------------------|
| Base | `btn` (required) |
| Variant | `btn-primary` \| `btn-secondary` \| `btn-ghost` \| `btn-ghost-danger` \| `btn-danger` \| `btn-outline` |
| Size | (default 36) \| `btn-xs` (28) \| `btn-sm` (32) \| `btn-lg` (40) |
| Shape | `btn-icon` (square; width/height = effective height) |
| Context | `dialog-close`, `workflow-card-menu-trigger`, `pagination-btn`, `nav-action`, `theme-toggle`, `account-trigger`, `js-*` |

### Template migration map

| From | To |
|------|----|
| `class="primary"` / `class="primary …"` | `class="btn btn-primary …"` |
| `class="secondary"` / `class="secondary …"` | `class="btn btn-secondary …"` |
| `class="danger"` | `class="btn btn-danger"` |
| `class="ghost dialog-close"` | `class="btn btn-ghost btn-icon dialog-close"` |
| `class="secondary secondary-small js-close-substep-override"` | `class="btn btn-ghost btn-icon js-close-substep-override"` |
| `class="user-action-button"` | `class="btn btn-ghost btn-icon btn-xs"` |
| `class="user-action-button user-action-button-danger"` | `class="btn btn-ghost-danger btn-icon btn-xs"` |
| `class="workflow-card-menu-trigger"` | `class="btn btn-ghost btn-icon workflow-card-menu-trigger"` |
| `class="nav-action theme-toggle"` | `class="btn btn-outline btn-icon btn-lg nav-action theme-toggle"` |
| `class="nav-action account-trigger"` | `class="btn btn-outline btn-icon btn-lg nav-action account-trigger"` |
| `class="secondary pagination-btn…"` | `class="btn btn-secondary pagination-btn…"` |
| `class="secondary substep-body-attachments-nav …"` | `class="btn btn-secondary substep-body-attachments-nav …"` |
| `a class="secondary"` (download) | `a class="btn btn-secondary"` |
| `a class="primary …"` / `a class="secondary …"` | same with `btn` + variant |

Files with button markup to touch:
- `server/templates/layout.html`
- `server/templates/pages/home.html`
- `server/templates/pages/stream.html`
- `server/templates/pages/process.html`
- `server/templates/pages/org_admin.html`
- `server/templates/pages/platform_admin.html`
- `server/templates/pages/dpp.html`
- `server/templates/pages/login.html`
- `server/templates/pages/signup.html`
- `server/templates/pages/invite.html`
- `server/templates/pages/reset_request.html`
- `server/templates/pages/reset_set.html`
- `server/templates/components/substep_body.html`
- `server/templates/attachment_carousel.html`
- `server/templates/substep_override_editor.html`

---

### Task 1: Add tokens + `button.css` + wire import + docs row

**Files:**
- Modify: `web/src/styles/tokens.css`
- Create: `web/src/styles/components/button.css`
- Modify: `web/src/styles/components.css`
- Modify: `docs/css.md` (CSS-only table row + control sizing note)
- Modify: `.agents/skills/attesta-ui-components/SKILL.md` (CSS-only list)

**Interfaces:**
- Consumes: existing color tokens (`--primary`, `--muted`, `--destructive`, `--border`, `--card`, `--nav-hover`)
- Produces: `.btn`, `.btn-primary`, `.btn-secondary`, `.btn-ghost`, `.btn-ghost-danger`, `.btn-danger`, `.btn-outline`, `.btn-xs`, `.btn-sm`, `.btn-lg`, `.btn-icon`, tokens `--btn-height`, `--btn-height-xs`, `--btn-height-sm`, `--btn-height-lg`, `--btn-padding-x`, `--btn-padding-x-sm`

- [ ] **Step 1: Add button size tokens to `tokens.css`**

After the spacing block in `:root`, add:

```css
  /* Buttons */
  --btn-height-xs: 1.75rem; /* 28px — dense icon actions */
  --btn-height-sm: 2rem; /* 32px */
  --btn-height: 2.25rem; /* 36px — default */
  --btn-height-lg: 2.5rem; /* 40px — nav / touch */
  --btn-padding-x: var(--space-3);
  --btn-padding-x-sm: var(--space-2);
```

Mirror the same tokens under `[data-theme="dark"]` only if that block redefines layout tokens; if dark theme inherits `:root` spacing, do **not** duplicate (check existing pattern — spacing is root-only today).

- [ ] **Step 2: Create `button.css` with full composition**

Create `web/src/styles/components/button.css`:

```css
/*
 * Button — composable control (CSS-only component).
 *
 * <button|a class="btn btn-{variant} [btn-{size}] [btn-icon] [context…]">
 *
 * Variants (pick one):
 *   btn-primary | btn-secondary | btn-ghost | btn-ghost-danger | btn-danger | btn-outline
 * Sizes (optional):
 *   btn-xs (28) | btn-sm (32) | default (36) | btn-lg (40)
 * Shape (optional):
 *   btn-icon — square; uses effective height for width/height
 *
 * Context modifiers live in domain CSS (dialog-close, workflow-card-menu-trigger, …).
 */

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  box-sizing: border-box;
  min-height: var(--btn-height);
  padding: 0 var(--btn-padding-x);
  border: 1px solid transparent;
  border-radius: 4px;
  font-family: var(--font-sans);
  font-size: var(--text-base);
  font-weight: 600;
  line-height: var(--leading-none);
  text-decoration: none;
  cursor: pointer;
  flex-shrink: 0;
}

a.btn:hover {
  text-decoration: none;
}

.btn:disabled,
.btn[disabled],
.btn.is-disabled {
  cursor: not-allowed;
  opacity: 0.6;
  pointer-events: none;
}

/* Variants */

.btn-primary {
  background: var(--primary);
  border-color: var(--primary);
  color: var(--primary-foreground);
}

.btn-primary:hover:not(:disabled):not(.is-disabled) {
  background: color-mix(in srgb, var(--primary) 90%, var(--card));
}

.btn-secondary {
  background: transparent;
  border-color: var(--primary);
  color: var(--primary);
}

.btn-secondary:hover:not(:disabled):not(.is-disabled) {
  background: color-mix(in srgb, var(--primary) 16%, var(--card));
}

.btn-ghost {
  background: transparent;
  border-color: transparent;
  color: var(--muted-foreground);
}

.btn-ghost:hover:not(:disabled):not(.is-disabled) {
  background: var(--muted);
  color: var(--foreground);
}

.btn-ghost-danger {
  background: transparent;
  border-color: transparent;
  color: var(--destructive);
}

.btn-ghost-danger:hover:not(:disabled):not(.is-disabled) {
  background: color-mix(in srgb, var(--destructive) 10%, var(--card));
  color: var(--destructive);
}

.btn-danger {
  background: var(--destructive);
  border-color: var(--destructive);
  color: var(--destructive-foreground);
}

.btn-danger:hover:not(:disabled):not(.is-disabled) {
  background: color-mix(in srgb, var(--destructive) 90%, var(--card));
}

.btn-outline {
  background: transparent;
  border-color: var(--border);
  color: var(--foreground);
}

.btn-outline:hover:not(:disabled):not(.is-disabled) {
  background: var(--nav-hover, var(--muted));
}

/* Sizes */

.btn-xs {
  min-height: var(--btn-height-xs);
  padding: 0 var(--space-1);
  font-size: var(--text-sm);
}

.btn-sm {
  min-height: var(--btn-height-sm);
  padding: 0 var(--btn-padding-x-sm);
  font-size: var(--text-sm);
}

.btn-lg {
  min-height: var(--btn-height-lg);
  padding: 0 var(--space-4);
}

/* Icon shape */

.btn-icon {
  width: var(--btn-height);
  height: var(--btn-height);
  min-height: 0;
  padding: 0;
}

.btn-xs.btn-icon {
  width: var(--btn-height-xs);
  height: var(--btn-height-xs);
}

.btn-sm.btn-icon {
  width: var(--btn-height-sm);
  height: var(--btn-height-sm);
}

.btn-lg.btn-icon {
  width: var(--btn-height-lg);
  height: var(--btn-height-lg);
}

.btn-icon .icon-svg {
  display: block;
}
```

- [ ] **Step 3: Import from `components.css`**

Insert before `shared.css`:

```css
@import url("./components/button.css");
```

Full top of file becomes:

```css
@import url("./components/page-header.css");
@import url("./components/panel.css");
@import url("./components/dialog.css");
@import url("./components/button.css");
@import url("./components/shared.css");
```

- [ ] **Step 4: Document in `docs/css.md` and skill**

In the CSS-only components table, add:

```markdown
| `button.css` | `.btn`, `.btn-primary`, `.btn-secondary`, `.btn-ghost`, `.btn-ghost-danger`, `.btn-danger`, `.btn-outline`, `.btn-xs`, `.btn-sm`, `.btn-lg`, `.btn-icon` | See file header in `web/src/styles/components/button.css` |
```

Update the control sizing sentence to:

```markdown
**Control sizing pattern:** buttons use the `btn` system in `button.css` (default height `--btn-height` / 36px). Form labels use `--text-sm` (`forms.css`); inputs use `--text-base`.
```

In the template ↔ CSS index, change shared buttons references to `components/button.css` (panel row “Also uses”, login forms row, etc.).

In `.agents/skills/attesta-ui-components/SKILL.md` CSS-only list, add:

```markdown
- `button` — composable controls (`.btn` + variants/sizes/`btn-icon` in `web/src/styles/components/button.css`)
```

- [ ] **Step 5: Lint CSS**

Run:

```bash
task css:lint
```

Expected: PASS (stylelint + breakpoint + inline-style checks).

- [ ] **Step 6: Commit**

```bash
git add web/src/styles/tokens.css web/src/styles/components/button.css web/src/styles/components.css docs/css.md .agents/skills/attesta-ui-components/SKILL.md
git commit -m "$(cat <<'EOF'
feat(css): add composable button.css system and size tokens

Introduce btn + variants/sizes/icon shape as the shared control contract.
EOF
)"
```

---

### Task 2: Update markup tests for new button classes (red)

**Files:**
- Modify: `server/cmd/server/dialog_markup_test.go`
- Modify: `server/cmd/server/panel_markup_test.go`
- Modify: `server/cmd/server/home_handler_test.go`

**Interfaces:**
- Consumes: Task 1 class names
- Produces: failing tests that Task 3–4 make green

- [ ] **Step 1: Update dialog markup expectations**

In `dialog_markup_test.go`, replace both occurrences of:

```go
`class="ghost dialog-close"`,
```

with:

```go
`class="btn btn-ghost btn-icon dialog-close"`,
```

- [ ] **Step 2: Update panel markup expectations**

In `panel_markup_test.go`, replace:

| Old assertion | New assertion |
|---------------|---------------|
| `` `class="secondary js-download-link"` `` | `` `class="btn btn-secondary js-download-link"` `` |
| `` `class="primary"` `` (process new instance / platform create) | `` `class="btn btn-primary"` `` |
| `` `class="primary js-share-link"` `` | `` `class="btn btn-primary js-share-link"` `` |

Update any `strings.Index` lookups that use the old substrings to match.

- [ ] **Step 3: Update home handler expectations**

In `home_handler_test.go`:

1. Where New instance asserts `` `class="primary"` ``, change to `` `class="btn btn-primary"` ``.
2. Where menu trigger asserts `` `class="workflow-card-menu-trigger"` ``, change to assert:

```go
`class="btn btn-ghost btn-icon workflow-card-menu-trigger"`
```

Keep assertions that only check absence of the trigger substring as:

```go
if strings.Contains(body, `workflow-card-menu-trigger`) {
```

(still valid — substring match).

- [ ] **Step 4: Run tests — expect FAIL**

```bash
cd server && go test ./cmd/server/ -run 'DialogMarkup|PanelMarkup|TestHome' -count=1
```

Expected: FAIL because templates still emit old classes.

- [ ] **Step 5: Commit failing test expectations**

```bash
git add server/cmd/server/dialog_markup_test.go server/cmd/server/panel_markup_test.go server/cmd/server/home_handler_test.go
git commit -m "$(cat <<'EOF'
test: expect btn-* classes for dialog, panel, and home controls

Pin the button composition contract before migrating templates.
EOF
)"
```

---

### Task 3: Migrate templates to `btn-*` (green tests)

**Files:**
- Modify all templates listed in the migration map above
- Modify: `web/src/styles/components/dialog.css` (slim close + dialog-actions selectors)

**Interfaces:**
- Consumes: Task 1 CSS; Task 2 test expectations
- Produces: all listed templates using `btn` composition; dialog close placement-only

- [ ] **Step 1: Migrate dialog close + dialog action footers**

Replace every `class="ghost dialog-close"` with `class="btn btn-ghost btn-icon dialog-close"` in:
- `home.html`, `stream.html`, `process.html`, `org_admin.html`, `platform_admin.html`, `substep_body.html`

Also migrate CTAs in those same files (`primary` → `btn btn-primary`, `danger` → `btn btn-danger`, etc.) while editing.

Replace override editor close:

```html
class="btn btn-ghost btn-icon js-close-substep-override"
```

- [ ] **Step 2: Slim `dialog.css`**

Update header close line to:

```css
 *       button.btn.btn-ghost.btn-icon.dialog-close
```

Replace `.dialog-close` block with placement-only:

```css
.dialog-close {
  transform: translate(var(--space-2), calc(-1 * var(--space-2)));
}
```

(If current transform uses different values, keep whatever is in the working tree today — only remove size/flex/padding rules that `.btn-icon` now owns.)

Update actions helpers:

```css
.dialog-actions .btn-primary {
  width: auto;
}

@media (--sm-down) {
  .dialog-head {
    flex-wrap: wrap;
    gap: var(--space-3);
  }

  .dialog-actions {
    flex-direction: column;
  }

  .dialog-actions .btn {
    width: 100%;
  }
}
```

- [ ] **Step 3: Migrate remaining pages and layout**

Apply the migration map to:
- `layout.html` — theme + account → `btn btn-outline btn-icon btn-lg nav-action …`
- `stream.html`, `process.html`, `org_admin.html`, `platform_admin.html`, `dpp.html`
- Auth pages: `login.html`, `signup.html`, `invite.html`, `reset_request.html`, `reset_set.html`
- `attachment_carousel.html` — download link + carousel nav
- Org/platform row actions → `btn btn-ghost btn-icon btn-xs` / `btn btn-ghost-danger btn-icon btn-xs`
- Home `workflow-card-menu-trigger` → `btn btn-ghost btn-icon workflow-card-menu-trigger`

Do **not** change `u-text-danger`, `process-meta-primary`, menu-item-danger, or text-link hash buttons.

- [ ] **Step 4: Run markup tests — expect PASS**

```bash
cd server && go test ./cmd/server/ -run 'DialogMarkup|PanelMarkup|StreamPreviewDialog|TestHomePicker|TestHomeStream|TestHomeWorkflowCard' -count=1
```

Expected: PASS (adjust `-run` if package test names differ; if narrow run misses failures, run full `go test ./cmd/server/ -count=1`).

- [ ] **Step 5: Commit**

```bash
git add server/templates web/src/styles/components/dialog.css
git commit -m "$(cat <<'EOF'
refactor(ui): migrate templates to composable btn-* classes

Replace bare primary/secondary/danger/ghost and domain icon buttons
with btn + variant + size + icon composition.
EOF
)"
```

---

### Task 4: Delete legacy button CSS and slim domain wrappers

**Files:**
- Modify: `web/src/styles/components/shared.css`
- Modify: `web/src/styles/components/org-admin.css`
- Modify: `web/src/styles/layout/chrome.css`
- Delete unused: `.hash-copy` in `shared.css` if still unused after grep

**Interfaces:**
- Consumes: templates already on `btn-*`
- Produces: no duplicate ghost/primary rules; domain classes placement-only

- [ ] **Step 1: Remove legacy button rules from `shared.css`**

Delete the entire block from `.primary, button {` through `button.ghost:hover…` (legacy primary/secondary/danger/ghost/secondary-small/disabled).

**Keep** a minimal unstyled `button` reset only if `reset.css` does not already cover it. Preferred: do **not** restyle bare `button` as primary — reliance on `.btn` is intentional. If any remaining `<button>` lacks `.btn`, fix the template in Task 3 instead of restoring the catch-all.

Slim `.workflow-card-menu-trigger` to:

```css
.workflow-card-menu-trigger {
  list-style: none;
  transform: translateY(-6px) translateX(6px);
  background: var(--card);
  color: var(--foreground);
}

.workflow-card-menu-trigger::marker {
  content: "";
}

.workflow-card-menu-trigger::-webkit-details-marker {
  display: none;
}

.workflow-card-menu-trigger:hover,
.workflow-card-menu[open] .workflow-card-menu-trigger {
  border-color: var(--primary);
  background: color-mix(in srgb, var(--primary) 12%, var(--card));
}
```

Remove width/height/padding/display flex from the trigger (owned by `btn-icon`). Keep `.workflow-card-menu` / dropdown / menu-item rules.

Delete `.hash-copy` if `rg hash-copy` shows CSS-only (no templates).

Move or keep `.pagination-btn` as:

```css
.pagination-btn {
  padding: var(--space-3) var(--space-2);
}
```

(Works with `btn btn-secondary pagination-btn`; optional later move into `button.css` — leave in `shared.css` for this task.)

- [ ] **Step 2: Delete `.user-action-button*` from `org-admin.css`**

Remove `.user-action-button`, `.user-action-button .icon-svg`, hover/disabled, and `.user-action-button-danger` entirely. Keep `.user-actions` flex container.

- [ ] **Step 3: Slim nav chrome**

In `chrome.css`, keep `.nav .nav-action` transitions/hover that use `--nav-hover`, but remove conflicting width/padding that fight `btn-lg` + `btn-icon`. Target structure:

```css
.nav .nav-action {
  /* domain only — sizing from btn btn-outline btn-icon btn-lg */
}

.nav .nav-action:hover,
.nav .account-menu[open] .nav-action {
  background: var(--nav-hover);
  text-decoration: none;
}

.nav .theme-toggle,
.nav .account-trigger {
  /* marker resets only if still needed */
}
```

Preserve account dropdown and theme icon show/hide rules. If removing the old bordered `nav-action` base breaks visual parity, ensure `btn-outline` provides the border; use `--nav-hover` in `.btn-outline:hover` already shipped in Task 1.

- [ ] **Step 4: Grep for leftovers**

```bash
rg -n 'class="(primary|secondary|danger|ghost|user-action-button)|button\.(primary|secondary|danger|ghost)|\.user-action-button' server/templates web/src/styles --glob '!**/plans/**'
```

Expected: no button class leftovers (ignore `u-text-danger`, `process-meta-primary`, `*-menu-item-danger`, workflow key named `secondary` in tests/config).

- [ ] **Step 5: Full verification**

```bash
task css:lint
cd server && go test ./cmd/server/ -count=1
```

Expected: CSS lint ok; Go tests PASS (or only unrelated failures — none expected for this change).

- [ ] **Step 6: Commit**

```bash
git add web/src/styles/components/shared.css web/src/styles/components/org-admin.css web/src/styles/layout/chrome.css
git commit -m "$(cat <<'EOF'
refactor(css): remove legacy button styles after btn-* migration

Slim domain wrappers to placement/open behavior; delete user-action-button.
EOF
)"
```

---

### Task 5: Manual spot-check checklist (no code unless bugs)

**Files:** none unless fixes required

- [ ] **Step 1: Visual QA** (dev server or Docker stack)

Verify:
1. Dialog close — ghost, squared, X centered, offset toward top-right corner
2. Home card ⋮ — ghost icon, open/hover still works
3. Org admin / platform admin row edit+delete — 28px icons; delete is `btn-ghost-danger`
4. Topbar theme + account — bordered outline 40px
5. Primary form submits + danger dialogs — filled styles
6. Stream/platform pagination — secondary + pagination padding
7. Attachment carousel prev/next — still readable over preview
8. Mobile (`--sm-down`) — dialog action buttons full-width column

- [ ] **Step 2: If bugs found, fix + commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
fix(css): polish button composition after visual QA
EOF
)"
```

Only if changes are needed; otherwise skip commit.

---

## Self-review

1. **Spec coverage:** Default 36px, xs 28 for org rows, `btn-*` rename, `btn-outline` for nav, `btn-ghost-danger` for destructive icons, full template migration, legacy CSS deletion — all have tasks.
2. **Placeholder scan:** No TBD/TODO; concrete CSS and class maps included.
3. **Type/name consistency:** `--btn-height-xs` / `btn-xs` / `btn-ghost-danger` / `btn-outline` used consistently across tasks.
4. **Out of scope called out:** text-link buttons and menu items intentionally excluded.

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-07-15-button-css-system.md`. Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
