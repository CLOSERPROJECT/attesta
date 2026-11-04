# 10: Overview group rows show load id (label)

**What to build:** Reopen deferred “labels under the lot on overview”: project an optional **`label`** on each `WorklistGroupInstance` (same chrome pattern as Job instance `label`). For plant lots, `label` is the minted Domain **`load`** entity id for that group instance. Overview entry rows display `label || id` + aggregate status. No plant-specific `lotId` / `loadId` wire field name — value is the load id.

**Blocked by:** 09 (overview instances UI)

**Status:** done

**Human (2026-11-03):** use the load id on the row (not group instance id `lots-N` as the primary title).

**Projection (Go):** when building `WorklistGroupInstance` in `projectWorklistGroup`, set `Label` from Domain: correlate `(jobId, jobInstanceId, groupId, groupInstance.id)` with nested Submissions → prefer an `IDsTouched` id that exists under entity kind `load`; else omit label (SPA falls back to `id`). Keep generic — do not hard-code Feragame task ids in the Worklist DTO shape; join via submissions + entity map is OK for v1 if documented. Prefer not inventing a second SPA Domain walk.

- [x] Go + TS: optional `label` on `WorklistGroupInstance`; tests cover load id after spawn+register
- [x] Overview model/UI: show `label || id` on group entry rows (`data-testid` stable)
- [x] ADR-0006: amend deferred line — overview may show projected group-instance label (load id); durable free-form labels still out of scope unless stated
- [x] No JobPanel; SPA does not read full Domain for this
