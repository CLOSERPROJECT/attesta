# Handoff addendum: `_new` sentinel + Nuovo turno `each_key_duplicate`

**Date:** 2026-10-30  
**Repo:** `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta`  
**Branch tip (when written):** `fix/smoke-issues` @ `c287ef5` (ahead includes `a808e51` tasks-array, smoke cascade, CLI living seed)  
**For:** agent already undoing / replacing the provisional **`_new`** Commit wire — paste as **additional context**, not a second owner of that work.

**Human stance (locked):** the **`_new` thing was a mistake.** Do not deepen the sentinel; remove or replace it with a correct provisional/mint story.

---

## Why this addendum

Smoke / browser after Nuovo **turno** (line Job) hit:

```text
Svelte error: each_key_duplicate
Keyed each block has duplicate key `1st` at indexes 0 and 1
```

Secondary console noise (ignore unless your fix touches forms): font 404, AJV `unknown format "date"`, CursorBrowser dialog override.

**Diagnoser:** [Diagnose each_key_duplicate turno](b35f303d-d985-421e-8d9e-7ed6a1200125) — confidence high.

---

## Root cause (for your `_new` rewrite)

| | |
| --- | --- |
| **UI** | JobPanel `{#each instances as inst (inst.id)}` — `app/src/lib/living/JobPanel.svelte` ~151 |
| **Line mint** | Fixture `instanceId: $.shift` → evaluated id is enum value (`1st`, `2nd`, …) |
| **Bug** | Provisional Commit sends `jobInstanceId: '_new'` (`TaskForm.svelte` after `27e30d7`). Mint uniqueness checks `hasJobInstance(..., "_new")`, **not** the evaluated id → second Nuovo turno with same shift **appends** another `{jobId:"line", id:"1st"}` → keyed-each crash |
| **Not caused by** | `a808e51` tasks-as-array (unrelated). **Exposed by** `_new` vs pre-`_new` client that sent `jobInstanceId: value.shift` |

Intake (`instanceId: $load.id`) can look fine on first mint; line shift reuse is the sharp edge.

---

## Evidence / red tests (keep or absorb)

Untracked (or absorb into your fix) repro:

- `server/internal/living/line_shift_new_dup_repro_test.go`
  - `TestCommitLineShiftNewTwiceMintsDuplicateInstanceIDs` — two `_new`+`shift:1st` → two Domain rows
  - related mint-gate expectation around evaluated id

Server loci to revisit with `_new` removal:

- `server/internal/living/instance_exprs.go` — `shouldMintJobInstance` / uniqueness vs client id
- `server/internal/living/mutate.go` — append JobInstance without `(jobId,id)` uniqueness
- `app/src/lib/living/TaskForm.svelte` — provisional Commit `'_new'`
- Cascade #1 history: `.scratch/living-flat-port/HANDOFF-smoke-instanceId-2026-10-30.md` (partially stale SHAs)

---

## Acceptance hints (when `_new` is gone)

1. Nuovo turno → submit `shift: 1st` twice → second Commit **denied** or idempotent; Domain has **one** `line`/`1st`; UI does **not** throw `each_key_duplicate`.
2. Nuovo camion / register still mints load id without requiring client plant `computeIntakeInstanceId` empty-default failure (the problem `_new` was papering over — fix that path properly).
3. Do **not** “fix” only by changing JobPanel each-key; uniqueness belongs on mint/Mutate + correct client instance id wire.

---

## Out of scope for *this* addendum

- Tasks-array / OrderedObject / re-seed (already on branch)
- Squash into `feat/attesta-rework` (branch coordinator)
- Ticket 13 rename, FormPlan polish

---

## One-liner for the other agent

`_new` makes mint uniqueness check the sentinel instead of the evaluated Job instance id (`$.shift` → `1st`), so duplicate `line`/`1st` rows crash JobPanel keyed-each — remove the sentinel and restore uniqueness against the **evaluated** id (intake `$load.id` path included).
