# HANDOFF — Slice D (many-submit history under home)

**Branch:** `feat/living-home-hard-cut`  
**Do not commit.**

## Changed paths

- `app/src/lib/living/ManySubmitHistory.svelte` — **new**; Accordion count → expand → lazy `TaskRecapList`; gates on `manySubmitHistoryVisible`; owns open state; `data-testid` prop (trigger = `${id}-trigger`).
- `app/src/lib/living/LivingShell.svelte` — under **task** act-now rows only: primary `WorklistRow` + `<ManySubmitHistory data-testid="home-many-history" …>`; can-start unchanged (no history).
- `app/src/lib/router/overview.svelte` — snippet + parent many-history accordion state removed; uses shared `<ManySubmitHistory data-testid="overview-many-history" />`.
- `app/src/lib/living/worklist-home.test.ts` — act-now enrichment asserts gate true for many+N>0 / false for once.

## API

`submit?`, `submissionCount?`, `request: TaskRecapRequest`, optional `data-testid` (default `many-submit-history`).

## Tests

- `cd app && bun run test:unit -- src/lib/living/worklist-home.test.ts src/lib/living/worklist-overview.test.ts src/lib/living/http-task-recap-port.test.ts` → **pass** (30).
- Autofixer: `ManySubmitHistory.svelte` + `LivingShell.svelte` **clean**; overview only pre-existing `$effect`+`navigate` note.

## Not touched

- Overview route kept; Commit nav; e2e (Slice E).
- `manySubmitHistoryVisible` / `formatSubmissionCount` stay in `worklist-overview.ts`.

## Follow-ups / blockers

- Slice E: e2e still asserts old home band headings; manual smoke: many-submit twice → home count 2 → expand → recap links; Apri still opens Task.
