# 14: Worklist four-value status (`actionable`)

**What to build:** Worklist Task status on the wire is `locked | available | actionable | done`. Unlock/Submission law still yields locked/unlocked/done; unlocked splits by Config-role eligibility into `available` (not for this viewer) vs `actionable` (Per te). Home Available, task-form mount, and overview navigation use `actionable`. Retire `viewerAvailable[]`. Group-occurrence aggregate prefers any child `actionable`, else `available`, else all `done`, else `locked`.

**Blocked by:** None (can start immediately)

**Status:** done

- [x] Go projects four-value status on Job instance Tasks and group-occurrence child Tasks
- [x] `viewerAvailable` removed from Worklist wire + SPA types
- [x] Home Available / CollectAvailable / mount gates read `actionable`
- [x] SPA overview/home treat `actionable` as the navigable/mine state
- [x] Group-occurrence aggregate prefers `actionable` over mere `available`
- [x] Tests cover role-split: same unlock → `actionable` for eligible viewer, `available` for other
