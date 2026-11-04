# Locked — lineLoad readyWhen + Unlock fan-out (2026-11-04)

Branch: `feat/attesta-rework` (tip includes squash `89eb056` one-pallet weigh + QR).  
Grill + Config walk. **Do not re-litigate** without human. Board files only here — no engine code from the cutting session.

## Product

1. **Concurrent after `openTurno`:** once `openTurno` has a Submission on the Job instance, both `palletWeight` (group) and `lineLoad` are **path-unlocked**.
2. **`lineLoad` (Segna avanzamento) gated by `readyWhen`:** Worklist + Commit require Config `readyWhen`: Domain has ≥1 pallet with `$exists(weight) and $not(progress = 'done')`. Covers prior-shift leftovers **and** this-shift weighs.
3. **Not** unlock-on-first-child-submission of weigh. Closing `palletWeight` does **not** open Segna avanzamento.
4. **No Feragame-specific Go/TS.** Config only. Engine stays generic. Tests must fail closed if someone special-cases `"lineLoad"` in Go.
5. **Worklist folds `readyWhen`:** Unlock true ∧ `readyWhen` false → new status `unavailable`. Do **not** overload `available` (that remains role handoff). Home / mount stay **actionable-only** (unavailable never appears in Available bands or mount-ok).
6. **Done law unchanged:** one `lineLoad` Submission ⇒ `done`; Domain reads do not reopen.
7. **`containerReg`** still unlocks after `lineLoad` Submission (linear successor of `lineLoad`).

## Rejected for this board

- Unlock-on-childSubmission / unlock-on-first-weigh as the fan-out mechanism.
- Plant-named Go hooks (`lineLoad`, `feragame`, leftovers completeness grammar).
- Reviving completeness / leftovers as product Unlock grammar (Domain `readyWhen` + reads only).

## Unlock fan-out Config grammar (ONE locked)

Today `IsTaskUnlocked` always takes predecessor = previous ordered sibling (`OrderedIDs[index-1]`). Linear order cannot unlock two successors of `openTurno`.

**Locked opt-in key:** `unlockFrom`

- Optional string on a Task (same Job sibling list — top-level Job Tasks, or children under one group).
- Meaning: predecessor for Unlock satisfaction is the named Task’s satisfaction (leaf → one Submission; group → **Close**, not one child Submission), **not** the previous ordered sibling.
- **Default (omit):** previous ordered sibling — current linear law unchanged (e.g. intake `lots` → `weightCheck` after Close).
- **First Task** in a sibling list stays always unlocked; do not author `unlockFrom` on it (validate reject).
- **Validate:** `unlockFrom` must be a non-empty string id of another Task in the **same** sibling list, at a **strictly earlier** authored index (no self, no forward/cycle). Unknown id → Config load error.
- Scope of this board: top-level Job fan-out (`lineLoad.unlockFrom: openTurno`). Nested child lists keep the same grammar if authored later; no plant need yet.

**Not** experiment-era `unlockFrom: instance` (Job-instance unlock). Value is always a **Task id**.

## Frozen Config — plant `line` Job fragment

```yaml
- id: openTurno
  # … unchanged …
- id: palletWeight
  kind: group
  # default unlockFrom = previous sibling = openTurno (linear)
  # … unchanged group / weigh …
- id: lineLoad
  unlockFrom: openTurno
  readyWhen: >-
    $count($domain.pallet.*[$exists(weight) and $not(progress = 'done')]) >= 1
  roles: [line]
  # … reads / schema / onSubmit unchanged …
- id: containerReg
  # default unlockFrom = previous sibling = lineLoad (linear)
  # … unchanged …
```

After `openTurno` Submission: `palletWeight` unlocked (linear); `lineLoad` unlocked via `unlockFrom` but Worklist/`Commit` still require `readyWhen`.

## Worklist status law (amend ADR-0005 in ticket 01)

Wire enum becomes:

`locked` | `unavailable` | `available` | `actionable` | `done`

Fold order (leaves):

1. Path Unlock / Submission → `locked` | unlocked | `done` (unchanged).
2. If unlocked and not done: evaluate Task `readyWhen` (omit/blank ⇒ true; eval fail ⇒ false, fail closed). False ⇒ **`unavailable`** (stop — do not apply role split).
3. Else role split: eligible ⇒ `actionable`, else `available`.

Home Available, task-form mount, and overview navigation remain **`actionable` only**. Overview may show `unavailable` chrome (visible, not navigable) — SPA label/tag only in 01; no second gate body in TS.

Commit path already evaluates `ReadyWhenOk` for Cerbos; Worklist must use the same `evaluateReadyWhen` / `readyWhenOkForCommit` semantics (Domain + catalogs), not a SPA twin.

## Deferred

- Glossary/`unavailable` wording polish beyond ADR-0005 amend (ok to touch GLOSSARY Worklist enum line in 01 if needed for consistency).
- Nested-group `unlockFrom` plant usage (grammar locked; no Feragame authoring).
- Prefetch / toast copy for unavailable Segna avanzamento (SPA tag is enough for 01).
