# 04: Worklist Home + Task routes

**What to build:** Living `/app` landing is **Home**: two always-visible bands (**Available**, **Can start**) from the viewer-scoped Worklist—flat Available rows keyed by `jobId` / `jobInstanceId` / `taskId`, sorted Job → instance → Task; Can start by Job order with `startLabel`. Opening Available or Start goes to `/app/{jobId}/{jobInstanceId|new}/{taskId}` with a lazy Task form bundle (Start = provisional first Task; mint on Commit). Abandon/back without Commit returns to Home; Draft behavior unchanged. Replace the JobPanel board; remove About from nav. No Draft badge on home; no locked/done rows on home.

**Blocked by:** 01 (Viewer-scoped Worklist chrome), 03 (Lazy Task bundle + Commit navigation payload)

**Status:** done

- [x] `/app` shows Home with Available + Can start bands (empty-state copy when a band has no rows)
- [x] Available rows use three-part identity and the agreed sort; display uses path/label + Task title chrome as designed
- [x] Can start uses `startLabel` and opens provisional `/…/new/{firstTaskId}`
- [x] Task route loads one bundle and supports Commit; Annulla/back → Home; Draft persists as today
- [x] JobPanel-style inline multi-form board is gone; About nav entry is removed
- [x] Home Load does not depend on bulk Task form bundles
- [x] Basic e2e or component coverage for Home → Task → back
