# Demo fixture seed invoke stays host-specific

Coolify Preview and local worktree Compose both gate on demo seed and prepare an empty Appwrite + fixture before (or as) the app serves. Shared orchestration must live in one Go prepare/seed seam so hosts do not fork wait/bootstrap/seed logic in shell.

**Decision:** Hosts invoke that seam through thin adapters that only choose an invocation **policy** (preview hard-fails; worktree soft-skips missing credentials/bootstrap). Adapters do not reimplement prepare/seed outside Go. Manual re-seed remains a separate operator entrypoint.

**Rejected:** A shared shell runner or seed-command escape hatch (duplicates gate/wait outside Go); separate soft flags instead of one host policy.
