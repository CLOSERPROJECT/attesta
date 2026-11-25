# AGENTS.md

## Sessions

**Interactive** (humans, Cursor, and other harnesses you drive): ask before a breaking change to routes, workflow configuration, or persistence. Preserve unrelated work in a dirty tree. Commit, push, and change external systems only when the person asked.

**Superagent** (`@forkboteu` tickets): follow [`.superagent/protocol.md`](.superagent/protocol.md). That protocol overrides the interactive commit/push/merge gates for those tickets only. Open Superagent issues with the `write-superagent-issues` skill.

## Working here

- Keep changes minimal and localized. Match established patterns after reading the relevant code.
- **Comments are _why_, never _what_.** Keep a comment only when deleting it would lose intent, a constraint, or a gotcha the code cannot express. In every region you touch, delete comments that restate identifiers, types, or control flow. Prefer rename or restructure over narrating the obvious.
- **Titles:** commits, issues, and PRs use [Conventional Commits](https://www.conventionalcommits.org/): `type(optional-scope): imperative summary` (lowercase type; typical `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`, `build`, `perf`). Enforce this; do not infer style from recent `git log`.
- **Commits:** Interactive only. When asked to commit, partition the dirty tree by independent *why* (separately reviewable or revertable), not by path; leave unrelated dirty work untouched. One Conventional Commit per concern—stage with path or hunk adds, in dependency order. Keep mechanically coupled edits together (feature + its test, template + CSS, fixture YAML + seed followers). Split when concerns would differ in Conventional type/scope or review story, not per file. Each commit leaves the tree buildable (required on `main`/`master`; preferred on feature branches). On an ambiguous partition, propose the commit list once and wait. One commit only when the person explicitly asks for a single commit.
- **Markdown:** flow prose; do not hard-wrap at a column. Soft-wrap in the editor. Keep intentional blank lines, lists, tables, and fences.
- **Toolchain:** pinned in `mise.toml`. Run `mise install` in the checkout before install/test/lint/build; use mise shims or `mise exec --`. Add or bump a tool in `mise.toml` in the same PR that needs it.

## Source guidance

- **Architecture:** how product and package seams connect → [`docs/architecture.md`](docs/architecture.md).
- **Operations:** pitch/env → [`README.md`](README.md); day-1 prepare/seed/worktrees (Cursor: native `/worktree`, not in-repo `.worktrees/`) → [`QUICKSTART.md`](QUICKSTART.md); Compose/Coolify/lifecycle → [`DOCKER.md`](DOCKER.md). Scripts and configuration remain the source of truth.
- **Glossary layer:** before naming product nouns → [`GLOSSARY-MAP.md`](GLOSSARY-MAP.md).
- **Glossary edit:** before adding or changing an entry → [`docs/agents/glossary.md`](docs/agents/glossary.md).
- **ADRs:** before writing or editing `docs/adr/` → [`docs/adr/README.md`](docs/adr/README.md).
- **Issues:** create, update, triage, or wayfind GitHub tickets → [`docs/agents/issue-tracker.md`](docs/agents/issue-tracker.md). Superagent-shaped issues (`@forkboteu`, Depends on, UAT) → `write-superagent-issues` skill and [`.superagent/protocol.md`](.superagent/protocol.md).
- **Org Config authoring:** platform gates, `onCommit`, or transforms → [`docs/agents/org-config-authoring.md`](docs/agents/org-config-authoring.md).
- **Feragame fixture:** `server/internal/org/fixtures/feragame-config.yaml` (or seed/test followers) → [`docs/agents/feragame-fixture-authoring.md`](docs/agents/feragame-fixture-authoring.md).
- **DPP/UNTP:** Digital Link or credential behavior → [`docs/untp.md`](docs/untp.md).
- **Templates/CSS:** `server/templates/` or `web/src/styles/` → [`docs/css.md`](docs/css.md); format with `task templates:fmt` / `templates:check` / `templates:lint` (djLint, not a generic HTML formatter). Component extract/place → `attesta-ui-components` skill.
- **`/app` i18n:** locale wiring on the Org SPA → [`docs/agents/app-i18n.md`](docs/agents/app-i18n.md) (tracks: ADR-0013).

## Invariants

- **Dumb executor:** Org Go (`server/internal/org`) and `/app` evaluate Org Config; plant law (entity shapes, transforms, `readyWhen`, open/close) lives in Config, not Feragame- or domain-named branches in executors. See [`docs/org/GLOSSARY.md`](docs/org/GLOSSARY.md) and Org Config authoring.
- **Progress keys:** `.` → `_` at Mongo write, normalize on read. Same rule for Stream Process and Org Domain persistence.
