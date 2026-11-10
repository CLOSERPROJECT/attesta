# Resource model for the Interpretation package

Design the authoring model of the Interpretation package (`docs/vision.md`):
how an org's configuration declares resources, their lifecycles, and the
events operators report on them. Replaces Task/array-form authoring as seen
in `server/internal/living/fixtures/feragame-config.yaml`.

Status: decisions settled with the human on 2026-11-09. Docs only, no code.

## Pointers

- Target architecture: `docs/vision.md` (Observation, Reconstruction, Review
  case, Seal, Interpretation package).
- First sketch (to be rewritten): `docs/sketch.yaml`.
- Pain points: Feragame `line/lineLoad/mark` (lines 489–537, `mapTable`,
  `patchEach`, `reads.rows.where`) and `line/containerReg/register` (lines 548–714,
  8 material arrays, `initialData` continuation, `continuesContainerId`,
  `presenceMark`, `supersede` bind, `mintMany` with `'C-' & count+1`).
  Engine special cases they force: `form_project.go`, `form_bundle.go`,
  `prep.go` partial-map mode, `rows.go`, `form_compile.go` path escaping,
  repeatable groups used only to allow resubmitting.
- Current vocabulary: `GLOSSARY.md`. ADRs touched: 0005, 0006, 0007.
- ADR format: `/Users/giovanniabbatepaolo/.agents/skills/domain-modeling/ADR-FORMAT.md`
  and the style of `docs/adr/0007-living-passport-production.md`.

## Vocabulary

- **Resource type / resource**: a kind of material thing the org tracks
  (lot, pallet, container) and one instance of it. Has a `quantity`; flows.
- **Scope type / scope** (2026-11-10, ADR-0010): a frame work happens in
  (working day, shift, truck). Lifecycle, containment, can be a subject;
  never a quantity, never flows. Not "context": context is the navigation
  path (`$context.lot`).
- **Agent / Flow / Process**: REA declarations (`agents`, `rea` per event
  type, `processes`). Declared and validated, not computed yet.
- **State**: where a resource is in its lifecycle. Owned by the engine,
  derived from the events that happened to it. Config never writes it.
- **Event type**: declared in config. An **Observation** asserts that an
  event of type X happened to subject Y. One Observation per event for now;
  correlation of several Observations into one event stays as in vision.md.
- **Subject**: the one resource whose lifecycle an event advances.
- **Reference**: context the event happened in (shift, current lot, truck).
- **Creates**: resources an event brings into existence.
- **Correction**: an Observation that supersedes an earlier one.
- **View**: how available actions are presented (table, hierarchy).
- **Check / Finding**: a rule over the reconstructed world, and the
  inconsistency it reports to review.
- **Prompt**: an urgent suggested action shown at capture.

## Decisions

1. **Event-driven lifecycle.** State is derived from events. States declare
   the properties a resource must have in that state (`requires`). Events
   declare `from` states, `to` state, inputs, and how values are written.
   Properties are gained or updated, never removed (history keeps every
   value; "not shown in this state" is expressed by the state). Static
   rule, enforceable later by a config validator: every event type with
   `to: S` supplies what `S` requires.
2. **Corrections are not transitions.** A correction references the
   Observation it supersedes; state is recomputed. State machines show only
   what physically happens. Example: a `done` pallet goes back to
   `partially_loaded` only through a correction.
3. **One subject per event.** Context is a reference, never a second
   subject. Events may `creates` resources (with a count). "Same event on
   many resources" is N events, never one event with N subjects. REA
   `consumes` / `produces` with quantities is deferred until a real
   transformation needs it (likely inter-company).
4. **Tables are views.** Rows = resources of the subject type in one of the
   event's `from` states, visible to the operator's roles. Columns = event
   inputs. Event types that create resources show an "add row". A row is
   sent automatically when it is valid **and settled** (operator leaves the
   row or presses Enter), never per keystroke. Editing a sent row sends a
   correction. "Submit all" is an optional per-view setting for scenarios
   that need it, and still produces N Observations.
5. **The board.** An operator's available actions are every (event type,
   subject resource) pair allowed for their roles. Grouped by event type they
   form tables; grouped by containment they form a hierarchy (working day →
   shift → actions on resources). Containment is a reference marked `within`.
   GUI presentation (tabs or otherwise) is free.
6. **Never block capture.** Ordering rules become prompts at capture and
   findings at review. Numbers that do not add up are review findings
   produced by checks.
7. **Identity.** Resource IDs are minted on the device and globally unique.
   Display labels (e.g. `C-12`) are separate from identity and may be
   assigned later.
8. **Screens (2026-11-10).** The board is presented as one screen per
   resource; the navigated path is the top bar and the screen's context.
   Any screen may have lists, tabs, and an actions menu; there is no fixed
   leaf level. Defaults come from `within`, subjects, and roles; views list
   only overrides (list rows by a rule, tabs combining event types with
   `add` and `group_by`, labels, table versus list-then-form). Feragame
   paths: administration working day → shift → truck; logistics and line
   working day → shift → lot (the truck is skipped on purpose).
9. **References resolved at capture** from the screen context
   (`$context.lot`) and stored in the Observation; corrections fix them.
10. **`available_when`** filters the board only. Capture is not checked;
    an Observation captured while it did not hold becomes a finding.
11. **Lot lifecycle** `registered` → `processing` (`lot_started`) →
    `processed` (`lot_processed`), both by logistics. Pallets are weighed
    only while their lot is processing; weighing the first pallet of a
    registered lot starts it (`lot_started` is a follow-up, `on_board:
    false`), and only when no other lot is processing.
12. **Container handoff is a state**: `open` → `handed_off` → `open`
    (`container_taken_over`, sent for every handed-off container as a
    follow-up of `lot_started`) → `full`. "Full" and "handed
    off" are two event types (different states, only "full" creates a
    successor); the containers tab shows both as actions on one row. A
    container can carry partial weights for any number of lots.
13. **Follow-ups (`also`)**: one operator action may send several
    Observations, each with one subject (keeps ADR-0009). Resolved at
    capture from the device's world model and stored; chains allowed,
    cycles rejected; each Observation corrected on its own. Missed
    follow-ups (offline) surface as prompts and checks.
14. **Scopes and REA declarations (2026-11-10, ADR-0010).** Top-level
    `scopes:` (working_day, shift, truck) next to `resources:` (lot,
    pallet, container); no `entities` wrapper. Scopes are not REA
    processes. Clients are `agents`; each resource type names its
    `quantity` properties, one per unit (pallet: kg and panels); lot
    registration is a `transfer` (ownership, not only custody); event
    types that move material declare `rea`
    (Valueflows action, resource, quantity, provider/receiver);
    `lot_processing` is a declared process (pallets consumed in,
    containers produced out). Quantities are written only by measuring
    events. Reconstruction ignores these until flows are computed.

## Feragame facts (for the sketch and worked examples)

- Chain: working day → shift → truck → lot → pallet. Lots are registered
  one at a time on an unloading truck (client, waste code, FIR number,
  pallet count, truck weight after unload, notes); registering a lot
  creates its pallets. Role slugs stay `line`, `logistic`, `administration`.
- Pallet weighing records gross weight, tare kind (industrial 25 kg, euro
  20 kg, or custom), average panel weight, an overridable derived panel
  count, and IR/NIR.
- **Pallet**: created → weighed → loading → done. A line load is real: an
  operator takes a weighed pallet and loads its panels onto the line one by
  one; that is the pallet's `loading` phase. If the shift ends mid-load, the
  operator explicitly marks only pallets started but not completed as
  `partially_loaded`; they resume next shift. `done` → `partially_loaded`
  only by correction. Two pallets may load at once (rare); two lots are
  never mixed on the line, so "the current lot" is well defined.
- **Lot**: once a truck is unloaded its lots go to the storage area; each
  lot is then processed on its own. Logistics starts it and marks it
  processed; the line marks pallet progress. Marking processed changes the
  GUI (the lot leaves the shift's list).
- **Container**: one resource across lots. The operator reads the **total**
  weight off the scale. At lot end: weigh every open container, mark each
  full or handed off, mark the lot processed, start the next lot, which
  takes over the handed-off containers. **Not** weighed at shift end.
  Weighing as full also creates its empty successor (same material).
  Weight per lot = this total minus the previous total, credited to the
  weighing's lot; any number of lots per container (generalizes
  `secondarySourceLoadId`); genealogy is computed, not entered. Getting
  this right is a core Feragame need and the proof of non-linear flows.
- Container weights per lot not matching the lot's weight → review finding.

## Open points (record, do not decide)

- Syntax for views, checks, prompts, and computed properties.
- How a reference is expressed when it is not on the navigation path.
- Whether correcting a trigger (voiding the first `pallet_weighed`) should
  void its follow-ups (`lot_started`, takeovers) automatically, or only
  offer to.
- Computing flows: wanted for building the passport and for inter-company
  transfers, not needed now. Open thread; do not design it yet.

## Deliverables

The coordinator writes all docs (human request, 2026-11-09); subagents may
only review.

| File | Status |
|---|---|
| `docs/interpretation-package.md` (new) | drafted |
| `docs/sketch.yaml` (full Feragame example) | drafted |
| `docs/adr/0008-resource-event-lifecycle.md` | drafted |
| `docs/adr/0009-one-subject-per-event.md` | drafted; scopes wording 2026-11-10 |
| `docs/adr/0010-scopes-and-rea-declarations.md` | drafted 2026-11-10 |
| `docs/vision.md` (links, "Guidance instead of gating", ADR note) | edited |

## Rules for agents

- Touch only the files your step owns. Do not commit.
- Follow `AGENTS.md`; use `GLOSSARY.md` vocabulary for current concepts and
  this file's vocabulary for target ones. Do not edit `GLOSSARY.md`.
- Flag conflicts with ADR-0005, ADR-0006, ADR-0007 explicitly.
- Write for a reader new to the codebase: complete sentences, no shorthand.
