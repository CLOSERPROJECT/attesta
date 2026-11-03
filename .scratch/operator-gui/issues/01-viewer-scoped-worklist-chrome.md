# 01: Viewer-scoped Worklist chrome

**What to build:** An affiliated operator’s living Load/Commit **Worklist** only treats Tasks/Jobs as board-actionable when their **OperatorIdentity** Config roles intersect the Task’s roles (Can start = `canStart` and first Task role-eligible). The projection also carries Config-free Task **title** and **roles** chrome beside existing `taskIds` / `startLabel` / `repeatable`, so the SPA never invents those from a full Org Config. Unlock / startsWhen / `locked`|`available`|`done` stay server-projected (ADR-0005 / ADR-0006); board scope uses slug∩roles, not Cerbos.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] Worklist Available affordances for the viewer exclude Tasks whose Config roles do not intersect OperatorIdentity roles
- [x] Can start is false for the viewer when they cannot complete the Job’s first Task, even if startsWhen would allow a start
- [x] Worklist chrome includes per-Task display title and Config roles (or equivalent) without shipping full Org Config
- [x] Tests cover at least two role-split OperatorIdentity cases on the same Domain (e.g. administration vs logistic)
- [x] SPA Worklist types accept the new chrome fields; existing `/app` still loads
