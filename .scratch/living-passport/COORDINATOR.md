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
| 01 | done — `a83f91a` + `d1f3b25` |
| 02 | done — `be99ffc` |
| 03 | done — `1ac683b` |
| 04 | done — `e863138` |
| 05 | done — `42c8701` (+ stamp `810792a`) |

**Active (2026-11-04):** tickets **01–05** APPROVE. **Rebase eval:** use `--onto` (not merge / not plain rebase). Awaiting human OK to run.

```bash
# after committing/stashing dirty .scratch/living-passport/*
git rebase --onto origin/feat/attesta-rework 705bdab feat/living-passport
```

Then: Compose smoke (`task seed -- --replace`); later squash-merge PR into rework. Do not push without ask.

## Done notes

| # | Commits | Review |
| --- | --- | --- |
| 01 | `a83f91a`, `d1f3b25` | Spec+Standards APPROVE |
| 02 | `be99ffc` | Spec+Standards APPROVE (soft: snapshot eval after Domain Put) |
| 03 | `1ac683b` | Spec+Standards APPROVE |
| 04 | `e863138` | Spec+Standards APPROVE |
| 05 | `42c8701`, `810792a` | Spec+Standards APPROVE |

## Human smoke (Compose, after seed refresh)

1. `task seed -- --replace` (wipes Domain demo staging — human OK required)
2. Register-close a load → Worklist **Blocca passaporto** → select → live recap → Invia
3. Expect Commit DPP links + public Digital Link/UNTP GET; locked load leaves select enum

## Verify (regression)

```bash
cd server && go test ./internal/living/... -count=1
cd app && bun run test:unit -- --run src/lib/living
```
