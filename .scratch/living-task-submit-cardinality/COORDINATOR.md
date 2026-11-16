# Coordinator — submit cardinality + lot session

DRI: Cos session. Subagents do the work; this file is the status board.

## Pointers (do not duplicate into prompts)

- [`LOCKED.md`](./LOCKED.md)
- [`IMPLEMENTATION.md`](./IMPLEMENTATION.md)
- [`docs/adr/0008-living-task-submit-cardinality.md`](../../docs/adr/0008-living-task-submit-cardinality.md)
- [`docs/agents/plant-config.md`](../../docs/agents/plant-config.md)
- [`GLOSSARY.md`](../../GLOSSARY.md)
- [`AGENTS.md`](../../AGENTS.md) — dumb executor; conventional commits if committing

## Status

| Phase | Owner | State |
|-------|-------|-------|
| 0 Docs | done | landed |
| 1 Engine `submit` | [Phase 1 submit engine](7706169a-7539-4056-b39b-667c5a6ed931) | **done** → [`HANDOFF-phase-1.md`](./HANDOFF-phase-1.md) |
| 2 `load`→`lot` + status | [Phase 2 load to lot](bde135af-f9fd-4420-9e7d-e775ca8a256d) | **done** → [`HANDOFF-phase-2.md`](./HANDOFF-phase-2.md) |
| 3 Intake flatten | [Phase 3 intake flatten](f797bbcd-1131-46e1-8383-f9a3822bbcc2) | **done** → [`HANDOFF-phase-3.md`](./HANDOFF-phase-3.md) |
| 4 Line lot session | [Phase 4b](0a8462b7-9a85-45a3-88ef-9fe84c5299ba) **done** → [`HANDOFF-phase-4.md`](./HANDOFF-phase-4.md) (WIP from [Phase 4](8abadc47-bc13-4398-8477-baa9503b2bb0) archived) |
| 5 SPA chrome | [Phase 5-6 SPA cleanup](87b587e3-87c1-428b-96f0-e03f35e5307c) | **done** → [`HANDOFF-phase-5-6.md`](./HANDOFF-phase-5-6.md) |
| 6 Passport/cleanup | same | **done** → same handoff |
| 7 Drop groups/unlockFrom | — | later, explicit ask |
| UX many-submit recap lines | Cos | **done** (implement complete; commit on ask) → [`LOCKED-many-submit-recap.md`](./LOCKED-many-submit-recap.md); plan [`IMPLEMENTATION-many-submit-recap.md`](./IMPLEMENTATION-many-submit-recap.md); closeout [`HANDOFF-ux-slice-e.md`](./HANDOFF-ux-slice-e.md) |
| UX Slice A LinkStore ts | [Slice A LinkStore timestamp](774030d1-f25e-4b85-bfe8-5766b6465e05) | **done** → [`HANDOFF-ux-slice-a.md`](./HANDOFF-ux-slice-a.md) |
| UX Slice B submissionCounts | [Slice B submissionCounts](8d1ab0ec-4439-46df-a8ea-d975eac214c2) | **done** → [`HANDOFF-ux-slice-b.md`](./HANDOFF-ux-slice-b.md) |
| UX Slice C list recap | [Slice C list recap API](72e7263d-d889-4694-bda5-336448f7ff07) | **done** → [`HANDOFF-ux-slice-c.md`](./HANDOFF-ux-slice-c.md) |
| UX Slice D overview accordion | [Slice D overview accordion](2a3e44db-53bb-4fa0-acd5-3b7b5c2c1451) | **done** → [`HANDOFF-ux-slice-d.md`](./HANDOFF-ux-slice-d.md) |
| UX Slice E closeout | [Slice E closeout verify](7ea37674-9f30-4779-a6b3-526751699d09) | **done** → [`HANDOFF-ux-slice-e.md`](./HANDOFF-ux-slice-e.md) |

## Phase 1 locked rule (job finished)

When projecting In progress / Completed / “all tasks done”:

- Only **`submit: once`** (default) Tasks count toward instance completion via Submission → `done`.
- **`submit: many`** Tasks never block completion by lacking `done`; they may still show as `actionable`/`unavailable` on overview while the instance remains In progress for other reasons (open once bookends). Prefer: instance is **Completed** only when every **once** Task on the Job is `done` (many ignored for that predicate).

## Environment / pit of success

- Prefer synthetic mini-configs in `internal/living` tests; no Feragame Task id branches in Go.
- Validate `submit` only on leaf Tasks; reject on `kind: group` if still present.
- After Phase 1: fixture may omit `submit` (default once) until Phase 3–4 author `many`.
- **Phase 5 debt:** ship `submit` on WorklistTask wire (or equivalent) so SPA
  `instanceTaskProgress` can ignore `many` — see HANDOFF-phase-1.

## Subagent return contract

Each implementer ends with a short note at
`.scratch/living-task-submit-cardinality/HANDOFF-phase-N.md`:

- What changed (paths)
- Tests run + pass/fail
- Follow-ups / blockers for next phase
- Do **not** commit unless Cos asks

## Schedule suggestion

After each phase HANDOFF is green: Cos reviews HANDOFF → dispatches next phase (or fix-up). Optional: daily resume check if a phase stalls >1 session.
