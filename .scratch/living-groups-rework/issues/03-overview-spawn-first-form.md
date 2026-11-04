# 03: Job page — Spawn opens the first form

**What to build:** On a Job instance overview, Spawn for a Task group navigates to the Task chrome for the first child with a provisional group instance, operator fills and Commits with spawn mint, and lands per server `next` (overview while the group stays open when there are no download links). Deep links use a path that includes job, group, group instance (`new` when provisional), and task. JobPanel is not restored.

**Blocked by:** 02 — Server — Spawn via first-leaf Commit, Close API, and where-next

**Status:** done

- [x] Overview shows Spawn for groups that may Spawn under this Job instance
- [x] Spawn navigates to Task route with provisional group instance and loads the first-child Task form
- [x] Successful first-leaf Commit uses spawn mint and updates Worklist; without download links, operator returns to overview
- [x] Nested draft / path addressing works for provisional and minted group instances
- [x] Automated UI or session-level tests cover the happy path; no JobPanel
