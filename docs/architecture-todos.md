# Architecture todos

Deferred deepening items surfaced during architecture reviews. Not committed work — tracking only.

## Process completion — transactional writes

**Context:** [architecture review, candidate #2 — process completion pipeline]

**Current behavior (preserve for now):**
- `UpdateProcessProgress` → `InsertNotarization` → `finalizeProcessIfDone` run sequentially.
- Notarization failure after progress is saved returns 500; substep remains `done`.
- DPP/status finalization failures are logged, not returned as errors.

**Deferred enhancement:**
- Make completion atomic: Mongo transaction or compensating rollback so progress and notarization succeed or fail together.
- Would change the `CompleteSubstep` module contract and `TestHandleCompleteSubstepStoreFailures` expectations.
- Evaluate when partial-failure repair (`EnsureCompletionArtifacts`) becomes insufficient or confusing in production.

**Trigger to revisit:** repeated support issues from “done substep, no notarization” or when Mongo multi-document transactions are already in use elsewhere.
