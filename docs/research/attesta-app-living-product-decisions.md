# Living product into `/app` — decisions dump

Status: working notes (2026-10-29). Not an ADR. Captures a grill on transplanting living Attesta (`attesta-experiment`) into `forkbomb/attesta` `/app` + Go JSON, without losing deferred work.

**Related**

- Move plan (source): `attesta-experiment/docs/research/attesta-move-into-attesta-app.md`
- SPA handoff: temp `attesta-svelte-app-handoff.md` (branch `feat/attesta-rework`, PR #276)
- Server layout epic: [#249](https://github.com/CLOSERPROJECT/attesta/issues/249) `refactor(server): thin cmd/server and extract deep internal packages`
- Settled product law (experiment): ADR-0025 client/server ports; living language: root [`GLOSSARY.md`](../../GLOSSARY.md)
- Legacy product language: root `CONTEXT.md` (Stream / Stream instance); map: [`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md)

---

## Destination (intent)

Transplant the living single-org product (ports, pure Modules, UI) into `attesta`’s Go-hosted Svelte SPA at `/app`, with Go as Commit/Load authority. Legacy HTMX `/my` + `web/` stay; additive only. Do not replace old surfaces in this effort.

Living product nouns match the locked root [`GLOSSARY.md`](../../GLOSSARY.md) (flat-port language). **The decisions themselves are locked**; this dump’s wording follows that glossary.

---

## Decisions settled

### Domain model (locked 2026-10-29)

Supersedes experiment-centric whole-config execution (historical name **Config run**) as the runtime identity for this transplant. Multi-org linear **Stream** flow remains legacy (`CONTEXT.md` / `/my`); living `/app` is single-org.

- **Org Config (versioned):** the durable authored document for an Organization — **Role catalog** + the map of **Jobs/Tasks** (the org’s whole operating picture). Not “a blueprint you instantiate many times as whole-config executions.” Not multiple streams per org.
- **Domain (per org):** one accumulating Domain store (+ Submissions / work log) keyed by **org**. Tasks write entities/knowledge; later Tasks read and extend them. Connections are explicit (sequence, spawn-many) and/or Domain conditions (`readyWhen`).
- **What repeats:** operators repeat **Jobs** (and their Tasks) inside the Org Config many times (**Job instances**). They do **not** run the whole config as one whole-config execution.
- **Config versioning:** Org Config documents are versioned (**Config version**); the org has an **Active Config version** for new work and for Load unless a pin policy says otherwise. Pin-in-flight-Job-to-old-version policy is deferred; Domain outlives many Config versions.
- **Roles:** both a **Role catalog** on the Org Config and **per-Task** which **Config roles** may complete it; **Cerbos** is allow/deny for Commit (see Authorization).
- **Mechanics keep, labels reworked:** unlock / Leftovers / `readyWhen` / `startsWhen` stay as mechanisms. Experiment Step / Substep / Step instance and ports like `LoadRunPort` / `RunStore` are rejected/historical code names — living nouns are Job / Task / Job instance and org Domain Load/Commit (see [`GLOSSARY.md`](../../GLOSSARY.md)).
- **Retired for living product:** whole-config execution as primary runtime identity; bare `run` / `/api/runs` / `RunStore` / `LoadRunPort` as the public seam names. Seed/reset targets **Domain** (or fixtures), not minting a new run id.

**Job / Task nesting (locked 2026-10-29)**

Challenge context: Feragame client intake is truck unload with **N lots** (gross weigh → per lot unload/weigh/reweigh/labels → final tare vs sum). That needs spawn-many under a truck. Handoff for the challenge: [`docs/research/attesta-glossary-scenario-challenge-handoff.md`](attesta-glossary-scenario-challenge-handoff.md). Living terms: root [`GLOSSARY.md`](../../GLOSSARY.md).

- **Job** is **top-level only** in the Org Config map (e.g. `intake`, `line`) — the chapter operators start (“Nuovo camion”). Jobs are not nested inside Jobs.
- Under a Job: a **Task tree** (any depth later). Two kinds (do not overload one noun):
  - **Task** (leaf) — one Commit / Submission / schema / Config roles / Leftovers (`completeness`). Cerbos Commit targets leaf Tasks only.
  - **Task group** (deferred product name; working label in glossary) — children, may be `repeatable` with parent-scoped instances; **no** Commit of its own. This is spawn-many (e.g. lot under truck).
- **Rejected:** jobs all the way down; untyped “tasks at any depth” without Task group vs leaf Task; modeling spawn-many lots as Leftovers `completeness: some` (leftover batching ≠ unknown-N full registers; `some` cannot unlock next / be a `startsWhen` target — experiment ADR-0023).
- **Nesting / Task groups / lot-shaped intake:** product decision locked; **implementation deferred** (see Port cut + Deferred). First transplant keeps today’s **flat** Job → leaf Tasks (current `truck-loop-steps.yaml`).

**Seams**

- **Load** ≈ hydrate the org’s Domain plus the Active Config version (or a pinned Config version later).
- **Commit** ≈ accept a sealed leaf **Task** into that Domain (Cerbos on the way in). Address by **Task path + Job instance** id (e.g. `intake/palletWeight` + instance) so deeper paths (`intake/lot/weighPallets`) stay additive later — not a single-level child index.
- **Persist** Domain by org; Org Config by org + Config version.
- **Submission** keys use the same Task path form; Unlock order / `startsWhen` keep speaking `job/…/task`.

### Port cut — now vs later (locked 2026-10-29)

Goal: port working experiment logic into `/app` + Go **soon**, without a second identity rewrite when nesting lands.

**Do now (correct flat port)**

- Runtime identity: org Domain + Job instances — never whole-config execution / `/api/runs` / `RunStore` as product shape.
- Seams: Load / Commit / persist as above; public nouns Job / Task / Job instance / Submission.
- Org Config **content**: current flat `intake` + `line` as in experiment `truck-loop-steps.yaml` (register → codiceFir → palletWeight → weightCheck; line as today).
- Mechanics: Unlock order, `startsWhen`, `readyWhen`, Leftovers `all`/`some`, ELT, chrome — same behavior.
- Cheap future-proofing: path-shaped Commit + Submission ids; do not hard-require “Tasks are only direct children” in public contracts if a Task path (or recursive children) is easy. Flat authored map for v1 is fine.
- Glossary: **Task** stays the submit leaf; note **Task group** / spawn-many as deferred — do not implement composites or call them Tasks in code yet.

**Postpone (no premature build)**

- Task group / nesting runtime + authoring.
- Feragame **lot-based** intake reshape (gross → N lots → final tare); demo old intake until nesting exists.
- Config version pin-in-flight policy; full Role catalog UX; Cerbos on Load / `/app`; DraftPort; attachments; SSE; OpenAPI codegen as a gate.

**Do not “fix later”**

- Run-id Domain; Step/Substep as public product names; Commit that can only address a hard one-level child list with no Task path; faking lots with Leftovers `completeness: some`.

### Code seam placeholders (locked 2026-10-29)

Use these **temporary** engineering names in early slices so plans stay consistent. They are **not** product glossary nouns. A **final renaming slice + discussion** renames them once the flat port works.

| Placeholder | Use for |
| --- | --- |
| `server/internal/living` | Go package for living Domain / Org Config / Load / Commit HTTP |
| `DomainStore` | Persistence Interface for org Domain |
| `ConfigStore` | Persistence Interface for versioned Org Config documents |
| `GET /api/living/domain` | Load (org from session; Active Config version resolved server-side) |
| `POST /api/living/commit` | Commit (body: Task path + Job instance id + sealed value / mint) |
| `living` Cerbos resource kind (draft) | Commit allow/deny until real kind names land in rename |

Rejected as public shape even as placeholders: `/api/runs`, `RunStore`, `LoadRunPort`.

### Experiment → attesta port policy (locked 2026-10-29)

Source: `attesta-experiment`. Destination: `attesta` `/app` + `server/internal/living` (placeholders). Rule of thumb: **same behaviour, new home, living nouns → copy + adapt**; **authority / persistence / product HTTP → rewrite in Go**; **if a copy would drag whole-config-execution or run-id identity → rewrite the seam**, keep only pure functions.

**Copy (then adapt to living glossary)**

- Pure TS Modules + Vitest that do not need Kit: Domain store helpers used only where still needed, Leftovers/submit prep while FormPlan remains client-side, ELT runner references, FormPlan / live chrome (until **Task form bundle**), wire DTOs, Local Commit/Load for unit tests only → into `app/` (or thin lib under `app/`).
- **Superseded for Unlock / startsWhen chrome:** do **not** keep production TS copies of Unlock order or startsWhen for Job chrome. Go projects a **Worklist** on Load/Commit; see [`docs/adr/0005-living-worklist-server-projection.md`](../adr/0005-living-worklist-server-projection.md). Delete SPA gate plants once Worklist ships.
- Rename Step / Substep / Step instance → Job / Task / Job instance on the Commit path as files are ported (no big-bang rename for untouched files).
- Flat truck-loop fixture: copy content; prefer **one** `steps`/`substeps` → Job/Task key rename when ConfigStore first loads Org Config so Load does not speak two languages.
- Truck-loop / stack UI → `app/` + sv-router; HTTP Adapters → `/api/living/…`. Drop harness gear.

**Rewrite (do not copy as authority)**

- Kit `+server`, SQLite RunStore, run-id routes — behaviour reference only; production = Go living package + DomainStore / ConfigStore.
- Go Commit accept path — reimplement (resolve Org Config, re-gate, re-prep, ELT, Cerbos). TS Local Commit stays for Vitest only.
- Go **Worklist** projection (Unlock / startsWhen / Leftovers openness → Task statuses + `canStart`) — server-only; SPA consumes JSON.
- Harness AuthPort passthrough and snapshot-replace PUT — do not port; verified session + seed/wipe instead.

**Do not copy**

- Spikes, archived kernels, experiment harness product UI, postponed work (Task group, lot intake), scaffold noise.

### Process / branching

- Base feature branch: `feat/attesta-rework`.
- Work in **slice branches** off that base; **no PR required** for slices; **squash merge** into `feat/attesta-rework` after a successful human smoke.
- [#249](https://github.com/CLOSERPROJECT/attesta/issues/249) and the living transplant are **parallel**. Transplant does not wait on Phase 0/1. Opportunistic extract from `cmd/server` only when the transplant needs a shared Seam (e.g. session gates / thin actor map later) — not a big-bang `internal/` migration first.
- **Slice map / ordering** is **not** locked here; design slices later with a dedicated skill/session.
- A **wayfinder** map is a good fit for **remaining decisions + research**, not as the implementation backlog. Persist this dump either way so chat settlements are not the only copy.

### Go layout

- New living Go code lands in a **deep `server/internal/…` package** (name TBD — see Deferred). Demonstrate [#249](https://github.com/CLOSERPROJECT/attesta/issues/249)’s target shape by birth, not by waiting on legacy extraction.
- Living package is **not** `internal/stream`. `#249`’s `internal/stream` is for legacy Stream/instance types. Living vocabulary is **Org Config / Domain / Jobs / Tasks** (see Domain model above and [`GLOSSARY.md`](../../GLOSSARY.md)). Experiment ADR-0020’s “config ≠ Stream” still holds; whole-config execution as runtime identity does not.
- **Plain `net/http`** handlers colocated in that package; thin mux registration from `cmd/server`. **Not** a new Goa service for Commit/Load in the first cut (Goa remains available elsewhere; OpenAPI/codegen is a later research track).
- **Do not** break existing routes, persistence encoding (progress-key `.` → `_`), or workflow config without explicit approval. Additive handlers only.

### Auth / session

- `/app` stays verified-session gated (`requireVerifiedPage` pattern).
- Living JSON Commit/Load: **same verified session** required.
- **Wire first**: keep session helpers in `cmd/server`; inject verified user into living handlers. Extract `internal/identity` (or similar) only when AuthPort leaves harness-style mapping or duplication hurts.
- **No product harness** surface in `attesta`: no SPA harness gear, no actor-impersonation UI, no `PUT` snapshot-replace API for scenarios.
- Dev/demo data: **Go seed / fixtures** + Vitest Memory; optionally a **dev user with broad roles**. Scenario “reset” = re-seed / wipe **Domain** — not browser snapshot replace, not minting a whole-config execution id.

### Authorization (Cerbos)

- **Cerbos is the sole EFFECT_ALLOW / EFFECT_DENY** for living **Commit** (mutate). Amend experiment ADR-0025’s “unlock ∧ roles ∧ readyWhen first / Cerbos optional” for the **attesta Go cut**. Vitest/tests may stub an Authorizer Adapter.
- **Hybrid domain predicates** (CEL is not a drop-in for authored JSONata):
  - **`readyWhen` and other JSONata** → evaluate in Go → boolean (and similar) **resource attrs** → Cerbos must allow.
  - **Light structural checks** (Config roles, org, Unlock order over a **compact** Submissions summary, `sequenceOk`-like flags) → prefer **CEL in Cerbos** when policies stay readable; if attrs get huge, fall back to Go boolean attrs.
  - **Never** dump a full DomainSnapshot into Cerbos.
- Living Commit uses **new living resource/action shapes** (names TBD), not legacy `CanComplete(… Process / WorkflowSub …)` as-is.
- **v1 scope**: Cerbos on **Commit**. **Load (GET)** and **`/app` shell** stay **session-only** until we explicitly add Cerbos actions for them.

### Persistence

- **Share Mongo wiring** (same cluster/DB from `cmd/server` construction) with legacy.
- Living persistence: **separate Interfaces** (names TBD after package naming) for (1) **Domain** authority document and (2) **versioned Org Config** documents. Memory Adapters first are fine. Not a per–whole-config-execution row store (`RunStore` rejected as the product shape).
- **Do not** extend the god Process `Store` interface with living methods; **do not** reuse `UpdateProcessProgress` / Process progress documents as the living Domain store.
- **Attachments**: not in the first Commit/Load cut (sealed JSON values only). Later, a living attachment Seam may **reuse GridFS wiring** behind its own Interface — not Process attachment helpers as the product Seam.

### Frontend / law / docs

- Living UI/ports go under `app/` (Kit static shell + sv-router); Kit `+server` is not production authority ([ADR-0025](https://github.com/CLOSERPROJECT/attesta-experiment/blob/feat/client-server-ports/docs/adr/0025-client-server-ports.md)).
- **Two glossaries until merge**: living terms in root [`GLOSSARY.md`](../../GLOSSARY.md) (locked for the flat port); `CONTEXT.md` Stream language for legacy. **Merge later**. This dump’s wording follows the living glossary; decisions stay as locked above.
- Copy/fence living ADRs into `attesta` when docs land; amend ADR-0025 for Cerbos + Domain Load/Commit (not run-id / whole-config execution hydrate). Point AGENTS / architecture at `/app` + living law for product UI work.

### Explicit non-goals (this effort / first cuts)

- Replacing legacy HTMX complete handlers with the SPA API wrapper.
- Merging new UI into `web/` or `server/templates/`.
- `packages/` monorepo split without a new ADR.
- DraftPort, domain chunks, real IdP redesign, SSE in the first transplant.
- Completing [#249](https://github.com/CLOSERPROJECT/attesta/issues/249) as a prerequisite.
- OpenAPI → TypeScript codegen as a gate on first Commit/Load (hand-mirrored DTOs / golden JSON OK until research lands).

---

## Deferred (tracked so nothing is lost)

| Item | Notes |
| --- | --- |
| **Living glossary lock** | **Done for flat-port core nouns** (Org Config, Config version, Active Config version, Domain, Job, Task, Job instance, Task path, Unlock order, Leftovers, startsWhen, readyWhen, Submission, Draft, Role catalog, Config role, Commit, Load, OperatorIdentity) — see root [`GLOSSARY.md`](../../GLOSSARY.md). **Still TBD:** final English for **Task group** (working label only). |
| **Task group / nesting** | Implement Task groups under Job (repeatable, parent-scoped instances); Unlock order / `readyWhen` over child instances. Decision locked above; code after flat port works. |
| **Feragame lot intake** | Reshape `intake` for truck lots (gross → per-lot register → final tare vs sum). Depends on Task group. Do not fake with Leftovers `completeness: some`. |
| **Code seam rename (final slice)** | After flat port smokes: discussion + slice to replace placeholders (`internal/living`, `DomainStore` / `ConfigStore`, `/api/living/...`, draft Cerbos kind) with lasting names aligned to glossary. |
| **Config version pin policy** | Whether in-flight Jobs pin to the Config version they started on vs always follow Active Config version. |
| **Slice design** | Done: `.scratch/living-flat-port/issues/` (01–13 flat port + rename). |
| **Wayfinder map** | Chart remaining **decisions + research** (not build tasks); destination ≈ “way clear to implement.” |
| **Glossary merge** | Merge living [`GLOSSARY.md`](../../GLOSSARY.md) + `CONTEXT.md`; Stream terms historical where appropriate. |
| **ADR-0025 amend in attesta** | Cerbos-as-production-Authorizer + hybrid predicates + Domain Load/Commit (retire run-id / whole-config execution hydrate). |
| **Identity package extract** | When living AuthPort needs session → `OperatorIdentity` beyond wiring. |
| **Cerbos on Load / `/app`** | After Commit path is solid. |
| **Living attachments Seam** | GridFS reuse when a Task needs files. |
| **OpenAPI / TS types codegen** | See Research. |
| **[#249](https://github.com/CLOSERPROJECT/attesta/issues/249) Phases 0–2** | Legacy `cmd/server` thin-out; link, don’t block. |
| **SPA polish from handoff** | Product UI beyond placeholders; Paraglide vs sv-router; CI `app` check/build; Coolify Bun stage; whether `/app` stays auth-gated long-term; trim scaffold noise from PR #276. |
| **Move-plan “later Seams”** | DraftPort; domain chunks; Config CDN; retire legacy HTMX when product says so. |
| **Freeze `attesta-experiment`** | After truck-loop works on `/app` + Go; single edit target. |

---

## Research (to ticket or run later)

1. **TypeScript types from Go contracts** — Options: Goa OpenAPI → `openapi-typescript`; hand OpenAPI + `openapi-typescript` / `oapi-codegen`; golden JSON contract tests until then. Goal: frontend consumes generated types for Commit/Load DTOs without dual drift.
2. **Cerbos living policy design** — Resource kinds, actions, attr schema for Commit under the hybrid rule; how compact to encode Unlock order / Submissions for CEL; policy layout next to existing `cerbos/policies/substep_policy.yaml`.
3. **CEL weight check** — Spike: which structural gates stay readable in CEL vs stay as Go→boolean attrs; document the cutoff so “B if not too heavy” is operational.
4. **Living Mongo document shape** — Domain document + versioned Org Config documents; indexes; collection naming (after package/naming).
5. **GridFS / attachments for living Tasks** — When needed: Interface, authz (Cerbos), and share surface with legacy attachment storage.
6. **Dev seed without harness** — How demo Domain + Config versions and all-roles users land via existing `attesta seed` / new seed paths; parity with experiment scenarios.
7. **Glossary merge approach** — Single root glossary vs map; how Stream (legacy) vs Org Config / Domain (living) coexist in docs/agents without lying about runtime.

---

## Interesting opportunistic extracts (from legacy, only if transplant needs them)

| Candidate | Verdict |
| --- | --- |
| Session gates (`requireVerifiedPage`, …) | High value to **share via wiring** now; package extract later. |
| Session → actor / `OperatorIdentity` map | When AuthPort is real. |
| Cerbos client helper (`checkResourceAction`) | Likely share; living policies are new. |
| God `Store` / Process progress | **Do not** share as the living Interface. |
| Mongo client / DB | **Share** wiring. |

---

## Changelog

- 2026-10-29: Initial dump from grill + move-plan/handoff/#249 context.
- 2026-10-29: Lock domain model — versioned Org Config, org-scoped living domain, repeat jobs not config-runs, dual role catalog + per-task roles / Cerbos; defer glossary wording rewrite.
- 2026-10-29: Lock Job top-level only + Task tree (leaf vs Task group); reject jobs-all-the-way-down and lots-via-`completeness: some`; lock port-now (flat truck-loop + org Domain seams + path Commit) vs postpone nesting/lot intake.
- 2026-10-29: Wording pass to locked root `GLOSSARY.md` (flat-port nouns); decisions unchanged; Task group final English still TBD; Config run retired as living noun in favor of “whole-config execution” / historical code names.
- 2026-10-29: Lock code seam placeholders (`internal/living`, DomainStore/ConfigStore, `/api/living/domain` + `/commit`); defer real names to a final rename slice + discussion.
- 2026-10-29: Flat-port tickets published under `.scratch/living-flat-port/issues/` (01–13).
- 2026-10-29: Lock experiment→attesta copy/adapt vs rewrite vs do-not-copy policy for the flat port.
