# HANDOFF — UX Slice B (`submissionCounts`)

## Paths changed

- `server/internal/living/worklist.go` — `SubmissionCounts` on `WorklistInstance` + `WorklistGroupInstance`; count while `ProjectWorklist` has `domain.Submissions` (flat vs nested address)
- `server/internal/living/submit_cardinality_test.go` — 0 / 1 / N flat + group child counts
- `app/src/lib/living/types.ts` — mirror optional `submissionCounts`
- `app/src/lib/living/http-commit-port.ts` — `parseSubmissionCounts` + normalize on Commit worklist parse
- `app/src/lib/living/worklist.test.ts`, `http-commit-port.test.ts` — fixture / parser coverage

## Tests

| Command | Result |
|---------|--------|
| `go test ./internal/living/ -run 'TestProjectWorklistSubmissionCounts\|TestProjectWorklistGroupSubmissionCounts\|…'` | **pass** (when package compiles) |
| `bun run test:unit -- src/lib/living/worklist.test.ts src/lib/living/http-commit-port.test.ts` | **pass** (14) |

**Note:** Parallel Slice C mid-flight (`LoadTaskRecap` → `TaskRecapList` in `task_recap.go`, tests not yet updated) can make `./internal/living` fail to compile. Not Slice B. Re-run Slice B `-run` after Slice C lands.

## Slice D follow-ups

- Gate accordion on `taskChrome?.submit === 'many'` **and** `(inst.submissionCounts?.[taskId] ?? 0) > 0`
- Closed label: short count from `submissionCounts` (no recap fetch)
- Expand → Slice C list-recap; keep primary row actionable → form
- Once Tasks: ignore counts for chrome (done accordion only)
- No Feragame-named branches
