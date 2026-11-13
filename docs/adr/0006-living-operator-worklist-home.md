# Living `/app` home is a single Worklist act-now list (Task + stay; overview unlinked)

Living operators land on a **Worklist** home that is one **act-now** list: viewer
**`actionable`** Tasks and **Can start** Jobs (peer row kinds, shared list layout)—not
legacy `/my`, not peer **In progress** / **Completed** bands, and not an Archive
surface yet. Rows are keyed by `jobId` / `jobInstanceId` / `taskId` (or Can start →
first Task with provisional `new`). Working a Task uses
`/app/{jobId}/{jobInstanceId|new}/{taskId}`. **Job instance overview** at
`/app/{jobId}/{jobInstanceId}` may remain routed for later reuse but is **not** a
Commit, Continue, home, or Task-back target. After Commit, auto-advance applies when
the same **OperatorIdentity** has another role-eligible actionable Task on that
instance and Config allows it; otherwise **home** (or **stay** when authored). Product
law is evaluated **server-side**: response **`navigation`** is `task` | `home` |
`stay` (legacy wire/Config `overview` maps to **home**); when `task`,
`nextTaskFormBundle`; when **`stay`**, **`continueNavigation`** is deferred `home` |
`task` (+ bundle when Continue targets a Task). SPA navigates from **`navigation`**
only—no Worklist re-walk for destination, no inferring `stay` from HTTP links.
Authored **`onCommit.navigation`** (`home` | `overview` | `advance` | `stay`; omit ⇒
`advance`; `overview` ⇒ home) and **`onCommit.advanceTarget`** (bool; omit ⇒ true)
override linear advance without plant branches. When only `advanceTarget: false`
applies, destination is **home**. For **`submit: many`** act-now rows, Submission
history chrome may sit under the row when `submissionCounts` N > 0 (closed count;
open = lazy list recap + links)—not a full instance accordion on home. **`submit:
once`** done recaps are not on home (gap until Archive). Home Load ships
viewer-scoped Worklist (title + roles + counts chrome); Task form bundles load on
Task open (`GET /api/living/task-form`) or Commit advance. **readyWhen** stays
Commit-path law (ADR-0005). Amends ADR-0005’s bulk Available bundles-on-Load for the
home path. Stream Operator home stays out of scope. Task-group Spawn/Close as a
product hub is out of scope (groups pending full removal).

**Considered options:** (1) Keep four home bands + overview handoff — rejected:
home should be act-now only; finished/non-actionable rediscovery waits for Archive.
(2) Delete overview route — rejected for now: keep path for later reuse; unlink from
product flows. (3) Always stay on overview for role handoff — rejected: handoff is
the next viewer’s act-now list. (4) Role-blind client filtering from Org Config —
rejected (ADR-0005/0006). (5) SPA re-derives post-Commit destination from Worklist —
rejected; Commit ships `navigation`.

**Amended by [ADR-0008](0008-living-task-submit-cardinality.md):** leaf `submit:
once|many`; Feragame multi-pass uses `many` + stay, not Task groups. Many-submit
Submission history on home reuses list-recap / timestamped LinkStore.

**Consequences:** Demo seed still needs role-split users so each home differs. SPA
must not treat overview as the board, must not show In progress/Completed on home,
must back from Task to home, and must not invent stay from links. Archive hierarchy
(working day → shift → lot) is explicitly later. Prior home-band and
overview-as-navigation amends of this ADR are superseded by the act-now hard cut.
