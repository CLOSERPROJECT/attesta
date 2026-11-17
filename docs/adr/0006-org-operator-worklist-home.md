# `/app` home is a single Worklist act-now list

Operators need a place to act next without a second rule engine or a board of finished work. Legacy multi-band homes and overview-as-handoff fight that.

**Decision:** `/app` home is one **act-now** Worklist: viewer **`actionable`** Tasks and **Can start** Jobs. Working a Task uses the Task route; **Job instance overview** may remain routed but is not a Commit, Continue, home, or Task-back target. **Archive** rediscovers past/completed Job instances as a flat list into that overview—not a Domain calendar tree, and not on home. After Commit, the server chooses the next destination (`task` | `home` | `stay`, including authored Config overrides); the SPA follows that outcome only—no Worklist re-walk and no inferring stay from links. Home Load ships viewer-scoped Worklist chrome; Task form bundles hydrate on Task open or Commit advance. readyWhen stays Commit-path law ([ADR-0005](0005-org-worklist-server-projection.md)). Stream Operator home and Task-group hubs are out of scope.

**Amended by [ADR-0008](0008-org-task-submit-cardinality.md):** multi-pass same-form work uses `submit: many` (+ stay), not Task groups; many-submit history may appear under act-now rows.

**Considered options:** (1) Four home bands + overview handoff — rejected: home is act-now only; rediscovery is Archive. (2) Delete overview — rejected for now: keep path; link from Archive only. (3) Always stay on overview for role handoff — rejected: handoff is the next viewer’s act-now list. (4) Role-blind client filtering from Org Config — rejected (ADR-0005). (5) SPA re-derives post-Commit destination from Worklist — rejected. (6) Domain calendar Archive hierarchy — rejected.

**Rejected:** In progress/Completed on home; overview as the board; inventing stay from HTTP links.
