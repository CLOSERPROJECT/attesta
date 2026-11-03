# 02: Server — Spawn via first-leaf Commit, Close API, and where-next

**What to build:** Operators open a new group instance by Committing the first child Task with spawn mint (one step); Close finishes the group over HTTP; while the group is still open, a child Commit’s `next` prefers Job instance overview (not the next Job-level Task); after Close, normal next-Task / overview / home applies. Nested Task form hydrate (addressed fetch with group coordinates) works for provisional and real group instances.

**Blocked by:** 01 — Server — Task groups under a Job (Domain, unlock, Worklist)

**Status:** done

- [x] Spawn mint on Commit creates the group instance and records the first leaf Submission in one success path
- [x] Close HTTP exists; Close updates Domain and returns Commit-shaped navigation (`next`, updated Worklist; no bulk home bundles)
- [x] While a group is open, child Commit `next` is overview (unless a later ticket owns stay-for-links in the SPA)
- [x] After Close, `next` follows the usual auto-advance → overview → home law
- [x] Addressed Task form load accepts group + group instance (including provisional `new`) with the same viewer mount rules as flat Tasks
- [x] HTTP / Go tests cover spawn Commit, Close, and nested form fetch; Cerbos still on leaf Commit only
