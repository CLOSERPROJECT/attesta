# Open work

This repository has no configured issue tracker. This document is the local
register for unresolved work; remove an item when it lands or move it to the
chosen tracker if one is adopted.

## Transactional completion durability

**Current behavior:** `CompleteSubstep` writes process progress, then a
notarization, then finalizes status and DPP artifacts. A notarization failure
after the progress write leaves the substep done and returns an error. Status
and DPP finalization failures are logged, and
`EnsureCompletionArtifacts` repairs completion artifacts on later reads.

**Desired result:** progress and its notarization either persist together or
the operation compensates reliably; completion status and DPP artifacts have a
clear, observable repair contract.

**Revisit when:** support reports repeatedly show completed substeps without a
notarization, repair behavior is confusing, or Mongo transactions are already
introduced elsewhere.

## Remaining terminology migration

Align implementation names with the canonical vocabulary in
[`CONTEXT.md`](../CONTEXT.md). The remaining migration covers `Workflow` versus
stream, `Process`/`ProcessStep` versus stream instance and substep progress,
and legacy process-oriented template and route names. Scope a rename as a
dedicated compatibility-aware change; routes and persistence require explicit
approval before breaking changes.

## Timeline UX, performance, and deepening

- Add step-level progress or rollup status to timeline headers.
- Preserve open accordion state across SSE refreshes and decide whether step
  accordions are single-open or multi-open by design.
- Replace the timeline's `HideStatus` flag with an explicit display mode and
  extract the reusable timeline-step template when that seam next changes.
- Consider lazy substep bodies and targeted SSE partials only when large
  timelines or refresh latency make them necessary.
- Clarify whether the dashboard's blueprint dialog should remain named a
  timeline preview.
