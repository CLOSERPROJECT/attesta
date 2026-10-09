---
title: "Minimal: array Substeps, last-Substep seal"
status: proposed
multiplicity: array-fields
kind_to_substep: per-kind
sequence: keep-partial
open_close: first-last-substep
seal_granularity: whole-step
cost_posture: minimal
---

# Minimal: array Substeps, last-Substep seal

Design exploration for the cheapest coherent matrix cell that can still meet Feragame pilot acceptance criteria C1–C6. Not an ADR; not a product decision.

---

## 1. Cell summary

This cell stretches today’s Substep/Formata model instead of introducing child event records or a Step-level approve API.

| Axis | Pick | Meaning here |
|------|------|----------------|
| **Multiplicity** | `array-fields` | Each repeating event kind is an array inside one Formata Substep payload. Operators append/edit rows during the day; rows are draft until seal. |
| **Kind → Substep** | `per-kind` | One Substep for batch loads, one for UdC registrations (more kinds = more Substeps later). Stations are fields on a UdC row, not separate Substeps. |
| **Sequence** | `keep-partial` | Sequential unlock remains for the open and close Substeps only. Repeating-event Substeps are marked non-sequencing / draftable so they can accept concurrent draft entry after open. |
| **Open / close** | `first-last-substep` | First Substep opens the day; last Substep seals and closes. No new Step-level action. |
| **Seal granularity** | `whole-step` | Completing the last Substep freezes all of that Step’s evidence (all kind arrays) in one irreversible seal. |
| **Cost posture** | `minimal` | Prefer blueprint + small server changes over a workflow-engine rewrite. |

### Why this cell (cost justification)

Compared with other matrix values:

- `array-fields` reuses Formata schemas (Attesta already stores `ProcessStep.Data` as a map and already has JSON Schema `type: array` examples in `server/config/stream.yaml`). It avoids a new event collection (`child-event-log`) or multi-complete Substep semantics (`reenter-substep`).
- `per-kind` keeps kind distinction (C5) as Substep identity plus schema, without one giant mixed log or one Substep per station.
- `keep-partial` is cheaper than inventing full `steps-only` multi-org sequence semantics or dropping all in-Step sequence globally: open/close stay ordered; only the middle Substeps bypass unlock.
- `first-last-substep` + `whole-step` simulate Step seal with existing `CompleteSubstep` — no new “approve Step” concept.

The stretch is real: today there is no durable draft progress and no concurrent Substep availability. This exploration names the **smallest** code seams for those two gaps rather than redesigning Streams.

### Pilot blueprint shape

One Organization, one Step, four Substeps (illustrative IDs):

| Order | Substep | Role in the day | Sequencing |
|------:|---------|-----------------|------------|
| 1 | Open production day | Manual open; short confirmation form | Sequencing |
| 2 | Batch loads | Array of batch-load rows | Non-sequencing / draftable |
| 3 | UdC registrations | Array of UdC rows (station, weight, identity, …) | Non-sequencing / draftable |
| 4 | Close / seal day | Manual close; seals whole Step evidence | Sequencing |

One Stream instance ≈ one Feragame production day (all shifts that belong to that manually opened day). Shifts are data on rows or on the open form, not separate instances.

---

## 2. Behavior (operator day)

### Morning — open

1. An authorized Feragame operator creates (or opens) the Stream instance for today’s production day.
2. Only **Open production day** is available (`computeAvailability` / “Locked by sequence” behavior for later Substeps, unchanged for this gate).
3. Operator completes Open (ordinary `CompleteSubstep`: progress `done` + notarization). That is the explicit manual open (C6).

### During the day — interleaved draft entry

4. After Open is done, **Batch loads** and **UdC registrations** are both available for edit at once. Neither must be completed before the other accepts rows.
5. At a station, an operator adds a UdC row (weight, station id, container identity, …) and **saves draft**. The Substep stays incomplete; the array grows.
6. Elsewhere (or later), another operator adds a batch-load row and saves draft on the Batch Substep.
7. Events interleave freely across stations and kinds (C2). Many rows per kind per instance (C1).
8. Rows remain editable: operators can correct a weight or delete a mistaken row while the day is open (C3, pre-seal).

Close stays locked by sequence until the operator is ready to seal (see below). Operators do **not** “complete” Batch or UdC during the day in today’s sense — that would notarize early and unlock Close too soon under classic rules.

### End of day — seal via last Substep

9. When the day’s work for Feragame is finished, an authorized operator opens **Close / seal day**.
10. Completing Close is the irreversible seal:
    - Draft payloads on Batch and UdC are frozen (marked done with their final arrays, notarized).
    - Close itself is completed and notarized.
    - Whole-Step evidence is immutable afterward (C3). Kind distinction remains: two Substeps, two payloads (C5).
11. If this is the only Step in the pilot Stream, the instance has run out of Substeps and can mark process done the same way final completion does today (`isProcessDone` / `finalizeProcessIfDone` in `process_completion.go`).

Night work that crosses midnight stays on this instance if it belongs to the same manually opened production day (per the brief). No automatic midnight close.

### What operators do *not* do in this cell

- They do not finish “all of station 1” before station 2.
- They do not complete Batch before UdC (or vice versa) just to unlock the next form.
- They do not get a separate Step Approve button; Close is that moment.

---

## 3. Attesta impact (what changes / what stays)

### What stays (reuse)

- Stream / Stream instance / Step / Substep layering; one org owning the Step.
- Formata Substeps (`WorkflowSub` with `inputType: formata`, `Schema` / `UISchema`) and role gates via Cerbos (`CanComplete` with `sequenceOk`).
- Progress map on `Process` (`Progress[substepID] → ProcessStep`) and Mongo key encoding (`.` → `_`).
- HTTP complete path: `handleCompleteSubstep` → `ProcessService.CompleteSubstep` → `UpdateProcessProgress` + `InsertNotarization`.
- Display of nested/array values already walks slices in `flattenDisplayValues` / `collectDisplayValues`.
- No new Attesta layer between Step and Substep.

### What today’s code does (collision points)

Grounded in `server/cmd/server/`:

1. **Sequential unlock** — `computeAvailability` and `isSequenceOK` (in `main.go`) treat Substeps as a flat ordered list: exactly one incomplete Substep is available; later ones get UI reason `"Locked by sequence"` (`substep_views_builder.go`). Completing out of order returns 409 from `handleCompleteSubstep`.
2. **Complete = done + notarize** — `CompleteSubstep` always writes `State: "done"` and inserts a notarization. There is no first-class draft progress write for operator answers.
3. **No Step seal** — Steps group Substeps and own Organization; there is no Step-level approve/close. “All Substeps done” is how an instance finishes (`isProcessDone`).
4. **Overrides are not an event log** — local Formata adaptation (`SaveSubstepOverride`) is schema override before completion, not durable multi-row operational history.

### Smallest coherent change set

Aimed at C1–C6 without a workflow-engine rewrite.

#### A. Workflow flag: non-sequencing / draftable Substeps

Add an optional field on `WorkflowSub` (name illustrative), e.g. `draftable: true` or `sequencing: false`, set on Batch and UdC Substeps in the Feragame blueprint YAML.

Semantics:

- **Sequencing Substeps** (Open, Close): still participate in order gates.
- **Draftable Substeps**: after all preceding *sequencing* Substeps are done, they become available **together**, without requiring sibling draftable Substeps to be done. They do not block each other.

Concrete touch points:

- `computeAvailability` — when walking `orderedSubsteps`, skip draftable Substeps for the “only one available” cursor; mark all draftable Substeps available once the last completed sequencing predecessor (Open) is done; keep Close unavailable until the operator may seal (Close remains sequencing; available when Open is done and Close is the next sequencing Substep — draftable middles are ignored for the sequencing cursor).
- `isSequenceOK` — for a draftable Substep, return true iff Open (all prior sequencing Substeps) are done; do not require other draftable Substeps done. For Close, require prior sequencing done; do **not** require Batch/UdC `done` yet (they seal as part of Close).
- Cerbos still receives `sequenceOk`; policy can stay as today if the server computes the flag correctly.
- `buildSubstepViews` picks up availability via `computeAvailability` — UI unlock follows.

This is the core of `keep-partial`: sequence preserved for open/close only.

#### B. Draft save for array payloads

Add a narrow “save draft” path (new handler next to `handleCompleteSubstep`, e.g. `…/substep/{id}/draft`) that:

- Authorizes like complete (role + org + `sequenceOK` under the new rules).
- Writes `ProcessStep` with something other than terminal done — e.g. `State: "draft"` (or keep `pending` but persist `Data`) with the full Formata payload (arrays included).
- Does **not** insert notarization.
- Rejects writes if the Substep is already `done` (post-seal).

`UpdateProcessProgress` can likely be reused; today it is only called from completion with `done`. Draft is a second caller with different state/payload rules.

Formata UI: array field components already appear in catalog schemas; the pilot schemas need **repeating object arrays** (batch-load item, UdC item) plus add/edit/remove in the form. That is blueprint + Formata UI work more than a new domain store.

#### C. Last-Substep seal (overload Close completion)

On `CompleteSubstep` for the Close Substep (detect via blueprint convention: last Substep in the Step, or an explicit `sealsStep: true` flag — flag is clearer and still cheap):

1. Load draft Data for all draftable Substeps in the Step.
2. For each, write `State: "done"`, set `DoneAt` / `DoneBy`, persist final `Data`, insert notarization (same shape as today’s complete).
3. Then complete Close itself as today.
4. Reject Close if Open is not done; optionally warn if arrays are empty (product choice — not required by C1–C6).

After seal, draft saves and further edits return conflict (“Already completed” / immutable).

No separate Step entity transition is required for the pilot: whole-Step seal is “all Substeps in the Step now done with notarized payloads,” triggered by Close.

#### D. What we deliberately do *not* change in this cell

- No child event collection or per-row notarization.
- No Step-level HTTP resource for approve/close.
- No drop of sequence across Steps (future multi-org Streams keep today’s Step-to-Step unlock).
- No requirement that stations map to Substeps.

### Illustrative sequencing after the change

```text
Open (sequencing)     → complete once
Batch (draftable)     ─┐
UdC (draftable)       ─┴→ concurrent draft saves all day
Close (sequencing)    → complete once → seals Batch + UdC + Close
```

---

## 4. Tradeoffs vs C1–C6

| Criterion | Met? | How / tension |
|-----------|------|----------------|
| **C1** Many events per kind per day instance | Yes | Arrays grow via draft saves on per-kind Substeps. |
| **C2** Interleave across stations/kinds | Yes, with keep-partial | Concurrent draftable Substeps remove false kind sequencing. Stations are row fields, so station interleaving does not need per-station Substeps. |
| **C3** Editable until seal; immutable after | Yes, with draft + Close seal | Depends on shipping draft state and refusing post-done writes. Seal is simulated (last Substep), not a first-class Step API. |
| **C4** Low-change path exists | Yes — this cell | Touches availability helpers, one draft endpoint, seal side-effect on Close, Feragame blueprint/schemas. Avoids event-log redesign. |
| **C5** Sealed evidence distinguishes kinds | Yes | Separate Substeps and notarized payloads per kind. Future kinds = more draftable Substeps. |
| **C6** Open and close explicit and manual | Yes | First and last Substep completions. |

### Strengths

- Fits the “cheap path must remain possible” constraint.
- Kind separation is obvious in the timeline (one Substep card per kind).
- Operators keep a familiar Attesta complete gesture for open and seal.
- Multi-org Step sequencing elsewhere can remain untouched.

### Weaknesses / stretches

- **Seal is overloaded Substep completion**, not a real Step seal. Multi-org handoff later may want a clearer Step API; this cell accepts simulation for the pilot.
- **Draft state is new product surface** (save vs complete, crash recovery, concurrent editors on the same array). Minimal code path still needs conflict rules (last-write-wins vs optimistic locking) — unspecified here, called out below.
- **Large arrays in one Formata document** may hurt UX and payload size on a busy day; acceptable for pilot volume, weaker if UdC cycles explode.
- **Per-kind Substeps** do not model station as a first-class gate; reporting by station is query/filter over rows, not Substep structure.
- **`keep-partial` special-cases sequence** — a small conceptual fork in `isSequenceOK` / `computeAvailability` that future designs (`steps-only`) might replace cleanly.
- Notarization remains **per Substep at seal time**, not per event row. Row-level audit is weaker than `child-event-log`.

### Comparison note (not a decision)

An architectural cell with `child-event-log` + `steps-only` + `step-level-action` would fit the domain more cleanly and avoid draftable-Substep flags, at higher implementation cost. This exploration exists to keep a minimal cell on the table that still clears C1–C6.

---

## 5. Open questions

1. **Empty-day seal** — May Close succeed with zero batch loads and/or zero UdC rows, or must each array meet a minimum?
2. **Concurrency** — Two operators draft-saving the same kind array: last-write-wins, merge-by-row-id, or single-writer UX?
3. **Shift representation** — Shift on each event row, on the Open form only, or both?
4. **Flag naming / blueprint convention** — `draftable` vs `sequencing: false` vs `sealsStep` on Close; who authors the Feragame workflow YAML?
5. **Close availability** — Is Close available as soon as Open is done (operator can seal early), or only after an explicit “ready to close” cue? (Sequence rules allow early seal; policy may not want it.)
6. **Partial notarization story** — When Close seals, do we notarize each kind Substep separately (preferred above) or one bundled payload? Affects UNTP/trace mapping later.
7. **UI for draftable Substeps** — Show both kind Substeps as equal “available” cards; how do we avoid looking like two completeable forms under today’s complete CTA?
8. **Migration / generality** — Is `draftable` Feragame-only metadata, or a general Attesta feature other Streams may use?
9. **Failure mid-seal** — If Batch is marked done+notarized and UdC write fails, what rollback or retry makes evidence consistent?
10. **Future kinds** — Adding a third draftable Substep: does Close automatically seal all draftable siblings in the Step (yes in this design) — confirm that as a rule so seal stays whole-Step.

---

*End of exploration `01-minimal-array-substeps`.*
