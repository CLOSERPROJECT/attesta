# 08: Org catalogs

**What to build:** Org Config gains an authored `catalogs` bag for stable plant knowledge (materials, tareKinds, clients, …). Expr scope exposes `$catalogs`. Feragame stops inlining those lists in derived/chrome/transforms; Task-local `tareKinds` injection into form Domain bindings goes away. Demo plant still loads and commits.

**Blocked by:** None (can start immediately).

**Status:** done (squashed into `feat/attesta-rework`)

**Execution:** sequential subagents only (finish A before B).

### Subagent 08A — Kernel catalogs

- [x] Org Config accepts `catalogs`; Commit/Load expr bindings include `$catalogs`
- [x] Task-local `tareKinds` is no longer copied into form `bindings.domain` by the host
- [x] Go tests cover `$catalogs` visibility and absence of tareKinds injection

### Subagent 08B — Feragame migration

- [x] Feragame materials, tareKinds, and clients live under `catalogs`
- [x] Derived / chrome / transforms that used inlined lists read `$catalogs.*`
- [x] Living fixture / Commit paths that depended on those lists stay green

**Decisions (grill 2026-11-02):** catalogs ≠ Domain entities; schema/enum codegen from catalogs is later; Go stays dumb — catalogs are config data only.

**Out of scope:** `bind` op; Domain kind map; leftovers/`completeness` removal; schema generation from catalogs.

**Landed:** squash on integration after review+fix (`$catalogs` on ResolveRead/submit-http; glossary Org catalogs).
