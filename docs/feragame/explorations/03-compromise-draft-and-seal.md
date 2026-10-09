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

### Strategy split (classic untouched)

Ship **two strategies**, selected once from a Stream (blueprint) opt-in — default remains today’s behavior:

| Strategy | Who uses it | Behavior |
|----------|-------------|----------|
| **Classic** | Existing Streams (default) | Flat sequential unlock; Complete Substep → `done` + notarize; instance done when all Substeps are done. **Current functions stay as the classic implementation** — do not sprinkle mode `if`s through them. |
| **Draft-and-seal** | Feragame (and A/B) Streams that opt in | Parallel in-Step availability; draft progress; no mid-day Complete on kind Substeps; Seal Step freezes + notarizes + completes the pilot instance. |

Select the strategy at the edge (handler / view model / process service entry). Shared pieces underneath: progress map + store, Formata payloads, role/org gates, notarization insert/export shape, Stream instance create, display flattening, progress-key encoding.

### Draft-and-seal must add (classic path unchanged)

| Area | Change |
|------|--------|
| **In-Step availability** | Draft-and-seal strategy: all incomplete Substeps in an unsealed Step available together; predecessor lock only across Steps. Classic availability/sequence helpers remain the sequential implementation. |
| **Draft progress** | Draft-and-seal only: persist Substep payload while `state` is not `done`. Classic still arrives answers with completion. |
| **Step seal** | Draft-and-seal only: Step-scoped seal (auth, confirm, freeze, notarize leaves, reject further edits). Close half of `open_close: mixed`. |
| **Completion semantics for event Substeps** | Draft-and-seal only: no ordinary Complete on kind Substeps; seal owns terminal + notarization. Classic Complete untouched. |
| **Pilot workflow config** | Opt-in Stream: one-org, one-Step; per-kind Formata array Substeps; no open/close Substeps; strategy = draft-and-seal. |

### Shared (maximize reuse)

| Area | Why |
|------|-----|
| **Layers** | Step = Organization phase; Substep = in-org work. No new depth layer for the pilot. |
| **Progress map + Formata** | Events live as array fields on Substep progress data—same document shape, display flattening already walks arrays. |
| **Role / org gates** | Keep Substep role and Organization checks on edit, draft save, and seal. |
| **Instance = process** | One instance per production day; create remains the open. |
| **Notarized export shape** | Sealed Substep payloads still export under Steps → Substeps; merkle/notarization can stay Substep-leaf oriented if seal finalizes those leaves together. |
| **Inter-Step sequence** | Future multi-org Streams still order Organizations by Step. |
| **Classic completion + availability** | Left as the classic strategy implementation; not rewritten for Feragame. |

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

## 5. Grill outcomes (2026-10-09)

Stress-tested on branch `docs/feragame-grill-compromise`. **This cell is an A/B candidate**, not the locked pilot winner. Further explorations / spikes will be compared internally before choosing one pilot version.

### Settled for this candidate (if built)

| Topic | Decision |
|-------|----------|
| Cell axes | Keep as written: `array-fields` × `per-kind` × `steps-only` × `mixed` × `whole-step` |
| vs minimal | Real Step seal + drafts required; no last-Substep-as-seal theatre |
| vs architectural | Formata arrays remain the store of record; no child-event log in this spike |
| Open | Stream instance create only; no open-day Substep |
| In-Step sequence | All kind Substeps available together; sequence only between Steps |
| Seal | One irreversible whole-Step seal; empty kinds allowed; no reopen / per-kind seal |
| Kind Substeps mid-day | Draft saves only; no ordinary Complete (seal owns terminal + notarization) |
| Concurrency | Feragame pilot has a single registrar → defer concurrent-append design; last-write-wins acceptable for now |
| Seal auth | Elevated (supervisor / Org admin–class); row draft uses ordinary Substep roles |
| After seal | Pilot has one Step → seal completes the Stream instance |
| Validation | Row/field validation on draft save; seal freezes whatever is there (no min-count gate) |
| Notarization | At seal, notarize each kind Substep’s sealed payload (Substep-leaf export shape). Notary `CreatedAt` = seal time. Event times are optional row fields, **not** required for integrity |
| Row schema | No mandatory `registered_at` or `shift` for the spike |
| Draft persistence | Progress payload with non-`done` state; same Formata document shape |
| UX | Timeline shows parallel drafting kind Substeps; explicit Seal Step control (not a fake last Substep) |
| Strategies | **Classic** (default, untouched) vs **draft-and-seal** (opt-in). Select once at the edge; maximize shared store/Formata/auth/notarization/instance create. No mode `if`s inside classic completion/availability. |
| Product status | Remains `proposed` / candidate until A/B picks a winner; ADR only if Step seal becomes general Attesta law |

### Still open (implementation detail, not product forks)

1. Exact HTTP/HTMX mapping for draft save (append-row UX vs replace-whole-array under single-writer assumption).
2. Exact elevated role string / Cerbos rule for Seal Step.
3. Whether draft state is named `draft`, `in_progress`, or another non-`done` token in progress maps.
4. Failure/retry semantics if seal notarizes one Substep leaf and fails on the next (transactional seal vs compensating retry).
5. Station remains a field on rows (`per-kind`); `per-station` is a different matrix cell if needed later.
6. Where the strategy flag lives on the Stream blueprint (top-level vs Step-scoped) — prefer Stream-level opt-in for the pilot.

---

## Bottom line

Use **per-kind Formata array Substeps** as the repeating event log, **drop in-Step sequential unlock**, **open the day by creating the instance**, and **close by sealing the Step**—so Feragame gets interleaved draft capture without a new event store, and Attesta gains an honest Step seal instead of a last-Substep workaround. Treat this as one internal A/B candidate alongside at least the minimal exploration.
