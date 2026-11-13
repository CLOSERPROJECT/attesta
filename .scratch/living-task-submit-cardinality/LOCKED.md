# Locked — Task submit cardinality + lot session (2026-11-13)

Grill session. **Do not re-litigate** without human. Design only here until implementation is asked.

Supersedes plant assumptions in
[`.scratch/living-lineload-ready/LOCKED.md`](../living-lineload-ready/LOCKED.md)
(weigh gates load; linear/group Unlock fan-out as plant law;
repeatable groups for multi-pass line work). Engine may still implement
`unlockFrom` / Task groups; **Feragame must not author them** for this model.

Canonical ADR: [`docs/adr/0008-living-task-submit-cardinality.md`](../../docs/adr/0008-living-task-submit-cardinality.md).  
Implementation: [`IMPLEMENTATION.md`](./IMPLEMENTATION.md).

## Product

1. **Less fixed pipeline, more available actions.** Operator work is Tasks that
   are available given Domain conditions (`readyWhen` + roles), not a fixed
   weigh → load → register Unlock chain.
2. **Task = form with submit cardinality.** Authored `submit: once` (default) or
   `submit: many`. Not repeatable Task groups for multi-pass work.
3. **Once:** one successful Submission ⇒ Worklist `done` (unchanged law).
4. **Many:** Commit appends a Submission (audit / history); Worklist status stays
   `actionable` / `unavailable` from `readyWhen` + roles — **never** `done` from
   one Commit. No Spawn/Close on the Task. Prefer `onCommit.navigation: stay`
   when the operator should submit again immediately.
5. **No Close chrome on many-submit Tasks.** Phase endings are sibling **once**
   Tasks or Domain exhaustion (`readyWhen` false), not group Close.
6. **Peer many-submit inside an open lot; Domain weigh-before-progress.**
   `weigh` / `markProgress` / `containerReg` stay peer Tasks (no Unlock
   pipeline). `markProgress` only treats pallets that already carry Domain
   `weight` as progress-eligible (`readyWhen` + `reads.where`). Progress does
   not gate `containerReg`.
7. **Lot session (Domain).** Entity rename **`load` → `lot`**. Single status
   enum: **`open` | `parked` | `fullyProcessed`**.
   - `startLot` → `open` (at most one `open`/`parked` lot under the open working day).
   - `closeTurno` while lot is `open` → **`parked`** (soft park; do not require lot finished).
   - next `openTurno` while `parked` → **`open`** (auto-resume; no resume Task).
   - `closeLot` → **`fullyProcessed`** when every pallet on that lot has
     **weight** and **`progress = done`**. This Commit is the sole writer of
     terminal `fullyProcessed` (container register no longer sets it).
8. **`parked` freezes** weigh / markProgress / containerReg / closeLot until
   resumed to `open`.
9. **Intake:** many-submit lot `register` + once `weightCheck`. No lot-group
   Close. `weightCheck` available when ≥1 lot for the truck (`readyWhen`);
   after `weightCheck` Submitted, further register goes unavailable.
10. **Unlock:** engine **keeps** `unlockFrom` mechanics; **Feragame does not
    author it**. Prefer `readyWhen` whenever Domain already has the fact.
    Likely removed later — not part of the plant model.
11. **Task groups:** engine may retain; **Feragame fixture authors only leaf
    Tasks** (once or many). No `kind: group` for intake lots or line multi-pass.
12. **Dumb executor unchanged.** Plant law in Config only.

## Rejected

- Repeatable Task groups as the multi-submit / spawn-many product model for Feragame.
- Weigh → progress → register Unlock pipeline (Domain weigh-before-progress
  on `markProgress` alone is locked; peers stay non-pipeline).
- Forcing lot completion before `closeTurno`.
- Separate boolean `fullyProcessed` beside `open`/`parked`/`closed`.
- Explicit park/resume Tasks (park/resume are shift bookend transforms).
- Encoding plant open/close exclusivity in `startsWhen` or `unlockFrom`.

## Feragame line shape (target)

Job `line` (sketch — Italian titles stay Config content):

| Task | submit | Gate sketch |
|------|--------|-------------|
| `openTurno` | once | readyWhen: open working day / shift exclusivity (today) |
| `startLot` | once* | readyWhen: no lot `open`/`parked`; eligible lots exist |
| `weigh` | many | readyWhen: a lot is `open`; reads unweighed pallets of that lot |
| `markProgress` | many | readyWhen: lot `open` ∧ ≥1 weighted unfinished pallet; reads weighted unfinished pallets of that lot |
| `containerReg` | many | readyWhen: lot `open`; books containers for that lot |
| `closeLot` | once* | readyWhen: lot `open` ∧ all its pallets weighed ∧ progress done → set `fullyProcessed` |
| `closeTurno` | once | readyWhen: shift open; transform parks `open` → `parked` |

\*Once per *activation* in Worklist terms still means Submission can mark `done`
on that Job instance path — if `startLot`/`closeLot` must run again for a
*later* lot on the same shift Job instance, they are **`submit: many`** with
`readyWhen` (same cardinality rule as other multi-pass leaves). Prefer
**`submit: many`** for `startLot` / `closeLot` so one line Job instance can
process multiple lots across a shift without inventing group instances.

Locked cardinality for start/close lot: **`submit: many`** + Domain status
gates (not Task groups).

## Feragame intake shape (target)

| Task | submit | Gate sketch |
|------|--------|-------------|
| `truckArrive` | once | readyWhen: open working day |
| `register` | many | readyWhen: truck instance exists; not after weightCheck |
| `weightCheck` | once | readyWhen: ≥1 lot for truck |

## Glossary / ADR followers

- Root [`GLOSSARY.md`](../../GLOSSARY.md) — Task submit cardinality; lot entity;
  status enum; Submission/Worklist done law; Unlock plant guidance.
- [`docs/adr/0008-living-task-submit-cardinality.md`](../../docs/adr/0008-living-task-submit-cardinality.md).
- Amend note on ADR-0005 done law for `submit: many` (see ADR-0008).
