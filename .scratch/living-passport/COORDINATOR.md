# Coordinator — living Passport production

**Branch:** `feat/living-passport`  
**Spec:** [`SPEC.md`](./SPEC.md)  
**ADR:** [`docs/adr/0007-living-passport-production.md`](../../docs/adr/0007-living-passport-production.md)  
**Hard constraint:** Go/SPA are dumb executors of Org Config — no plant-named Passport engine logic.

## Tickets

| # | File | Blocked by |
| --- | --- | --- |
| 01 | `issues/01-prefactor-extract-digital-link-untp-helpers.md` | — |
| 02 | `issues/02-passport-config-store-commit-producer.md` | 01 |
| 03 | `issues/03-public-serve-living-passport.md` | 02 |
| 04 | `issues/04-lock-load-recap-live-snapshot.md` | 02 |
| 05 | `issues/05-feragame-passport-lock-job.md` | 02, 03, 04 |

**Frontier:** 01

| # | Status |
| --- | --- |
| 01 | ready-for-agent |
| 02 | ready-for-agent |
| 03 | ready-for-agent |
| 04 | ready-for-agent |
| 05 | ready-for-agent |

## Verify (when board advances)

```bash
cd server && go test ./internal/living/... -count=1
cd app && bun run test:unit -- --run src/lib/living
```

After Feragame fixture changes: `task seed -- --replace` when Active Config must refresh.
