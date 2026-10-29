# Handoff: challenge one plant scenario vs living glossary + experiment YAML

Use this in a **new chat**. Goal: pressure-test **one concrete plant/operations scenario** against **(A)** living Attesta glossary vocabulary and **(B)** `attesta-experiment` truck-loop YAML as it exists on disk today.

The human (or the session) **picks the scenario**. The agent should **not invent the primary scenario** unless asked to sketch example prompts. Keep the challenge observational: map beats → glossary nouns → YAML (or gaps); record keep / rename / structural YAML change / glossary fix. Do not redesign Cerbos, implement code, or rewrite the decisions dump in that session.

---

## 1. Purpose

| Actor | Job |
| --- | --- |
| Human | Bring one end-to-end plant story (who acts, what facts appear, what unlocks next). |
| Agent | Narrate that story only in living glossary terms; map each beat to a concrete YAML step/substep; surface vocab↔YAML friction; log disposition (keep / rename / YAML change / glossary fix). |

**Success:** a short challenge log that either confirms the scenario fits Org Config + Domain + Jobs/Tasks, or names exact mismatches (naming keys, missing catalog, run-id framing, etc.).

---

## 2. Locked domain (short)

Source of truth for **decisions** (nouns may still be provisional in older prose):  
`/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/docs/research/attesta-app-living-product-decisions.md`

| Locked | Meaning |
| --- | --- |
| **Org Config (versioned)** | One authored document per org: **role catalog** + job/task map for the whole operation. Not “a blueprint you instantiate many times as runs.” |
| **One living Domain per org** | Accumulating entities + submission history, keyed by org — not one domain per “run.” |
| **What repeats** | Operators repeat **jobs/tasks** (Job instances). They do **not** “run the whole config” as a config-run instance. |
| **Roles** | Org-level **Config role** catalog **and** per-task “which roles may complete this”; production allow/deny is **Cerbos** (later / on Commit). |
| **Config run** | **Rejected** as primary runtime identity for living `/app`. |

Mechanics from the experiment (Step / Substep / unlock / `startsWhen` / `readyWhen` / ELT) are kept; product labels move to Job / Task / Job instance.

---

## 3. Living glossary cheat-sheet

Source: `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/GLOSSARY.md`  
Map vs legacy Stream: `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/GLOSSARY-MAP.md`  
Legacy Stream only: `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/CONTEXT.md` (do **not** use for `/app` challenge prose).

| Living term | One-line def | Experiment legacy name |
| --- | --- | --- |
| **Org Config** | Versioned authored document for one org: role catalog + job/task map. | Config (Attesta YAML) |
| **Config version** | One immutable revision of an Org Config; org has an active version for new work. | *(absent in experiment fixture)* |
| **Domain** | Org’s accumulating facts: entities with stable ids + append-only submission history. One per org. | Domain store (often scoped under a **Config run**) |
| **Job** | Authored unit that groups ordered **Tasks**; may be repeatable (many Job instances). | **Step** |
| **Task** | One submit unit inside a Job: schema, gates, successful Submission; names Config roles. | **Substep** |
| **Job instance** | One occurrence of a repeatable Job within the org Domain (e.g. one truck load / one shift). | **Step instance** |
| **Config role** | Named role slug in the Org Config catalog; Tasks list which may complete them. | Substep `roles: [...]` (no catalog in YAML) |
| **startsWhen** | Job-level start gate: quantifier (`some` \| `all`) over Submissions of a named Task. | Same key on Step |
| **readyWhen** | Task-level domain readiness (boolean over Domain) ∧ role gate; false ⇒ visible but unavailable. | Same key on Substep |
| **Submission** | Record that a Task succeeded on a Job instance (time, actor, facts). Unlock rules read these. | Same |
| **Draft** | Working data of an in-progress Task before Submission; not Domain until Commit. | Same |
| **Commit** | Authoritative accept of a sealed Task into Domain (Cerbos in production). | CommitPort (experiment) |
| **Load** | Hydrate org Domain + applicable Config version for the operator UI. | LoadRunPort / hydrate by **run id** (experiment) |

**Mapping to remember in the challenge:** Job ← Step · Task ← Substep · Job instance ← Step instance.

---

## 4. Experiment YAML reality (today)

### Primary fixture (use this)

| | |
| --- | --- |
| **Absolute path** | `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/src/lib/fixtures/truck-loop-steps.yaml` |
| **Loader** | `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/src/lib/load-truck-loop.ts` — `loadTruckLoop(text)`; default raw import is the **flat** file below; plant tests pass `truck-loop-steps.yaml?raw`. |
| **Shape ADR** | `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/docs/adr/0023-attesta-config-steps-substeps.md` |

### Secondary / superseded flat fixture

`/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/src/lib/fixtures/truck-loop.yaml` — top-level `forms:` (`register` / `weigh` / `process`). ADR-0023 rejects top-level `forms`. Useful only as a smaller smoke path; **challenge against `truck-loop-steps.yaml`**.

### Top-level shape (`truck-loop-steps.yaml`)

```yaml
id: truck-loop
org: feragame
domain:
  entities:
    load: { fields: [...] }
    pallet: { fields: [...] }
    container: { fields: [...] }
steps:
  intake: { repeatable, startLabel, instanceId, label, substeps: {...} }
  line:   { repeatable, startLabel, startsWhen, instanceId, label, substeps: {...} }
```

- **No** top-level `roles` / role catalog.
- **No** config version field.
- `loadTruckLoop` returns `{ id, org?, forms?, steps? }` — it validates/parses **steps** (and optional forms) but **does not surface `domain` on `TruckLoopConfig`**; domain declaration still lives in the YAML and is consumed elsewhere (e.g. tests reading authored YAML).

### Steps / substeps (order = YAML key order)

| Step (`steps.*`) | Job framing in YAML | Substeps (Tasks) in order | Per-substep `roles` |
| --- | --- | --- | --- |
| `intake` | `repeatable: true`, `startLabel: Nuovo camion`, `instanceId: $load.id`, `label: $load.id` | `register` → `codiceFir` → `palletWeight` → `weightCheck` | `administration`, `administration`, `logistic`, `administration` |
| `line` | `repeatable: true`, `startLabel: Nuovo turno`, `instanceId: $.shift`, `label: "'Turno: ' & $.shift"` | `shift` → `lineLoad` → `containerReg` → `loadedExtracted` | `line`, `line`, `logistic`, `administration` |

### Role catalog vs roles on substeps

- **Catalog in YAML today:** none. Role slugs appear only as Task-level lists.
- Distinct slugs used: `administration`, `logistic`, `line`.

### `startsWhen` / `instanceId` / repeatable (excerpts)

```340:348:/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/src/lib/fixtures/truck-loop-steps.yaml
  line:
    repeatable: true
    startLabel: Nuovo turno
    startsWhen:
      substep: intake/palletWeight
      quantifier: some
    # Shift-once: first Substep submit mints the instance.
    instanceId: $.shift
    label: "'Turno: ' & $.shift"
```

```32:38:/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/src/lib/fixtures/truck-loop-steps.yaml
  intake:
    repeatable: true
    startLabel: Nuovo camion
    # After first Substep Submission + transform bindings (e.g. $load):
    instanceId: $load.id
    label: $load.id
    substeps:
```

### `readyWhen`

- **Not present** on any substep in `truck-loop-steps.yaml`.
- Flat fixture only: `forms.process.readyWhen` in `truck-loop.yaml` (`$count($domain.pallet.*[$exists(weight)]) >= 1`). Plant path uses Submission unlock + leftovers instead for the analogous `lineLoad` work.

### Other gates / batching callouts

- `palletWeight`: `dynamicProperties.weights.completeness: all` (eligible as `startsWhen` target / in-step unlock predecessor).
- `lineLoad`: `completeness: some` (batch progress; **cannot** unlock next or be a `startsWhen` target — loader enforces).
- `containerReg`: `unlockFrom: instance` (unlocked with the Job instance, not by `lineLoad` Submission).

### Domain: declared vs written (high level)

| Declared under `domain.entities` | Written by Task `onSubmit.transform` |
| --- | --- |
| `load` | `register` → `mint` load (+ `mintMany` pallets); `codiceFir` / `weightCheck` → `patch` load |
| `pallet` | `register` → `mintMany`; `palletWeight` → `patchEach` (weight, tareKind, idoneita, panelCount); `lineLoad` → `patchEach` progress |
| `container` | `containerReg` → `mintMany` |
| — | `shift`, `loadedExtracted` → empty `transform: []` (Submission / instance context only) |

Side effects: `register` also declares `onSubmit.http` (fake labels / CSV). Failures must not roll back Domain (living glossary **HttpPort**).

### Experiment runtime still “run”-shaped (context for friction)

Code/seams still speak **config run** / `LoadRunPort` / `/api/runs/:runId/commit` (see experiment `GLOSSARY.md`, ADR-0025). That is mechanics/history for the challenge — living product **rejects** Config run as the identity of Domain.

---

## 5. Vocab ↔ YAML friction points (poke these)

| Friction | Living expectation | YAML / experiment today |
| --- | --- | --- |
| **Config run vs org Domain** | One Domain per org; Load by org + Config version. | Fixture + ports still oriented around a **run** (day/session mental model; `LoadRunPort`, `/api/runs/...`). |
| **Step / Substep key names** | Product nouns: Job / Task. | Authored keys remain `steps` / `substeps` (and ADR-0023 language). |
| **Missing org-level role catalog** | Org Config declares Config roles; Tasks reference them. | Only per-substep `roles: [...]`; no catalog block; loader does not require a catalog. |
| **Versioning absent** | Config version + active version for new work. | Single unversioned document (`id: truck-loop`). |
| **“Day / turno” framing** | Repeatable Jobs inside one org Domain. | `line.startLabel: Nuovo turno`, shift enum includes `whole-day` — easy to narrate as “a run of the plant day” instead of Job instances in one Domain. |
| **`readyWhen` underused in plant fixture** | Task readiness over Domain is a first-class gate. | Plant YAML has none; flat `process` form is the only authored example. |
| **Domain not on loader type** | Org Config includes domain entity declarations. | YAML has `domain:`; `TruckLoopConfig` omits it — easy to treat “config” as steps-only. |
| **Config `id` vs org** | Org owns one Org Config; Domain is org-scoped. | Both `id: truck-loop` and `org: feragame` — clarify which is identity in living terms. |
| **Forms path still shippable** | Nested jobs/tasks only. | Default `loadTruckLoop()` still loads flat `truck-loop.yaml` `forms`. |

---

## 6. How to run the challenge (checklist)

1. **Pick one end-to-end scenario** (human): actors + Config roles, domain facts that appear (load / pallet / container / FIR / checks), and what unlocks next (in-Job Task order, `startsWhen`, leftovers, `unlockFrom`).
2. **Narrate ONLY in living glossary terms** — Org Config, Domain, Job, Task, Job instance, Config role, startsWhen, readyWhen, Submission, Draft, Commit, Load. Ban: Config run, Step/Substep (except when citing YAML keys), Stream, “run the config.”
3. **Map each beat** to `steps.<id>.substeps.<id>` (or note a **gap**: missing Task, missing catalog entry, missing version, missing readyWhen, run-id Load, etc.).
4. **For each beat, record disposition:**
   - **keep** — YAML + glossary already aligned enough
   - **rename** — same mechanism; product/key rename only
   - **structural YAML change** — catalog, versioning, key reshape, domain-on-config, etc.
   - **glossary fix** — living def wrong/incomplete for this plant beat
5. **Optional stress:** retell the same scenario once with forbidden “config run / plant day run” language, then rewrite — does the living model still cover every beat?

---

## 7. Starter prompts (copy-paste)

**Read these files first** (absolute paths):

1. `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/GLOSSARY.md`
2. `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/GLOSSARY-MAP.md`
3. `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/docs/research/attesta-app-living-product-decisions.md`
4. `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/src/lib/fixtures/truck-loop-steps.yaml`
5. `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/src/lib/load-truck-loop.ts`
6. `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/docs/research/attesta-glossary-scenario-challenge-handoff.md` (this file)

Optional mechanics:  
`/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/docs/adr/0017-step-substep-submission-unlock.md`,  
`.../0018-domain-store-and-elt-transforms.md`,  
`.../0019-substep-availability-ready-when.md`,  
`.../0023-attesta-config-steps-substeps.md`,  
`.../0025-client-server-ports.md`,  
`/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/GLOSSARY.md` (experiment / Config-run era).

---

**Prompt A — full truck intake**

> Using only living Attesta glossary terms from attesta `GLOSSARY.md`, narrate: administration registers a NIR truck for client NO, enters FIR, logistic weighs all pallets, administration does weight check. Map every beat to `truck-loop-steps.yaml` step/substep ids. Flag any place the story needs “config run” or misses a role catalog / config version. Output a disposition table (keep / rename / structural YAML / glossary fix). Do not invent a different scenario; do not implement code.

**Prompt B — line after weigh**

> Same glossary-only narration: after some intake has completed `palletWeight`, line starts a 2nd-shift Job instance, marks pallet progress in batches, logistic registers dual-source containers, administration does loaded-vs-extracted. Map to YAML including `startsWhen`, `completeness: some`, and `unlockFrom: instance`. Call out day/turno vs org-Domain friction. Disposition table. No Cerbos redesign, no code.

**Prompt C — friction audit only**

> Do not invent plant fiction. Walk the friction list in `attesta-glossary-scenario-challenge-handoff.md` §5 against the on-disk `truck-loop-steps.yaml` and `load-truck-loop.ts`. For each row: evidence (path + short excerpt), living-glossary ideal, suggested disposition. No implementation.

**Prompt D — human supplies scenario**

> I will describe one Feragame/plant scenario in plain English. Rewrite it strictly in living glossary nouns, then map each beat onto `truck-loop-steps.yaml` (or mark gap). End with disposition per beat. Read the files listed in the handoff §7 first. Do not redesign Cerbos or rewrite the decisions doc.

**Prompt E — Load/Commit seams**

> For one Job instance created by `intake/register`, explain Load and Commit in living terms vs experiment `LoadRunPort` / `/api/runs/:runId`. What must change in product language (not code this session) so Domain is org-scoped? Disposition only.

---

## 8. Out of scope (this challenge session)

- Redesigning Cerbos policies or CEL attr schemas  
- Implementing Go/Svelte/YAML changes  
- Rewriting `attesta-app-living-product-decisions.md`  
- Merging living glossary with `CONTEXT.md` Stream language  
- Challenging multiple unrelated scenarios in one pass (pick **one**)

---

## Quick path index

| Role | Path |
| --- | --- |
| Living glossary | `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/GLOSSARY.md` |
| Locked decisions | `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/docs/research/attesta-app-living-product-decisions.md` |
| Plant YAML | `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/src/lib/fixtures/truck-loop-steps.yaml` |
| Flat YAML (legacy forms) | `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/src/lib/fixtures/truck-loop.yaml` |
| Loader | `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/src/lib/load-truck-loop.ts` |
| Experiment glossary | `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment/GLOSSARY.md` |
