# 02: Plant — containerReg Task group + unlockFrom + park loadedExtracted

**What to build:** On the live line Job, Registrazione contenitori is a repeatable Task group with a `register` leaf (lots/weigh pattern). The group path-unlocks with `openTurno` (`unlockFrom`), has no `readyWhen`, supports Spawn/Close (`closeRequiresChildSubmission`), and shows group instances labeled by load like intake lots (`instanceLabelEntity: load`). `loadedExtracted` is removed from the live Task list; its Risultati chrome body is parked in-repo for later relocate. Multi-Spawn `register` works on one Job instance. DualSources form may remain until ticket 05. Engine stays Config-only.

**Blocked by:** None (can start immediately; parallel with 01 OK).

**Status:** done

- [x] Live line Tasks no longer include `loadedExtracted`; chrome body preserved outside the live list
- [x] `containerReg` is a repeatable group + `register` leaf; `unlockFrom: openTurno`; omit `readyWhen`
- [x] After `openTurno`, group is path-unlocked / actionable for logistic; Close requires a child Submission; no Unlock successor
- [x] Overview/Worklist labels group instances via load (lots pattern)
- [x] Plant/seed/flow tests updated; no new Go special-case on plant Task ids

**Residual (out of peel):** SPA `line-smoke.test.ts` still may need alignment with the unlockFrom group plant shape.
