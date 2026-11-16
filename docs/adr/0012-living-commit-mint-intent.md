# Living Commit mint is explicit client intent; instance id is server-evaluated after transform

Living **Commit** carries an explicit **mint** knob on the wire: `mint.kind` is
`startingNew`, `spawningNew`, or `none`. Omit `mint` or an empty `kind` ⇒
`none`. The server **does not** infer mint vs reuse from an empty
`jobInstanceId`, from “first Task in Job order” alone, or from provisional UI
state. **Client-required mint intent; server-evaluated instance id after mint.**

**`startingNew`** starts a new **Job instance** on the first leaf Task in
authored Job order: the client omits or sends empty `jobInstanceId` and any
client id is **cleared and ignored** before authorize and mutate. After
`onSubmit.transform`, the server evaluates Org Config **instance id** and
**label** expressions from submission facts and bindings, appends one
`JobInstance`, and returns the **evaluated** id on the Commit response (the SPA
re-binds navigation and draft keys from that response, not from a local
sentinel). A second Commit with `startingNew` that would mint the same evaluated
id is rejected at prep (no duplicate Domain rows). The wire sentinel
`jobInstanceId: "_new"` is always rejected—use `mint.startingNew` with empty id.

**`none`** continues work on an existing instance: non-empty **real**
`jobInstanceId` is required; Commit **never** mints a Job or group instance. For
nested task paths, a real `groupInstanceId` is required unless the operator is
on the group spawn path below.

**`spawningNew`** mints a **Task group** occurrence on a nested path: requires
a Job-scoped `jobInstanceId`, a nested `taskPath`, and the provisional group
instance id in the path; the first leaf in authored **group** order must Commit.
The engine keeps group Spawn law for configs that still author groups even when
a plant flattens multi-pass into `submit: many` leaves ([ADR-0008](0008-living-task-submit-cardinality.md)).

Provisional “start new” UI (empty instance bag) is **client-only** until the
first successful Commit: no separate create HTTP route; first Task Commit with
`startingNew` is the mint. Cerbos stays a single `commit` action; first-Task
start privilege remains `rolesAllowed` on that Task, with authorize allowing
empty `jobInstanceID` only under `startingNew` prep.

**Considered options:** (1) Infer mint when `jobInstanceId` is empty — rejected:
ambiguous mid-Job empty ids and duplicate mint bugs. (2) Accept `_new` on the wire
as “please mint” — rejected: sentinel leaked into Domain and collided with
evaluated ids. (3) Client-supplied instance id on first Commit — rejected: id
law lives in Org Config exprs after transform, not operator typing. (4) Mint from
Worklist or Unlock alone without `mint` — rejected: Commit is the Seam;
intent must be on the Commit body. (5) Drop `spawningNew` when plants remove
groups — rejected: engine/group path remains until removal pass; plants use
`none` + real ids on flat many-submit Tasks.

**Consequences:** Commit DTO and handler validate `mint.kind` and sentinel ids;
Go mint branches in `applyConfigCommit` enforce first-in-order and
post-transform instance expr evaluation; Mutate hardens duplicate Job instance
append. SPA TaskForm derives `mint` from provisional instance bag (`''` ⇒
`startingNew`, bound id ⇒ `none`, empty group bag ⇒ `spawningNew`) and never
sends `_new` on Commit. Tests cover second-`startingNew` duplicate denial and
mid-Job `none` commits. Draft keys may keep local provisional ids; wire and
Domain use evaluated ids from the Commit response.