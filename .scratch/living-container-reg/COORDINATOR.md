# Coordinator — containerReg group + incomplete continuity

**Branch:** `feat/living-container-reg`  
**Spec:** [`SPEC.md`](./SPEC.md)  
**Hard constraint:** Go/SPA are dumb executors of Org Config — no plant-named engine logic.

## Tickets

| # | File | Blocked by |
| --- | --- | --- |
| 01 | `issues/01-prefactor-config-initialdata-from-domain.md` | — |
| 02 | `issues/02-plant-containerreg-group-unlockfrom.md` | — |
| 03 | `issues/03-plant-fullyprocessed-load-filter.md` | 02 |
| 04 | `issues/04-plant-incomplete-freeze.md` | 02 |
| 05 | `issues/05-plant-continuation-delta-drop-dualsources.md` | 01, 03, 04 |

**Frontier:** none — board complete. Residual: Active Config needs `task seed -- --replace` when refreshing plant YAML.

| # | Status |
| --- | --- |
| 01 | done |
| 02 | done |
| 03 | done |
| 04 | done |
| 05 | done |

## Verify (board)

```bash
cd server && go test ./internal/living/... -count=1
cd app && bun run test:unit -- --run src/lib/living
```

Active Config refresh after plant YAML: `task seed -- --replace` (wipes Domain demo staging).
