# 18: Single-child group occurrence flattens to child content

**What to build:** On Job instance overview, when a **Task group** has exactly one **child Task**, each group occurrence does **not** nest a one-item child list. The occurrence presents that child’s interaction directly: `actionable` → navigate to that child Task form; `done` → accordion body is that child’s recap (facts+links); `locked`/`available` → status chrome only. Occurrence title stays `label || id` (instance identity). Multi-child groups unchanged (occurrence expands to child Task rows).

**Blocked by:** None (can start immediately; builds on 17)

**Status:** implemented (uncommitted)

- [x] Detect single-child via `childTaskIds.length === 1` (or equivalent overview model)
- [x] Single-child occurrence: no nested child Task row list
- [x] Same status/nav/recap rules as a leaf Task, addressed with group + occurrence ids
- [x] Multi-child groups keep nested child rows
- [x] Unit coverage for single-child vs multi-child
