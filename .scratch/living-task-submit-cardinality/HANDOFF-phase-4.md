# HANDOFF — Phase 4 Done

Phase 4 complete. No commit (Cos did not ask).

## Checklist (IMPLEMENTATION Phase 4)

| Item | Status |
|------|--------|
| `startLot` / `closeLot` (`submit: many`) + readyWhen | Done (prior + this) |
| Flatten line groups → leaf many-submit peers | Done (fixture prior) |
| Park/resume via closeTurno / openTurno transforms | Done (prior) |
| Exclusivity one `open`\|`parked` lot | Done (prior) |
| No Feragame line `unlockFrom` / `kind: group` | Done (prior; verified) |
| `containerReg` does not write `fullyProcessed` | Done (prior) |
| Seed / demo / plant flow followers | Done (prior + this) |
| plant-config verify green | Done (this) |

## Verify (this agent)

```text
go test ./internal/living/ -count=1          # ok
go test ./cmd/server/ -run Living -count=1   # ok
bun run test:unit -- --run src/lib/living    # 29 files / 171 tests ok (from app/)
```

## This agent’s follower sweep

Plant YAML / unlock many-skip **not** re-authored. Fixed tests still on group paths / old ids:

### Go (`server/internal/living/`)
- `commit_test`: weightCheck unlocks after arrive (many-skip); Worklist `unavailable` until ≥1 lot
- `container_reg_schema_test`: leaf `containerReg` (no group/`register` child)
- `cover_gate_test`: CloseGroup on leaf → unknown-target; boom Get via probe org with a real group
- `coverage_paths_test`: flat `line/weigh` roles
- `line_starts_when_test`: task id list = openTurno…closeTurno flattened
- `form_bundle_test` / `task_form_http_test` / `worklist_test`: flat `line/weigh`, `markProgress`, `containerReg`, `startLot`; no CloseGroup/Spawn chrome on line peers

### SPA (`app/src/lib/living/`)
- `line-smoke.test.ts`: startLot → markProgress → containerReg leaf path; no palletWeight/lineLoad
- QR apply / scan / http-qr fixtures: flat navigate to `line/weigh` (group path retained only as nested-coords regression)
- `form/ajv-form-validator.test.ts`: leaf containerReg schema/uiSchema

## Engine note (prior, kept)

`unlock.go` default predecessor skips `submit: many` leaves so once bookends unlock peer many Tasks; Domain `readyWhen` remains the gate. Do not remove without a synthetic test proving it wrong.

## Next

Phase 5 — SPA / overview chrome (see `IMPLEMENTATION.md`). Optional: delete or archive `HANDOFF-phase-4-wip.md`.
