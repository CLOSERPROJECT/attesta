# Org Config as Stream

Org Config work needs Stream-shaped discovery and timelines without a Domain per truck or a Formata runtime for plant law. **Decision:** each Org Config file is one **Org stream** with its own Active Config version and Domain; an **Org stream instance** is a filtered view over that Domain, anchored on one **anchor Job** instance (plus Job instances its `each` selects)—not a Domain of its own. `/my` catalogs and dashboards Org-backed streams; everyone lands on `/my`; `/app` routes are scoped to one Org stream instance. Streams of the same org are isolated and do not share entities.

**Amends [ADR-0005](0005-org-worklist-server-projection.md):** Load is keyed by (org, stream, Org stream instance); the Worklist is always filtered to that instance.

**Amends [ADR-0006](0006-org-operator-worklist-home.md):** `/app` is no longer the operator landing; Commit navigation `home` means the instance home. Archive and Job instance overview are removed for Org streams—the `/my` dashboard and timeline replace them.

**Amends [ADR-0011](0011-org-qr-resolve.md):** QR resolve is stream-scoped and may jump the operator to another Org stream instance of the same stream.

**Considered options:** (1) Domain per stream instance — rejected: plant-wide rules (one open lot, carry-over containers, plant-scoped ids) need one Domain per stream. (2) Run Org plants on Formata/Process — rejected: the Org executor stays a dumb Config evaluator. (3) Plant-wide `/app` board — rejected: every operator route is instance-scoped; cold QR jumps into the right instance.

**Rejected:** Sharing entities across streams of the same org; Stream-only features (substep overrides, termination, delete, Formata builder, Stream DPP, `/events`) for Org-backed streams.
