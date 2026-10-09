# Spec: Feragame compromise candidate — draftable array Substeps + Step seal

Status: ready-for-agent

Source: `docs/feragame/explorations/03-compromise-draft-and-seal.md` (grill outcomes 2026-10-09)  
Branch context: `docs/feragame-grill-compromise`  
Note: A/B candidate for internal compare — not the locked pilot winner.

---

## Problem Statement

Feragame needs to record many interleaved batch-load and UdC registration events during one manually opened production day, keep those records editable until the day is closed, then freeze evidence. Attesta today forces one incomplete Substep at a time and notarizes on Substep complete, so operators cannot draft a day’s event log inside one Organization Step without fighting the product.

## Solution

Build the **compromise** pilot candidate behind an opt-in **draft-and-seal** strategy so **classic Stream logic stays untouched**. Default Streams keep today’s sequential unlock and Complete→done+notarize path. Opt-in Feragame Streams: one Stream instance per production day (create = open); one Step with per-kind Formata array Substeps; parallel in-Step draft saves; no mid-day Complete on kind Substeps; elevated **Seal Step** freezes payloads, notarizes each Substep leaf, and completes the (single-Step) instance. Maximize reuse of shared store, Formata, auth, notarization shape, and instance create. No child-event store. Concurrent multi-writer append is deferred (single registrar for this pilot).

## User Stories

1. As a Feragame operator, I want to create a Stream instance for today’s production day, so that opening the day is an explicit manual act.
2. As a Feragame operator, I want both Batch loads and UdC registrations Substeps available right after create, so that I am not blocked by sequential unlock inside the Step.
3. As a Feragame operator, I want to append a batch-load row during the morning, so that incoming loads are recorded when they happen.
4. As a Feragame operator, I want to append a UdC registration row at any station at any time, so that fill rates that differ across stations do not force false ordering.
5. As a Feragame operator, I want to switch freely between kind Substeps during the day, so that interleaved station work matches the line.
6. As a Feragame operator, I want draft saves that do not mark a Substep done, so that I can keep editing rows until the day is closed.
7. As a Feragame operator, I want draft saves that do not notarize, so that mid-day captures are not frozen evidence.
8. As a Feragame operator, I want to edit a mistaken row before seal, so that typos and wrong weights can be fixed.
9. As a Feragame operator, I want to remove a mistaken row before seal, so that cancelled or duplicate events do not stay in the day log.
10. As a Feragame operator, I want validation feedback when I save an invalid row, so that bad data is caught before seal, not after confirm.
11. As a Feragame operator, I want to seal the day with zero rows on a kind if the line stopped early, so that an empty kind does not block close.
12. As a Feragame supervisor, I want a distinct Seal Step action, so that closing the day is not “complete the last form.”
13. As a Feragame supervisor, I want Seal Step to require confirmation, so that accidental freeze is hard.
14. As a Feragame supervisor, I want Seal Step to be irreversible, so that sealed evidence matches C3 immutability.
15. As a Feragame supervisor, I want Seal Step to freeze every kind Substep in the Step together, so that the day’s evidence commits as one unit.
16. As a Feragame supervisor, I want Seal Step to complete the Stream instance when this is the only Step, so that I do not need a second “close process” ritual.
17. As a station operator without elevated standing, I want to be unable to seal the Step, so that only supervisors close the day.
18. As a station operator, I want to keep drafting rows with ordinary Substep roles, so that day-to-day capture does not need admin rights.
19. As a Feragame operator, I do not want a Complete button on kind Substeps during the day, so that I am not taught the wrong commit model.
20. As a Feragame operator, I want the timeline to show both kind Substeps as drafting/available, so that parallel work does not look like a broken sequence lock.
21. As a Feragame operator, I want Seal Step visible as a Step-level control, so that close is clearly not another kind Substep.
22. As a Feragame operator, I want post-seal views to show kind Substeps as finished/immutable, so that I know the day is closed.
23. As a Feragame operator, I want post-seal edits and appends rejected, so that sealed evidence cannot drift.
24. As an auditor, I want sealed evidence to keep separate payloads per kind Substep, so that batch loads and UdC registrations remain distinguishable (C5).
25. As an auditor, I want notarization of each sealed Substep payload at seal time, so that integrity matches Attesta’s Substep-leaf export shape.
26. As an auditor, I want notarization timestamps to reflect seal time, so that commit time is not confused with optional per-event times in row data.
27. As a Feragame operator, I want optional event-time or shift fields only if the form schema includes them, so that schema stays minimal and notarization does not require `registered_at`.
28. As a Feragame operator, I want shifts (if recorded) to live as data inside the day instance, so that I do not open a new Stream instance per shift.
29. As a Feragame operator, I want cross-midnight work to stay on the same instance until seal, so that the business day is not forced to clock midnight.
30. As a platform admin configuring the pilot Stream, I want a one-Organization, one-Step blueprint with per-kind array Substeps, so that the pilot matches the agreed blueprint shape.
31. As a platform admin, I want no ceremonial open/close Substeps in that blueprint, so that lifecycle stays instance create + Step seal.
32. As a future multi-org Stream designer, I want inter-Step sequence preserved, so that a later Organization’s Step still waits on Feragame’s sealed Step.
33. As a product owner running internal A/B, I want this implementation to remain clearly the compromise candidate, so that choosing the pilot winner stays a later decision.
34. As a single Feragame registrar, I want simple whole-array draft replaces to be acceptable for now, so that concurrent-append engineering can wait until a second writer appears.
35. As a developer, I want draft progress stored in the existing progress map, so that we do not invent a parallel draft store.
36. As a developer, I want kind Substeps to refuse ordinary complete-and-notarize mid-day, so that seal remains the only commit path for those Substeps.
37. As a Feragame operator, I want to add more event kinds later as additional array Substeps of the same shape, so that new kinds do not require a new multiplicity model.
38. As a Feragame operator, I want station identity on UdC rows as form fields (not station Substeps), so that this candidate stays `per-kind`.
39. As a denied actor, I want unauthorized draft saves rejected, so that role and Organization gates still apply.
40. As a denied actor, I want unauthorized Seal Step rejected, so that elevated seal auth is enforced.
41. As a Feragame supervisor, I want seal to fail closed if the Step is already sealed, so that double-seal cannot rewrite evidence.
42. As a Feragame operator, I want sealed notarized JSON / downloads to still expose Step → Substep structure, so that existing export expectations keep working.
43. As a Feragame supervisor confirming seal, I want the confirm step to freeze current drafts even if some kinds are empty, so that “freeze what’s there” is the rule.
44. As a Feragame operator viewing a sealed day, I want clear drafting-vs-sealed language, so that Attesta’s usual “complete Substep to advance” mental model does not mislead.
45. As an internal evaluator, I want this spike comparable to the minimal last-Substep candidate, so that A/B can judge honesty of seal vs cheaper theatre.
46. As a platform operator of existing Streams, I want classic sequential Complete behavior unchanged, so that Feragame’s candidate does not regress current Streams.
47. As a developer, I want classic completion and availability kept as their own strategy implementation, so that draft-and-seal does not sprinkle mode branches through the current path.
48. As a developer, I want draft-and-seal to reuse progress store, Formata, auth gates, notarization records, and instance create, so that only divergent behavior is duplicated.

## Implementation Decisions

- **Candidate scope only.** Implement exploration `03` as an A/B spike/candidate. Do not mark it the sole pilot architecture in product docs or ADR unless a later decision says so.
- **Two strategies; classic untouched.** Default strategy = **classic** (today’s flat unlock + Complete→done+notarize + instance done when all Substeps done). Opt-in strategy = **draft-and-seal** (this candidate). Select once at the edge from a Stream blueprint flag (prefer Stream-level for the pilot). Do **not** rewrite classic functions with scattered mode `if`s — keep current completion/availability as the classic strategy body; add parallel draft-and-seal implementations for the divergent parts.
- **Maximize shared pieces.** Both strategies reuse: progress map + store updates, Formata schemas/payloads/display flattening, role and Organization gates, notarization insert + Substep-leaf export shape, Stream instance create, progress-key encoding, instance detail chrome that is mode-agnostic.
- **Divergent pieces (draft-and-seal only).** In-Step parallel availability/sequence checks; draft save (non-`done` progress); absent mid-day Complete on kind Substeps; Seal Step (elevated auth, confirm, freeze, notarize leaves, reject further edits, complete single-Step instance); timeline/UI for parallel drafting + Seal control.
- **Domain vocabulary.** Use Stream, Stream instance, Step, Substep, Organization, notarization as in `CONTEXT.md`. Feragame terms (UdC, production day, batch load) as in `docs/feragame/README.md`.
- **Blueprint.** Opt-in Stream: one Organization, one Step, ≥2 Formata Substeps (Batch loads, UdC registrations) with array item schemas. No open/close Substeps. Strategy = draft-and-seal.
- **Open.** Creating the Stream instance opens the day.
- **In-Step availability (draft-and-seal).** Within an unsealed Step, all incomplete kind Substeps are available together. Predecessor locking remains across Steps only (`steps-only`). Classic availability stays sequential.
- **Draft progress.** Persist Substep Formata payloads with a non-`done` progress state via the shared progress-update path (same document shape). No parallel draft store. Exact state token (`draft` / `in_progress` / etc.) is an implementation choice.
- **No mid-day Complete on kind Substeps.** Ordinary complete+notarize is absent/disabled for draft-and-seal kind Substeps; Seal Step owns the terminal transition. Classic Complete path unchanged.
- **Seal Step.** New Step-scoped action on draft-and-seal Streams: elevated auth, confirm, freeze all kind Substep payloads in the Step, mark them terminal/done (or equivalent sealed terminal), notarize each Substep payload leaf, reject further edits, and for the pilot single-Step Stream mark the Stream instance complete.
- **Seal granularity.** Whole-Step, irreversible; empty kinds allowed; no unseal / partial seal.
- **Validation.** Enforce per-row/field rules on draft save. Seal does not impose minimum event counts.
- **Notarization.** One notarization per sealed Substep leaf at seal time; digest over sealed payload; `CreatedAt` = seal time. Compatible with existing Substep-oriented notarized export. No per-event notarization. Shared notarization helpers/types where possible.
- **Row schema.** No mandatory `registered_at` or `shift`. Station (and any times/shifts) only if present as ordinary Formata fields.
- **Concurrency.** Single registrar assumed; whole-array last-write-wins acceptable. Do not build row-level append/ETags in this candidate unless needed for the Formata UX under one writer.
- **Auth.** Draft/edit: existing Substep role + Organization checks. Seal: narrower elevated standing (supervisor / Org admin–class — exact role wiring is implementation detail).
- **UI.** For draft-and-seal instances: timeline shows parallel drafting Substeps; Seal is a Step-level control. Classic UI/flow unchanged for classic Streams.
- **Progress key encoding.** Preserve Mongo progress-key dot→underscore boundary on write/read.
- **Primary module seam.** A small strategy selection at the process-completion / instance edge: classic strategy = existing `CompleteSubstep` + classic availability; draft-and-seal strategy = draft-save + Seal Step (+ its availability). Shared store, digest/notarization insert, and auth helpers underneath. Avoid a second workflow engine or child-event store.
- **Out of this candidate’s storage model.** No child-event entities, no re-enter-Substep multiplicity, no `per-station` Substep mapping.

## Testing Decisions

- Test **external behavior** per strategy: given Stream instance + blueprint (including strategy flag) + actor, assert progress state, payload mutability, notarization presence/absence, availability, and auth outcomes. Do not assert internal helper call graphs or Mongo encoding details except where the progress-key boundary is a documented invariant.
- **Primary seam under test:** strategy selection at the process-completion domain edge — classic path still matches today’s completion tests; draft-and-seal path covers draft save + Seal Step with in-memory store, fixed clock, seeded instances.
- **Regression gate:** existing classic completion and availability tests remain green without modification of expected classic behavior (add new draft-and-seal tests alongside, do not weaken classic assertions).
- **Secondary seam (only if not covered via the service):** draft-and-seal availability helpers for parallel in-Step vs cross-Step lock; classic availability tests stay on the classic helpers.
- **HTTP/integration prior art:** existing complete/integration tests for classic; parallel cases for draft (no notarization) and seal (notarizations + immutability + instance complete) on opt-in Streams.
- **Good tests include:** classic Stream still one-at-a-time + Complete notarizes; draft-and-seal draft does not notarize; seal notarizes each kind leaf once; second seal rejected; non-elevated actor cannot seal; post-seal draft rejected; parallel availability before seal; empty-kind seal succeeds; ordinary complete on draft-and-seal kind Substeps rejected or unavailable; strategy flag selects behavior.
- Prefer the strategy edge as the behavioral seam; add minimal UI/HTMX coverage only for draft-and-seal chrome (“Complete absent / Seal present / both Substeps available”) if templates can regress independently.

## Out of Scope

- Choosing this candidate as the final Feragame pilot architecture (A/B vs at least the minimal exploration comes later).
- Changing classic Stream completion or availability behavior (classic strategy must keep today’s semantics).
- ADR promoting Step seal as general Attesta product law.
- Child-event log / first-class event store (architectural exploration).
- Last-Substep-as-seal workaround (minimal exploration).
- Concurrent multi-writer append, row-level merge, ETags.
- Per-event or per-kind notarization / seal.
- Unseal, reopen, or partial seal.
- Mandatory `registered_at` or `shift` fields.
- Modeling stations as Substeps (`per-station`).
- Multi-Organization Streams beyond preserving inter-Step sequence for a future Step after seal.
- Clock-based auto-close at midnight or shift change.
- Rewriting Formata builder architecture beyond schemas needed for array kind Substeps.

## Further Notes

- Grill settlements live in `docs/feragame/explorations/03-compromise-draft-and-seal.md` §5.
- Hub index marks this exploration as a grilled A/B candidate, not chosen.
- If a second registrar appears, revisit concurrency before treating last-write-wins as acceptable beyond the pilot assumption.
- Ask before breaking routes, workflow configuration, or persistence shapes beyond this candidate’s draft + seal surface (`AGENTS.md`).
