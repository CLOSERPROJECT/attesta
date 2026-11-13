# Living Tasks are once- or many-submit leaves; Feragame drops group multi-pass

Living operator work is modeled as **leaf Tasks** with authored submit cardinality
`once` (default) or `many`, gated primarily by Domain **`readyWhen`** (+ roles).
**Repeatable Task groups** (Spawn/Close occurrences) are the wrong fit for
Feragame multi-pass forms (lot register, pallet weigh, progress, container
register): those are the same form submitted repeatedly while Domain still has
eligible subjects. Worklist **`done`** remains Submission law for `once` only;
`many` never becomes `done` from a single Commit. Feragame plant Config prefers
**`readyWhen`** over **`unlockFrom`** (engine keeps Unlock for now; plant does
not author it). Plant entity **`load` renames to `lot`** with status
`open | parked | fullyProcessed` for the line lot session (park on shift close,
resume on next open).

**Amends [ADR-0005](0005-living-worklist-server-projection.md):** “A Task is
`done` after Submission on the instance” holds only for `submit: once` (omit ⇒
once). For `submit: many`, unlocked status stays `unavailable` | `available` |
`actionable` from readyWhen + roles.

**Amends [ADR-0006](0006-living-operator-worklist-home.md) plant path:** Feragame
no longer relies on Task-group Spawn/Close home/overview chrome for intake or
line multi-pass; overview stays leaf-Task accordion for that fixture.

**Considered options:** keep repeatable Task groups for N submits; map/table
“leftovers” as unlock grammar; Unlock pipeline weigh→load→register. Rejected:
groups add Close/Spawn chrome without matching plant law; leftovers were already
rejected for spawn-many; plant needs parallel Domain-gated actions inside an
open lot, including incomplete lots parked across shifts.

**Consequences:** Config schema gains `submit`; Worklist/Commit/Cerbos honor
many-submit; Feragame fixture flattens groups and renames load→lot; seed, SPA
group affordances for that plant path, and passport subject bindings follow.
Task-group engine code may remain until a later removal pass.
