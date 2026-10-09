# 02: Parallel draft day — record kinds without completing

**What to build:** On a draft-and-seal Feragame-shaped Stream (one Organization, one Step, per-kind Formata array Substeps), creating a Stream instance opens the production day. Both kind Substeps are available together. An operator with ordinary Substep roles can draft-save array payloads (non-`done` progress, no notarization) and re-edit until seal. Mid-day Complete is absent or rejected for those kind Substeps. Classic Streams remain one-incomplete-Substep-at-a-time.

**Blocked by:** 01 (Strategy seam — classic untouched, draft-and-seal selectable).

**Status:** resolved

- [x] Opt-in Feragame-shaped Stream uses strategy `draft-and-seal`
- [x] After instance create, all incomplete kind Substeps in the unsealed Step are available together
- [x] Draft save persists Formata array payloads without marking Substeps `done` and without notarization
- [x] Draft payloads reload and remain editable; row/field validation runs on save
- [x] Mid-day Complete on kind Substeps is absent or rejected
- [x] Classic Streams still enforce sequential unlock and Complete→notarize
- [x] Demoable: interleave batch-load and UdC rows across kind Substeps and reload still editable

## Answer

**Symbols**
- `draftAndSealCompletionStrategy.ComputeAvailability` — parallel in-Step; cross-Step lock (`steps-only`)
- `draftAndSealCompletionStrategy.IsSequenceOK` — prior Steps must be done; in-Step peers unlocked
- `draftAndSealCompletionStrategy.SaveDraft` — progress state `draft`, no notarization, shared `UpdateProcessProgress`
- `draftAndSealCompletionStrategy.CompleteSubstep` — still `ErrCompletionUnsupported` (Seal owns terminal)
- `draftAndSealCompletionStrategy.SealStep` — still stubbed for ticket 03
- `progressStateDraft`, `ErrDraftValidation`
- `validateDraftFormataPayload`, `progressEntryDone`, `stepAllSubstepsDone`
- Test blueprint helpers: `feragameDraftAndSealDef`, `feragameArraySchema`

**HTTP:** No new routes. `ProcessService.SaveDraft` already exists at the strategy edge; demo coverage is domain-service tests (handlers do not yet call SaveDraft).

**Tests**
```text
cd server/cmd/server && go test -count=1 -timeout 120s . \
  -run 'TestCompleteSubstep|TestEnsureCompletion|TestProcessCompletion|TestComputeAvailability|TestSequenceAndDone|TestNextAvailable|TestNormalizeCompletion|TestWorkflowCompletion|TestClassicStrategy|TestDraftAndSeal|TestClassicSaveDraft|TestParseRuntimeConfigCompletion|TestWorkflowDefCompletion'
ok  	github.com/CLOSERPROJECT/attesta/server/cmd/server	0.342s
```
