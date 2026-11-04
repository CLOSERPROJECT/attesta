# Coordinator — lineLoad readyWhen + Unlock fan-out

**Branch:** `feat/attesta-rework` (tip includes `89eb056` one-pallet weigh + QR)  
**Frozen design:** [`LOCKED.md`](./LOCKED.md)  
**Pointers:** [`NOTES-for-implementers.md`](./NOTES-for-implementers.md)  
**Do not** push; prefer leave board files uncommitted until human asks.

## Tickets

| # | File | Blocked by |
| --- | --- | --- |
| 01 | `issues/01-worklist-readywhen-unavailable.md` | — |
| 02 | `issues/02-unlock-from-predecessor.md` | prefer after 01 (parallel OK if careful) |
| 03 | `issues/03-plant-lineload-readywhen-unlockfrom.md` | 01, 02 |

**Frontier:** board complete — human smoke (Active Config may need `task seed -- --replace`)

| # | Status |
| --- | --- |
| 01 | APPROVE (`13b1fbd`) — Worklist `unavailable` fold + ADR-0005 + SPA label/tag + synthetic tests |
| 02 | APPROVE (`87dad7c`) — `unlockFrom` Unlock predecessor + validate + fan-out / linear Close regression |
| 03 | APPROVE (`7bd5a0f`) — plant `lineLoad.unlockFrom` + locked `readyWhen`; plant_flow / pallet_weight / seed / fixture expectations |

### Settled hard calls

- Concurrent path unlock after `openTurno`; `lineLoad` gated by Domain `readyWhen`, not weigh Close / first child Submission.
- Unlock opt-in key name: **`unlockFrom`** (Task id predecessor). Default remains linear previous sibling.
- Worklist status `unavailable` ≠ role-handoff `available`. Home/mount stay actionable-only.
- No Feragame-named Go/TS; no leftovers/completeness revival.

### Next (human)

1. Smoke plant: openTurno → `lineLoad` unavailable until weighed leftover / this-shift weigh; Segna avanzamento actionable with weigh group still open; `containerReg` after `lineLoad` Submission.
2. If running stack still has pre-03 Active Config: `task seed -- --replace` (refreshes Config; wipes Domain demo staging — expected for Config pick-up).

## Verify

```bash
cd server && go test ./internal/living/... -count=1
cd app && bun run test:unit -- --run src/lib/living
```
