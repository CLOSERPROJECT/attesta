# Feragame × Attesta

Design brief for integrating Attesta into Feragame’s plant workflow.

**Reader:** You already know how Attesta works (Streams, Stream instances, Steps, Substeps, Organizations, role gates, notarization). You do not need prior knowledge of Feragame. This document is self-contained for the pilot problem and the architecture option space. It does not decide an implementation.

**Status:** Constraints agreed; architecture open. Concrete slices of the option space live as explorations under `explorations/` (see [Explorations](#explorations)).

---

## 1. Purpose

Feragame wants to use Attesta to register what happens on their solar-panel material extraction line: recurring operational events during a production day, then a clear close/seal when that day’s work for their organization is finished.

Attesta was shaped for multi-party Streams (Organization-owned Steps in sequence, Substeps completed one-by-one with notarization). Feragame’s pilot is mostly **one Organization**, with **rich activity inside a single Step**. The tension is not “can one org own a Step?” (yes) but “can operators record many interleaved events inside that Step without fighting sequential Substep unlock and immediate notarization?”

This brief captures the domain, the agreed pilot constraints, and a **matrix** of architecture choices so later work can explore cells without rewriting the problem each time.

---

## 2. Feragame

Feragame extracts materials from end-of-life solar panels.

### Line and stations

- They run a continuous extraction **line**.
- Along the line, **stations** each extract a different material (or perform a different operation).
- At each station, material goes into containers. Containers are filled, stocked, and replaced with empty ones in cycles.
- Stations do **not** advance in lockstep. Fill rates differ. A realistic day looks like: station 1 stocks a container, then station 2, then station 1 again, then station 3, and so on.

### Unità di Carico (UdC)

Containers are called **Unità di Carico** (**UdC**) in Italian — a load unit / material container at a station.

Before a filled UdC is stocked, operators **register its weight** (and related identity/context). That registration is one of the event types Attesta must capture. It is **not** the only event type.

### Batch loads

**Batch loads** (panels or material entering / moving through the process) also happen **many times** during the same production day. Like UdC registrations, they are repeating operational events, not a once-per-Stream-instance form.

### Shifts and the day

Feragame works **day by day**. A production day typically includes **two to three shifts**. Multiple batch loads and many UdC cycles occur in that window. Night work may cross calendar midnight; the business “day” is not assumed to equal a clock date boundary.

---

## 3. What must be recorded

### Repeating in-Step events

Inside Feragame’s work for a day, operators record **multiple kinds** of **repeating events**. Known kinds for the pilot:

| Kind | Meaning (pilot) |
|------|------------------|
| **Batch load** | A load event that can occur many times per day |
| **UdC registration** | Weigh / register a container before stock; many times per day, interleaved across stations |

More kinds may appear later. Designs should not assume UdC is the only row type.

### Interleaving

Events at different stations (and different kinds) **interleave in time**. Any design that forces “finish all of station 1 before station 2 can record” fails the line.

### Day boundary and close

- One Attesta **Stream instance** is **intended** to cover **one Feragame production day** (all shifts and loads that belong to that working day).
- The instance is **opened and closed manually**. No automatic close at midnight or at shift change.
- Shifts are **data inside** the day, not separate Stream instances (unless a future exploration argues otherwise).
- Cross-midnight shifts belong to whichever production day was manually opened for that work.

---

## 4. Mapping onto Attesta

### Layering rule (product direction)

Keep Attesta’s layers meaningful in general, not only for this pilot:

| Layer | Role |
|-------|------|
| **Step** | Organization-scoped phase in the Stream; the multi-org sequence lives here (one org after another). |
| **Substep** | Work **inside** an Organization’s Step: forms, registrations, gates. |

If more “depth” is needed inside an org later, that is a separate discussion. It is not required to invent a new layer for the pilot.

### Pilot blueprint shape

For the Feragame pilot, keep cost down:

- **One Organization** (Feragame) participates in the executable Stream.
- **One Step** owns that day’s Feragame work.
- **Multiple Substeps** structure work inside that Step (exact mapping of stations / event kinds → Substeps is an architecture choice — see the matrix).

A single-org Stream is allowed today: nothing requires alternating Organizations across consecutive Steps. The pilot’s novelty is **inside** the Step, not ownership rules between orgs.

### Open and close

Starting and finishing the day/Step must be **explicit and manual**. Two implementation families are both acceptable to explore:

1. **First and last Substep** act as open / close (or open / seal).
2. A **Step-level** action (e.g. approve / close Step) distinct from ordinary Substep completion.

Designs must say which family they use (or how they mix them).

### Seal

Today, Attesta completes **Substeps**: on completion, progress becomes done and notarization runs; there is no separate “complete Step” action — Steps group Substeps and carry Organization ownership. When all Substeps are done, the Stream instance has simply run out of Substeps.

For Feragame (and as a general product intuition for multi-org handoff), it is useful to think of **sealing the Step’s evidence** before that work is treated as finished for handoff or closure:

- During the day, event records stay **editable** (draft until seal).
- **Seal** makes that Step’s evidence **immutable**.
- Seal may be implemented as “complete the last Substep”, as a new Step-level approve, or another mechanism — see matrix. Explorations must state whether they overload Substep completion or introduce Step completion as a concept.

---

## 5. Gaps versus current Attesta behavior

These are facts about current product behavior that collide with Feragame’s day. Explained here so this brief stands alone.

### Sequential Substep unlock

Substeps are ordered flat across the whole Stream (Step order, then Substep order). Exactly **one** incomplete Substep is available to edit/complete at a time. Later Substeps may be visible as preview but are locked (“Locked by sequence”) until predecessors are done. Completion of the current Substep unlocks the next in that flat list — whether it sits in the same Step or the next Step.

**Collision:** Stations and event kinds must accept input in interleaved order. Forcing notarized completion of Substep A before Substep B can accept data does not match the line.

### No draft answers before completion

Completing a Substep writes done state and notarization together. There is no first-class “save many answers, seal later” path for Substep payloads. Overrides/adaptation before completion are not a substitute for a durable event log.

**Collision:** Operators need to accumulate many batch-load and UdC events during the day and only seal when the day/Step is closed.

### Steps do not complete themselves as a seal

There is no Step-level approve/close that freezes evidence independent of Substep completions.

**Collision:** “Commit the day” is a real operational moment; designs must either invent Step seal or carefully simulate it with Substeps.

---

## 6. Agreed constraints and acceptance criteria

### Constraints

1. **Pilot instance = one production day**, manually opened and closed.
2. **One Step, many Substeps**, one Organization for the pilot blueprint.
3. **Steps remain the multi-org sequence** in the product model; in-org work stays in Substeps.
4. **Repeating events** of multiple kinds (at least batch load and UdC registration), many per instance.
5. **Interleaved** recording across stations/kinds must work (sequential unlock inside the Feragame Step is a blocker to remove or bypass).
6. **Editable until seal**; sealed evidence is immutable.
7. **Do not lock UX/architecture prematurely** in conversation — compare coherent options via explorations.
8. **Cheap path must remain possible** — at least one exploration should avoid a full workflow-engine rewrite.

### Acceptance criteria (any viable design)

A design is viable for this pilot only if it can satisfy all of:

| # | Criterion |
|---|-----------|
| C1 | Many events per kind per production-day Stream instance |
| C2 | Events can interleave across stations (and kinds) without false sequencing |
| C3 | Records remain editable until Step/day seal; after seal they are immutable |
| C4 | A low-change implementation path exists in the option space (not every exploration must be cheap, but the matrix must include one) |
| C5 | Sealed evidence distinguishes event kinds (batch load vs UdC vs future kinds) |
| C6 | Open and close of the day/Step are explicit and manual |

### Soft requirement: Step seal capability

All serious options should support **draft many events → one irreversible seal for the Step’s evidence**. The exact control (last Substep vs Step-level approve vs other) is open. Prefer evaluating the **capability**, not a single button label.

---

## 7. Architecture matrix

The open decisions form a **matrix**, not a single fork. Full cross-product is too large to implement blindly. **Explorations** pick one coherent cell (or a small, named mix) and analyze it.

### Axes

| Axis | Key (frontmatter) | Values | Meaning |
|------|-------------------|--------|---------|
| **Multiplicity** | `multiplicity` | `array-fields` · `reenter-substep` · `child-event-log` · `mixed` | How many events fit in the Substep/model: array fields with drafts; complete/re-enter the same Substep pattern; first-class child event records; or a documented mix |
| **Kind → Substep** | `kind_to_substep` | `per-kind` · `per-station` · `line-log` · `mixed` | One Substep per event kind; one per station (type on the row); one log Substep for all kinds; or mixed |
| **Sequence** | `sequence` | `drop-in-step` · `keep-partial` · `steps-only` | Drop sequential unlock inside the Step; keep it for some Substeps only; keep sequence only across Steps (multi-org), not inside |
| **Open / close** | `open_close` | `first-last-substep` · `step-level-action` · `instance-lifecycle` · `mixed` | First/last Substep; Step-level approve/close; instance create + explicit close; or mixed |
| **Seal granularity** | `seal_granularity` | `whole-step` · `per-kind` · `other` | One seal for the whole Step’s evidence; seal per event kind; other (must explain) |

Use `null` in an exploration when that exploration does not take a stance on an axis.

### Optional non-axis tag

| Tag | Key | Values | Meaning |
|-----|-----|--------|---------|
| **Cost posture** | `cost_posture` | `minimal` · `architectural` · `compromise` · `other` | Why this slice was chosen for study — not a matrix axis, but useful for the planned three cuts |

### Matrix at a glance

Rows are independent axes. A design picks **one value per axis** (or `mixed` / `other` with an explanation).

| Axis | Option A | Option B | Option C | Option D |
|------|----------|----------|----------|----------|
| **Multiplicity** | `array-fields` — repeating entries in form/schema; draft saves | `reenter-substep` — Substep defined for one item; mode allows many completions/entries over time | `child-event-log` — events as records outside “one form submit = done” | `mixed` |
| **Kind → Substep** | `per-kind` — e.g. “Batch loads”, “UdC registrations” | `per-station` — station-scoped Substeps; kind on the event | `line-log` — single log for all kinds | `mixed` |
| **Sequence** | `drop-in-step` — no in-Step sequential lock | `keep-partial` — sequence only for some Substeps (e.g. open/close) | `steps-only` — sequence only between Steps (org handoff) | — |
| **Open / close** | `first-last-substep` | `step-level-action` | `instance-lifecycle` | `mixed` |
| **Seal granularity** | `whole-step` | `per-kind` | `other` | — |

### Planned cuts (not exclusive)

The first wave of explorations is expected to sample cost postures:

1. **Minimal** — cheapest cell that still meets C1–C6 (likely stretch today’s Substeps/forms; simulate seal with last Substep; loosen or bypass in-Step sequence).
2. **Architectural** — clean domain cell (repeating events as first-class; real Step seal; sequence = multi-org / Steps only).
3. **Compromise** — hybrid after seeing where (1) and (2) hurt.

Further explorations may add any other cell. The matrix is generative; these three are not the only valid architectures.

---

## 8. Explorations

Explorations are **design probes**, not ADRs. An ADR is appropriate only after a hard-to-reverse choice is actually made.

### Layout

```text
docs/feragame/
  README.md                 ← this brief + matrix
  explorations/
    NN-slug.md              ← one exploration per file
```

Use a two-digit prefix for reading order (`01`, `02`, …) and a stable slug.

### Frontmatter schema

Every exploration starts with YAML frontmatter using the matrix keys:

```yaml
---
title: "Short title"
status: proposed # proposed | active | superseded | rejected
multiplicity: null # array-fields | reenter-substep | child-event-log | mixed | null
kind_to_substep: null # per-kind | per-station | line-log | mixed | null
sequence: null # drop-in-step | keep-partial | steps-only | null
open_close: null # first-last-substep | step-level-action | instance-lifecycle | mixed | null
seal_granularity: null # whole-step | per-kind | other | null
cost_posture: null # minimal | architectural | compromise | other | null
---
```

When a new matrix axis is added later, update the table in this README and add the new key to new explorations; older explorations may omit the key or set it to `null`.

### Suggested body shape

1. **Cell summary** — restate the frontmatter picks in prose  
2. **Behavior** — how a production day would run for operators  
3. **Attesta impact** — what must change vs what can stay  
4. **Tradeoffs** — vs acceptance criteria C1–C6  
5. **Open questions**

### Index

| File | Title | Cost posture | Status |
|------|-------|--------------|--------|
| [01-minimal-array-substeps.md](explorations/01-minimal-array-substeps.md) | Minimal: array Substeps, last-Substep seal | minimal | proposed |
| [02-architectural-event-log.md](explorations/02-architectural-event-log.md) | Architectural: child event log + Step seal | architectural | proposed |
| [03-compromise-draft-and-seal.md](explorations/03-compromise-draft-and-seal.md) | Compromise: draftable array Substeps + Step seal | compromise | proposed |

Add a row when you add `explorations/NN-slug.md`.

### How to add an exploration

1. Copy the frontmatter schema above.  
2. Set every matrix axis you commit to; leave the rest `null`.  
3. Write the body for that cell only.  
4. Link it in the index table.  
5. Do not treat the exploration as a product decision until an ADR (or equivalent) records the choice.
