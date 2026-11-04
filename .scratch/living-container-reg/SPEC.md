# Spec: containerReg group with incomplete continuity

**Branch:** `feat/living-container-reg` (from `feat/attesta-rework` @ lineLoad readyWhen/`unlockFrom` squash)  
**Status:** ready-for-agent  
**Hard constraint:** Go and SPA are dumb executors of Org Config. No Feragame- or plant Task-id special-case logic in the backend or SPA.

Also published: https://github.com/CLOSERPROJECT/attesta/issues/284

## Problem Statement

Logistic operators need to register containers against intake loads during a line turno so that, for each load, the sum of pallet net weight can be reconciled with the sum of container weight attributed to that load. Today Registrazione contenitori is a single flat Task that unlocks only after Segna avanzamento Submission, cannot be repeated per load on the same Job instance, and models dual-source with a per-row checkbox plus a free-picked secondary load. Incomplete bins that continue across loads cannot freeze a partial weight for one load and later book only the delta when the operator enters a full scale reading. The final Controllo caricato vs estratto step is not wanted on the live line Job right now (its chrome should be kept for a later relocate).

## Solution

Reshape plant Registrazione contenitori into a repeatable Task group (same family as intake lots): each Spawn/Invia books one load. Path-unlock with the turno (`unlockFrom: openTurno`); no Domain readyWhen (always open once unlocked—floor timing is operator discipline for now). Hide loads already marked `fullyProcessed` from the primary select; set that fact on the load when registration succeeds. Drop the `dualSources` checkbox. Prefill continuation rows for still-open incomplete containers (hidden `continuesContainerId`); operator enters full scale weight; Org Config transform books the delta for this load and supersedes the prior incomplete. New bins are plain rows. Remove `loadedExtracted` from the live line Task list but keep its report body available in-repo for later relocation. Go and the SPA remain dumb executors of Org Config—no Feragame- or container-specific engine branches.

## User Stories

1. As a logistic operator, I want Registrazione contenitori available as soon as the turno is open, so that I am not blocked on Segna avanzamento Submission.
2. As a logistic operator, I want to Spawn a new registration per load, so that I can book containers for multiple loads in one turno.
3. As a logistic operator, I want each registration instance labeled with the load id, so that the overview matches intake lots labeling.
4. As a logistic operator, I want the primary load select to omit loads I already finished registering, so that I do not double-book a load.
5. As a logistic operator, I want the system to mark a load `fullyProcessed` when I successfully Invia its registration, so that it disappears from future selects.
6. As a logistic operator, I want to enter container weights for a load when its pallets are loaded, so that pallet net and container net can be compared for that load.
7. As a logistic operator, I want to mark a container row incomplete, so that I can freeze partial weight for this load while the physical bin stays open.
8. As a logistic operator, I want incomplete to default unchecked on every new form (including continuation rows), so that incomplete only means “freeze again for this load.”
9. As a logistic operator, on the next Spawn, I want prior open incompletes prefilled as rows, so that I do not re-pick which bin continues.
10. As a logistic operator, on a continuation row, I want to type the full scale reading (not a delta), so that the scale and the form match.
11. As a logistic operator, I want the system to attribute only the delta to the new load, so that load 1 keeps the frozen kg and load 2 gets full minus freeze.
12. As a logistic operator, I want the prior incomplete closed when I continue it, so that each physical bin has at most one open incomplete chain.
13. As a logistic operator, I want to add new bins on a later Spawn without a prior freeze, so that booked weight equals what I type.
14. As a logistic operator, I want to mark a continuation or new row incomplete again, so that I can freeze mid-bin for the current load.
15. As a logistic operator, I do not want a dualSources checkbox or a free secondary load dropdown, so that continuity replaces that UX.
16. As a logistic operator, I want optional Close on the containerReg group after at least one child Submission, so that I can finish group chrome without unlocking a further live step.
17. As a line operator, I want Segna avanzamento unchanged by this work (still unlockFrom openTurno + readyWhen), so that concurrent line progress keeps working.
18. As an administration operator, I no longer want Controllo caricato vs estratto on the live line Job, so that the turno ends at container registration for now.
19. As a developer, I want the Risultati finali chrome body preserved outside the live Task list, so that we can relocate reconciliation UI later without rewriting the report.
20. As a platform developer, I want all plant behavior authored in Org Config, so that Go and SPA stay generic executors.
21. As a platform developer, I want no Go/TS special-case on plant Task ids or Feragame names for this feature, so that other orgs can author the same patterns.
22. As a logistic operator, I want multiple Submissions of register under the group without the flat Task becoming done after the first Invia, so that multi-load booking works.
23. As a logistic operator, I want secondary/previous-load attribution derived from the continued container when relevant, so that reporting can still see cross-load bins without me picking secondary.
24. As a reviewer, I want Domain container facts to carry incomplete and enough identity for continuation, so that reconciliation can sum booked weights as stored.
25. As a reviewer, I want load.fullyProcessed as the select filter fact, so that “already registered” is Domain-visible.
26. As an operator on overview, I want group instances to show load ids via instanceLabelEntity load, so that scanning the turno is obvious.
27. As a logistic operator, I want closeRequiresChildSubmission on the group, so that I cannot Close with zero registrations.
28. As a future product owner, I want a later option to gate register on “all pallets of load loaded,” so that we can tighten timing without redesigning the group.
29. As a QA engineer, I want Commit tests that prove full reading minus frozen weight, so that delta booking cannot regress.
30. As a QA engineer, I want Load/bundle tests that prove prefilled continuesContainerId and fullyProcessed filtering, so that form seed stays Config-driven.
31. As a QA engineer, I want Worklist tests that prove containerReg unlocks with openTurno and loadedExtracted is absent from live line, so that plant shape stays honest.
32. As a logistic operator, I want new material rows to still use catalogs.materials, so that minting materials stays catalog-driven.
33. As a developer, I want hidden continuesContainerId on continuation rows only, so that new rows mint fresh containers.
34. As an operator, I want first Spawn of a Job instance to have no continuation rows, so that the first load has no secondary/delta behavior.
35. As a platform developer, I want existing Task group Spawn/Close/Unlock grammar reused, so that we do not invent Completeness or multi-done flat Tasks.
36. As a stakeholder, I want pallet-vs-container reconciliation to remain possible from Domain facts after this change, so that Risultati can return later without new booking rules.

## Implementation Decisions

- **Dumb executors (hard constraint):** Go living engine and SPA only execute Org Config (Unlock, readyWhen omit, Task groups, reads, transform, form seed from authored expressions/bindings). No Feragame-named branches, no hardcoded containerReg/lineLoad/load id logic in Go or TypeScript beyond generic mechanisms already used by other Jobs.
- **Plant line shape:** Remove loadedExtracted from the live line Task list. Keep its report/chrome body recoverable in-repo (scratch or non-live parking)—do not delete dataReport engine support.
- **containerReg becomes a Task group:** repeatable group id `containerReg`; single repeatable leaf `register` carrying today’s form/transform (reshaped). `unlockFrom: openTurno` on the group; omit readyWhen.
- **Group lifecycle:** `closeRequiresChildSubmission: true`; no Unlock successor after Close.
- **Instance identity:** Group instance id aligns with chosen `sourceLoadId`; `instanceLabelEntity: load` (lots pattern).
- **Domain load fact:** `fullyProcessed` set true on the primary load in onSubmit after successful register; primary (and any load enum) reads where not fullyProcessed.
- **Domain container facts:** `weight` is the booked kg for that load; `incomplete` boolean; identity stable for continuation; optional link fields as needed for prior container / secondary load attribution—authored in Config transform, not engine special cases.
- **Drop dualSources UX:** No per-row dualSources checkbox; no operator-picked secondarySourceLoadId enum for continuity.
- **Continuation seed:** Load of register after prior incompletes exist seeds array rows with hidden `continuesContainerId` (and display of prior freeze as needed). `incomplete` defaults false/unchecked.
- **Booking math (Config transform):** If `continuesContainerId` present: booked weight = entered full reading − prior container booked weight; supersede prior incomplete. If absent: booked weight = entered weight. Mint/patch via existing transform ops.
- **Secondary attribution:** When continuing, previous load id for reporting comes from the prior container’s `sourceLoadId` (Config), not a free select.
- **Reuse:** `unlockFrom`, Task groups, Worklist unavailable/readyWhen fold, enumReads/reads, bind/mintMany/patch—no Completeness / leftovers product grammar.
- **Glossary/ADR:** Update Domain field vocabulary if needed (`fullyProcessed`, `incomplete`, `continuesContainerId` as Config form fact). Amend plant expectations/tests; no plant id guards that encode Feragame beyond fixture assertions.
- **Seams:** (1) Org Config fixture, (2) Commit/Domain outcomes, (3) Worklist/path Unlock, (4) Task form bundle Load seed/enums.

## Testing Decisions

- Prefer external behavior: Commit changes Domain/Submissions/Worklist statuses; Load returns seeded formData and filtered enums; fixture parse/validate rejects invalid Config.
- Do not assert internal helper names or SPA re-implementation of booking math.
- Modules: living Commit + Domain store tests; Worklist projection; form bundle Load; fixture/plant_flow/seed expectation updates; SPA only if wire/status labels require it (no TS readyWhen/booking twins).
- Prior art: line_remaining_test (container mint), lots group tests, lineLoad unlockFrom/readyWhen plant tests, form_bundle Load seed, Worklist actionable/unavailable suites.
- Guard: tests that fail closed if Go gains plant Task-id special cases for this feature (extend existing engine source guards where appropriate).

## Out of Scope

- Relocating or shipping Risultati finali / loadedExtracted UI in this change.
- readyWhen that requires all pallets of a load loaded before register.
- Unique-load engine enforcement beyond Domain fullyProcessed + reads filter.
- secondaryWeight field; Completeness all/some; Unlock-on-childSubmission for this group.
- Feragame-specific Go/TS; changing Segna avanzamento law except incidental test updates.
- Physical bin QR identity (continuesContainerId from Domain seed is enough).
- Pushing/releasing; Mongo seed wipe policy beyond documenting seed --replace when Active Config must refresh.

## Further Notes

- Floor timing (“register when all pallets of the load are loaded”) is operator discipline until a later Config readyWhen.
- Reconciliation law: per load, Σ pallet net vs Σ Domain container booked weights for that load; continuation stores deltas so sums stay additive.
- Hidden `continuesContainerId` is the row↔prior-container link; previous load alone is insufficient when multiple incompletes exist.
- GitHub twin (optional): #284 — prefer this scratch file as working source of truth on the branch.
