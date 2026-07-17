# Stream timeline — improvements backlog

Running list of stream timeline view improvements. Updated 2026-07-14 after body-first shell + a11y tracks merged into `refactor/domain-vocabulary` (`0a6ec46`).

Plans for completed work: `docs/superpowers/plans/2026-07-14-timeline-substep-shell-body-first.md`, `docs/superpowers/plans/2026-07-14-stream-timeline-a11y.md`.

---

## Done

| # | Area | Item | Notes |
|---|------|------|-------|
| 1 | Data model | Body-first substep shell display | Minimal `Body` stub at build time; `substepShellDisplay` prefers `Body`; DPP deduped |
| 4 | A11y | Step/substep accessible names, status labels, focus-on-open | Stable DOM ids, `aria-labelledby`, `replace` template func |
| 4 | A11y | Status pill exposed to assistive tech | `aria-label="Status: …"` on shell status span |
| 5 | Tests | Timeline selection and status rendering gaps | `decorateTimelineSelection`, content partial `?substep=`, terminated/skipped shell markup |

---

## Architecture & code quality

### 6. Replace `HideStatus bool` with explicit timeline mode

`StreamTimelineView`, `StreamTimelineStepView`, `StreamTimelineSubstepView`, and `StreamInstanceDetailView` use `HideStatus bool` for dashboard preview and DPP traceability. Align with `SubstepBodyView.Mode`:

- Proposed: `StreamTimelineMode` — `interactive` | `preview` | `traceability`
- Call sites: `main.go` preview (`HideStatus = true`), DPP history step, process page (default interactive)
- Templates: dispatch on mode instead of `if not .HideStatus`

### 7. Extract `stream_timeline_step` to its own component file

`stream_timeline_step` is reused by DPP via `dpp_history_step.html` but defined inside `stream_timeline.html`. Per `attesta-ui-components` skill: move to `components/stream_timeline_step.html` with focused tests (mirror `substep_shell` extraction).

---

## UX & product

### 2. Step-level progress at a glance

`StepSummaryView` shows substep **count** and step-level “Completed at” when all substeps are done. Missing: **“2 / 5 complete”** or rollup status (not started / in progress / done) on step accordion headers. Data mostly available in `buildStepSummary` + per-substep status walk.

### 3. SSE refresh — preserve accordion state

On `process-updated`, HTMX replaces entire `#process-page-content`. Server re-expands only the step containing the selected substep (`decorateTimelineSelection`); other open steps collapse. Options:

- Pass open step ids in query; merge in `decorateTimelineSelection`
- Client: record open `<details>` before swap; restore after `htmx:afterSwap`
- Narrow partial: refresh only open substep body (larger change)

### 8. Consistent accordion behavior (step vs substep)

Substeps: JS enforces single-open via `toggle` on `.js-process-substep-panel`. Steps: multiple `<details class="stream-timeline-step">` can stay open. Decide and document intentional UX, or mirror single-open at step level.

### 9. Visual language for step chrome

`stream-timeline.css` is layout/hover only. Substeps have rich status styling (`substep-available`, etc.). Step rows could reflect rollup state (border/background when step complete vs active).

### 10. Dashboard “timeline preview” naming

Stream dashboard (`/w/:key/`) exposes blueprint preview in a dialog (`buildWorkflowPreviewProcess`, `HideStatus: true`), not per-instance timeline on list rows. Clarify in CONTEXT/AGENTS if product intent differs.

---

## Performance (large timelines)

### 11. Lazy substep bodies

All substep bodies render on every page load and SSE refresh, even when collapsed. Load body HTML on first expand (`hx-get` on shell) when substep count grows.

### 12. Targeted SSE partials

Refresh only the changed substep panel instead of timeline + termination + DPP + downloads on every `process-updated` event.

---

## Naming & migration debt

See `docs/domain-naming-debt.md`. Still open for timeline-adjacent surfaces:

- `process.html` / `ProcessPageView` → `stream_instance_detail_*`
- `Process` / `ProcessStep` (progress map) naming overload

---

## Suggested next order

1. **UX 2** — step progress in `stream_step_summary`
2. **UX 3** — SSE accordion preservation
3. **Architecture 6** — `StreamTimelineMode` replaces `HideStatus`
4. **Performance 11–12** — only when timeline size or refresh jank becomes a problem
