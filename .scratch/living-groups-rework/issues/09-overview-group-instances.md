# 09: Overview lists group instances then Spawn

**What to build:** On the Job instance overview, when a Config task is a Task group, show that group's existing instances (entries) **inside** the group item — each entry is **id + aggregate status** (no deep links, no durable labels). Then the existing Spawn control (start next instance), then Close when allowed. SPA-only from Worklist `groups[].instances[]`; no server/wire change.

**Blocked by:** —

**Status:** done

**Human (2026-11-03):** ship before double-Commit fix; entry chrome = id + aggregate (not child rows / links).

**Aggregate rule:** from the instance's child `tasks` map (prefer `childTaskIds` order when present): all `done` → `done`; any `available` → `available`; else `locked`.

- [x] `jobInstanceOverview` projects `groupInstances` on group rows (`id` + aggregate `status`); leave `OverviewGroupChrome` spawn/close-only
- [x] `overview.svelte` renders instances between the group status row and Spawn; entries are not linkable
- [x] Unit tests cover at least one group with instances + Spawn still present when `canSpawn`
- [x] No JobPanel; no ADR reopen for durable labels under the lot
