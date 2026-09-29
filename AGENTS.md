# AGENTS.md

## Working here

- Keep changes minimal and localized. Match established patterns after reading the
  relevant code, especially `server/cmd/server/main.go` for HTTP behavior.
- Ask before making a breaking change to routes, workflow configuration, or
  persistence.
- Preserve unrelated work in a dirty tree. Do not commit, push, or change
  external systems unless asked.

## Source guidance

- **Architecture:** for system boundaries or where behavior lives, read
  [`docs/architecture.md`](docs/architecture.md).
- **Operations:** for setup, commands, ports, Docker, or worktrees, read
  [`README.md`](README.md), [`QUICKSTART.md`](QUICKSTART.md), and
  [`DOCKER.md`](DOCKER.md); scripts and configuration remain the source of
  truth.
- **Domain language:** for product terms, read [`CONTEXT.md`](CONTEXT.md).
- **Open work:** for deferred design or migration work, read
  [`docs/open-work.md`](docs/open-work.md).
- **DPP/UNTP:** for Digital Link or credential behavior, read
  [`docs/untp.md`](docs/untp.md) and the implementation it names.
- **Templates and CSS:** before changing `server/templates/` or
  `web/src/styles/`, read [`docs/css.md`](docs/css.md) and use the
  `attesta-ui-components` skill for component extraction or placement.

## Invariants

- Progress keys containing dots are encoded for Mongo storage (`.` becomes
  `_`) and normalized on reads. Preserve that boundary when changing progress
  persistence.
- Format and check Go templates with djLint (`task templates:fmt`,
  `task templates:check`, `task templates:lint`); do not use a generic HTML
  formatter on Go templates.
- Every worktree owns a Compose project and generated-once host ports in its
  ignored `.env.worktree`; later manual edits are preserved. Compose project
  names are path-qualified for every checkout (primary, linked, or independent
  clone). Pre-worktree `deployment_*` volumes need a one-time
  `COMPOSE_PROJECT_NAME=deployment` in the primary `.env.worktree`. Use
  `scripts/worktree-compose.sh` (normally through Taskfile tasks); stop/reset/purge
  must remain scoped to the current checkout.
- Parallel worktrees keep one point-in-time, all-or-nothing bundle of MongoDB
  and durable Appwrite state in ignored `.worktree-data/`. Restore only when the
  stack is stopped and durable volumes are empty; later starts keep that
  worktree's data. `task reset` clears volumes so the next start re-applies the
  saved bundle. Reject partial or corrupt bundles. Bootstrap fails closed if the
  primary's managed project has no durable volumes while legacy `deployment`
  volumes still exist. Worktrunk `pre-remove` deletes Compose volumes.
