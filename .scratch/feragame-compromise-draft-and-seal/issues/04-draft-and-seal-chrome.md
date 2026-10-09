# 04: Draft-and-seal chrome — timeline language and Seal control

**What to build:** Instance UI for draft-and-seal Streams shows both kind Substeps as available/drafting (not fake sequence locks), omits Complete on kind Substeps, and exposes an explicit Step-level Seal control with confirmation. Copy distinguishes drafting vs sealed. Classic Stream instance UI and Complete flow stay unchanged so A/B evaluators can run a production day without using the API.

**Blocked by:** 03 (Seal Step — freeze, notarize, complete instance).

**Status:** resolved

- [x] Timeline/navigation shows parallel drafting kind Substeps for draft-and-seal instances
- [x] Kind Substeps do not offer mid-day Complete in the UI
- [x] Seal appears as a Step-level control with confirmation, not as a last Substep
- [x] Drafting vs sealed states are clear in operator-facing copy
- [x] Classic instance UI still shows sequential unlock and Complete as today
- [x] Demoable end-to-end in the browser for internal A/B on the compromise candidate

## Answer

**Chrome:** draft-and-seal instance timeline labels in-Step kind Substeps **drafting** (parallel available; not “Locked by sequence”). Kind bodies use **Save draft as** (no Complete) and restore Formata values from progress `draft`. Org admins for the Step org get a Step-level **Seal Step** dialog (`confirm=true`); sealed copy is **sealed** / frozen evidence.

**HTTP (reuse Formata save path):** `handleCompleteSubstep` calls `SaveDraft` when the Stream strategy is draft-and-seal (same `/substep/{id}/complete` POST). New `POST .../step/{stepID}/seal` → `ProcessService.SealStep`. Classic Complete + sequential lock unchanged.

**Templates:** cluster markup in `stream_timeline` (seal is not a Substep); `substep_body` hint + role dialog copy. CSS: `.stream-timeline-seal` in `stream-timeline.css`.

**Tests**
```text
cd server/cmd/server && go test -count=1 -timeout 120s . \
  -run 'TestBuildSubstepViewsDraftAndSeal|TestBuildStreamInstanceDetailViewDraftAndSeal|TestBuildStreamInstanceDetailViewClassic|TestStreamTimeline|TestSubstepBody|TestHandleCompleteSubstepDraftAndSeal|TestHandleSealStep|TestHandleProcessRoutesDispatchesSeal|TestHandleCompleteSubstepSuccess|TestDraftAndSeal|TestClassic'
ok  	github.com/CLOSERPROJECT/attesta/server/cmd/server

bash scripts/djlint.sh server/templates/components/stream_timeline.html \
  server/templates/components/substep_body.html --check
# 0 files would be updated
```
