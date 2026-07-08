# Plan: ADR-0004 — CSS polish (token hygiene, zero inline styles, maintainability)

- **ADR:** [0004-css-polish.md](../adr/0004-css-polish.md)
- **Prerequisite:** [ADR-0001](../adr/0001-css-architecture-refactor.md), [ADR-0002](../adr/0002-role-color-appwrite-source.md), and [ADR-0003](../adr/0003-role-palette-storage.md) merged on `master` (#141, #143, #144).
- **Branch:** `feat/css-polish` from `master`.
- **Date:** 2026-07-08

## Goal

**Phase 1:** Eliminate remaining template inline styles, consolidate theme tokens with `light-dark()`, fix the topbar token typo, and extract palette attribute maps to `role-palette.css`.

**Phase 2:** Improve long-term maintainability — split `components.css`, align utility naming, add stylelint and visual regression, address responsive and accessibility gaps.

**Explicitly not in scope:** Backend role resolution, Appwrite storage shape, `data-role-palette` renaming, Formata shadow-DOM overrides, applying the obsolete `feat/css-refactor-2` stash wholesale.

## Settled decisions (from design review)

| Topic | Decision |
|-------|----------|
| Attribute name | Keep `data-role-palette` (not `data-palette` from stash) |
| Stream status | `data-stream-status="{{ .Status }}"` on `.stream-status-section-head` |
| Progress bar | `style="--progress: {{ .Percent }}%;"` → `width: var(--progress, 0%)` in CSS |
| Token consolidation | `light-dark()` in `:root`; remove duplicate role hues from dark block |
| Palette CSS file | `role-palette.css` after `tokens.css` in import stack |
| Stash / old branch | Do not merge; cherry-pick CSS ideas only per ADR-0004 |
| Formata `!important` | Separate future effort; not Phase 1 or 2 |

## Current vs target

### Template inline styles

```
# Before (master)
stream.html: style="--stream-color: var(--stream-{{ .Status }});"
stream.html: style="width: {{ .Percent }}%;"

# After (Phase 1)
stream.html: data-stream-status="{{ .Status }}"
stream.html: style="--progress: {{ .Percent }}%;"   # sole allowed pattern
# Target end state: zero style= in templates (progress uses custom prop only)
```

### Role tokens (`tokens.css`)

```
# Before
:root { --role-red-bg: oklch(...); … }
[data-theme="dark"] { --role-red-bg: oklch(...); … }   # 34 duplicate lines

# After
:root { --role-red-bg: light-dark(oklch(light), oklch(dark)); … }
[data-theme="dark"] { /* no role hue duplicates */ }
```

### CSS layer stack

```
# Before
tokens.css → reset.css → utilities.css → layout.css → components.css (2133 lines, includes palette maps) → …

# After (Phase 1)
tokens.css → role-palette.css → reset.css → … → components.css (smaller)

# After (Phase 2)
… → components/index.css (barrel) → components/timeline.css, actions.css, …
```

### Lint allowed patterns

```
# Before
--stream-color:|width:[[:space:]]*\{\{

# After
--progress:
```

---

## Phase 1 — Token consolidation (`light-dark()`)

**Objective:** Single definition per role hue and stream status color; shrink `[data-theme="dark"]` block.

### 1.1 Role palette tokens

File: `web/src/styles/tokens.css`

- Convert `--role-ink` and all 17 `--role-*-bg` / `--role-*-border` pairs to `light-dark(light, dark)` using existing values from `:root` (light) and `[data-theme="dark"]` (dark).
- Delete lines 101–135 (duplicate role hues in dark block).
- Leave all non-role dark overrides untouched.

### 1.2 Stream status tokens

Same file, `:root` block:

```css
--stream-active: light-dark(#5385d1, #7aa3e8);
--stream-done: light-dark(#b3b3b3, #6e6e6e);
--stream-terminated: light-dark(#dd8379, #c4786f);
```

### 1.3 Topbar typo

- `tokens.css`: `--tobar-pa-bg` → `--topbar-pa-bg` (both `:root` and dark block).
- `layout.css`: `var(--tobar-pa-bg)` → `var(--topbar-pa-bg)`.

### 1.4 Tests / verification

- `cd web && npm run build`
- Manual: toggle dark mode; verify role pills, timeline substeps, org-admin pills, stream section headers.

**Exit criteria:** Built CSS contains `light-dark(` for role hues; dark block has no `--role-red-bg` etc.; topbar renders in light and dark.

---

## Phase 2 — Extract `role-palette.css`

**Objective:** Move attribute-map rules out of `components.css`.

### 2.1 Create `web/src/styles/role-palette.css`

Move from `components.css` (approx. lines 463–640):

- `.role-pill[data-role-palette=…]` (17 keys + fallback)
- `.substep[data-role-palette=…]` (17 keys + fallback)

Add new rules:

```css
.stream-status-section-head[data-stream-status="available"] { --stream-color: var(--stream-available); }
.stream-status-section-head[data-stream-status="active"] { --stream-color: var(--stream-active); }
.stream-status-section-head[data-stream-status="done"] { --stream-color: var(--stream-done); }
.stream-status-section-head[data-stream-status="terminated"] { --stream-color: var(--stream-terminated); }
```

### 2.2 Update `web/src/styles.css`

```css
@import "./styles/tokens.css";
@import "./styles/role-palette.css";
@import "./styles/reset.css";
/* … */
```

### 2.3 Delete moved rules from `components.css`

Move-only; do not rename selectors or change specificity.

**Exit criteria:** `rg 'data-role-palette' web/src/styles/components.css` returns no matches; `role-palette.css` contains all palette maps.

---

## Phase 3 — Template inline style elimination

**Objective:** Replace stream inline styles; achieve zero or single-pattern `style=` usage.

### 3.1 `server/templates/stream.html`

Replace stream status head:

```html
<div class="stream-status-section-head" data-stream-status="{{ .Status }}">
```

Replace progress fill:

```html
<div class="process-progress-fill" style="--progress: {{ .Percent }}%;"></div>
```

### 3.2 `web/src/styles/components.css` (or `stream.css` after Phase 2 split)

Ensure `.process-progress-fill` uses:

```css
width: var(--progress, 0%);
```

(Verify rule exists or add if currently `width` without custom property.)

### 3.3 Lint script

File: `deployment/scripts/check-template-inline-styles.sh`

```bash
allowed_pattern='--progress:'
```

### 3.4 Grep verification

```bash
rg 'style=' server/templates/
```

Expected: only `--progress:` lines in `stream.html` (or zero if a non-inline approach is adopted later).

**Exit criteria:** `task css:lint` passes; stream page renders correct status colors and progress bars.

---

## Phase 4 — Documentation (Phase 1)

| Doc | Update |
|-----|--------|
| `docs/css.md` | Add `role-palette.css` to layer table; update inline-style section to `--progress` only; reference ADR-0004 |
| `docs/adr/0004-css-polish.md` | Tick Phase 1 acceptance criteria when done |
| `AGENTS.md` | Optional one-line: CSS layers include `role-palette.css` |

**Exit criteria:** `docs/css.md` matches implementation.

---

## Phase 5 — Split `components.css` (Phase 2)

**Objective:** Move-only decomposition; no visual changes.

### 5.1 Create `web/src/styles/components/` directory

| File | Approx. content |
|------|-----------------|
| `timeline.css` | `.timeline-*`, `.substep-*` (excluding palette maps already in `role-palette.css`) |
| `actions.css` | `.action-*`, action cards, completion UI |
| `forms.css` | Inputs, labels, buttons, form layout |
| `org-admin.css` | `.roles-*`, `.role-palette-*`, org-admin tables/dialogs |
| `stream.css` | `.stream-*`, `.process-progress-*`, instance cards |
| `shared.css` | `.panel`, `.pill`, `.stack` modifiers, pagination, etc. |

### 5.2 Barrel file

Replace `components.css` body with:

```css
@import "./components/shared.css";
@import "./components/timeline.css";
/* … */
```

Or keep `components.css` as the barrel at `web/src/styles/components.css` importing `./components/*.css`.

### 5.3 Process

One sub-module per commit recommended for easier review/revert.

**Exit criteria:** No single file > ~800 lines; `npm run build` unchanged output (diff `web/dist` optional).

---

## Phase 6 — Utility naming (Phase 2)

**Objective:** Resolve `mx-auto` / `max-w-*` vs `u-*` inconsistency.

### 6.1 Inventory

```bash
rg 'mx-auto|max-w-prose|max-w-7xl' server/templates web/src
```

### 6.2 Preferred path: rename to `u-*`

| Current | Target |
|---------|--------|
| `.mx-auto` | `.u-mx-auto` |
| `.max-w-prose` | `.u-max-w-prose` |
| `.max-w-7xl` | `.u-max-w-7xl` |

Update `utilities.css` and all template/class references.

### 6.3 Alternative

If rename churn is too high, document the exception in `docs/css.md` § Utilities and do not add new unprefixed layout utilities.

**Exit criteria:** Grep-clean or documented; no mixed naming for new code.

---

## Phase 7 — Token hygiene audit (Phase 2)

**Objective:** Reduce scattered color literals outside `tokens.css`.

### 7.1 Audit

```bash
rg 'rgba\(|#[0-9a-fA-F]{3,8}' web/src/styles --glob '!tokens.css'
```

### 7.2 Triage

- Repeated values → new tokens in `tokens.css`.
- One-off compositional values → leave with comment if justified.

**Exit criteria:** No new hex outside `tokens.css`; audit list empty or documented exceptions in `docs/css.md`.

---

## Phase 8 — Responsive improvements (Phase 2)

### 8.1 Breakpoint tokens

Add to `tokens.css`:

```css
--bp-phone: 640px;   /* match phone.css */
--bp-tablet: 900px;
--bp-desktop: 1200px;
```

Use in media queries when touching breakpoint files (optional migration).

### 8.2 Page audits

Manual check on phone width:

- `/org-admin/users`, `/org-admin/roles`
- `/process/:id`
- `/w/:workflow/stream` (instances)

Add overrides to `phone.css` for identified gaps.

**Exit criteria:** No horizontal scroll on 375px viewport for audited pages.

---

## Phase 9 — stylelint (Phase 2)

### 9.1 Install

```bash
cd web && npm install -D stylelint stylelint-config-standard
```

### 9.2 Config (`.stylelintrc.json` or `stylelint.config.js`)

Initial rules:

- `color-no-hex: true` with `ignoreFiles: ['**/tokens.css']`
- `declaration-no-important: true` with ignore for known `layout.css` exception
- `max-nesting-depth: 3` (match existing nesting style)

### 9.3 Task integration

Extend `Taskfile.yml` `css:lint`:

```yaml
css:lint:
  cmds:
    - bash deployment/scripts/check-template-inline-styles.sh
    - cd web && npx stylelint "src/styles/**/*.css"
```

**Exit criteria:** `task css:lint` runs both checks; passes on main branch after fixing violations or scoped allowlist.

---

## Phase 10 — Visual regression (Phase 2)

### 10.1 Tooling

Add Playwright to repo (or extend existing test harness if present).

### 10.2 Golden pages

| Route | Notes |
|-------|-------|
| `/` or `/dashboard` | Layout, footer, topbar |
| `/process/:id` | Timeline, role pills (use seed process) |
| `/01/{gtin}/10/{lot}/21/{serial}` | DPP public page |
| `/org-admin/roles` | Role pills, palette picker |
| Stream instances | Workflow-scoped stream page |

Capture light + dark where theme matters.

### 10.3 CI

Add job or step to `tests.yml` (optional: compare against stored snapshots).

**Exit criteria:** ≥5 pages with screenshot baselines; CI fails on unintended visual drift.

---

## Phase 11 — Accessibility (Phase 2)

### 11.1 Focus

- Add or verify `:focus-visible` styles on buttons, links, pagination, org-admin dialogs.
- Add `@media (prefers-reduced-motion: reduce)` to disable non-essential transitions.

### 11.2 Contrast

After Phase 1 `light-dark()` change, spot-check role pills in dark mode (WCAG AA for text on pill backgrounds).

**Exit criteria:** Checklist in ADR-0004 ticked or explicit deferrals documented.

---

## Implementation order

### Phase 1 (first PR — `feat/css-polish`)

```
Phase 1 (tokens + typo)
    ↓
Phase 2 (role-palette.css extract)
    ↓
Phase 3 (stream template + lint)
    ↓
Phase 4 (docs)
```

Phases 1–3 can be a single commit if preferred; separate commits ease review.

### Phase 2 (follow-up PRs on `master`)

```
Phase 5 (components split)
    ↓
Phase 6 (utilities) + Phase 7 (token audit)   # parallel OK
    ↓
Phase 8 (responsive)
    ↓
Phase 9 (stylelint) → Phase 10 (visual regression)
    ↓
Phase 11 (a11y)
```

Phases 9–10 can land together. Formata shadow-DOM work is **not** scheduled here.

## Verification

```bash
# Phase 1
task css:lint
cd web && npm run build
cd server && go test ./cmd/server/... -run 'Template' -count=1

# Phase 2 (when applicable)
task css:lint          # includes stylelint
cd web && npm run build
# Playwright visual tests (TBD command)
```

### Manual QA (Phase 1)

1. **Light/dark toggle** — role pills on process, action list, DPP, org-admin show correct hues.
2. **Stream instances** — section headers colored per status (available/active/done/terminated) in both themes.
3. **Progress bars** — stream instance cards show correct fill width.
4. **Platform admin topbar** — `.topbar-pa` background correct in light (#000) and dark (#fff) after typo fix.

### Manual QA (Phase 2)

1. **Responsive** — org-admin and process usable at 375px width.
2. **Visual regression** — review screenshot diffs on intentional CSS changes.
3. **Keyboard** — tab through org-admin role dialog; focus visible.

## File checklist

| File | Phase |
|------|-------|
| `web/src/styles/tokens.css` | 1 |
| `web/src/styles/layout.css` | 1 |
| `web/src/styles/role-palette.css` | 2 (new) |
| `web/src/styles.css` | 2 |
| `web/src/styles/components.css` | 2, 5 |
| `server/templates/stream.html` | 3 |
| `deployment/scripts/check-template-inline-styles.sh` | 3 |
| `docs/css.md` | 4 |
| `docs/adr/0004-css-polish.md` | 4 |
| `web/src/styles/components/*.css` | 5 |
| `web/src/styles/utilities.css` | 6 |
| `web/package.json` | 9 |
| `Taskfile.yml` | 9 |
| Playwright config + tests | 10 |

## Out of scope reminders

- `feat/css-refactor-2` branch and `stash@{0}` — do not merge; obsolete backend/template changes
- `data-palette` rename — keep `data-role-palette`
- Appwrite / Go role resolution — ADR-0002/0003 complete
- Formata `web/src/main.js` shadow overrides — future ADR
- Editing `server/config/*.yaml`
- New palette keys beyond existing 17

## Cleanup after Phase 1 merges

- Delete remote branch `feat/css-refactor-2` if no longer needed.
- Drop `git stash` entry `stash@{0}` (CSS ideas captured in this plan).
- Archive `/tmp/attesta-css-followup-handoff.md` pointers to ADR-0004.
