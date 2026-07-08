# ADR-0004: CSS polish — token hygiene, zero inline styles, and maintainability

- **Status:** Proposed
- **Date:** 2026-07-08
- **Scope:** `web/src/styles/`, `web/src/styles.css`, `server/templates/stream.html`, `deployment/scripts/check-template-inline-styles.sh`, `docs/css.md`. Phase 2 additionally touches breakpoint files, lint tooling, and test infrastructure.
- **Out of scope:** Backend role resolution (covered by [ADR-0002](0002-role-color-appwrite-source.md) and [ADR-0003](0003-role-palette-storage.md)); Formata embed shadow-DOM overrides (`web/src/main.js`, `formata_builder.go`, `formata-arch/`); renaming `data-role-palette` to `data-palette`; editing `server/config/*.yaml`; Cerbos; new palette keys beyond the existing 17.
- **Related:** [ADR-0001](0001-css-architecture-refactor.md) (layered CSS, allowed inline patterns), [ADR-0002](0002-role-color-appwrite-source.md) (`data-role-palette`), [ADR-0003](0003-role-palette-storage.md) (palette-only storage), [implementation plan](../plans/0004-css-polish.md), `docs/css.md`.

## Context

ADR-0001 through ADR-0003 delivered a layered CSS architecture, Appwrite-backed role palettes, and palette-only persistence. The main app is in good shape but several items were **deferred** during the role-color work and remain on `master`:

| Signal | Current state on `master` |
|--------|---------------------------|
| Inline `style=` in templates | **2** — stream status color + progress bar width in `stream.html` |
| Role hue tokens | **Duplicated** — 17 palette pairs defined separately in `:root` and `[data-theme="dark"]` (~68 lines) |
| Stream status tokens | **No dark-theme overrides** — `--stream-active`, `--stream-done`, `--stream-terminated` are light-theme hex only |
| Token typo | `--tobar-pa-bg` used in `tokens.css` and `layout.css` (should be `--topbar-pa-bg`) |
| Palette attribute maps | **~180 lines** of `[data-role-palette=…]` rules embedded in `components.css` (2,133 lines total) |
| `components.css` size | Largest layer file; mixes timeline, actions, forms, org-admin, stream UI |
| Utility naming | Unprefixed `.mx-auto`, `.max-w-*` coexist with `u-*` utilities |
| Lint guardrails | Template inline-style script only; no stylelint |
| Visual regression | None |
| Formata `!important` overrides | 16 in `web/src/main.js` (ADR-0001 out of scope) |

A parallel branch (`feat/css-refactor-2`) and an unmerged stash explored overlapping CSS improvements under a different ADR numbering (`0002-role-palette-consolidation`). That work is **superseded** by master ADR-0002/0003 for backend and attribute naming. This ADR captures the **still-valid CSS-only improvements** from that research without re-opening palette storage or `data-role-palette` contracts.

### Prioritized delivery

Work splits into two phases:

- **Phase 1 (this ADR, first PR):** Token consolidation, typo fix, eliminate remaining template inline styles, extract palette maps to a dedicated layer file. Low risk; no Go behavior changes.
- **Phase 2 (follow-up PRs):** Structural and quality improvements — `components.css` split, utility naming, responsive gaps, stylelint, visual regression, accessibility pass. Formata shadow-DOM styling remains a separate effort.

## Decision

### Phase 1 — Token hygiene and zero inline styles

#### 1. Consolidate role hue tokens with `light-dark()`

Replace duplicated light/dark role palette definitions with a single `:root` block using CSS `light-dark()`:

```css
--role-red-bg: light-dark(oklch(50.5% 0.213 27.518), oklch(88.5% 0.062 18.334));
```

- Apply to all 17 palette pairs (`--role-*-bg`, `--role-*-border`) and `--role-ink`.
- Remove the duplicate role hue entries from `[data-theme="dark"]`; keep non-role dark overrides (panel, ink, substep status, etc.) in the dark block.
- Rely on existing `color-scheme: light` / `color-scheme: dark` on `:root` and `[data-theme="dark"]` for `light-dark()` resolution. The `data-theme` toggle in `web/src/main.js` remains unchanged.

#### 2. Add dark-theme stream status tokens via `light-dark()`

```css
--stream-active: light-dark(#5385d1, #7aa3e8);
--stream-done: light-dark(#b3b3b3, #6e6e6e);
--stream-terminated: light-dark(#dd8379, #c4786f);
```

`--stream-available` continues to reference `var(--accent)` (already theme-aware).

#### 3. Fix `--tobar-pa-bg` → `--topbar-pa-bg`

Rename the token in `tokens.css` and update the consumer in `layout.css`. No template changes.

#### 4. Replace stream inline style with `data-stream-status`

In `stream.html`, replace:

```html
style="--stream-color: var(--stream-{{ .Status }});"
```

with:

```html
data-stream-status="{{ .Status }}"
```

Add CSS rules mapping `data-stream-status` values (`available`, `active`, `done`, `terminated`) to `--stream-color` on `.stream-status-section-head`.

#### 5. Replace progress width inline style with `--progress` custom property

In `stream.html`, replace:

```html
style="width: {{ .Percent }}%;"
```

with:

```html
style="--progress: {{ .Percent }}%;"
```

Update `.process-progress-fill` to use `width: var(--progress, 0%)`.

After this change, **no** `style=` attributes remain in `server/templates/` (org-admin palette picker swatches use client-side JS, not server templates).

#### 6. Extract palette attribute maps to `role-palette.css`

Move all `[data-role-palette=…]` rules for `.role-pill` and `.substep`, plus new `data-stream-status` rules, from `components.css` into `web/src/styles/role-palette.css`.

Import order in `web/src/styles.css`:

```
tokens.css → role-palette.css → reset.css → …
```

Keep the attribute name **`data-role-palette`** (not `data-palette`) to match ADR-0002/0003 and existing templates.

#### 7. Tighten inline-style lint

Update `deployment/scripts/check-template-inline-styles.sh`:

- Remove `--stream-color` and `width: {{` from the allowed pattern.
- Allow only `--progress:` (the sole remaining runtime custom property).
- Goal: lint passes with **zero** `style=` in templates, or the script is removed entirely if the grep finds none.

Update `docs/css.md` to reflect zero inline styles and the new layer file.

### Phase 2 — Structural and quality improvements (deferred)

#### 8. Split `components.css` into sub-modules

Split the ~2,100-line file into move-only sub-modules under `web/src/styles/components/`:

| File | Contains |
|------|----------|
| `timeline.css` | `.timeline-*`, `.substep-*` (non-palette) |
| `actions.css` | `.action-*`, completion forms |
| `forms.css` | Shared form controls, inputs, buttons |
| `org-admin.css` | Org-admin dialogs, role picker, user tables |
| `stream.css` | Stream instance cards, status sections (non-palette) |
| `misc.css` | Remaining shared components |

Re-export via `components.css` as a barrel `@import` list. **No selector changes** in the first pass.

#### 9. Utility naming alignment

Either:

- **(Preferred)** Rename `.mx-auto`, `.max-w-prose`, `.max-w-7xl` to `u-mx-auto`, `u-max-w-prose`, `u-max-w-7xl` and update template references, or
- Document unprefixed layout utilities as an intentional exception in `docs/css.md` with a one-line rationale.

Do not introduce both patterns for new utilities.

#### 10. Token hygiene for scattered `rgba()`

Audit `layout.css`, `components.css`, and `pages.css` for hardcoded `rgba()` / hex values. Move repeated values to `tokens.css`; leave one-off compositional values only when they are truly local.

#### 11. Responsive improvements

- Introduce `--bp-phone`, `--bp-tablet`, `--bp-desktop` tokens in `tokens.css` matching current breakpoint values.
- Audit org-admin and process pages for mobile layout gaps; add overrides to `phone.css` where needed.
- Prefer breakpoint files over inline or page-scoped hacks.

#### 12. stylelint guardrails

Add stylelint to the web build with rules:

- No hex/rgb outside `tokens.css` (with explicit allowlist for `reset.css` if needed).
- No new `!important` in CSS files (existing `layout.css` exception documented).
- Run via `task css:lint` alongside the template inline-style script.

#### 13. Visual regression tests

Add Playwright (or equivalent) screenshot tests for golden pages:

- Layout/footer (light + dark)
- Process timeline with role pills
- DPP traceability
- Org-admin roles list
- Stream instances page

Store baselines in-repo; run in CI on template/CSS changes.

#### 14. Accessibility pass

Systematic review and fixes for:

- [x] `:focus-visible` on interactive controls missing visible focus rings — global ring on `button`, `a`, `input`, `select`, `textarea`, and `summary` in `reset.css`; existing component rules retained.
- [x] `prefers-reduced-motion` for animations/transitions added in CSS layers — `@media (prefers-reduced-motion: reduce)` block in `reset.css`.
- [x] Role pill contrast in dark mode (validate after `light-dark()` token change) — `.role-pill` uses `color: var(--role-ink)` on solid `var(--pill-bg)` backgrounds.

#### 15. Formata shadow-DOM styling (separate effort)

Extract JS string overrides from `web/src/main.js` into a dedicated CSS file injected into the Formata embed host; reduce `!important` usage; explore `::part()` where supported. **Not part of Phase 1 or Phase 2 delivery** — track as a future ADR if pursued.

## Consequences

### Positive

- **Zero template inline styles** — simpler CSP story, lint can be exhaustive.
- **~34 fewer lines** of duplicated role tokens; single edit point per palette hue.
- **Dark-theme stream colors** render correctly without a separate token block.
- **`role-palette.css`** isolates attribute-map rules from general components; easier to navigate.
- Phase 2 improvements reduce `components.css` churn risk and catch visual regressions early.

### Negative / trade-offs

- **`light-dark()` browser support** — requires browsers with `color-scheme` + `light-dark()` (Baseline 2024; acceptable for this app).
- **`--progress` still uses `style=`** in Phase 1 unless replaced by a class-per-bucket approach (rejected: progress is continuous 0–100).
- **Phase 2 stylelint** may surface many existing violations; fix incrementally or scope rules to changed files initially.
- **Visual regression** adds CI maintenance (baseline updates on intentional visual changes).

### Risks and mitigations

| Risk | Mitigation |
|------|------------|
| `light-dark()` renders wrong in dark mode | Manual smoke test light/dark on role pills, stream headers, org-admin |
| Palette maps break after file move | Move-only refactor; no selector renames; run template tests |
| Stash/branch conflicts if applied wholesale | This ADR explicitly rejects stash backend/template changes; CSS-only cherry-picks |
| `components.css` split causes import-order bugs | Barrel file preserves order; one commit per sub-module move |

## Supersedes

- Informal follow-up items from post–ADR-0001 analysis (`/tmp/attesta-css-followup-handoff.md`) that overlap with Phase 1 and Phase 2 here.
- Stash `0002-role-palette-consolidation` naming and `data-palette` attribute proposal — **do not apply**; use `data-role-palette` per ADR-0002/0003.

## Acceptance criteria

### Phase 1

- [x] Role hue tokens use `light-dark()`; `[data-theme="dark"]` has no duplicate role palette entries.
- [x] Stream status tokens use `light-dark()` for active/done/terminated.
- [x] `--topbar-pa-bg` typo fixed in tokens and layout.
- [x] `stream.html` uses `data-stream-status` and `--progress`; no other `style=` in templates.
- [x] `role-palette.css` imported after `tokens.css`; palette maps removed from `components.css`.
- [x] `docs/css.md` updated; `task css:lint` passes; `cd web && npm run build` succeeds.
- [x] Go template tests pass (`go test ./cmd/server/... -run Template`).

### Phase 2

- [ ] `components.css` split into sub-modules (move-only).
- [ ] Utility naming resolved (rename or documented exception).
- [ ] stylelint integrated into `task css:lint`.
- [ ] ~~Visual regression suite covers ≥5 golden pages.~~ Dropped to keep the toolchain simple; may be revisited in a future ADR.
- [x] Accessibility checklist items addressed or ticketed with explicit deferrals.

## References

- `web/src/styles/tokens.css` — role and stream tokens
- `web/src/styles/components.css` — palette attribute maps (to move)
- `web/src/styles/layout.css` — `--tobar-pa-bg` consumer
- `server/templates/stream.html` — remaining inline styles
- `deployment/scripts/check-template-inline-styles.sh` — template lint
- `docs/css.md` — style guide
