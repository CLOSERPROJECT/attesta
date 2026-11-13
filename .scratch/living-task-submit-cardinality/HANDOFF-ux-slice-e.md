# HANDOFF — UX Slice E (verify + board closeout)

## Done

Many-submit overview/recap UX slices A–D verified green. COORDINATOR UX rows
and parent “UX many-submit recap” marked **done** (implement complete; commit
still on ask). ADR-0008 / glossary churn skipped.

## Verify

| Command | Result |
|---------|--------|
| `go test ./internal/living/ -count=1` (from `server/`) | **pass** |
| `bun run test:unit -- --run src/lib/living/worklist-overview.test.ts src/lib/living/http-task-recap-port.test.ts src/lib/living/worklist.test.ts src/lib/living/http-commit-port.test.ts` (from `app/`) | **pass** (4 files / 34 tests) |
| Paraglide compile (`bun x @inlang/paraglide-js compile --project ./project.inlang --outdir ./src/lib/paraglide --emit-ts-declarations` from `app/`) | **pass** — `submission_count_one` / `submission_count_many` in generated outdir (gitignored) |

No feature code changes in this slice.

## Board

- [`COORDINATOR.md`](./COORDINATOR.md) — UX many-submit recap + Slice E → **done**
- [`IMPLEMENTATION-many-submit-recap.md`](./IMPLEMENTATION-many-submit-recap.md) — Slices A–E ticked ✅
- ADR / `docs/adr/` / glossary — not touched

## Wipe reminder (LinkStore)

Address-only link docs are orphaned after Slice A timestamp keys. Before relying
on recap download links locally/pre-prod:

- `task reset` / `task purge` for this checkout’s volumes, **or**
- drop Mongo collection `living_submission_links`

No migration job. Details: [`HANDOFF-ux-slice-a.md`](./HANDOFF-ux-slice-a.md).

## Follow-ups

- Commit / PR when Cos asks (conventional commits).
- Optional integrator note in ADR-0008 for `submissionCounts` + list `items`
  wire — still ask before doc churn.
- Manual smoke (not run here): many-submit twice → count accordion → expand →
  oldest-first mini-recaps with per-timestamp links.
