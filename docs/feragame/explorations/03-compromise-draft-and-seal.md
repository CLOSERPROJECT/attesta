---
title: "Compromise: draftable array Substeps + Step seal"
status: proposed
multiplicity: array-fields
kind_to_substep: per-kind
sequence: steps-only
open_close: mixed
seal_granularity: whole-step
cost_posture: compromise
---

# Compromise: draftable array Substeps + Step seal

Design probe for the Feragame × Attesta pilot. Not an ADR. Assumes the reader knows Attesta (Streams, instances, Steps, Substeps, Organizations, role gates, notarization) and has read `docs/feragame/README.md`.

---

## 1. Cell summary

This exploration sits between a cheap “stretch Substeps” path and a clean first-class event log.

| Axis | Pick | What it means here |
|------|------|--------------------|
| **Multiplicity** | `array-fields` | Each event kind is an array on a Formata Substep. Operators append rows all day. Payload stays draft until Step seal. No separate event collection / child-event store. |
| **Kind → Substep** | `per-kind` | One Substep per known kind (at least “Batch loads” and “UdC registrations”). Kind identity is the Substep itself; rows are instances of that kind. |
| **Sequence** | `steps-only` | Inside Feragame’s single Step, Substeps are all available together. Flat “one incomplete Substep at a time” unlock is removed for in-Step work. Sequence remains between Steps for multi-org handoff later. |
| **Open / close** | `mixed` | **Open** = creating the Stream instance (manual day start). **Close** = a real Step-level seal action. Substeps never act as open or seal gates. |
| **Seal granularity** | `whole-step` | One irreversible seal freezes every draftable Substep payload in that Step. |
| **Cost posture** | `compromise` | Reuse forms/progress for the event log; pay for draft saves + Step seal + in-Step parallel availability. Avoid a new event store and avoid last-Substep-as-seal. |

### Why this hybrid

- **Cheaper than architectural:** Events remain Substep form payloads (arrays). Progress maps, Formata schemas, display flattening, and notarized export of Substep data stay the storage shape. No child-event records or parallel workflow engine.
- **Cleaner than minimal:** Seal is not “complete the last Substep.” Completing a Substep today means done + notarize immediately (`handleCompleteSubstep` / progress `state: done`). Overloading that for “close the day” conflates row capture with evidence freeze and fights the line. This cell introduces **Step seal** as the freeze moment and keeps Substeps as **kind buckets**, not lifecycle gates.
- **`open_close: mixed` (concrete):** Instance create opens; Step seal closes. Defended below.

### Open = instance create; close = Step seal

Two acceptable open/close families in the brief are first/last Substep and Step-level action. This cell mixes **instance lifecycle for open** with **Step-level action for close**:

1. **Opening with a Substep is a bad fit once sequence is `steps-only`.** An “Open day” Substep that must be completed before event Substeps unlock reintroduces in-Step sequencing the line cannot tolerate. An open Substep that stays forever incomplete is a fake gate. Instance create is already an explicit, manual act that matches “we started this production day.”
2. **Closing with a last Substep is the minimal trap.** It either forces operators to finish event Substeps first (back to sequential unlock / premature notarization) or leaves event Substeps “done” while still editable—an honesty problem for C3. A Step-level seal says what the plant means: commit this day’s evidence for the Organization’s Step.
3. **Substeps stay for C5 (kind distinction) and C1 (many rows), not for day boundaries.** That keeps Attesta’s layering rule: Step = org-scoped phase; Substep = work inside the org.

Shifts stay data on events or day metadata inside the instance—not separate instances—per the brief.

---

## 2. Behavior (operator day)

Pilot blueprint: one Organization (Feragame), one Step for the day’s line work, two (or more) Substeps under that Step—e.g. **Batch loads** and **UdC registrations**—each with an array schema for repeating rows.

### Morning: open the day

1. An authorized operator **creates a Stream instance** for today’s production day (label/metadata as needed: plant, nominal date, notes). That create is the open. No “Open day” Substep.
2. The Feragame Step is active. Both kind Substeps show as **available** (not locked by sequence relative to each other).
3. Arrays start empty. Nothing is notarized yet for these Substeps’ final evidence.

### During the day: interleave and draft

4. Station work proceeds out of lockstep. Operators open whichever kind Substep matches the event:
   - Batch load arrives → append a row on **Batch loads** (time, shift, load identity/context, …).
   - UdC ready to stock → append a row on **UdC registrations** (station, weight, container identity, shift, …).
5. Saves are **draft progress updates**: persist array contents without marking the Substep `done` and without running completion notarization. Operators can edit or remove mistaken rows until seal.
6. Switching kinds and stations freely is allowed because availability is not “exactly one incomplete Substep in the flat ordered list” (`computeAvailability` today). Inside this Step, both Substeps remain editable in parallel.
7. More kinds later = more Substeps of the same shape (array + draft), not a new multiplicity model.

### End of day: seal the Step

8. When the production day is finished for Feragame (all shifts that belong to this manually opened day), an authorized operator runs **Seal Step** (name in product UI can vary: approve / close / seal). Confirmation required; action is irreversible for that Step’s evidence.
9. Seal:
   - freezes every kind Substep payload in the Step (arrays become immutable);
   - marks those Substeps completed for progress/timeline purposes as part of the seal transaction (or equivalent “sealed” terminal state that reads as finished);
   - runs the integrity path once for the sealed Step evidence (notarization of sealed payloads—exact leaf strategy is an open question, but seal—not per-row Substep complete—is the commit);
   - closes the day for this Organization’s Step. Further append/edit is rejected.
10. No automatic close at midnight or shift change. Cross-midnight work stays on this instance until someone seals.

### After seal

11. Operators (and later consumers) see a finished Step: distinct kind Substeps, each holding its sealed array, immutable.
12. If a future multi-org Stream places another Organization’s Step after Feragame’s, **inter-Step** sequence still applies: that next Step unlocks only after Feragame’s Step is sealed (and any product rules for Step readiness are satisfied). That is the intended home of sequence under `steps-only`.

### What operators do *not* do

- Complete event Substeps one-by-one during the day to “unlock” the next kind.
- Treat the last kind Substep as “close day.”
- Open a second instance for the next shift of the same production day.

---

## 3. Attesta impact (what changes / what stays)

Grounded in current server behavior: Substeps are ordered flat; `computeAvailability` exposes only the first incomplete Substep; completion writes `done` and notarizes together; there is no Step-level approve/close.

### Must change

| Area | Change |
|------|--------|
| **In-Step availability** | For Substeps that share an open (unsealed) Step, mark all incomplete Substeps available—or scope the flat predecessor rule so it does not lock siblings inside the same Step. Preserve predecessor locking **across** Steps (`steps-only`). Touches availability helpers (`computeAvailability`, `isSequenceOK`), timeline/status (“Locked by sequence”), and complete-path sequence checks (today: completing out of order → conflict). |
| **Draft progress** | Allow persisting Substep payload while `state` is not `done` (draft / in-progress). Today, durable answers arrive with completion. Need save (and load) of draft Formata values, authorization, and UI that does not imply “Completed.” |
| **Step seal** | New explicit action at Step scope: authorize, confirm, freeze all Substep payloads in the Step, transition progress to sealed/done, notarize sealed evidence, reject further edits. This is the close half of `open_close: mixed`. |
| **Completion semantics for event Substeps** | Ordinary mid-day “Complete Substep” must not be the path that freezes the day’s event log. Either hide/disable complete on draftable kind Substeps until seal owns the terminal transition, or redefine complete so it is not used for these Substeps. Seal is the commit. |
| **Pilot workflow config** | One-org, one-Step blueprint; per-kind Formata Substeps with array item schemas; no open/close Substeps. |

### Should stay

| Area | Why |
|------|-----|
| **Layers** | Step = Organization phase; Substep = in-org work. No new depth layer for the pilot. |
| **Progress map + Formata** | Events live as array fields on Substep progress data—same document shape, display flattening already walks arrays. |
| **Role / org gates** | Keep Substep role and Organization checks on edit, draft save, and seal. |
| **Instance = process** | One instance per production day; create remains the open. |
| **Notarized export shape** | Sealed Substep payloads still export under Steps → Substeps; merkle/notarization can stay Substep-leaf oriented if seal finalizes those leaves together. |
| **Inter-Step sequence** | Future multi-org Streams still order Organizations by Step. |

### Deliberately deferred (vs architectural)

- First-class child event entities / event store.
- Per-event notarization or per-kind seal.
- Re-enter-Substep multiplicity (many completions of the same Substep as the event mechanism).
- Modeling stations as Substeps (`per-station`); station stays a field on UdC (and similar) rows.

### Implementation cost sketch (order of magnitude)

Relative to “minimal” (bypass sequence + last Substep seal + maybe weak drafts): this adds a real seal API/UI and honest draft state. Relative to “architectural” (child-event log + Step seal + steps-only): this skips the new store and keeps Formata arrays as the log. Main risks are draft concurrency on shared arrays and teaching the product that “Substep complete” ≠ “day closed.”

---

## 4. Tradeoffs vs C1–C6

| Criterion | Verdict | Notes |
|-----------|---------|-------|
| **C1** Many events per kind per day instance | **Met** | Array fields grow without one Substep completion per event. |
| **C2** Interleave across stations/kinds | **Met** | `steps-only` removes in-Step sequential unlock; kinds are parallel Substeps; station order lives in timestamps/row data, not Substep order. |
| **C3** Editable until seal; immutable after | **Met** | Draft saves until Step seal; seal freezes whole-Step evidence. Stronger than last-Substep theatre. |
| **C4** Low-change path exists in the option space | **Partial for this cell** | This exploration is not the cheapest cell; it still avoids a new event store. The matrix’s cheap path remains other explorations. Local cost: availability + draft + seal—not a workflow rewrite. |
| **C5** Sealed evidence distinguishes kinds | **Met** | Per-kind Substeps; sealed payloads are separate by Substep ID/title. |
| **C6** Open and close explicit and manual | **Met** | Open = instance create; close = Step seal. Both manual; no clock automation. |

### Soft requirement: draft many → one Step seal

Supported directly: accumulate on arrays, one whole-Step seal.

### Tension this cell accepts

- **Arrays as a log** are weaker than a first-class event stream (indexing, per-row ACL, concurrent append UX, audit of single-row edits). Acceptable for pilot volume if draft saves are solid and seal is rare (once per day).
- **Seal marks Substeps done** without mid-day Substep completion may surprise existing Attesta mental models (“progress advances by completing Substeps”). Product copy and timeline must show “drafting / sealed” clearly.
- **Open without an open Substep** means the timeline may show event Substeps available immediately after create—which is correct for Feragame, but different from Streams that use Substep 1 as a ceremonial start.

### Comparison intent (without reading sibling files)

- vs **minimal:** pays for Step seal and real drafts so close/editability are not faked; still stretches forms instead of inventing events.
- vs **architectural:** declines child-event-log multiplicity; accepts Formata arrays as the compromise store of record for the pilot.

---

## 5. Open questions

1. **Draft persistence API** — Reuse `UpdateProcessProgress` with a non-`done` state, or a dedicated draft endpoint? How do HTMX Formata saves map onto append-row vs replace-whole-array?
2. **Concurrent editors** — Two stations appending UdC rows at once: last-write-wins on the whole array is unsafe. Need row-level append, ETags/versions, or short-term “one writer per kind Substep” operational rule for the pilot.
3. **Notarization at seal** — One notarization per Substep leaf at seal time, one Step-level notarization, or both? Must remain compatible with existing notarized JSON / merkle export expectations.
4. **Empty kinds** — May the day seal with zero batch loads or zero UdC rows? Probably yes (line stopped early), but confirm with Feragame.
5. **Seal authorization** — Same roles as Substep edit, a narrower “supervisor” role, or Organization admin only?
6. **Post-seal Stream instance** — Is the process “done” when the single Feragame Step seals, or only when all Steps (future multi-org) are sealed? Pilot likely: seal ⇒ instance complete.
7. **Shift representation** — Field on every row, day-level metadata, or both? Brief says shifts are data inside the day; pick a schema convention before build.
8. **UI for parallel Substeps** — Timeline today centers “the” available Substep. Drafting two open kind Substeps needs navigation that does not look like a broken sequence.
9. **Migration of `complete` button** — For draftable kind Substeps, remove complete entirely vs keep it disabled with explanation until seal exists in UI.
10. **Validation at seal** — Required fields per row, minimum row counts, or “seal whatever is there”? Affects whether seal can fail after confirm.
11. **Station as data vs Substep** — This cell keeps `per-kind`. If Feragame later wants station-scoped queues, that is a different matrix cell (`per-station`), not a silent tweak here.
12. **Partial seal / reopen** — Out of scope for C3’s irreversible seal; confirm no pilot need to unseal or seal per kind.

---

## Bottom line

Use **per-kind Formata array Substeps** as the repeating event log, **drop in-Step sequential unlock**, **open the day by creating the instance**, and **close by sealing the Step**—so Feragame gets interleaved draft capture without a new event store, and Attesta gains an honest Step seal instead of a last-Substep workaround.
