# Implementation plan — many-submit overview Submission recap

Depends on [`LOCKED-many-submit-recap.md`](./LOCKED-many-submit-recap.md).
Parent board: [`COORDINATOR.md`](./COORDINATOR.md). Scout:
[`SCOUT-many-submit-recap.md`](./SCOUT-many-submit-recap.md).

**Dumb executor:** no Feragame-named branches in Go/SPA. Use
`WorklistTask.submit` + `submissionCounts` only.

Do not start until explicitly asked. Prefer small slices in this order.
Pre-prod: wipe Mongo LinkStore / Domain as needed after timestamp keys land
(no migration job).

---

## Slice A — LinkStore: address + timestamp ✅

**Goal:** each many-submit Commit keeps its own download links.

1. Extend LinkStore key (memory + Mongo) with `Submission.Timestamp`
   (RFC3339Nano already set at Commit).
2. Commit `Put` uses address + that Commit’s timestamp.
3. Recap `Get` takes timestamp (required for lookups after wipe).
4. Drop any assumption of one doc per address-only key.
5. Tests: two Commits on same many-submit address → two link docs; each
   recoverable by timestamp.

**Done when:** `go test` covering LinkStore + Commit link Put/Get with
timestamp keys green. Local wipe documented in HANDOFF (e.g. `task reset` /
purge LinkStore collection).

**Touch (expected):** `link_store.go`, `link_store_mongo.go`, `commit.go`,
related tests.

---

## Slice B — `submissionCounts` on Worklist ✅

**Goal:** overview knows N without calling recap.

1. While `ProjectWorklist` has `domain.Submissions`, count matches per
   instance leaf address (`jobId`, `jobInstanceId`, `taskId` [, group fields]).
2. Wire `submissionCounts map[string]int` (or equivalent JSON) on
   `WorklistInstance` and nested `WorklistGroupInstance` if still projected.
3. SPA types + any Worklist fixtures/parsers.
4. Tests: 0 / 1 / N appends → correct counts; once Task with one Submission
   still countable if useful (overview may ignore for once chrome).

**Done when:** Load JSON includes counts; living projection tests green.

**Touch (expected):** `worklist.go`, `types.ts` (SPA), projection tests,
maybe Commit response Worklist assert.

---

## Slice C — List recap API (`items`) ✅

**Goal:** one wire for once and many.

1. Change `GET /api/living/task-recap` (or add `task-recaps` and retire single
   response — prefer **one** endpoint) to return
   `{ items: TaskRecap[] }` always.
2. Filter Domain Submissions by address; sort by `timestamp` **ascending**
   (oldest first).
3. For each item, attach LinkStore links via address + that item’s timestamp.
4. Include titles when already resolved today for single recap.
5. Update Go HTTP tests + SPA `task-recap-port` / `http-task-recap-port` +
   unit tests.
6. Migrate `TaskRecapPanel` to consume `items` (once → render `items[0]` or
   empty state).

**Done when:** once done-accordion still works; many path can return N items
with per-row links in tests (SPA chrome for many may still be Slice D).

**Touch (expected):** `task_recap.go`, `http.go`, `task_recap_test.go`,
`task-recap-port.ts`, `http-task-recap-port.ts`, `TaskRecapPanel.svelte`.

---

## Slice D — Overview many-submit accordion ✅

**Goal:** locked chrome.

1. For leaf rows with `submit === 'many'` and `submissionCounts[taskId] > 0`:
   keep primary row as today (actionable link / status).
2. Under the row: accordion closed label = short count text from
   `submissionCounts` (Italian copy consistent with overview).
3. On expand: lazy list-recap request; render each `items[]` as mini-recap
   (`<dl>` + `HttpLinkStrip`) — reuse panel internals or thin list wrapper.
4. No accordion when N = 0 / missing.
5. Once Tasks: leave done-only accordion; they already use Slice C wire.
6. Overview / chrome tests as exist for accordion patterns.

**Done when:** many-submit leaf shows count → expand → oldest-first rows with
facts + links; primary row still opens the form when actionable.

**Touch (expected):** `overview.svelte`, maybe `OverviewRowChrome.svelte`,
`worklist-overview.ts`, overview tests; optional
`TaskRecapList.svelte`.

---

## Slice E — Docs / board closeout ✅

1. ADR-0008 / glossary: **skipped** (Cos: ask before doc churn).
2. HANDOFF + COORDINATOR row → done → [`HANDOFF-ux-slice-e.md`](./HANDOFF-ux-slice-e.md).
3. Glossary skipped — “submission count” remains wire-only.

---

## Suggested PR / commit order

| Slice | Standalone? | Notes |
|-------|-------------|--------|
| A | yes | Wipe after deploy/local |
| B | yes | Can land before SPA uses it |
| C | after A | Needs timestamp Get |
| D | after B + C | UX consumer |
| E | last | |

Rough size: ~8–12 files across A–D (scout estimate + LinkStore redesign in v1).

## Explicit non-goals

- Phase 7 group/`unlockFrom` removal
- Feragame fixture changes (unlikely)
- Committing parent Phases 1–6 unless asked

## Verify (each slice)

- `go test` for touched `internal/living` packages
- SPA: `bun run test:unit` for living/overview/recap as applicable
- Manual: many-submit twice → count 2 → expand → two rows, both link strips;
  actionable title still opens form
