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
| 06 | `issues/06-home-completed-band.md` | — |
| 07 | `issues/07-canstart-first-task-readywhen.md` | — |
| 08 | `issues/08-overview-allow-finished.md` | 06 |

| 09 | `issues/09-seed-replace-clear-passports.md` | — |

| 10 | `issues/10-omit-passport-commit-links.md` | — |
| 11 | `issues/11-clear-session-links-on-nonstay-navigate.md` | — |

**Frontier:** **11** (revert 10 + clear session links on non-stay navigate)

| # | Status |
| --- | --- |
| 01 | done — `cacde9f` + `2cade42` |
| 02 | done — `59abb2b` |
| 03 | done — `41df5a7` |
| 04 | done — `bc5ed25` |
| 05 | done — `b274121` (+ stamp `9080c00`) |
| 06 | done — `84872f5` |
| 07 | done — `73ae2c9` |
| 08 | done — `e26e8af` + `0dd1165` |
| 09 | done — `f341c89` |
| 10 | **reverted** by 11 — Commit.Links was OK |
| 11 | done — see Done notes (implementer); leave Active for coordinator |

**Active (2026-11-04):** **11** — implementer finished (revert 10 + clear session links on non-stay navigate). Awaiting review.

### Investigation note (eligibility)
Passport Can start always-on: no Job `startsWhen`; Domain gate only on select enum + lock readyWhen. Prefer Go fold of first-Task `readyWhen` into `canStart` + Feragame `select.readyWhen` count eligible loads.

## Done notes

| # | Commits (post-rebase) | Review |
| --- | --- | --- |
| 01 | `cacde9f`, `2cade42` | Spec+Standards APPROVE |
| 02 | `59abb2b` | Spec+Standards APPROVE |
| 03 | `41df5a7` | Spec+Standards APPROVE |
| 04 | `bc5ed25` | Spec+Standards APPROVE |
| 05 | `b274121`, `9080c00` | Spec+Standards APPROVE |
| 06 | `84872f5` | Spec+Standards APPROVE |
| 07 | `73ae2c9` | Spec+Standards APPROVE |
| 08 | `e26e8af`, `0dd1165` | Spec+Standards APPROVE |
| 09 | `f341c89` | Spec+Standards APPROVE |
| 10 | `6a2f835` (reverted by 11 / `6666b5f`) | Spec+Standards APPROVE then human override |
| 11 | `6666b5f` + SPA clear commit | implementer done |
| board | `f217a86` ledger stamp | — |

## Human smoke (Compose, after seed refresh)

1. `task seed -- --replace` (wipes Domain demo staging — human OK required)
2. Register-close a load → Worklist **Blocca passaporto** → select → live recap → Invia
3. Expect Commit DPP link on lock form only (not on next TaskForm); overview/Completed recap + public Digital Link/UNTP GET; locked load leaves select enum

## Verify (regression)

```bash
cd server && go test ./internal/living/... -count=1
cd app && bun run test:unit -- --run src/lib/living
```
