# HANDOFF — Phase 2 (`load` → `lot` + status enum)

## Paths changed

### Fixture / Domain shape
- `server/internal/living/fixtures/feragame-config.yaml` — entity **`load` → `lot`**; field **`fullyProcessed` (bool) → `status`** (`open` \| `parked` \| `fullyProcessed`); pallet **`loadId` → `lotId`**; container **`sourceLoadId` / `secondarySourceLoadId` → `sourceLotId` / `secondarySourceLotId`**; passport subject/select/`instanceId`; reads key `loads` → `lots`; containerReg still sets `status: "fullyProcessed"` (terminal until Phase 4 `closeLot`)
- `server/internal/living/fixtures/parked-loaded-extracted.yaml` — same Domain vocabulary in dataReport exprs

### Seed / demo
- `server/internal/living/seed.go` — `Kind("lot")`, `sourceLotId` Commit values; seed ids (`SeedLoadCOID` / `SeedLoadDAID`) unchanged
- `server/internal/living/demo_commits.go` — minted kind `lot`

### Living Go tests + cmd followers
- `server/internal/living/*_test.go` (fixture, seed, line_remaining, passport, worklist, data_report, coverage, form_bundle, …)
- `server/cmd/server/living_test.go`, `server/cmd/server/seed_test.go`
- `server/cmd/seed-loaded-extracted/main.go`
- `services/living-http/artifacts.go`, `artifacts_test.go`, `handlers_test.go` (field renames where present)

### SPA (`$living-fixtures` = same YAML)
- `app/src/lib/living/line-smoke.test.ts`
- `app/src/lib/living/form/ajv-form-validator.test.ts`
- `app/src/lib/living/form/task-form-bundle.test.ts`
- `app/src/lib/living/form/deep-local-draft-sync.svelte.ts`
- `app/src/lib/living/form/domain-snapshot.ts`
- `app/src/lib/living/worklist-overview.ts`, `worklist-overview.test.ts`

## Tests

```text
go test ./internal/living/ -count=1
→ ok

go test ./cmd/server/ -run Living -count=1
→ ok

bun run test:unit -- --run src/lib/living   # from app/
→ 29 files / 171 tests ok
```

## Follow-ups / blockers for Phase 3

- Task groups still present (intake `lots`, line `palletWeight` / `lineLoad` / `containerReg`); Phase 3 flattens intake.
- Fixture still omits `submit: many` (default once) until Phases 3–4 author cardinality on leaves.
- No `startLot` / `closeLot` / park-resume writers yet: register → `status: fullyProcessed` remains the terminal writer (Phase 4 moves that to `closeLot`; park/resume on shift bookends).
- Seed constant names `SeedLoadCOID` / `SeedLoadDAID` still say “Load”; optional rename to `SeedLot*` later.
- Local stacks with old Domain docs need `task seed:feragame-config` or full seed after this YAML shape change.
- Do **not** commit unless Cos asks.
