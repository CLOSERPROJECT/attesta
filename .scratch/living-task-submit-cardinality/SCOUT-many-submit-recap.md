# Scout — many-submit overview Submission recap

Research-only (2026-11-13). No code changes. Context: Phase 5+6 shipped
`WorklistTask.submit`; many-submit never Worklist `done` ([LOCKED.md](./LOCKED.md)).

---

## Verdict

**Today:** overview shows **one** Submission per Task path, and **only** when
status is `done` (accordion body → lazy `TaskRecapPanel`).

**For `submit: many`:** that chrome never opens (status stays
`actionable`/`unavailable`). There is **no** list-many recap API; Domain appends
many Submissions under the **same** address; `task-recap` + LinkStore are
**single-record / first-or-overwrite**.

**Reuse:** the closest “one row per past submit + download links” UI is **Task
group occurrence list** under the group accordion (each occurrence = unique
address → one `TaskRecap`). Not a separate summary component — same
`TaskRecapPanel` / `HttpLinkStrip`. Faster UX pattern than inventing new chrome;
API must still grow to list many under one leaf path.

**Recommend:** **C hybrid** — extend recap to return **N** rows (facts + links),
render like occurrence list under the **leaf** Task card; hang history on
**non-`done`** many-submit rows (not only done accordion).

---

## 1. Overview Task card / “done” body (SPA)

| Piece | Path | Role |
|-------|------|------|
| Route | `app/src/lib/router/overview.svelte` | Accordion; `taskStatusRow` snippet |
| Model | `app/src/lib/living/worklist-overview.ts` | `jobInstanceOverview` rows |
| Recap UI | `app/src/lib/living/TaskRecapPanel.svelte` | Lazy fetch; `<dl>` facts + `HttpLinkStrip` |
| Port | `app/src/lib/living/http-task-recap-port.ts` → `task-recap-port.ts` | GET helper |
| Links | `app/src/lib/living/form/HttpLinkStrip.svelte` | Post-Commit download/open |
| Fact format | `formatFactValue` in `worklist-overview.ts` | Already JSON.stringifies objects |
| Chrome | `OverviewRowChrome.svelte` | Title + status tag; chevron |

**Done path:** `status === 'done'` → Accordion Item → expand →
`<TaskRecapPanel active={doneExpanded} request={recap} />`.

**Non-done:** `actionable` → plain `<a>` to Task form (no body). Else status-only
`<div>` (no expand).

**Group path:** group accordion → `groupInstances` → each occurrence (or nested
child) uses the **same** `taskStatusRow` + `TaskRecapPanel` when that child is
`done`, with `groupId` + `groupInstanceId` on the recap request.

---

## 2. Current assumption: one Submission per path

**Yes — single.**

- SPA: one `TaskRecapRequest` / one panel per done row.
- Go `findSubmission` (`task_recap.go`): first Domain Submission matching
  `(jobId, jobInstanceId, taskId[, groupId, groupInstanceId])`.
- LinkStore: one key per address; `Put` **replaces** (memory + Mongo upsert).

Group “many rows” worked because **each occurrence minted a distinct
`groupInstanceId`**, so each Submission had a unique address — still one recap
per address, N addresses under the group card.

Flat `submit: many` reuses the **same** leaf address → N Submissions, 1 key.

---

## 3. Prior “Task-group occurrence summary”

**Not a dedicated summary component.** It is overview group chrome:

- Issues: `.scratch/living-groups-rework/issues/09-overview-group-instances.md`,
  `17-overview-accordion-chrome.md`, `16-lazy-task-recap.md`,
  `18-single-child-group-instance-flatten.md`
- Render: `overview.svelte` group branch + `projectGroupInstances` in
  `worklist-overview.ts`
- Data shape: Worklist `inst.groups[groupId].instances[]` (`id`, `label`,
  child `tasks` statuses) — **not** Submission payloads
- Per-occurrence facts/links: lazy **`GET /api/living/task-recap`** when child
  `done` (same `TaskRecap` wire)

**Historical plant mapping:** intake `lots` group → each Spawn occurrence ≈ one
`register` Submission + labels/CSV links via LinkStore. Feragame now flat
`register` `submit: many` (Phase 3+); group list is the UX ancestor for “one
line per past submit,” not a reusable group-only API.

---

## 4. Server APIs

| Endpoint | Payload | Many under same path? |
|----------|---------|------------------------|
| `GET /api/living/task-recap` | Single `TaskRecap`: address, timestamp, userId, facts, titles?, links? | **No** — first match |
| `GET /api/living/domain` (Load) | Worklist (+ chrome); **no** Submission history for overview | N/A |
| `GET /api/living/task-form` | Form bundle + Domain slice for that mount | Not overview recap |
| Commit response | Updated Worklist + stay links for **that** Commit | Ephemeral; not history |

**Handler:** `server/internal/living/http.go` → `LoadTaskRecap` in
`task_recap.go`. Comment says “not-done → ErrNotFound”; implementation is
“no Submission at address” (no Worklist `done` check). So after a many-submit
Commit, recap **would** return the **first** Submission if called — but SPA
never mounts the panel unless status is `done`.

**Domain:** append-only `[]Submission` (`domain.go` / `mutate.go`). Queryable
in Go by filtering the slice; **no** public list-recap endpoint yet.

**Links:** `LinkStore` (`link_store.go`, `link_store_mongo.go`) keyed like
Submission address. Second many-submit **overwrites** links for that path —
older download URLs are **not** recoverable from LinkStore today.

---

## 5. Wire already available (`submit: many`)

**Yes (Phase 5).**

- Go: `WorklistTask.Submit` `"once"|"many"` — `worklist.go`
- SPA: `WorklistTask.submit?: TaskSubmit` — `types.ts`
- Overview completion already filters many out of `finished`
  (`worklist-overview.ts`); home progress same (`worklist-home.ts`)
- **Overview UI does not yet branch on `submit`** for card chrome / history

Usable as the flag: `taskChrome(job, taskId)?.submit === 'many'`.

---

## 6. Product nuance — where history chrome hangs

Many-submit may stay `actionable` / `unavailable` forever → **never** enter the
current `done` accordion body.

| Hang point | Fit |
|------------|-----|
| Done accordion only | **Wrong** for many |
| Actionable `<a>` only | No place for history |
| **Leaf Task accordion while non-done** (expand = past Submissions; form via link/button when actionable) | **Preferred** — mirrors group card: expand for list, separate CTA for next work |
| Always-visible strip under row | OK if expand feels heavy; still needs `submit === 'many'` |

Conflict today: `OverviewRowChrome` uses ChevronRight for `actionable` (nav) vs
ChevronDown for expand. Many-submit actionable that also expands needs a
**split affordance** (e.g. row expands for history + explicit “Apri” / title
link to form), similar to group body + Spawn.

---

## 7. Options

### A) Stringify each Submission facts

- New/list API: all Submissions for path (+ optional titles)
- SPA: one line/row; `formatFactValue` / `JSON.stringify` OK short-term
- **Links:** broken for prior Commits unless LinkStore key gains submission id /
  timestamp (or links inlined into recap rows at Commit time)

### B) Reuse group occurrence + task-recap as-is

- Fast visually only if plant still had groups; Feragame flattened; Phase 7 drops
  group chrome. Does **not** solve flat many address collision. **Reject** as
  sole path.

### C) Hybrid (recommended)

1. **API:** extend `task-recap` (or `task-recaps`) to return
   `submissions: TaskRecap[]` (or `{ items: [...] }`) for the address filter;
   keep single-object response for once/compat **or** always array of 0..N.
2. **Links:** short-term attach LinkStore get to **latest** row only; track
   follow-up to key links per Submission (timestamp / index) so register labels
   survive multi-submit.
3. **SPA:** if `submit === 'many'`, leaf row is expandable (any status with
   history, or always when many); body = list of mini-recaps (reuse
   `TaskRecapPanel` internals or thin loop: timestamp + stringify facts +
   `HttpLinkStrip`). Keep once Tasks on current done-only accordion.
4. Mirror group occurrence list density (one row per past Commit).

---

## 8. Minimal touch estimate (C)

| Area | Files |
|------|--------|
| Go list + HTTP | `task_recap.go`, `http.go`, `task_recap_test.go` (+ optional LinkStore key later: `link_store*.go`, `commit.go`) |
| SPA port/types | `task-recap-port.ts`, `http-task-recap-port.ts` (+ tests) |
| Panel / list UI | `TaskRecapPanel.svelte` or new `TaskRecapList.svelte`; `HttpLinkStrip` reuse |
| Overview chrome | `overview.svelte`, maybe `OverviewRowChrome.svelte`, `worklist-overview.ts` (+ tests) |

Rough: **~6–10 files** for facts list + many-row expand without LinkStore redesign;
**+2–3** if per-Submission links must be correct in v1.

---

## Pointers

- Handoff wire: [HANDOFF-phase-5-6.md](./HANDOFF-phase-5-6.md)
- Product lock: [LOCKED.md](./LOCKED.md) §4 many never `done`
- ADR: `docs/adr/0008-living-task-submit-cardinality.md`
- Group accordion handoff: `.scratch/living-groups-rework/HANDOFF-overview-accordion-2026-11-03.md`
