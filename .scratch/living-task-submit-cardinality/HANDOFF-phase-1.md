# HANDOFF — Phase 1 (engine `submit: once | many`)

## Paths changed

- `server/internal/living/tasks.go` — parse/validate `submit: once|many` (default once); reject on groups
- `server/internal/living/unlock.go` — `Done` only for once; `sequenceOk` = Unlocked ∧ ¬Done (many repeats; once second Commit denied)
- `server/internal/living/worklist.go` — comments + `worklistInstanceHasRemainingWork(cfg, …)` skips `submit: many`
- `server/internal/living/commit.go` — call site for remaining-work signature
- `server/internal/living/submit_cardinality_test.go` — synthetic mini-config tests (no plant Task ids)

## Tests

```text
go test ./internal/living/ -run 'Submit|ValidateTaskSubmit|RemainingWorkIgnores' -count=1
→ ok

go test ./internal/living/ -count=1
→ ok
```

## Follow-ups / blockers for Phase 2

- SPA In progress / Completed (`app/src/lib/living/worklist-home.ts` `instanceTaskProgress`) still counts all `taskIds`; until Phase 5 (or a wire `submit` on WorklistTask), a Job with only-once-done + actionable many would stay In progress in the SPA even though Go remaining-work treats it finished.
- Phase 2 (`load` → `lot` + status enum) unblocked for fixture/seed work; no dependency on further engine cardinality changes.
- Feragame fixture still omits `submit` (default once) until Phases 3–4 author `many`.
