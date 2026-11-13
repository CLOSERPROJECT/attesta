# HANDOFF — Slice E (closeout)

**Branch:** `feat/living-home-hard-cut`  
**Do not commit. Do not push.**  
**Board ready for Cos to commit when asked.**

## Changed

- `app/src/routes/[...path]/page.svelte.e2e.ts` — act-now home (`worklist-act-now`); Task back → Home; unauthorized Task → Home; overview direct-URL tests kept.
- `app/src/lib/living/line-smoke.test.ts` — fake Commit remaining-work emit `home` (no `overview`).

## Grep (home→overview product entry)

- LivingShell: Task via `availableRowTarget`; Can start → `new`+first Task. No overview hrefs.
- Overview route still registered (`/:jobId/:jobInstanceId`).
- `inProgressRowTarget` / `completedRowTarget` remain (helpers + unit tests only; unused by home).

## Untouched (ask Cos)

- `.scratch/commit-navigation/LOCKED.md` still says default destination **Job instance overview** when `advanceTarget: false`. Leave as-is; pointer board is this folder / ADR-0006.

## Verify

- `go test ./internal/living/` → pass
- SPA unit (`worklist-home`, `worklist-nav`, `nav-place`, `worklist-overview`, `http-commit-port`, `line-smoke`, `navigate-target`) → **69 pass**
- e2e `page.svelte.e2e.ts` → **7 pass**

## Manual smoke (from IMPLEMENTATION Verify)

1. `/app` → only act-now (actionable + can start); no In progress / Completed.
2. Start Job → Task → back → home (not overview).
3. Close leaf / `advanceTarget: false` → lands home.
4. Many-submit twice → home row count 2 → expand → two recap rows + links; title/Apri still opens form.
5. Manual URL `/app/{job}/{instance}` still loads overview (orphan OK).

## Follow-ups

- Cos: commit when asked; optionally refresh `commit-navigation` scratch LOCKED.
- No Phase 7 group removal in this PR.
