# `/app` instance home is a single Worklist act-now list

Operators need a place to act next without a second rule engine or a board of finished work. Legacy multi-band homes and overview-as-handoff fight that. Discovery and finished-work rediscovery live on `/my` ([ADR-0014](0014-org-config-as-stream.md)).

**Decision:** Everyone lands on `/my`. `/app` is scoped to one **Org stream instance**; its home is one **act-now** Worklist for that instance: viewer **`actionable`** Tasks and **Can start** Jobs. Working a Task uses the Task route. After Commit, the server chooses the next destination (`task` | `home` | `stay`, including authored Config overrides); Commit navigation **`home`** means that instance home—the SPA follows that outcome only—no Worklist re-walk and no inferring stay from links. Home Load ships viewer-scoped Worklist chrome; Task form bundles hydrate on Task open or Commit advance. readyWhen stays Commit-path law ([ADR-0005](0005-org-worklist-server-projection.md)). **Archive** and **Job instance overview** are removed for Org streams; the `/my` dashboard (with its done filter) and instance timeline replace them. Formata Stream Operator home and Task-group hubs remain out of this Decision’s Org path.

**Amended by [ADR-0008](0008-org-task-submit-cardinality.md):** multi-pass same-form work uses `submit: many` (+ stay), not Task groups; many-submit history may appear under act-now rows.

**Amended by [ADR-0014](0014-org-config-as-stream.md):** landing, instance-scoped home, and removal of Archive / overview for Org streams as above.

**Considered options:** (1) Four home bands + overview handoff — rejected: home is act-now only. (2) Keep Archive / overview for Org streams — rejected: `/my` dashboard and timeline cover finished work. (3) Always stay on overview for role handoff — rejected: handoff is the next viewer’s act-now list. (4) Role-blind client filtering from Org Config — rejected (ADR-0005). (5) SPA re-derives post-Commit destination from Worklist — rejected. (6) Domain calendar Archive hierarchy — rejected. (7) `/app` as the operator landing — rejected (ADR-0014).

**Rejected:** In progress/Completed on instance home; overview as the board; inventing stay from HTTP links; treating `/app` (unscoped) as the affiliated landing.
