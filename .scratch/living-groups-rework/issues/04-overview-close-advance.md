# 04: Job page — Close group (confirm) and advance

**What to build:** On the Job instance overview, Close on a group asks for confirmation, closes the group, then follows server `next`: the operator’s next ready Task if any, otherwise overview, otherwise home.

**Blocked by:** 03 — Job page — Spawn opens the first form

**Status:** done

- [x] Overview exposes Close when the group may Close
- [x] Confirm dialog before Close; cancel leaves Domain unchanged
- [x] Successful Close refreshes Worklist and navigates from Commit-shaped `next` (next Task / overview / home)
- [x] Closing unlocks the next Job-level work as in plant flows (covered by test or fixture assertion)
