---
title: "Architectural: child event log + Step seal"
status: proposed
multiplicity: child-event-log
kind_to_substep: per-kind
sequence: steps-only
open_close: step-level-action
seal_granularity: whole-step
cost_posture: architectural
---

# Architectural: child event log + Step seal

Design probe for the Feragame × Attesta pilot. Not an ADR. Cell chosen: first-class repeating events, real Step seal, and sequential unlock only between Organization Steps — not inside Feragame’s day Step.

---

## 1. Cell summary

This exploration takes the clean domain cell from the architectural cut:

| Axis | Choice | Intent |
|------|--------|--------|
| **Multiplicity** | `child-event-log` | Batch loads and UdC registrations are durable child records under a Substep, not one form submit that marks the Substep done. |
| **Kind → Substep** | `per-kind` | Blueprint has one Substep for “Batch loads” and one for “UdC registrations” (more kinds become more Substeps later). |
| **Sequence** | `steps-only` | Flat “one unlocked Substep at a time” does **not** apply inside the Feragame Step. Sequence remains the multi-org handoff between Steps. |
| **Open / close** | `step-level-action` | Open and seal are Step-scoped actions, not first/last Substep completions. |
| **Seal granularity** | `whole-step` | One irreversible seal freezes all event evidence for that Step. |
| **Cost posture** | `architectural` | Prefer a coherent model over stretching today’s Substep completion path. |

**Why this cell (not a cheaper mix):** Feragame’s day is many interleaved events of several kinds, editable until an explicit close. Today Attesta treats Substep completion as “write `progress` done + notarize payload” (`ProcessService.CompleteSubstep` in `process_completion.go`), and unlock as a flat predecessor chain (`computeAvailability` / `orderedSubsteps` in `main.go`). Simulating a day log with array fields or re-entry would paper over that seam. This cell names the missing concepts: **child events**, **draft until Step seal**, and **Step as a sealable unit** while keeping Steps as the multi-org sequence.

**Pilot blueprint shape (unchanged from the brief):** one Organization (Feragame), one Stream instance per production day, one Step owning that day’s work, multiple Substeps as kind-scoped homes for event logs. Shifts remain data on events or on the instance — not separate instances.

**New concepts (relative to today’s Attesta):**

1. **Child event** — a first-class record belonging to a Stream instance and a parent Substep (kind). Many events per Substep; kinds stay distinguishable.
2. **Draft vs sealed (Step)** — while the Step is open, events are mutable; sealing the Step makes that Step’s event evidence immutable and is the notarization / handoff boundary.
3. **Step open / Step seal** — explicit Step-level actions. Substeps no longer need to act as open/close gates.

---

## 2. Behavior (operator day)

### Start of day — open the Step

1. An authorized Feragame operator creates (or selects) the Stream instance for the production day — still the existing instance lifecycle.
2. They **open** Feragame’s Step with a Step-level action (e.g. “Open day” / “Start Step”). Until open, kind Substeps do not accept new events (or accept only as preview — product detail open).
3. Opening does **not** complete a Substep and does **not** run Substep notarization. It records Step state: open, who, when.

There is no “open Substep” that must be finished before batch/UdC work can start.

### During the day — append and edit events

Inside the open Step, operators work against **kind Substeps** that behave as **log panels**, not as one-shot forms:

| Substep (kind) | What operators do |
|----------------|-------------------|
| **Batch loads** | Create a child event per batch load (identity/context fields TBD). Many times per day. |
| **UdC registrations** | Create a child event per weigh/register before stock. Many times per day, interleaved across stations. |

**Interleaving:** Station 1 stocks a UdC, then station 2, then station 1 again, with batch loads mixed in. Both kind Substeps are available **at the same time**. Creating or editing a UdC event never requires “completing” the Batch-loads Substep first. That removes today’s “Locked by sequence” behavior for in-Step Substeps (`substep_views_builder.go` reason string when status is locked).

**Draft editing:** Each child event stays editable while the Step is open: correct a weight, fix station context, delete a mistaken row (policy TBD). Persistence is ordinary create/update/delete of event records — not `CompleteSubstep`.

**Shifts:** Operators may tag shift (or infer from time) on the event or instance. Crossing midnight does not auto-close anything; the instance stays the manually opened production day.

**What Substeps are not used for:** They are not the unit of “done + notarize.” Completing a kind Substep mid-day would be the wrong metaphor: the Substep is “still collecting” until the Step seals.

### End of day — seal the Step

1. When the day’s Feragame work is finished, an authorized operator runs **Seal Step** (approve / close Step).
2. Preconditions (illustrative): Step is open; optional checks (e.g. at least one event, or role confirmation) — exact gates are open questions.
3. On seal:
   - All child events under that Step become **immutable**.
   - Evidence for the Step is treated as committed (notarization / digest strategy at Step or event-set level — see open questions).
   - The Step is closed for further Feragame work on this instance.
4. Kind Substeps may flip to a terminal “sealed with N events” presentation; they are **not** individually completed via today’s `/substep/:id/complete` path as the seal mechanism.

### After seal

- No further create/edit/delete of events in that Step.
- Downstream consumers (exports, DPP/UNTP mapping later, another org’s Step if the Stream grows) see a frozen Feragame evidence set.
- For the pilot’s single-org Stream, sealing the only Step may also drive instance “done” (or an explicit instance close may remain — see open questions). The important operational moment is **Step seal**, not “last Substep completed.”

### Multi-org later (same model)

If a second Organization’s Step follows Feragame’s:

1. Sequence applies **between Steps**: Org B’s Step stays unavailable until Feragame’s Step is sealed (and any product rules for handoff).
2. Inside Org B’s Step, the same pattern can apply: their own kind Substeps + child events + their own seal — without forcing their internal work through flat Substep unlock.
3. Feragame’s sealed event log is the handoff package: kinds preserved, counts and timestamps retained, no need for Org B to re-enter Feragame’s forms.

---

## 3. Attesta impact (what changes / what stays)

Grounding: today Substeps are ordered flat across the Stream (`orderedSubsteps`), exactly one incomplete Substep is available (`computeAvailability`), completion writes `ProcessStep{State: "done", ...}` and inserts a `Notarization` keyed by `substepId` (`CompleteSubstep`), and Steps mainly group Substeps plus `OrganizationSlug` (`WorkflowStep`). Step summaries treat a Step as “completed” when all its Substeps are done (`buildStepSummary`) — there is no Step approve/close API.

### What stays

| Area | Why it can stay |
|------|-----------------|
| **Stream / instance** | One instance ≈ one production day; create/list/status still apply. |
| **Workflow blueprint layers** | Step = org-scoped phase; Substep = in-org work container. No new layer between Step and Substep. |
| **Org ownership on Step** | `WorkflowStep.OrganizationSlug` still gates who may act for that Step. |
| **Role gates (Cerbos / authorizer)** | Still decide who may open, append events, edit drafts, seal — possibly new action names. |
| **Kind → Substep mapping** | Blueprint YAML still declares Substeps with schemas; schemas describe **one event** of that kind, not “the whole day’s array.” |
| **Progress key encoding** | Mongo `.` → `_` boundary for progress keys remains for any Substep-keyed state that stays. |

### What changes

#### A. Child event persistence (new)

- New store surface: create / update / delete / list child events by `processId` + parent `substepId` (and filters: station, time, kind).
- Event document (sketch): id, process id, step id, substep id (kind), kind discriminator, payload, actor, created/updated timestamps, optional station/shift fields, draft flag or inherited from Step seal state.
- HTTP/UI: routes for event CRUD under the instance (separate from `.../substep/:id/complete`).
- UI: kind Substep view = list + add/edit, not a single JSON Form submit that finishes the Substep.

**Code boundaries (likely):** new types + store methods beside `Process` / `ProcessStep`; handlers near process routes in `main.go`; templates/views that today assume one `SubstepBodyView` form body (`substep_views_builder.go`, process templates).

#### B. Sequence policy: Steps only inside this model

- `computeAvailability` / `isSequenceOK` today: predecessors in the flat Substep list must be `done`. That blocks interleaved kinds.
- This cell: **inside a Step**, all kind Substeps that belong to an **open** Step are available together (subject to org/role). **Across Steps**, unlock still requires the previous Step to be sealed (not merely “some Substeps done”).
- Authorization attributes (`sequenceOk` in `authorizer.go`) need a Step-aware meaning for event actions vs Substep complete.

#### C. Step open and Step seal (new lifecycle)

- Persist Step-level state on the process (e.g. per-step: `pending | open | sealed`, openedBy/At, sealedBy/At). Today process status is largely instance-level (`available` / `done` / terminated) with progress only at Substep granularity.
- New commands: `OpenStep`, `SealStep` (names illustrative), with authorization distinct from `CanComplete` on a Substep.
- Seal is **whole-step**: all child events under all Substeps of that Step freeze together.
- Notarization: move the irreversible commit moment from per-Substep `InsertNotarization` on complete to **seal time** (one Step notarization and/or notarization of the sealed event set). Mid-day event saves do not notarize as “done Substep.”

#### D. Substep completion path — narrowed role

- Kind Substeps in the Feragame Step **do not** use `CompleteSubstep` as the day-close mechanism.
- Options for coexistence with the rest of the product:
  - **Preferred for this cell:** Substep `complete` remains for traditional one-shot Substeps in other Streams; Feragame kind Substeps are marked (blueprint flag or input type) as `event-log` and never become `progress[substepId].state = done` via the old path — Step seal accounts for them.
  - Instance done: either “all Steps sealed” or “sole Step sealed” replaces “all Substeps done” (`isProcessDone`) for workflows that use Step seal.

#### E. What Step-level open/approve does to Substeps

| Concern | Behavior in this cell |
|---------|------------------------|
| Open day | Step action; **replaces** a first “Open” Substep. |
| Close / seal day | Step action; **replaces** a last “Seal” Substep. |
| Kind Substeps | Remain in the blueprint as **homes** for each event kind (structure, schema, roles, UI entry points). |
| Completing a kind Substep | Not the operator’s primary action; optional internal bookkeeping on seal (mark substeps reflected/sealed) without mid-day notarized completion. |

So Substeps are not deleted from the model; some of their **lifecycle duties** (open/close/sequence/notarize-on-complete) move up to the Step or out to child events.

### What this deliberately does not do

- Does not invent a new layer between Step and Substep.
- Does not put sequence back inside the Feragame Step for “Batch loads before UdC.”
- Does not seal per kind (`seal_granularity: whole-step` only).
- Does not require multi-org for the pilot; it only keeps the Step boundary ready for it.

---

## 4. Tradeoffs vs C1–C6

| Criterion | Verdict | Notes |
|-----------|---------|--------|
| **C1** Many events per kind per day instance | **Met** | Child-event log; unbounded rows per kind Substep. |
| **C2** Interleave across stations/kinds | **Met** | `steps-only` sequence; both kind Substeps concurrently available when Step is open. |
| **C3** Editable until seal; then immutable | **Met** | Draft events until whole-Step seal; seal freezes evidence. |
| **C4** Low-change path exists in the option space | **Not this cell’s job** | This exploration is explicitly `architectural`. A minimal exploration should cover cheap stretch of Substeps/forms. Cost here is real: new persistence, Step lifecycle, availability rewrite, notarization timing. |
| **C5** Sealed evidence distinguishes kinds | **Met** | `per-kind` Substeps + kind on each child record; seal does not flatten kinds into one opaque blob (payload packaging can still nest by kind). |
| **C6** Open and close explicit and manual | **Met** | Step-level open and seal; no midnight/shift auto-close. |

**Soft requirement (Step seal capability):** First-class. Seal is not simulated by completing a last Substep.

**Strengths**

- Domain language matches the line: “log an event,” “seal the day.”
- Clear product generalization: multi-org Streams get a real handoff object (sealed Step), not “the previous org finished all Substeps.”
- Avoids overloading `CompleteSubstep` with draft semantics it does not have today.

**Costs / risks**

- Largest implementation surface among the planned cuts: store, routes, UI, authorizer actions, process-done rules, notarization/DPP/UNTP assumptions that today key off completed Substeps (`untp.go` maps completed substeps).
- Two Substep modes in one product (classic complete vs event-log) need a sharp blueprint flag so other Streams do not regress.
- Whole-Step seal is coarser than per-kind seal: cannot finalize UdC while leaving batch loads open (acceptable for Feragame’s day close; call out if a future kind needs earlier freeze).

**Compared to other matrix corners (qualitative)**

| Alternative | Why this cell differs |
|-------------|------------------------|
| `array-fields` | Keeps one Substep payload; fights draft/seal and size/UX; still tempted to “complete” once. |
| `reenter-substep` | Many completions of one Substep pattern; clashes with done+notarize and sequence unless heavily bent. |
| `first-last-substep` open/close | Cheaper ceremony, but reuses Substep completion for day boundary and keeps seal coupled to the wrong unit. |
| `drop-in-step` only (without Step seal) | Unlocks interleaving but leaves “commit the day” undefined. |
| `line-log` single Substep | One log for all kinds; weaker kind structure in blueprint/roles; this cell prefers kind Substeps as homes. |

---

## 5. Open questions

1. **Notarization grain at seal** — One digest over the whole Step’s event set, one notarization per event at seal time, or both? How should exports and UNTP ModifyEvents represent many child events vs today’s one event per completed Substep?

2. **Blueprint marker for event-log Substeps** — New `inputType`, flag on `WorkflowSub`, or Step-level “mode”? Needed so classic Streams keep `CompleteSubstep` behavior.

3. **Instance done vs Step seal** — For a one-Step pilot Stream, does seal alone mark the process `done` and trigger DPP finalization (`finalizeProcessIfDone`), or is a separate instance close required?

4. **Open Step vs create instance** — Is creating the instance enough to open the Step, or must open stay an explicit second action (C6 prefers explicit; UX may want one click)?

5. **Delete and audit of drafts** — Soft-delete with tombstones for sealed history, or hard delete only while open? Any requirement to show corrected values after seal?

6. **Station modeling** — Station as a field on the UdC (and batch) event only, or also filter facets in UI? (Stations are not Substeps in this cell.)

7. **Authorization actions** — New Cerbos actions (`open_step`, `seal_step`, `create_event`, `update_event`) vs overloading `complete`? Prefer explicit actions for audit clarity.

8. **Concurrent editors** — Multiple operators appending events on the same open Step: last-write-wins per event, or stricter locking?

9. **Empty day seal** — Allowed to seal with zero events (e.g. line down), or blocked?

10. **Migration of `isProcessDone` / availability** — Feature-flag per workflow vs global Step-seal-aware engine? Affects how safely this ships beside existing Streams.

11. **Shifts as data** — Field on each event, a day-level shift list on the Step, or both? Brief forbids shift = separate instance unless a future exploration argues otherwise; this cell does not reopen that.

12. **Partial seal / reopen** — Out of scope for whole-step irreversible seal; confirm product never needs reopen after seal for the pilot (corrections would be a new instance or a future “amendment” concept).
