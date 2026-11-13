# Coordinator — living `/app` home hard cut

DRI: Cos session. Subagents do the work; this file is the status board.

## Pointers (do not duplicate into prompts)

- [`LOCKED.md`](./LOCKED.md)
- [`IMPLEMENTATION.md`](./IMPLEMENTATION.md)
- [`GLOSSARY.md`](../../GLOSSARY.md)
- [`docs/adr/0006-living-operator-worklist-home.md`](../../docs/adr/0006-living-operator-worklist-home.md)
- [`AGENTS.md`](../../AGENTS.md) — dumb executor; no commit unless Cos asks

**Branch:** `feat/living-home-hard-cut`  
**Repo:** `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta`

## Status

| Slice | Owner | State |
|-------|-------|-------|
| A+B Nav (overview→home, Task back→home) | [nav hard cut](a10bdb86-b5bc-48f9-9ebb-89fb3a65c510) | **done** → [`HANDOFF-slice-a-b.md`](./HANDOFF-slice-a-b.md) |
| C Home act-now list | [act-now home](46f09559-bc6e-4fb2-ab37-91677c5331a4) | **done** → [`HANDOFF-slice-c.md`](./HANDOFF-slice-c.md) |
| D Many-history on home | [many-history home](51b095ab-3e45-4759-b439-1533514ff7bd) | **done** → [`HANDOFF-slice-d.md`](./HANDOFF-slice-d.md) |
| E Closeout | [closeout](e230680f-253e-4c4d-8cf3-ee76fd608b56) | **done** → [`HANDOFF-slice-e.md`](./HANDOFF-slice-e.md) |

## Subagent return contract

End with `.scratch/living-home-hard-cut/HANDOFF-slice-<id>.md`:

- What changed (paths)
- Tests run + pass/fail
- Follow-ups / blockers
- **Do not commit**

## Pit of success

- A+B share `worklist-nav.ts` → **one agent**, not two parallel.
- C owns `worklist-home.ts` + `LivingShell.svelte`; do not touch commit.go.
- D extracts shared many-history; wait for C’s `submit` + `submissionCount` on act-now task rows.
- Sort for act-now: **actionable first, Can start after** (IMPLEMENTATION default).
