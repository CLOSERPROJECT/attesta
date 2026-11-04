# 03: Plant feragame YAML + integration tests

**What to build:** Author plant `line` Config so `lineLoad` unlocks from `openTurno` and is Worklist/`Commit`-gated by Domain `readyWhen`. Update plant_flow / pallet_weight / line_load / seed expectations. Prove leftover-with-open-weigh ⇒ actionable; no weighed leftovers ⇒ unavailable; `containerReg` still linear after `lineLoad`.

**Blocked by:** 01 (Worklist readyWhen → unavailable), 02 (Unlock `unlockFrom`).

**Status:** implemented

## Build exactly

1. In `server/internal/living/fixtures/feragame-config.yaml` on `lineLoad`, author exactly (per [`LOCKED.md`](../LOCKED.md)):

   ```yaml
   unlockFrom: openTurno
   readyWhen: >-
     $count($domain.pallet.*[$exists(weight) and $not(progress = 'done')]) >= 1
   ```

   Leave `palletWeight` / `containerReg` on default linear Unlock. Do not add plant-named Go.

2. Rewrite expectations that today assume “Close `palletWeight` unlocks `lineLoad`”:
   - After `openTurno` only (no weighed pallets): `lineLoad` is **unlocked path** but Worklist **`unavailable`** (readyWhen false); `palletWeight` spawnable as today.
   - After ≥1 pallet has `weight` and not `progress: done` (prior leftover **or** this-shift weigh), even while `palletWeight` group still open: `lineLoad` **`actionable`** for line role — Close weigh is **not** required.
   - Closing weigh does not by itself open Segna avanzamento when readyWhen is already true/false independently.
   - One `lineLoad` Submission ⇒ `done`; Domain reads do not reopen.
   - `containerReg` stays locked until `lineLoad` Submission; then actionable for logistic (regression).

3. Touch tests / seed as needed: `plant_flow_test.go`, `pallet_weight_test.go`, `line_remaining_test.go`, `worklist_test.go`, `seed_test.go` / demo seed helpers — only expectation + fixture updates, still generic engine.

4. Assert elsewhere: no new Go branch on `"lineLoad"` / `"feragame"`.

## Verify

```bash
cd server && go test ./internal/living/ -count=1 -run 'Plant|PalletWeight|Line|Seed|Worklist'
cd server && go test ./internal/living/... -count=1
cd app && bun run test:unit -- --run src/lib/living
```

## When done

- [x] Feragame `lineLoad` has `unlockFrom: openTurno` + locked `readyWhen` expr
- [x] Leftover / weighed pallet + open weigh group ⇒ `lineLoad` actionable
- [x] openTurno only, no eligible pallets ⇒ `lineLoad` unavailable
- [x] Close weigh does not special-case open Segna avanzamento
- [x] `containerReg` after `lineLoad` Submission still works
- [x] Seed / plant_flow / related tests green; no plant id special-cases in Go
