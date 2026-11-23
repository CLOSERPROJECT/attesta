# Superagent protocol (attesta)

Harness law for `@forkboteu` tickets on this repository. Product law stays in root [`AGENTS.md`](../AGENTS.md). Interactive sessions ignore this file.

Deployment prompts (`delivery`, `claude-review`, `uat-test`) still own the worker loop; this file is the **product-side contract** Claude review and issue authors share with those prompts.

## Ticket → merge

1. An allowlisted person opens a GitHub issue whose **created** description mentions `@forkboteu` (a later edit is invisible; comment instead).
2. The Superagent coder implements on a ticket branch, opens one PR, and names the issue with `Closes #<n>` (advances only: `Refs #<n>`).
3. When the issue includes a valid UAT section and the deployment has UAT on, `uat-test` must pass for the PR head before `claude-review`.
4. `claude-review` must PASS for that exact head; required CI must pass; then `@forkboteu` squash-merges (`--match-head-commit`). No interactive approval gate.

Scope stays inside the issue. Park unrelated findings as new issues. Conventional Commits for PR titles (squash subject) and commits. Never `--admin` or bypass branch protection.

## Issue body

One PR-sized slice. Acceptance criteria a reviewer can check, including tests to add. Optional dependency line, exactly:

`Depends on #<a>, #<b>`

Omit the line when there are none. Never ask the agent to wait, hold merge, or seek approval.

Open Superagent issues with the `write-superagent-issues` skill (human’s `gh`, never the agent account).

## User Acceptance Tests

Omit the whole section when no end-to-end check applies. Without the section, UAT is `NOT_REQUIRED`.

When present, the heading must be exactly `## User Acceptance Tests`. A near-miss heading blocks the PR. Optional first line: `Environment: local` (default here). Use only environment names the deployment allowlists.

Each scenario: `### UAT-<n>: <title>` with unique positive `n` and non-empty steps (Given/When/Then recommended). Observable outcomes only (HTTP, pages, records). Test data only — no production credentials.

Sandbox config: [`.superagent/uat.yaml`](uat.yaml) (`mise run uat:setup` / `uat:start` / `uat:cleanup`). Setup may use network; start keeps the app answering `ready` without rewriting committed files.

Editing the UAT section re-runs UAT and withdraws an earlier pass; editing the rest of the issue does not.
