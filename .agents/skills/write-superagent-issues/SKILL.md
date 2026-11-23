---
name: write-superagent-issues
description: >
  Open Superagent backlog issues for @forkboteu: split a plan into
  one-PR slices, wire Depends on, and draft User Acceptance Tests.
  Use when filing agent tickets, sizing a Superagent backlog, or
  adding UAT scenarios to an issue.
---

# Write Superagent issues

Turn a plan or change request into GitHub issues the Superagent pipeline can run. Body shape and UAT rules live in [`.superagent/protocol.md`](../../../.superagent/protocol.md); product constraints live in root `AGENTS.md`. This skill is the procedure.

## 1. Load the contract

Read `.superagent/protocol.md` and the product sources the person named (architecture, ADR, glossary, or the files under change). Confirm the agent mention is `@forkboteu`.

Done when: protocol hard rules are in context (mention in the **created** body, one PR per issue, acceptance checks, `Depends on`, exact UAT heading or none).

## 2. Split

Break the work into **slices** — each issue is one reviewable PR.

- Split when two outcomes can be accepted separately; keep producer and consumer together when either alone would be an invalid contract.
- Keep Context paths inside a mid-tier budget (~40k tokens of must-read files); more than one subsystem → split further.
- Order slices so every prerequisite precedes its dependents; no cycles.
- Park out-of-scope findings as separate issues instead of enlarging a slice.

Done when: each slice has an imperative title, a one-paragraph ask, and a place in the dependency order.

## 3. Draft each body

```markdown
@forkboteu <one paragraph: what to build and why, with pointers to the product docs or paths that define done>

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

| Field | Rule |
| ----- | ---- |
| Mention | `@forkboteu` in the body at create time |
| Acceptance | Checks a reviewer can verify, including tests to add |
| UAT | Heading exactly `## User Acceptance Tests`, or omit the whole section. Default environment here: `local` (see `.superagent/uat.yaml`) |
| Scenarios | `### UAT-<n>: <title>`, unique positive `n`, non-empty steps |
| Depends on | One line `Depends on #<a>, #<b>`, or omit |

Never ask the agent to wait, hold merge, or seek approval.

Done when: every draft has the mention, acceptance checks, a valid Depends on line or none, and either a parse-ready UAT section or no UAT heading.

## 4. Confirm, then open

Show the person the backlog: title, Depends on, UAT yes/no. Iterate until they approve the split.

Create issues with **that person’s** `gh` (never the agent account), in dependency order so each `Depends on` names issues that already exist:

`gh issue create --repo <owner>/<repo> --title "<imperative title>" --body-file <file>`

Done when: every approved draft exists on GitHub with the intended numbers, and Depends on lines match those numbers.
