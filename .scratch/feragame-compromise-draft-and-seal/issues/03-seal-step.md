# 03: Seal Step — freeze, notarize, complete instance

**What to build:** An elevated actor can confirm **Seal Step** on a draft-and-seal Stream instance. Seal freezes every kind Substep payload in the Step (empty kinds allowed), notarizes each Substep leaf at seal time, rejects further drafts, and completes the single-Step Stream instance. Double-seal fails closed. Ordinary Substep editors cannot seal. Classic Streams gain no Seal action.

**Blocked by:** 02 (Parallel draft day — record kinds without completing).

**Status:** resolved

- [x] Elevated auth required for Seal Step; ordinary Substep roles cannot seal
- [x] Seal requires confirmation and is irreversible for that Step’s evidence
- [x] Seal freezes all kind Substep payloads in the Step (including empty arrays)
- [x] One notarization per sealed Substep leaf at seal time; digest over sealed payload; CreatedAt = seal time
- [x] Post-seal draft/edit/append rejected
- [x] Second seal rejected (fail closed)
- [x] Pilot single-Step Stream instance is complete after seal
- [x] Notarized export still exposes Step → Substep structure
- [x] Demoable: open → draft → seal → immutable day

## Answer

**Elevated auth:** Org admin standing (`org-admin` / `org_admin` on `Actor.RoleSlugs` or `Actor.Role`) for the Step’s Organization (`Actor.OrgSlug` must match `WorkflowStep.OrganizationSlug`). Ordinary Substep roles (e.g. `registrar`) cannot seal.

**Symbols**
- `draftAndSealCompletionStrategy.SealStep` — confirm, elevate-auth, freeze all kind leaves, notarize each, finalize single-Step instance
- `ErrSealUnauthorized`, `ErrSealNotConfirmed`, `ErrSealAlreadySealed`
- `findWorkflowStep`, `actorCanSealStep`, `actorHasOrgAdminStanding`, `sealActor`, `sealedSubstepPayload`
- Shared classic leaf path reused: `UpdateProcessProgress` → `done` + `InsertNotarization` (`digestPayload`, `CreatedAt` = seal time) + `finalizeProcessIfDone`
- `SaveDraft` still rejects done/sealed Substeps and closed instances

**HTTP:** No new routes. `ProcessService.SealStep` remains the strategy edge; coverage is domain-service tests (handlers do not yet call SealStep — UI is ticket 04).

**Tests**
```text
cd server/cmd/server && go test -count=1 -timeout 120s . \
  -run 'TestCompleteSubstep|TestEnsureCompletion|TestProcessCompletion|TestComputeAvailability|TestSequenceAndDone|TestNextAvailable|TestNormalizeCompletion|TestWorkflowCompletion|TestClassicStrategy|TestDraftAndSeal|TestClassicSaveDraft|TestParseRuntimeConfigCompletion|TestWorkflowDefCompletion'
ok  	github.com/CLOSERPROJECT/attesta/server/cmd/server	0.378s
```
