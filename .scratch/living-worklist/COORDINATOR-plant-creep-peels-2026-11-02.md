# Coordinator — plant-creep peels 08–11

**Integration branch:** `feat/attesta-rework`  
**Process:** one ticket → one feature branch → sequential subagents → reviewer → fixer → **squash merge** into integration.  
**Tickets:** `.scratch/living-worklist/issues/08` … `11` (ordered; each blocked by previous).  
**Grill decisions:** embedded in each ticket body; do not re-litigate.

## Branch naming

`ticket/08-org-catalogs` → `ticket/09-bind-containerReg-flatten` → `ticket/10-domain-kinds-map` → `ticket/11-drop-leftovers-grammar`

## Per-ticket loop (CoS)

1. Implementer on ticket branch (worktree OK); sequential in-ticket subagents A→B→C; conventional commits; run living Go (+ SPA if touched) tests.
2. Reviewer: Matt Pocock **code-review** skill (Standards + Spec vs ticket) — **not** Bugbot.
3. Fixer addresses review only; new commits on same branch.
4. Squash-merge into `feat/attesta-rework`; delete ticket branch.
5. Mark ticket status done in the scratch issue file (checkbox + Status).

## Constraints

- `AGENTS.md`, conventional commits, minimal diffs, comments = why only.
- No push unless CoS asks.
- Leave unrelated dirty scratch boards alone.
- Mongo wipe only in ticket 10C.

## Frontier

**08–11 done** (all squashed into `feat/attesta-rework`). Frontier empty. Residual: rename server `leftover*` wire identifiers when ready.
