# Task Recap projects Domain entities (or submissions) from Config

Overview expand needs a board table for a Task at a Submission address. Stamping `createdBy` on every Domain entity would duplicate the actor already on Submissions and freeze authorship across later patches. Snapshotting cell values at mint would drift from the live Domain the operator is working.

**Decision:** Submissions record `entitiesTouched` (mint/patch by kind + id). Task Recap is a server projection: when Config authors `recap:`, one row per touched entity of that kind with **current** Domain fields and actor/time/links from the latest touching Submission at that address; otherwise one row per Submission (fact union). See [Recap](../org/GLOSSARY.md).

**Rejected:** Entity-level createdBy; mint-time field snapshots as the overview body; SPA re-deriving entity rows from Domain.
