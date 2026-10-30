# 17: Server — readyWhen → Cerbos attr

**What to build:** Evaluate Task `readyWhen` (Domain expression) in Go and pass a compact boolean (or small attrs) into Cerbos on Commit. Cerbos remains sole EFFECT_ALLOW/DENY. No full DomainSnapshot dump.

**Blocked by:** 16: Server — config-driven Commit (ELT)

**Status:** done — reviewer APPROVE; squash into rework; human smoke

- [x] readyWhen false → Commit denied via Cerbos; Domain unchanged
- [x] readyWhen true (with roles + sequenceOk) → Commit allowed when otherwise eligible
- [x] Attrs stay compact; hybrid rule matches decisions dump
- [x] Tests cover deny/allow with fixture or synthetic readyWhen

**Refs:** ticket 15 server table; decisions Authorization; experiment `ready-when`.
