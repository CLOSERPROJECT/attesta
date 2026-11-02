# 05: Job instance overview + post-Commit routing

**What to build:** After Commit, route by payload: auto-advance into the next role-eligible `available` Task on the same Job instance; else open **Job instance overview** at `/app/{jobId}/{jobInstanceId}` with Task statuses, handoff copy when the next work is for other Config roles, and links when a Task is available and yours; if the instance is finished, go to Worklist Home. Deep links to a Task the viewer cannot do land on overview (or Home if unknown). Overview is reachable from Task/Available exit as well as post-Commit handoff.

**Blocked by:** 03 (Lazy Task bundle + Commit navigation payload), 04 (Worklist Home + Task routes)

**Status:** ready-for-agent

- [ ] Overview route renders instance label + per-Task statuses from Worklist projection
- [ ] Handoff copy names Config roles when next `available` work is not role-eligible for the viewer
- [ ] Role-eligible `available` Tasks on overview link to the Task route; locked/done are status-only
- [ ] Post-Commit navigation implements auto-advance → overview → Home per ADR-0006
- [ ] Unauthorized/locked Task URL resolves to overview or Home, not an empty form
- [ ] Coverage for advance, handoff, and finished-instance paths (tests and/or e2e)
