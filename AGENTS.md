# AGENTS.md

## Working here

- Keep changes minimal and localized. Match established patterns after reading the
  relevant code, especially `server/cmd/server/main.go` for HTTP behavior.
- Ask before making a breaking change to routes, workflow configuration, or
  persistence.
- Preserve unrelated work in a dirty tree. Do not commit, push, or change
  external systems unless asked.
- **Comments are *why*, never *what*.** Keep a comment only when deleting it
  would lose non-obvious intent, a constraint, or a gotcha the code cannot
  express. In every region you touch, delete comments that restate identifiers,
  types, or control flow. Prefer renaming or restructuring over explaining the
  obvious. Narration, section banners, and commented-out code are out.
- **Titles:** commits, issues, and pull-request titles use
  [Conventional Commits](https://www.conventionalcommits.org/):
  `type(optional-scope): imperative summary` (lowercase type; typical types
  `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`, `build`, `perf`).
  Enforce this convention; do not infer title style from recent `git log`.

## Source guidance

- **Architecture:** for system boundaries or where behavior lives, read
  [`docs/architecture.md`](docs/architecture.md).
- **Operations:** for setup, commands, ports, Docker, or worktrees, read
  [`README.md`](README.md), [`QUICKSTART.md`](QUICKSTART.md), and
  [`DOCKER.md`](DOCKER.md); scripts and configuration remain the source of
  truth.
- **Domain language:** before naming product nouns or editing a glossary —
  pick the layer (shared membership, Org `/app`, Stream `/my`, Feragame plant)
  via [`GLOSSARY-MAP.md`](GLOSSARY-MAP.md).
- **ADRs:** before writing or editing `docs/adr/`, read
  [`docs/adr/README.md`](docs/adr/README.md).
- **Active work:** unresolved design and implementable work live as GitHub
  issues; use [`docs/agents/issue-tracker.md`](docs/agents/issue-tracker.md)
  when creating, updating, triaging, or wayfinding tickets.
- **Domain exploration:** before codebase exploration that depends on glossary
  or ADR vocabulary, read
  [`docs/agents/domain.md`](docs/agents/domain.md).
- **Org Config authoring:** before changing platform Config gates, `onCommit`,
  or transform authoring patterns, read
  [`docs/agents/org-config-authoring.md`](docs/agents/org-config-authoring.md).
- **Feragame fixture authoring:** before editing
  `server/internal/org/fixtures/feragame-config.yaml` (or its seed/test
  followers), read
  [`docs/agents/feragame-fixture-authoring.md`](docs/agents/feragame-fixture-authoring.md)
  and [`docs/feragame/GLOSSARY.md`](docs/feragame/GLOSSARY.md).
- **DPP/UNTP:** for Digital Link or credential behavior, read
  [`docs/untp.md`](docs/untp.md) and the implementation it names.
- **Templates and CSS:** before changing `server/templates/` or
  `web/src/styles/`, read [`docs/css.md`](docs/css.md) and use the
  `attesta-ui-components` skill for component extraction or placement.

## Invariants

- Org Go (`server/internal/org`) and the `/app` SPA are **dumb executors** of Org Config: plant
  business rules (entity shapes, transforms, `readyWhen`, open/close laws) live
  in Config, not as Feragame- or domain-named branches in executors. See
  [`docs/org/GLOSSARY.md`](docs/org/GLOSSARY.md) (**Dumb executor**) and
  [`docs/agents/org-config-authoring.md`](docs/agents/org-config-authoring.md).
- Progress keys containing dots are encoded for Mongo storage (`.` becomes
  `_`) and normalized on reads. Preserve that boundary when changing progress
  persistence.
- Format and check Go templates with djLint (`task templates:fmt`,
  `task templates:check`, `task templates:lint`); do not use a generic HTML
  formatter on Go templates.
- Every worktree owns a Compose project and generated-once host ports in its
  ignored `.env.worktree`; later manual edits are preserved. Compose project
  names are path-qualified for every checkout (primary, linked, or independent
  clone). Use `scripts/worktree-compose.sh` (normally through Taskfile tasks);
  stop/reset/purge must remain scoped to the current checkout.
- Demo data for empty stacks comes only from the Go seed (`attesta seed` /
  `ATTESTA_SEED`). There is no worktree volume-copy of MongoDB or Appwrite data.
  `task reset` / `task purge` clear this checkout's volumes; the next start can
  re-apply the Go demo seed when credentials exist. When a worktree manager
  runs a pre-remove hook, Compose volumes are deleted with the checkout;
  otherwise run `task reset` / `task purge` before removal, or
  `task worktree:gc` / `task worktree:gc:apply` for orphaned resources.
