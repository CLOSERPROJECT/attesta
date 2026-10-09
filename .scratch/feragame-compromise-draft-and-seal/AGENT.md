# Agent notes — draft-and-seal A/B candidate

## Pointers (read these; do not re-derive)

- Spec: `.scratch/feragame-compromise-draft-and-seal/spec.md`
- Tickets: `.scratch/feragame-compromise-draft-and-seal/issues/`
- Design: `docs/feragame/explorations/03-compromise-draft-and-seal.md`
- Brief: `docs/feragame/README.md`
- Domain: `CONTEXT.md`, `AGENTS.md`

## Hard constraints

1. **Two strategies.** Default `classic` = today’s Complete + sequential availability — leave those function bodies as the classic strategy. Opt-in `draft-and-seal` is a separate strategy. No mode `if`s sprinkled inside classic completion/availability.
2. **Shared only:** progress map/store, Formata, role/org gates, notarization insert/export shape, instance create, progress-key encoding (`.` → `_` for Mongo).
3. **Do not** invent a child-event store, change classic Streams, or crown this the pilot winner.
4. **Ask before** breaking unrelated routes / workflow config / persistence shapes (`AGENTS.md`).
5. Prefer extending the process-completion domain edge; keep classic regression tests green and unweakened.

## Ticket order

`01` → `02` → `03` → `04`. Claim (`Status: claimed`) before writing; resolve with `## Answer` + `Status: resolved` when done.

## Landed (01)

See `issues/01-strategy-seam.md` ## Answer. Edge: `newCompletionStrategy`, `ProcessService.SaveDraft` / `SealStep`, `substepAvailability` / `substepSequenceOK`.

## Landed (02)

See `issues/02-parallel-draft-day.md` ## Answer. Draft-and-seal: parallel in-Step availability, `SaveDraft` → progress `draft` (no notarization), `CompleteSubstep` rejected.

## Landed (03)

See `issues/03-seal-step.md` ## Answer. `SealStep` implemented: Org-admin standing, confirm, freeze + notarize leaves, instance complete. `ErrSealUnauthorized`, `ErrSealNotConfirmed`, `ErrSealAlreadySealed`.

## Landed (04)

See `issues/04-draft-and-seal-chrome.md` ## Answer. Timeline parallel drafting + Save draft; Step seal dialog; Formata POST `/complete` → `SaveDraft` for draft-and-seal; classic Complete unchanged. Chain `01`–`04` complete.

## Landed (03)

See `issues/03-seal-step.md` ## Answer. `SealStep`: Org admin standing for Step org, confirm required, freeze all kind leaves (empty OK), one notarization per leaf at seal time, reject further drafts and second seal, complete single-Step instance. No UI (04).

## Landed (04)

See `issues/04-draft-and-seal-chrome.md` ## Answer. Instance chrome: parallel drafting labels, no Complete on kind Substeps, Step-level Seal with confirm. Formata POST `/complete` → `SaveDraft` on draft-and-seal; `handleSealStep` on `/step/{id}/seal`. Classic UI unchanged.

## Verify

Run the existing Go completion/availability tests for classic; add draft-and-seal tests alongside. Do not weaken classic assertions.
