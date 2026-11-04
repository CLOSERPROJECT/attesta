# 06: Home Completed band (below In progress)

**What to build:** Worklist home gains a **Completed** band **below In progress**: finished Job instances (Config `taskIds` progress complete), deep-link to Job instance overview where done-Task accordion already shows stored post-Commit links (incl. Passport/DPP). Operators rediscover DPP without relying on Commit flash only.

**Blocked by:** None (platform overview links already exist; Passport smoke motivated this).

**Status:** ready-for-agent

- [x] Home shows **Completed** band directly under **In progress**
- [x] Rows = finished Job instances (inverse of In progress progress rule); Job → instance order matches In progress
- [x] Row navigates to Job instance overview (same target pattern as In progress)
- [x] Amend ADR-0006: home bands = Available + Can start + In progress + **Completed**
- [x] Unit tests for row projection; no plant-/passport-named SPA branches
- [x] Italian/UI copy consistent with existing home bands (e.g. Completati / Completed — match locale patterns in LivingShell)
