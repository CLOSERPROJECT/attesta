# 04: Intake — Unlock order for simple Tasks

**What to build:** Flat intake Tasks that are not Leftovers-heavy (`codiceFir`, and the simple gate into later intake work) respect Unlock order on the same Job instance. Out-of-order Commit is denied (via attrs → Cerbos). HTTP-demoable after a successful register.

**Blocked by:** 03: Cerbos on living Commit

**Status:** done

- [x] In-order Commit of the simple intake Tasks succeeds and Submissions unlock the next
- [x] Out-of-order Commit is denied; Domain unchanged
- [x] Unlock order is distinct from startsWhen / readyWhen in behaviour
- [x] Tests cover allow and deny ordering
