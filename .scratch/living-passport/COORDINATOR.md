# Coordinator — living Passport production

**Branch:** `feat/living-passport`  
**Cursor worktree:** `~/.cursor/worktrees/living-passport-a3f7c2b1/attesta-714729698316`  
**Spec:** [`SPEC.md`](./SPEC.md)  
**ADR:** [`docs/adr/0007-living-passport-production.md`](../../docs/adr/0007-living-passport-production.md)  
**Coordinator handoff:** [`HANDOFF-coordinator.md`](./HANDOFF-coordinator.md)  
**Pipeline:** ticket → implementer → code review (Standards+Spec) → fixer → …; human smoke / rebase after container-reg squash.  
**Hard constraint:** Go/SPA are dumb executors of Org Config — no plant-named Passport engine logic.

## Tickets

| # | File | Blocked by |
| --- | --- | --- |
| 01 | `issues/01-prefactor-extract-digital-link-untp-helpers.md` | — |
| 02 | `issues/02-passport-config-store-commit-producer.md` | 01 |
| 03 | `issues/03-public-serve-living-passport.md` | 02 |
| 04 | `issues/04-lock-load-recap-live-snapshot.md` | 02 |
| 05 | `issues/05-feragame-passport-lock-job.md` | 02, 03, 04 |

**Frontier:** board complete — awaiting human

| # | Status |
| --- | --- |
| 01 | done — `cacde9f` + `2cade42` |
| 02 | done — `59abb2b` |
| 03 | done — `41df5a7` |
| 04 | done — `bc5ed25` |
| 05 | done — `b274121` (+ stamp `9080c00`) |

**Active (2026-11-04):** rebased `--onto origin/feat/attesta-rework` (cutoff `705bdab`) — clean, 10 commits. Tip `f217a86` on `0e69179`. Next: Compose smoke (`task seed -- --replace`); later squash-merge PR into rework. Do not push without ask.

## Done notes

| # | Commits (post-rebase) | Review |
| --- | --- | --- |
| 01 | `cacde9f`, `2cade42` | Spec+Standards APPROVE |
| 02 | `59abb2b` | Spec+Standards APPROVE |
| 03 | `41df5a7` | Spec+Standards APPROVE |
| 04 | `bc5ed25` | Spec+Standards APPROVE |
| 05 | `b274121`, `9080c00` | Spec+Standards APPROVE |
| board | `f217a86` ledger stamp | — |

## Human smoke (Compose, after seed refresh)

1. `task seed -- --replace` (wipes Domain demo staging — human OK required)
2. Register-close a load → Worklist **Blocca passaporto** → select → live recap → Invia
3. Expect Commit DPP links + public Digital Link/UNTP GET; locked load leaves select enum

## Verify (regression)

```bash
cd server && go test ./internal/living/... -count=1
cd app && bun run test:unit -- --run src/lib/living
```
