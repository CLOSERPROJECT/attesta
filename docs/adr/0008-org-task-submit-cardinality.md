# Org Tasks are once- or many-submit leaves

Operator work is modeled as **leaf Tasks** with authored submit cardinality
`once` (default) or `many`, gated primarily by Domain **`readyWhen`** (+ roles).
**Repeatable Task groups** (Spawn/Close occurrences) are the wrong fit for
multi-pass same-form work: those are the same form submitted repeatedly while
Domain still has eligible subjects. Worklist **`done`** remains Submission law
for `once` only; `many` never becomes `done` from a single Commit. When Domain
already holds the prerequisite, author **`readyWhen`** and omit **`unlockFrom`**
(engine keeps Unlock for now).

**Amends [ADR-0005](0005-org-worklist-server-projection.md):** “A Task is
`done` after Submission on the instance” holds only for `submit: once` (omit ⇒
once). For `submit: many`, unlocked status stays `unavailable` | `available` |
`actionable` from readyWhen + roles.

**Amends [ADR-0006](0006-org-operator-worklist-home.md):** product path no
longer relies on Task-group Spawn/Close chrome for multi-pass. Many-submit
Submission history attaches under **Worklist** home act-now rows when counts
allow; Job instance overview is unlinked from Commit/Continue/Task-back
(**Archive** may open it; see ADR-0006).

**Considered options:** keep repeatable Task groups for N submits; map/table
“leftovers” as unlock grammar; Unlock pipeline as the only multi-pass gate;
park-across-shift mid-states as platform law. Rejected: groups add Close/Spawn
chrome without matching Domain-gated parallel actions; leftovers were already
rejected for spawn-many; plants need Domain-gated actions inside an open subject
without a fixed Unlock chain. Plant mid-states (if any) are Config payload—see
[`docs/feragame/GLOSSARY.md`](../feragame/GLOSSARY.md).

**Consequences:** Config schema gains `submit`; Worklist/Commit/Cerbos honor
many-submit. Task-group engine code may remain until a later removal pass. Demo
plant (Feragame) flattens groups, prefers `readyWhen`, and authors lot session /
entity renames in fixture payload—illustration, not this Decision.
