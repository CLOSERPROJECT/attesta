# Org Tasks are once- or many-submit leaves

Multi-pass same-form work is “submit again while Domain still has eligible subjects.” Repeatable Task groups (Spawn/Close) add chrome without matching that Domain-gated loop.

**Decision:** Operator work is **leaf Tasks** with authored submit cardinality `once` (default) or `many`, gated primarily by Domain **`readyWhen`** (+ roles). Worklist **`done`** is Submission law for `once` only; `many` never becomes `done` from a single Commit and stays `unavailable` | `available` | `actionable` from readyWhen + roles. Sibling / Domain prerequisites live in **`readyWhen`** (and Job **`startsWhen`** for start). Platform **Unlock** / authored **`unlockFrom`** are removed — Config rejects `unlockFrom` like `kind:group`. Worklist path availability is `done` | `available` (no Unlock `locked`); `sequenceOk` is `!Done` ∧ `startsWhen` only. Task groups are not the product path for multi-pass; many-submit history may sit under act-now rows ([ADR-0006](0006-org-operator-worklist-home.md)).

**Amends [ADR-0005](0005-org-worklist-server-projection.md)** and **[ADR-0006](0006-org-operator-worklist-home.md)** as above.

**Considered options:** Repeatable Task groups for N submits; leftovers as unlock grammar; Unlock pipeline as the only multi-pass gate; park-across-shift mid-states as platform law. Rejected: groups mismatch Domain-gated parallel actions; leftovers already rejected for spawn-many; plants need Domain gates inside an open subject without a fixed Unlock chain; plant mid-states stay Config payload.

**Consequences:** Config schema gains `submit`; Worklist/Commit/Cerbos honor many-submit. Task-group engine, Close API, and `spawningNew` mint are removed (Config rejects `kind:group`). Unlock engine and `unlockFrom` validate are removed (Config rejects `unlockFrom`). Demo plant (Feragame) flattens groups, prefers `readyWhen`, and authors lot session / entity renames in fixture payload—illustration, not this Decision.
