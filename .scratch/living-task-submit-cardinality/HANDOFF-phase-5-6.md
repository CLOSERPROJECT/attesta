# HANDOFF — Phases 5 + 6 Done

Phases 5 (SPA / Worklist chrome) and 6 (passport + leftovers) complete. No commit.

## Checklist

| Item | Status |
|------|--------|
| Ship `submit` (`once`\|`many`) on `WorklistTask` wire | Done |
| SPA `instanceTaskProgress` / In progress / Completed ignore `many` | Done |
| Overview `finished` ignores `many`; group Spawn/Close gated on Worklist `groups` | Done |
| Stay + Continue remain server-driven (no SPA invent) | Confirmed |
| Passport eligibility: lot `status = fullyProcessed` ∧ not locked | Confirmed (fixture) |
| Grep stragglers (`load`, boolean `fullyProcessed`, plant `kind: group` / `unlockFrom`) | Clean (engine + synthetic grouporg left alone) |
| `HANDOFF-phase-4-wip.md` | One-line superseded pointer |

## Paths changed

### Go
- `server/internal/living/worklist.go` — `WorklistTask.Submit`; `projectWorklistTasks` sets `taskSubmit`
- `server/internal/living/worklist_test.go` — assert intake `register=many`, `truckArrive`/`weightCheck=once`

### SPA
- `app/src/lib/living/types.ts` — `TaskSubmit`; optional `WorklistTask.submit`
- `app/src/lib/living/worklist-home.ts` — `taskCountsTowardCompletion`; progress counts once only
- `app/src/lib/living/worklist-home.test.ts` — many-ignore + Available still shows flat many
- `app/src/lib/living/worklist-overview.ts` — `finished` over once rows only
- `app/src/lib/living/worklist-overview.test.ts` — finished + no group chrome without `groups`

### Scratch
- `.scratch/.../HANDOFF-phase-4-wip.md` → superseded by `HANDOFF-phase-4.md`

## Phase 5 notes

- **Home Available:** provisional Spawn rows only when `inst.groups?.[taskId]` exists; Feragame flat Jobs never invent group `new` rows.
- **Stay + Continue:** Commit wire already carries `navigation: stay` + `continueNavigation` for many-submit Tasks in Config; SPA (`worklist-nav` / session) only follows server values.
- **i18n:** Config Italian titles unchanged; no copy pass.

## Phase 6 grep

- Passport Job: `status = "fullyProcessed" and $not(locked = true)` (select + lock readyWhen).
- No Domain kind `load`, no boolean `fullyProcessed: true`, no plant `kind: group` / `unlockFrom` in `feragame-config.yaml`.
- Remaining `kind: group` / `unlockFrom` hits are engine support + synthetic tests (`task_groups_test`, `unlock_test`, grouporg) — left alone (Phase 7).

## Verify

```text
go test ./internal/living/ -count=1          # ok
go test ./cmd/server/ -run Living -count=1   # ok
bun run test:unit -- --run src/lib/living    # 29 files / 173 tests ok (from app/)
```

Manual `/app` smoke: **skipped**.

## Follow-ups

- Phase 7 (explicit ask): drop Task-group engine + SPA chrome; drop unused `unlockFrom` engine support.
- Optional: Cos update `COORDINATOR.md` Phase 5–6 → done.
