# 03: Plant — fullyProcessed load filter

**What to build:** After a successful `register` Commit, the primary load is marked `fullyProcessed` on Domain. Load enums on the next Spawn omit fully processed loads so logistic cannot re-select them. Authored entirely in Org Config (transform + reads/enumReads); dumb executors only.

**Blocked by:** 02 (containerReg group + register leaf).

**Status:** done

- [x] Successful `register` sets `load.fullyProcessed` on the chosen primary load
- [x] Subsequent `register` Load omits those loads from primary (and any shared load enum used the same way)
- [x] Tests: two Spawns — second cannot pick the first load; Domain fact visible
- [x] No plant-named Go beyond fixture/tests
