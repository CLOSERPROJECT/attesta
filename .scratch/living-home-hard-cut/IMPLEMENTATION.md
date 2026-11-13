# Implementation plan — living `/app` home hard cut

Depends on [`LOCKED.md`](./LOCKED.md). Docs already landed (glossary + ADR-0006/0008);
this plan is code only.

**Dumb executor:** no Feragame-named branches. Gate many-history on
`WorklistTask.submit === 'many'` + `submissionCounts` only.

Do not start until explicitly asked. Prefer small slices in this order.

---

## Slice A — Commit navigation: overview → home ✅/⬜

**Goal:** product wire never targets overview; legacy maps to home.

1. In `resolveCommitDestination` / `resolveDefaultCommitDestination`
   (`commit.go`): wherever today’s default returns `CommitNextOverview`
   (remaining work, nested-open group), return **`CommitNextHome`** instead.
2. Authored `onCommit.navigation: overview` → emit **`home`** on the wire
   (compat), not `overview`.
3. `continueNavigation` must not be `overview` (map/default to `home`).
4. Keep `CommitNextOverview` const only if still useful for parsing old
   clients—or treat inbound unknown/`overview` as home in one place.
5. Update Go tests that assert `overview` for close/remaining-work/
   group-open paths → expect **`home`**. Group tests may still exist until
   Phase 7; change expectations only, no new group product work.
6. SPA `worklist-nav.ts`: parsing Commit `navigation` / `continueNavigation`
   — if wire still sends `overview`, navigate **home** (defense in depth).
   Prefer Go never sending it.

**Done when:** `go test` for commit navigation + related living tests green;
no successful Commit returns `navigation: "overview"` in new assertions.

**Touch (expected):** `commit.go`, `commit_navigation_test.go`,
`groups_commit_test.go` (expect home), `worklist-nav.ts` + tests,
maybe `http-commit-port.ts` if it normalizes nav enums.

---

## Slice B — Task back / exit → home ✅/⬜

**Goal:** leaving a Task returns to Worklist home.

1. `resolveTaskExit` → always **`{ kind: 'home' }`** (or home whenever not
   provisional—lock: always home).
2. `nav-place.ts` / Task chrome back affordance follow that; drop overview
   as `backTarget`.
3. Update `nav-place.test.ts`, `worklist-nav.test.ts`.

**Done when:** unit tests show Task back → home for minted and provisional.

**Touch (expected):** `worklist-nav.ts`, `nav-place.ts`, their tests.

---

## Slice C — Home act-now list (drop peer bands) ✅/⬜

**Goal:** one list = actionable ∪ Can start; no In progress / Completed UI.

1. Add `actNowRowsFromWorklist` (or equivalent) that merges:
   - existing `availableRowsFromWorklist` order, and
   - `canStartRowsFromWorklist` as peer entries (discriminated union:
     `{ kind: 'task', ...AvailableRow }` | `{ kind: 'start', ...CanStartRow }`).
   Decide sort: recommend **Can start after actionable** within the same
   flat list (or Job-stable: keep available Job→instance→task order, then
   append can-start Jobs not already represented—document choice in HANDOFF).
2. Enrich **task** act-now rows with `submit` + `submissionCount` from
   Worklist chrome / `instance.submissionCounts` (needed for Slice D).
3. `LivingShell.svelte`: single section (one heading or none); render both
   row kinds; remove In progress / Completed sections and their empty
   states.
4. Stop linking home rows to overview targets (`inProgressRowTarget` /
   `completedRowTarget` unused by home).
5. Paraglide: reuse or add a single home heading if needed; orphaned
   `in_progress` / `completed` strings can remain unused for now.
6. Tests: `worklist-home.test.ts` for merge + enrichment; adjust any
   shell/component tests.

**Done when:** home shows only act-now rows; can start still mints via
`new` + first Task; no overview deep-links from home.

**Touch (expected):** `worklist-home.ts` + tests, `LivingShell.svelte`,
maybe small row component extract if the markup gets heavy.

**Note:** Keep `inProgressRowsFromWorklist` / `completedRowsFromWorklist`
helpers for now (progress math still used elsewhere / overview route) unless
they become dead—optional delete in a cleanup slice, not required.

---

## Slice D — Many-submit history under home act-now rows ✅/⬜

**Goal:** re-host overview many-history chrome on home task rows.

1. Extract shared snippet/component from `overview.svelte`’s
   `manySubmitHistory` (Accordion + count + `TaskRecapList`) into something
   home and overview can both use (e.g. `ManySubmitHistory.svelte`), **or**
   duplicate minimally on home and leave overview as-is (route kept; no
   polish pass—prefer **extract once** to avoid drift).
2. On home **task** rows with `submit === 'many'` and count N > 0: under
   the primary Apri/link row, closed count → expand → lazy list recap
   (same request shape as overview: job/instance/task [+ group coords]).
3. Primary row remains the Task navigation control (do not put expand on
   the same control).
4. Can start rows: no history.
5. Tests: unit for visibility gate; component/unit as existing overview
   many-history tests pattern.

**Done when:** home many-row shows N → expand → oldest-first items + links;
Apri still opens the Task.

**Touch (expected):** extract component; `LivingShell.svelte` (or home row
partial); optionally thin `overview.svelte` to use the extract; messages
already have `submission_count_*`.

---

## Slice E — Closeout / verify ✅

1. Grep SPA for home→overview entry points; leave overview route registered.
2. Confirm `.scratch/commit-navigation/LOCKED.md` if it still says overview
   default—add a one-line pointer to this board or leave untouched (ask
   before editing unrelated scratch).
3. HANDOFF note on this folder; manual smoke checklist.
4. No Phase 7 group removal in this PR.

**Done when:** checklist below green; branch ready to commit when asked.

---

## Suggested order

| Slice | Depends on | Notes |
|-------|------------|--------|
| A | — | Go + SPA nav parse |
| B | — | Parallel with A |
| C | — | Parallel with A/B; needs submit/count on rows for D |
| D | C (+ recap stack already on branch) | UI host move |
| E | A–D | |

Rough size: Go nav is small; home merge + LivingShell is the bulk; D is
mostly extract + wire.

## Explicit non-goals

- Delete overview route or rewrite overview UI
- Archive hierarchy / stub
- Phase 7 Task-group / `unlockFrom` removal
- Feragame Config changes (unless a test fixture asserts overview nav)
- Once-done recap on home
- Push/PR unless asked

## Verify

Per slice:

- `go test` for touched `server/internal/living` (at least commit + worklist)
- SPA: `bun run test:unit` for `worklist-home`, `worklist-nav`, `nav-place`,
  living shell/recap as touched

Manual (after D):

1. Load `/app` → only act-now list (actionable + can start).
2. Start Job → Task → back → home (not overview).
3. Close leaf / `advanceTarget: false` → lands home.
4. Many-submit twice → home row shows count 2 → expand → two recap rows with
   links; title still opens form.
5. Manual URL `/app/{job}/{instance}` still loads overview (orphan OK).
