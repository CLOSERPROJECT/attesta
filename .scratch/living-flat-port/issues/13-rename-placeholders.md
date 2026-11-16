# 13: Rename placeholders (discussion + code)

**What to build:** Hard-cut rename to names locked in `13-rename-placeholders-discussion.md`: Go `internal/org`, SPA `lib/org`, `/api/org/…` only, Cerbos `task`/`commit`, types `DomainLoader` / `TaskCommitter` / `API` / `DPPStore`, Mongo `org_*` / `org_dpps`, Feragame `*@feragame.com` emails. No backwards compatibility. Preserve Stream `/my`.

**Blocked by:** Quiet tree (no parallel agents editing living seams).

**Status:** names locked + hard-cut amended 2026-11-16 — implement now on a quiet tree.

- [x] Agreed lasting names recorded
- [x] Package, Interfaces, HTTP paths, and Cerbos kind renamed consistently
- [x] Client Adapters and seed/compose/env paths updated
- [x] Flat-port smoke (Load + intake + line) still passes under new names
