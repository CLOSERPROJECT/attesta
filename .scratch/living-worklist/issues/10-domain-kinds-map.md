# 10: Domain kinds map

**What to build:** Domain storage and wire use one kind→entities map. Privileged Go fields for load/pallet/container go away. SPA Domain snapshot is dynamic over config kinds. Mongo for living domains may be wiped. Feragame keeps kind names like `load` / `pallet` / `container` as **config nouns**, not Go struct slots. Load / Commit / Worklist still demo.

**Blocked by:** 09 Bind + containerReg flatten.

**Status:** ready-for-agent

**Execution:** sequential subagents only (A → B → C). Expand–contract; green promised after C.

### Subagent 10A — Expand

- [ ] Domain persists a unified kind→map (or equivalent) while existing wire keys for known kinds still round-trip
- [ ] Entity resolution / mutate / clone prefer the unified map
- [ ] Tests prove read/write for both former well-known kinds and Extra-style kinds

### Subagent 10B — Migrate callers

- [ ] Go tests and helpers stop depending on privileged Domain field accessors
- [ ] SPA Domain snapshot / form bindings accept dynamic kinds from the server bag
- [ ] Feragame `$domain.load` / `.pallet` / `.container` exprs still work via kind names

### Subagent 10C — Contract

- [ ] Privileged Load/Pallet/Container Go fields (or equivalents) are deleted
- [ ] Living Mongo volumes/data wiped or migrated; fresh seed works
- [ ] No host API requires the old three-slot Domain shape

**Decisions (grill 2026-11-02):** kill well-known Domain slots in Go; kinds are Org Config vocabulary; Mongo wipe OK.

**Out of scope:** leftovers/`completeness` removal; catalogs (already 08); rename of living package.
