# Org Worklist is server-projected; SPA does not re-gate

Operator chrome needs startsWhen and readyWhen. Copying those rules into the SPA would create a second rule body beside Cerbos-authoritative Commit.

**Decision:** Go evaluates Submission relations and Task readyWhen into a viewer-scoped **Worklist** on org Load and Commit. The SPA reads Task statuses (`unavailable` | `available` | `actionable` | `done`) and Job `canStart`; it must not re-implement startsWhen, readyWhen, or “mine” eligibility. Path availability is `done` | `available` only (platform Unlock / Worklist `locked` removed; Config rejects `unlockFrom`). `unavailable` means readyWhen false (visible, not navigable); when readyWhen passes, role eligibility splits `available` (handoff) vs `actionable` (this viewer). Load/Commit do not ship full Domain or Org Config; server-evaluated display chrome the SPA cannot invent is projected, and live draft/operator chrome may stay client-side. A `submit: once` Task becomes `done` after Submission on the instance (Domain reads fill forms but do not reopen Worklist status).

**Amended by [ADR-0006](0006-org-operator-worklist-home.md):** home Load is lazy for Task form bundles; post-Commit navigation is server-directed.

**Amended by [ADR-0008](0008-org-task-submit-cardinality.md):** `submit: many` stays readyWhen-gated and does not become `done` from a single Commit.

**Rejected:** Production TypeScript copies of startsWhen/readyWhen; a parallel viewer-id list beside status; shipping full Domain or Org Config for the SPA to re-project.
