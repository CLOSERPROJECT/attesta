# Living Tasks are once- or many-submit leaves; Feragame drops group multi-pass

Living operator work is modeled as **leaf Tasks** with authored submit cardinality
`once` (default) or `many`, gated primarily by Domain **`readyWhen`** (+ roles).
**Repeatable Task groups** (Spawn/Close occurrences) are the wrong fit for
Feragame multi-pass forms (lot register, pallet weigh, progress, container
register): those are the same form submitted repeatedly while Domain still has
eligible subjects. Worklist **`done`** remains Submission law for `once` only;
`many` never becomes `done` from a single Commit. Feragame plant Config prefers
**`readyWhen`** over **`unlockFrom`** (engine keeps Unlock for now; plant does
not author it). Plant entity **`load` renames to `lot`** with **lot session
status** `open | fullyProcessed` only (mint/open on `startLot`; terminal on
`closeLot`). There is no `parked` value and no park-on-turno / resume-on-openTurno
law.

**Amends [ADR-0005](0005-living-worklist-server-projection.md):** “A Task is
`done` after Submission on the instance” holds only for `submit: once` (omit ⇒
once). For `submit: many`, unlocked status stays `unavailable` | `available` |
`actionable` from readyWhen + roles.

**Amends [ADR-0006](0006-living-operator-worklist-home.md) plant path:** Feragame
no longer relies on Task-group Spawn/Close chrome for intake or line multi-pass.
Many-submit Submission history attaches under **Worklist** home act-now rows when
counts allow; Job instance overview is unlinked from Commit/Continue/Task-back
(Archive may open it; see ADR-0006 amend).

**Amended (lot session; park/resume overturned):** Product law in
[`.scratch/living-two-job-plant/LOCKED.md`](../../.scratch/living-two-job-plant/LOCKED.md)
overturns the prior `open | parked | fullyProcessed` lot session and
park-on-`closeTurno` / resume-on-`openTurno` rules in this ADR. Lot session is
`open | fullyProcessed` only; an open lot stays open until `closeLot` (or
equivalent). Working-day / shift Domain entities and turno bookend Tasks are out
of plant law. At most one lot may be `open`; next `startLot` waits until that lot
is `fullyProcessed`. Leaf `submit: once|many`, dumb executor, and Feragame
multi-pass via `many` + stay remain in force.

**Considered options:** keep repeatable Task groups for N submits; map/table
“leftovers” as unlock grammar; Unlock pipeline weigh→load→register; park lots
across shifts via turno close/open. Rejected: groups add Close/Spawn chrome
without matching plant law; leftovers were already rejected for spawn-many; plant
needs Domain-gated actions inside an open lot without a parked mid-state or
day/shift Domain.

**Consequences:** Config schema gains `submit`; Worklist/Commit/Cerbos honor
many-submit; Feragame fixture flattens groups and renames load→lot with
`open|fullyProcessed` lot session (no parked/turno); seed, SPA group affordances
for that plant path, and passport subject bindings follow. Task-group engine code
may remain until a later removal pass.
