# Agent instructions

## Project

Attesta turns regulated, multi-party streams into verifiable digital records. Each organization sees the actions it is allowed to complete, submits structured data or file evidence, and follows a live timeline from intake to final notarization. It is built for traceability demos where accountability matters: policy checks enforce roles and step order, MongoDB stores process state and attachments, Appwrite manages organizations and users, and completed streams can publish a GS1 Digital Link / Digital Product Passport for external verification.

Read [`docs/architecture.md`](docs/architecture.md) before any task: it maps how the Stream (`/my`) and Org (`/app`) product surfaces connect, and these instructions only add how work is done.

Project constraints:

- **Dumb executor:** Org Go (`server/internal/org`) and `/app` evaluate Org Config; plant law (entity shapes, transforms, `readyWhen`, open/close) lives in Config, not Feragame- or domain-named branches in executors.
- **Progress keys:** `.` → `_` at Mongo write, normalize on read, for both Stream Process and Org Domain persistence.

The [Invariants](#invariants) section states both rules in full.

## Working here

- Keep changes minimal and localized. Match established patterns after reading the relevant code.
- Ask before a breaking change to routes, workflow configuration, or persistence.
- Preserve unrelated work in a dirty tree. Do not commit, push, or change external systems unless asked.
- **Comments are _why_, never _what_.** Keep a comment only when deleting it would lose intent, a constraint, or a gotcha the code cannot express. In every region you touch, delete comments that restate identifiers, types, or control flow. Prefer rename or restructure over narrating the obvious.
- **Titles:** commits, issues, and PRs use [Conventional Commits](https://www.conventionalcommits.org/): `type(optional-scope): imperative summary` (lowercase type; typical `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`, `build`, `perf`). Enforce this; do not infer style from recent `git log`.
- **Markdown:** flow prose; do not hard-wrap at a column. Soft-wrap in the editor. Keep intentional blank lines, lists, tables, and fences.

## Source guidance

- **Architecture:** how product and package seams connect → [`docs/architecture.md`](docs/architecture.md).
- **Operations:** pitch/env → [`README.md`](README.md); day-1 prepare/seed/worktrees → [`QUICKSTART.md`](QUICKSTART.md); Compose/Coolify/lifecycle → [`DOCKER.md`](DOCKER.md). Scripts and configuration remain the source of truth.
- **Glossary layer:** before naming product nouns → [`GLOSSARY-MAP.md`](GLOSSARY-MAP.md).
- **Glossary edit:** before adding or changing an entry → [`docs/agents/glossary.md`](docs/agents/glossary.md).
- **ADRs:** before writing or editing `docs/adr/` → [`docs/adr/README.md`](docs/adr/README.md).
- **Issues:** create, update, triage, or wayfind GitHub tickets → [`docs/agents/issue-tracker.md`](docs/agents/issue-tracker.md).
- **Org Config authoring:** platform gates, `onCommit`, or transforms → [`docs/agents/org-config-authoring.md`](docs/agents/org-config-authoring.md).
- **Feragame fixture:** `server/internal/org/fixtures/feragame-config.yaml` (or seed/test followers) → [`docs/agents/feragame-fixture-authoring.md`](docs/agents/feragame-fixture-authoring.md).
- **DPP/UNTP:** Digital Link or credential behavior → [`docs/untp.md`](docs/untp.md).
- **Templates/CSS:** `server/templates/` or `web/src/styles/` → [`docs/css.md`](docs/css.md); format with `task templates:fmt` / `templates:check` / `templates:lint` (djLint, not a generic HTML formatter). Component extract/place → `attesta-ui-components` skill.
- **`/app` i18n:** locale wiring on the Org SPA → [`docs/agents/app-i18n.md`](docs/agents/app-i18n.md) (tracks: ADR-0013).

## Invariants

- **Dumb executor:** Org Go (`server/internal/org`) and `/app` evaluate Org Config; plant law (entity shapes, transforms, `readyWhen`, open/close) lives in Config, not Feragame- or domain-named branches in executors. See [`docs/org/GLOSSARY.md`](docs/org/GLOSSARY.md) and Org Config authoring.
- **Progress keys:** `.` → `_` at Mongo write, normalize on read. Same rule for Stream Process and Org Domain persistence.

## How work happens

- Work starts from GitHub issues. An issue that mentions `@forkboteu` and is opened by an allowed person becomes a task for the dsh-superagent agent, which implements it on a branch and opens one pull request for it.
- Claude Code reviews every pull request the agent opens. The agent fixes the blocking findings until the review passes, then squash-merges the pull request itself once required CI checks and user acceptance tests pass.
- No human approval is needed for any step. Issues run in the order their `Depends on` lines give, and independent issues run in parallel.
- A person stops or redirects work with a comment on the issue or pull request.

## Code and tests

- Every change comes with tests that exercise it, and the existing tests pass. Run `task cover` before opening a pull request; CI runs the same suite with the 90% coverage gate.
- The toolchain (runtimes, package managers, CLIs) is managed with [mise](https://mise.jdx.dev/) and pinned in `mise.toml` at the repository root; CI uses the same versions. Before installing dependencies, running tests, linters, type checks or builds, run `mise install` in the checkout, then use the tools it provides (through mise's shims on `PATH`, or `mise exec -- <command>`); never rely on a version installed some other way, and never install a runtime globally. A tool the work needs is added to `mise.toml`, and a version changed there, in the same pull request as the code that needs it.
- Keep changes within the scope of their issue; record anything else found on the way as a new issue instead of fixing it in the same pull request.
- Never commit secrets, credentials or personal data, and never deploy to production.

## Commits and pull requests

- Commit messages and pull request titles follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/): `<type>(<optional scope>): <description>`, with type one of `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore` or `revert`; `!` and a `BREAKING CHANGE:` footer mark a breaking change. The squash merge uses the pull request title as the commit subject.
- One pull request per issue. Its description says `Closes #<n>` for the issue it completes, and `Refs #<n>` for an issue it only advances.
- Never create a pull request as a draft; open it ready for review.

## Merging

The agent account `@forkboteu` squash-merges a pull request it opened, without further human approval, once an independent Claude Code review of the pull request's exact head commit passes, the user acceptance tests for that commit pass, and all required CI checks pass (`gh pr merge --squash --match-head-commit <reviewed sha>`). It never merges any other pull request, never uses `--admin` or bypasses branch protection, and does not merge work whose issue or comment explicitly forbids merging. Every other merge needs human approval.

## Long-lived branch

Issues that implement #300 open their pull request against `feat/org-config-as-stream-v2`, cut from `master`; that branch, not `master`, receives the work.

The branch is not the default branch, so `Closes #n` in a pull request into it does not close the issue. The reviewer closes the implementing issue as completed with `gh issue close <n> --reason completed` once its pull request is squash-merged.

CI runs on pull requests and pushes for `feat/org-config-as-stream-v2` as well as `master`, and every such pull request reports the `⏳ Run tests` check even when it only changes root Markdown. The check can therefore be required on the branch without staying pending.

## Writing issues

These rules are for anyone, person or agent, who opens issues for this repository; an agent asked to plan a feature or a backlog opens the issues following them.

Every issue:

- Is opened by a person's account that the deployment allows to request work, never by `@forkboteu`: the agent does not take requests from itself. An agent opening issues does so with that person's `gh`.
- Mentions `@forkboteu` in the description it is created with. A mention added later by editing the description is not seen; post a comment instead.
- Asks for exactly one pull request's worth of work: one module or feature slice, reviewable on its own, typically up to a few hundred lines. Split anything larger, such as a redesign across several modules.
- States the acceptance criteria as checks a reviewer can verify, including the tests to add.
- Lists its prerequisites on one line, `Depends on #<a>, #<b>`, naming only the issues whose merged code it needs. Omit the line when there are none. An issue starts once every listed issue is closed as completed; one closed as not planned blocks it.
- Never asks the agent to wait, not to begin, not to merge or to get approval: the agent obeys such a request and does not lift it by itself.

Opening a backlog:

1. Break the work into issues as above and order them so each one's prerequisites come before it; there must be no cycles.
2. Make the first issue set up `mise.toml` with the toolchain, the build, the test suite and a CI workflow that installs the same versions from `mise.toml` (for example with `jdx/mise-action`), and make that workflow's checks required on the default branch, so every later merge has to pass them.
3. Create the issues in that order with `gh issue create --repo <owner>/<repo> --title "<title>" --body-file <file>`, so each `Depends on` line names issues that already exist.
4. Open them all at once: the agent starts every issue whose prerequisites are done, up to its concurrency limit, and the rest as their prerequisites merge.

Issue template:

```markdown
@forkboteu <one paragraph: what to build and why, referring to the relevant
sections of docs/architecture.md>

## Acceptance criteria
- [ ] <verifiable behavior>
- [ ] <tests that cover it>

## User Acceptance Tests
Environment: local

### UAT-1: <observable behavior>
Given <starting state>
When <what a user or client does>
Then <what they observe>

Depends on #<a>, #<b>
```

The title is a short imperative summary, such as `Add the stream instance route`; the agent writes the conventional pull request title from it. The `## User Acceptance Tests` section is optional; leave it out when no end-to-end check applies.

## User acceptance tests

When the deployment enables them, an independent DeepSeek tester runs an issue's user acceptance tests (UAT) against the pull request's exact head commit after the agent pushes it, before the Claude Code review, and the pull request merges only after UAT passed for that commit; the tester's report is posted on the pull request. UAT checks what a user or client observes from the running application (HTTP responses, pages, files, command output), not unit tests or the code itself: the tester runs every scenario and keeps evidence of what it saw.

- Write the heading exactly `## User Acceptance Tests`. Without it, no UAT runs and the check says so; a heading written another way (`## User acceptance test`) blocks the pull request instead of being skipped.
- Optionally start the section with one `Environment: <name>` line: `local` (the default) builds and starts the pull request's code in a disposable sandbox, `preview` tests the pull request's preview deployment. Text before the first scenario is context for the tester.
- Each scenario is a `### UAT-<n>: <title>` heading with a unique positive number, followed by its steps; Given/When/Then is recommended. Name exact, observable outcomes (status codes, messages, records), and the fixtures or test users a scenario needs. Another `### ` heading, a repeated number or a scenario without steps blocks the pull request.
- Keep scenarios to test data: they never run against production or need real credentials.
- Editing the section re-runs UAT and withdraws an earlier pass; editing the rest of the issue does not.
- A failing scenario goes back to the agent with what was expected, what was observed and how to reproduce it, and so does a scenario the tester cannot run against the pull request, with what it lacked; after three such rounds on one pull request it stops and is reported blocked, unmerged. A scenario is never passed without being run.
- A repository that needs setup to run locally describes it in `.superagent/uat.yaml` (commands run only inside the sandbox). Setup and the application may create files anywhere (dependencies, build output, data), but committed files are read-only to them: a command that rewrites a committed file, such as a lockfile update or a formatter, fails the run, so build into untracked paths and commit lockfiles as installed:

  ```yaml
  version: 1
  default_environment: local
  environments:
    local:
      type: local
      setup: mise run uat:setup        # optional; with network access
      start: mise run uat:start        # keeps running; no network access
      ready: http://127.0.0.1:3000/health
      cleanup: mise run uat:cleanup    # optional
  ```

### Preview

`Environment: preview` tests the pull request's preview deployment at `https://{pr}.attesta.apps.forkbomb.eu`, where `{pr}` is the pull request number. The preview is seeded with the Feragame demo, including the staged truck `261102-1` with closed lots `CO-261102-1` and `DA-261102-1`.

Seeded users all have the password equal to their email: `admin@feragame.com` (all roles), `administration@feragame.com`, `logistic@feragame.com`, `line@feragame.com`, and `admin@org1.com` (another organization).

Preview environment variables live in Coolify, outside this repository. If a Turnstile challenge blocks login on the preview, the pull request says so instead of claiming the login scenario passed.
