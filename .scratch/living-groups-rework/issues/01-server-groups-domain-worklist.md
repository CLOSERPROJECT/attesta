# 01: Server — Task groups under a Job (Domain, unlock, Worklist)

**What to build:** An org Domain can hold Task group instances under a Job instance; unlock and “group open/closed” rules match the settled Task group model; the viewer-scoped Worklist can describe each group (including whether the operator may Spawn or Close). Proven with Go tests. Product Commit effects still use Sidecar HttpPort (never restore FakeHttp as the product default).

**Blocked by:** None (can start immediately).

**Status:** done

- [x] Domain persists open/closed group instances under a Job instance without breaking existing flat Jobs
- [x] Unlock / path availability treats group-closed and nested children per settled law (shared with later Commit)
- [x] Viewer Worklist projection includes group chrome (can Spawn / can Close, instances, child statuses) under each Job instance
- [x] Go tests cover spawn/close domain verbs and Worklist shape; product HttpPort wiring unchanged (Sidecar / env soft-fail)
