# Next thin-SPA candidates (context after Worklist 01–05)

**Date:** 2026-10-30  
**Tip:** `feat/attesta-rework` @ `346546d` (squash `70298a3`)  
**Sources:** `$TMPDIR/handoff-living-thin-spa-next-candidates-2026-10-30.md`, `$TMPDIR/architecture-review-20261030-133840.html`, codebase scouts.

## Arc status

Tickets **01–05 done** and squashed. Safe to schedule follow-ons. Do not re-open JobChrome delete, twin truck-loop DRY, feragame rename, or SPA Unlock/startsWhen (ADR-0005).

## Ranked follow-ons

| # | Candidate | Strength | Size | Status now |
| --- | --- | --- | --- | --- |
| **1** | Chrome Domain projectors → bindings | Strong | M–L | **Live SPA** `project-final-results.ts` + plant tare/`palletWeight`; Go only ships DomainSnapshot bag |
| **2** | Harden FormPlan residual | Strong | M | Compile/`projectFormSchema`/`resolveRead` **prod-dead**; delete/quarantine so they cannot re-grow |
| **6** | Orphan Commit-prep twins | Explore | S | TS `flattenContainerRegFacts` / `filterFilledObjectWeights` **dead** — pure delete |
| **4** | Authored-shape classifiers | Explore | S | TS classifiers feed dead compile only; Go owns live classification |
| **5** | Domain→Draft re-seed | Explore | S | Runtime seed = bundle `initialData`; SPA seed modules **test/orphan** |
| **3** | Seal / progress omit | Explore | S–M | Client seal still live; Go prep authoritative — twin by design; harden contract or peel later |
| **7** | Roles / readyWhen chrome | Speculative | — | ADR defers; don’t grow SPA gates |

**Cleanup (#6→4→5→2):** squashed into `feat/attesta-rework` as `fc0d1e2`. Worktree removed.

**Recommended first grill:** **#1**. Ticket 05 removed Domain from the wire; `finalResults` still walks `bindings.domain` client-side with plant hardcodes — second Domain seam unless aggregates land in the bag.

### Candidate 1 detail (pointers)

- SPA: `form/project-final-results.ts`, `live-task-chrome.svelte.ts`, `container-reg-materials.ts`
- Prod chain: `JobPanel` → `TaskForm` → `LiveTaskForm` → live chrome → `projectFinalResults`
- Go today: `buildFormBindings` (snapshot + tareKinds); **no** `projectFinalResults`
- Slice option: precompute finalResults display into bindings; drop SPA projector; keep JSONata recap/valuePrint client until a later slice
- Risks: dual materials with `container_reg.go`; Italian labels/bands; tare defaults drift

### Quick wins after or beside #1

Order for deletion-only: **#6 → #4 → #5 → #2**.

### Pause

- 130731 “Deep Unlock + startsWhen” TS+Go twin for chrome — superseded by Worklist
- readyWhen / Roles chrome until product asks
