# HANDOFF — Phase 3 (Feragame intake flatten)

## Paths changed

### Fixture / plant law
- `server/internal/living/fixtures/feragame-config.yaml` — intake drops `lots` **Task group**; leaf **`register`** (`submit: many`, `onCommit.navigation: stay`, lot mint / labels / CSV HTTP preserved); `readyWhen` blocks register after `weightCheck` Submission; **`weightCheck`** once with `readyWhen` ≥1 lot for truck (no plant `unlockFrom`)
- `docs/agents/plant-config.md` — stay example points at intake `register`

### Seed / demo
- `server/internal/living/seed.go` — `intake/register` Commits (no Mint spawn / no CloseGroup)
- `server/internal/living/demo_commits.go` — same for `commitIntakeThroughLotsClosedCtx`

### Living Go tests + cmd
- `server/internal/living/plant_flow_test.go` — helpers + plant path lock (many register → weightCheck → register unavailable)
- `server/internal/living/fixture_test.go` — intake task order `truckArrive, register, weightCheck`
- Followers: `worklist_test`, `seed_test`, `commit_*`, `cerbos_commit`, `form_bundle`, `task_form_http`, `task_recap`, `link_store`, `sidecar_http`, `cover_gate`, `line_remaining`, `coverage_paths`, …
- `server/cmd/server/living_test.go` — flat `intake/register` Commit body + worklist asserts
- Synthetic **grouporg** / path-parser tests that still use id `lots` left alone (engine groups remain for line)

### SPA
- `app/src/lib/living/line-smoke.test.ts` — intake taskIds / Submission use `register`
- Other SPA tests that mock generic group chrome with synthetic `lots` unchanged (Phase 5 / line still uses groups)

## Tests

```text
go test ./internal/living/ -count=1
→ ok

go test ./cmd/server/ -run Living -count=1
→ ok

bun run test:unit -- --run src/lib/living   # from app/
→ 29 files / 171 tests ok
```

## Follow-ups / blockers for Phase 4

- Line still has `palletWeight` / `lineLoad` / `containerReg` groups + plant `unlockFrom`
- No `startLot` / `closeLot` / park-resume yet; `containerReg` still writes lot `status: fullyProcessed`
- After register, Unlock + readyWhen make `weightCheck` actionable while register stays many-actionable until weightCheck Submits (Continue from stay → weightCheck)
- Local stacks need `task seed:feragame-config` (or full) after this YAML shape change
- Do **not** commit unless Cos asks
