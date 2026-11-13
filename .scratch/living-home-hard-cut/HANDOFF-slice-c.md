# HANDOFF — Slice C (home act-now list)

**Branch:** `feat/living-home-hard-cut`  
**Do not commit.**

## Sort choice

**Actionable first, Can start after** (flat list: `availableRowsFromWorklist` order, then `canStartRowsFromWorklist`). Documented in `actNowRowsFromWorklist` JSDoc; matches COORDINATOR / IMPLEMENTATION default.

## Changed paths

- `app/src/lib/living/worklist-home.ts` — `ActNowRow`, `actNowRowsFromWorklist`, `actNowRowKey`; AvailableRow enriched with `submit` + `submissionCount` (flat from `instance.submissionCounts`; nested from group-instance counts, mirror overview).
- `app/src/lib/living/worklist-home.test.ts` — merge order + enrichment + empty act-now.
- `app/src/lib/living/LivingShell.svelte` — one `worklist-act-now` section; task + start row kinds; In progress / Completed / peer Available+Can-start bands removed; no overview links; can-start still mints via `new` + `firstTaskId`.

## Kept (intentionally)

- `inProgressRowsFromWorklist` / `completedRowsFromWorklist` (+ row target helpers) for overview / progress math.
- Overview route untouched.
- No many-history accordion (Slice D).
- No Go commit navigation changes (A+B).

## Tests

- `cd app && bun run test:unit -- src/lib/living/worklist-home.test.ts` → **pass** (10).
- Svelte autofixer on `LivingShell.svelte` → **clean**.

## Follow-ups / blockers

- E2E `app/src/routes/[...path]/page.svelte.e2e.ts` still asserts `worklist-available` / `worklist-can-start` headings — update in Slice E (or when touching e2e).
- Slice D can host many-history under home **task** rows using `submit` + `submissionCount`.
