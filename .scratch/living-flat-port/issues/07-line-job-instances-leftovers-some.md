# 07: Line — Job instances + Leftovers some

**What to build:** Multiple line Job instances can exist in Domain. `lineLoad` uses Leftovers `completeness: some` (batch progress only): it must not unlock the next Task and must not be a startsWhen target.

**Blocked by:** 06: Line — startsWhen opens Job

**Status:** done

- [x] Two (or more) line Job instances can be progressed independently
- [x] lineLoad `some` accepts batch Commits without unlocking the next Task
- [x] lineLoad cannot be used as a startsWhen target
- [x] Tests cover multi-instance and `some` non-unlock rules
