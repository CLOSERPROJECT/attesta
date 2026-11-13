# 01: feat(living): gate bancale progress on weight

**What to build:** Plant law — a **bancale** (pallet) cannot progress until it has been **weighted**. On an open lot, **Segna avanzamento** (`markProgress`) must only treat pallets that already carry Domain `weight` (from a successful **Pesa bancale** / `weigh` Commit) as progress-eligible. Author the gate in Org Config (`server/internal/living/fixtures/feragame-config.yaml`) via Domain-driven **readyWhen** and matching `reads.where` / row projection. Go and the SPA stay **dumb executors**; no Feragame-named engine branches.

**Revises LOCKED.md:** This plant law **revises** [`.scratch/living-task-submit-cardinality/LOCKED.md`](../LOCKED.md), which currently says weighing does not gate progress and that `markProgress` has no weigh gate / reads unweighed pallets — product §6 (“Weighing does not gate loading… `weigh` / `markProgress` / `containerReg` are peer…”) and the Feragame line table (`markProgress` | many | readyWhen: lot `open`; reads that lot’s pallets (incl. unweighed)). Implementation.md Phase 4 likewise said “**no** weigh gate on progress.” Update those locked statements when implementing; leave weigh↔load and progress↔containerReg peer (no pipeline) unless Cos expands scope.

**Blocked by:** None.

**Status:** Done

- [x] Open-lot `markProgress` excludes unweighted pallets (Domain `weight` required)
- [x] Unweighted bancali cannot be marked partial/done until weigh Commit has set `weight`
- [x] LOCKED.md (and Phase 4 notes as needed) revised to match the weigh-before-progress law
- [x] Fixture + living plant expectations updated; no plant id special-cases in Go/SPA
