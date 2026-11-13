# HANDOFF — UX Slice D (many-submit overview accordion)

## Done

Locked chrome: primary leaf row unchanged; under it, accordion only when
`submit === 'many'` **and** `submissionCount > 0`. Closed = Load count copy
(`N invio` / `N invii`). Open = lazy list-recap; **all** `items[]` as
`<dl>` + `HttpLinkStrip`. Once Tasks: done-only accordion (Slice C wire).

## Paths changed

- `app/src/lib/living/worklist-overview.ts` — `submit` / `submissionCount` on
  leaf + group-child rows; `manySubmitHistoryVisible`, `formatSubmissionCount`
- `app/src/lib/living/worklist-overview.test.ts` — count projection + gate
- `app/src/lib/living/TaskRecapFacts.svelte` — shared mini-recap facts + links
- `app/src/lib/living/TaskRecapList.svelte` — lazy fetch; render all `items[]`
- `app/src/lib/living/TaskRecapPanel.svelte` — once path uses `TaskRecapFacts`
- `app/src/lib/router/overview.svelte` — nested many-history accordion under
  non-done primary row (`overview-many-history`)
- `app/messages/it.json`, `app/messages/en.json` — `submission_count_one` /
  `submission_count_many` (recompile paraglide locally)

## Tests

```text
bun run test:unit -- --run \
  src/lib/living/worklist-overview.test.ts \
  src/lib/living/http-task-recap-port.test.ts \
  src/lib/living/worklist.test.ts
# 25 pass
```

## Manual verify

1. Many-submit twice → overview shows actionable primary link + closed
   `2 invii` (no recap network until expand).
2. Expand → two mini-recaps oldest-first; each has facts + link strip when
   LinkStore has per-timestamp URLs (Slice A).
3. Primary title/chevron still opens the Task form.
4. Once done Task → only existing done accordion (no many-history strip).
5. Many with N = 0 → no accordion under the row.

## Slice E follow-ups

- Optional ADR-0008 / overview integrator note for `submissionCounts` + list
  `items` wire (ask before doc churn).
- COORDINATOR / board row → done; Glossary skip unless a term sticks.
- No commit/push unless Cos asks.
