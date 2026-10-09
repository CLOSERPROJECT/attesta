# 01: Strategy seam — classic untouched, draft-and-seal selectable

**What to build:** A Stream blueprint can declare a completion strategy. Default remains **classic**: sequential Substep unlock and Complete → done + notarize behave exactly as today. Opt-in **draft-and-seal** is recognized at the process/instance edge and routed to a separate strategy body (stub that fails closed for divergent ops until later tickets). Shared progress store, Formata, auth gates, and notarization types stay the common layer — classic functions are not rewritten with mode branches.

**Blocked by:** None (can start immediately).

**Status:** resolved

- [x] Stream blueprint carries a strategy flag; unset/default is `classic`
- [x] Edge selects classic → existing Complete + sequential availability path (no behavior change for classic Streams)
- [x] Existing classic completion and availability tests remain green without weakened assertions
- [x] Opt-in `draft-and-seal` is parseable and selects a distinct strategy stub (divergent ops fail closed until later tickets)
- [x] No scattered mode `if`s inside classic completion/availability implementations

## Answer

**Symbols**
- `WorkflowDef.CompletionStrategy` (`yaml/bson: completionStrategy`)
- `CompletionStrategyClassic`, `CompletionStrategyDraftAndSeal`
- `ErrCompletionUnsupported`
- `normalizeCompletionStrategy`, `workflowCompletionStrategy`, `newCompletionStrategy`
- `completionStrategy` interface; `classicCompletionStrategy`; `draftAndSealCompletionStrategy`
- `ProcessService.CompleteSubstep` (edge) → `classicCompleteSubstep` (classic body)
- `ProcessService.SaveDraft`, `ProcessService.SealStep`
- `SaveDraftCmd`, `SealStepCmd`
- `substepAvailability`, `substepSequenceOK` (availability/sequence edge)

**Tests**
```text
cd server/cmd/server && go test -count=1 -timeout 120s . \
  -run 'TestCompleteSubstep|TestEnsureCompletion|TestProcessCompletion|TestComputeAvailability|TestSequenceAndDone|TestNextAvailable|TestNormalizeCompletion|TestWorkflowCompletion|TestClassicStrategy|TestDraftAndSeal|TestClassicSaveDraft|TestParseRuntimeConfigCompletion|TestWorkflowDefCompletion'
ok  	github.com/CLOSERPROJECT/attesta/server/cmd/server	0.358s
```

Broader related suite (`TestComplete|TestHandleComplete|TestIntegrationComplete|…`) also `ok`.
