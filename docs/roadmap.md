# Roadmap

The order in which Living Attesta (`/app`) moves toward the target described
in [`vision.md`](vision.md). The vision says what the platform is and why;
this file says what gets built first. It replaces the "Suggested sequence" in
`vision.md` as the delivery order: the operator app comes first, and the
data-protection work runs alongside it.

Design sources: [`vision.md`](vision.md) (target architecture, gap analysis,
open decisions), [`interpretation-package.md`](interpretation-package.md)
(authoring model, open questions), [`sketch.yaml`](sketch.yaml) (Feragame in
the new shape), and ADRs 0008, 0009, and 0010.

## Overview

| Phase | Goal | Status |
|---|---|---|
| 0. Operator app | Routing, layout, and screens of the target board | Next; one decision pending |
| A. Protect today's data | Fix defects that lose or cap data now | Any time, alongside Phase 0 |
| 1. New config | The Interpretation package replaces Jobs and Tasks | After Phase 0 |
| 2. Observations and reconstruction | Capture is recorded as Observations; the Domain is rebuilt from them | After Phase 1 |
| 3. Online and offline | Capture works without a connection and syncs later | After Phase 2 |
| 4. Review, seals, publication | Passports come from sealed Review cases | After Phase 3, or earlier if needed |
| 5. Flows and federation | REA flows, inter-company exchange, machines | Deferred until asked |

Each phase lists its goal, the decisions it needs first, its topics, what it
leaves out, and when it is done.

## Phase 0: Operator app (routing, layout, UI)

**Goal.** An operator app whose navigation and screens follow the target
board model: one screen per scope or resource, the navigated path in the top
bar, and lists, tabs, and an actions menu on each screen (`interpretation-package.md`,
"Screens"). Tablets and phones on the plant floor; desktop for
administration.

**Decisions needed first.**

- **What the screens are built against.** Options: a screen contract fed by
  a hand-written fixture of the Feragame screens; the same contract fed by an
  adapter over today's `/api/living` worklist; or today's Job and Task model.
  To be decided after the human reviews `sketch.yaml`.
- **Fate of the Stream surface (`/my`).** Retire, freeze, or migrate. It
  shares the Go server and session with `/app`, so routing depends on it.
- **The `living` placeholder name.** Renaming is cheapest before new routes
  and API paths are added.

**Topics.**

1. Screen contract: the data the UI consumes for a screen (path, lists, tabs,
   available actions as event type and subject, rows, form inputs, read-only
   values, prompts), if the contract option is chosen.
2. Routing: the URL is the navigated path (working day, shift, lot); deep
   links and back behavior; QR scans land on a resource screen; paths differ
   by role (administration goes through the truck, logistics and line skip
   it); keep `sv-router` or move to SvelteKit routes.
3. App shell: top bar with the path, role display, scan, actions menu, and a
   slot for sync status (filled in Phase 3); responsive layout for floor and
   desktop.
4. Screen building blocks: lists, tabs with `add` and `group_by`, tables,
   list-then-form, forms (reusing `@sjsf/form`), read-only values, a prompts
   area, and a resource's history.
5. Row-level sending in tables: a row is sent when it is valid and settled
   (the operator leaves the row or presses Enter), never per keystroke.
6. Feedback states: loading, errors, empty screens, toasts, draft markers.
7. Design system and plant ergonomics: shadcn-svelte and Tailwind
   components, large touch targets, accessibility, Italian and English copy.
8. Session handling in the app: unverified users, login redirect, role
   display, logout, and organization switching if needed.
9. UI tests: Playwright end-to-end and component tests against the screen
   data.

**Out of scope.** Behavior that needs the new engine: follow-ups (`also`),
evaluating view and prompt rules, live derived values that stay overridable,
and corrections when a sent row is edited. These arrive in Phases 1 and 2.

**Done when** every Feragame screen in `sketch.yaml` is reachable for each
role through path-based routing on floor and desktop layouts, with tests.

## Track A: Protect today's data

Independent of the phases; can land any time.

1. **Lost updates.** `Commit` reads the org's Domain, changes it, and writes
   the whole document back with `ReplaceOne` filtered by org only
   (`server/internal/living/store_mongo.go`), so two concurrent commits can
   silently lose one. Add optimistic concurrency.
2. **Domain document size.** The whole Domain, including every Submission,
   is one MongoDB document per org, and MongoDB caps documents at 16 MB.
   Measure current sizes and growth, then decide whether to split
   Submissions out.
3. **Observation envelope under today's Commit (recommended early).** Store
   the raw payload, a client-generated ID used for idempotency, the config
   version, and the three timestamps, and record group close as an envelope
   too (`CloseGroup` in `server/internal/living/groups.go` writes no
   Submission today). History captured before this cannot be replayed. It is
   listed again in Phase 2 in case it waits until then.
4. **Doc fixes already decided.** In `vision.md`: the replay test must cover
   group close and reuse each record's stored timestamp; fix the shortened
   path `fixtures/feragame-config.yaml` and the link to a `.scratch/` file. In
   `interpretation-package.md`: allow `$today` only as a capture-time default,
   never during reconstruction.

## Phase 1: New config (Interpretation package)

**Goal.** Feragame runs on an Interpretation package of scopes, resources,
and event types (ADR-0008, ADR-0009, ADR-0010) instead of Jobs, Tasks, and
array forms, and the Phase 0 screens are generated from the package.

**Decisions needed first** (`interpretation-package.md`, "Open questions").

- Top-level structure of the package and syntax for views, prompts, checks,
  computed properties, and expressions.
- What derived inputs may read (the `tare` case in `pallet_weighed`).
- How a reference is expressed when it is not on the navigated path.
- Whether an event's target state may depend on its input.
- The order behind display labels such as `C-12` (`$seq`).

**Topics.**

1. Package format and an end-to-end parity check of `sketch.yaml` against
   `server/internal/living/fixtures/feragame-config.yaml`.
2. Parser and validator: every event type that enters a state supplies what
   the state requires; follow-up chains have no cycles; REA declarations are
   well formed.
3. Package registry and versioning.
4. Engine: state derived from events, `creates`, follow-ups, references
   resolved at capture, `available_when`, prompts, computed properties.
5. Port Feragame; retire Task, Task group, and dynamic-map authoring;
   migrate existing Domain data; rebuild the demo seed.
6. Effects under the new model: label PDFs, CSV registers, and QR codes from
   the `living-http` sidecar; legal registers and exports.
7. Documentation: reopen ADR-0005 and ADR-0006; an ADR for screens,
   references at capture, and follow-ups; promote target terms into
   `GLOSSARY.md` as code lands.
8. Admin: publishing and switching package versions.

**Out of scope.** Ungated capture and reconstruction (Phase 2). Commit may
keep checking `from` states until then.

**Done when** Feragame runs end to end on the new package, the old Task
authoring is gone, and the operator app renders from the package.

## Phase 2: Observations and reconstruction

**Goal.** Every operator action is stored as an Observation, capture is
never blocked by world state, and the Domain is rebuilt from Observations by
a deterministic engine.

**Decisions needed first.** Who may correct an Observation; whether voiding
a trigger also voids its follow-ups or only offers to.

**Topics.**

1. Observation envelope (if Track A did not land it): raw payload, client ID,
   package version, occurred, recorded, and received times; idempotent
   ingestion.
2. Ungated capture: Cerbos authorizes the source (who or which device may
   report this event type), not the world state; ordering rules become
   prompts.
3. Reconstruction engine: replay into revisions with snapshots and a content
   hash; the Domain becomes a projection; a replay test proves equality.
4. Checks and findings on each revision.
5. Corrections: replace or void an Observation; state is recomputed; editing
   a sent row in the UI sends a correction.
6. Attachments and evidence storage.

**Done when** the stored Domain can be deleted and rebuilt identically from
Observations, and capture succeeds regardless of state, with findings in its
place.

## Phase 3: Online and offline

**Goal.** Operators keep capturing without a connection; Observations sync
later and the app keeps evolving its screens from a local world model.

**Decisions needed first.** Whether a device runs a whole workflow offline or
only its workstation's slice; the portable engine (Go compiled to WebAssembly,
or JSONata-only rules with a shared evaluator); whether effects such as label
printing must run offline.

**Topics.** Device observation log (IndexedDB); service worker and
installable app; device identity; sync protocol with acknowledgements and
retries; local reconstruction and local world model; sync status in the top
bar; encryption at rest; offline effects if decided.

**Done when** a device can work through a shift offline, sync, and match the
central reconstruction.

## Phase 4: Review, seals, publication

**Goal.** Passports and other publications come only from sealed Review
cases, replacing today's passport lock (reopens ADR-0007).

**Decisions needed first.** Review case granularity (lot, shift, passport
subject, period); identity merge policy for resources created separately on
two devices.

**Topics.** Review workspace; Review cases and Seals bound to a revision;
reviewer corrections as Observations; passport, GS1 Digital Link, and UNTP
built from Seals; identity resolution (merge and split).

Today's passport lock keeps working until this phase. If sealing is needed
before offline, this phase can move ahead of Phase 3.

## Phase 5: Flows and federation (deferred)

Not designed until the human asks: computing REA flows and genealogy,
passports built from flows, transfers between companies, correlating several
Observations into one event, machine and IoT ingestion, and planning.

## Cross-cutting

| Topic | Needed by |
|---|---|
| Stream surface (`/my`) fate | Phase 0 |
| `living` placeholder rename | Phase 0 (routes) or Phase 1 (API) |
| Data migrations and backups for the model change | Phase 1 |
| Reports and legal registers | Phase 1 |
| Admin screens: members, roles, package versions, devices | Phases 1 and 3 |
| Audit logging and operations | Phase 2 |
