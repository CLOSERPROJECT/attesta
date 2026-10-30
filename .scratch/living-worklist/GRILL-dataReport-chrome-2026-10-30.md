# Grill settled: Chrome Domain projectors → `dataReport`

**Date:** 2026-10-30  
**Status:** shared understanding (await user confirm before tickets/implement)

## Decisions

1. **Problem:** After thin wire (ticket 05), SPA still aggregates Domain for finalResults chrome (`project-final-results.ts` + plant paths). That reopens a Domain seam.
2. **Generic seam:** Any Domain-only aggregate chrome is authored logic Go **runs**; SPA **only renders**. Case #1 is today’s Feragame finalResults; the mechanism is not finalResults-specific.
3. **Language:** **Pure JSONata** produces the **full display DTO** (feasible via existing `evaluateExpr`). No plant projector Module in Go/TS; no declarative-ops DSL in v1.
4. **Kind:** `kind: dataReport` — Load/Commit server eval → `bindings.chromeDisplay[fieldId]`. Widget: `dataReportField`. Distinct from live `recap` / `readonly` / `valuePrint` (those stay client until a later slice).
5. **Refresh:** Recompute on Load + successful Commit only (Domain snapshot, not draft keystrokes).
6. **DTO shape:** Keep today’s sections/rows/tone display model (genericized); YAML expr targets it.
7. **Feragame purge:** Remove Feragame-specific ids/widgets/materials hardcodes from living BE + FE; plant copy and aggregation live in YAML (field id + expr + title).
8. **Docs:** Amend ADR-0005 + GLOSSARY (Chrome / Task form bundle / Load)—no new ADR.
9. **Delivery:** One local ticket after (or with) cleanup squash (`chore/living-thin-spa-cleanup` @ `e85374f`).

## Out of this slice

- Moving `recap` / `readonly` / `valuePrint` / Derived to server
- Roles / readyWhen chrome
- JobChrome delete, twin fixture DRY (still paused)

## Next (after confirm)

1. Optional: `/apply-worktree` cleanup → squash into `feat/attesta-rework`
2. `/to-tickets` → `.scratch/…/issues/` for dataReport
3. Implement / review loop
