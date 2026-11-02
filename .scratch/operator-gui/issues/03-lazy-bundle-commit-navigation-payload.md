# 03: Lazy Task bundle + Commit navigation payload

**What to build:** Operators can hydrate a single **Task form bundle** by Task address (including provisional start with empty/`new` instance). Successful **Commit** returns an updated viewer-scoped **Worklist** and, when auto-advance applies (another role-eligible `available` Task on that Job instance), the next Task form bundle—so the SPA can route without a blind full home re-Load. Home Load must not *require* bulk bundles for every Available Task (expand: add addressed fetch + Commit fields; contract bulk once Home/Task UI consumes them—ADR-0006).

**Blocked by:** 01 (Viewer-scoped Worklist chrome)

**Status:** done

- [x] An authenticated living API can return one Task form bundle for `jobId` + `taskId` + `jobInstanceId` (provisional start supported)
- [x] Bundle endpoint denies or does not mount work the viewer cannot role-complete (aligned with board scope)
- [x] Successful Commit response includes viewer-scoped Worklist
- [x] When the same operator has a next role-eligible `available` Task on the instance, Commit includes that Task’s form bundle (or unambiguous ref + bundle)
- [x] When auto-advance does not apply, Commit still returns Worklist suitable for overview/home routing without inventing Unlock client-side
- [x] Tests cover advance vs handoff vs finished-instance payload shapes
- [x] Home Load can omit bulk `taskFormBundles` without breaking the new fetch path (old bulk may remain briefly only if still required by pre-04 UI)

## Notes for 04

- `GET /api/living/task-form?jobId=&taskId=&jobInstanceId=` (+ `new`/empty provisional) is the on-demand mount path.
- Commit wire adds `nextTaskFormBundle` (omit when no advance).
- Home Load omits bulk `taskFormBundles` (ticket 04); Commit may still ship interim bulk until post-Commit routing (05) drops it. Task mounts use addressed fetch.
