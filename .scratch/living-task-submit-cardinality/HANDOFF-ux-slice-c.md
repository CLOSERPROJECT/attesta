# HANDOFF — UX Slice C (list recap `items`)

## Done

`GET /api/living/task-recap` always returns `{ items: TaskRecap[] }` (same
endpoint). Matches filter by Submission address; sort `timestamp` ascending;
each item `LinkStore.Get(org, addr, sub.Timestamp)` + shared schema titles.
Empty match → still `404`. SPA port/panel consume `items` (once → `items[0]`).

## Paths changed

- `server/internal/living/task_recap.go` — `TaskRecapList`; list Load
- `server/internal/living/http.go` — encode list
- `server/internal/living/task_recap_test.go`
- `app/src/lib/living/task-recap-port.ts` — `TaskRecapList`
- `app/src/lib/living/http-task-recap-port.ts` (+ `.test.ts`)
- `app/src/lib/living/TaskRecapPanel.svelte` — `items[0]` or empty

## Tests

```text
go test ./internal/living/ -count=1
# includes TestLoadTaskRecapManySubmitOldestFirstWithPerRowLinks

bun run test:unit -- src/lib/living/http-task-recap-port.test.ts
```

**Pass.**

## Slice D follow-ups

- Many-submit chrome: `submit === 'many'` + `submissionCounts[taskId] > 0`
  → accordion under leaf; closed = count copy; open = lazy this endpoint.
- Render **all** `items[]` (not only `[0]`) as mini-recap `<dl>` +
  `HttpLinkStrip`; optional `TaskRecapList.svelte` wrapper.
- Once Tasks: keep done accordion; already on Slice C wire.
- Avoid touching Worklist `submissionCounts` projection if Slice B still
  landing — consume the field once present.
