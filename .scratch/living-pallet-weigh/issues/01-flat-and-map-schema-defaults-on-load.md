# 01: Schema defaults seed on every Task Load

**What to build:** Opening any Task form — mapTable or flat — puts authored schema `default` values (and the existing first-enum fallback for required string enums without defaults) into `initialData`. Operators see tare / idoneita / average panel weight defaults without re-deriving them client-side. Map forms keep today’s row seeding behaviour; flat forms stop getting an empty draft solely because they have no dynamic map.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] Flat Task Load includes schema property defaults in `initialData`
- [ ] Map / cell-map Tasks still seed as they do today (no regression)
- [ ] Covered by unit tests for both shapes
