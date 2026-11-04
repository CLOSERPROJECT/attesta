# Handoff — coordinator for Task groups on rework

**Date:** 2026-11-03  
**Your job:** chief-of-staff DRI for `.scratch/living-groups-rework/` until tickets **01–08** are implemented, reviewed, fixed, and human-ready to land on `feat/attesta-rework`.

**Mode:** all implement / review / fix via **subagents**; coordinator keeps ledger + pipeline; surface **hard calls only**. Do not re-grill settled product (see COORDINATOR).

---

## Mission

Port Task groups onto the **rework spine** (new Worklist home / overview / task chrome + Sidecar HttpPort), using `feat/task-groups` as a **donor** only. Success = operator can Spawn lots from a Job overview, get real labels, loop Start another / Close, and advance after Close—without JobPanel or product FakeHttp.

---

## Git map

| Ref | SHA / tip | Role |
| --- | --- | --- |
| **`feat/attesta-rework`** | `aedfb5e` | **Integration spine** — new GUI (#283) + living-http Sidecar (`ea16276`) |
| Donor **`feat/task-groups`** | `f221679` | Groups Domain/Unlock/Commit/Close SPA JobPanel era; **do not** implement here |
| Donor worktree | `/Users/giovanniabbatepaolo/.cursor/worktrees/task-groups-22bdc3ff/attesta-714729698316` | Read / cherry-pick source only |
| **Feature branch (create)** | e.g. `feat/living-groups-rework` from `aedfb5e` | Preferred implement line; or work directly on rework if human prefers |

**Policy:** Prefer a feature branch off rework + optional new worktree (`scripts/worktree-compose.sh` / Taskfile). Do **not** disturb primary checkout `task dev` if using a worktree. Do not push unless human asks. Conventional Commits. Comments = why only (`AGENTS.md`).

**Uncommitted on rework tip (commit early as docs/chore on the feature branch):**

- `GLOSSARY.md` — Task group / Spawn / Commit / overview updates  
- `docs/adr/0006-living-operator-worklist-home.md` — Task groups amend  
- `.scratch/living-groups-rework/**` — this board  

Ignore unrelated dirty `.scratch/living-worklist/*` unless human says otherwise.

Donor tip also has **uncommitted** `board-wire` SPA slice — **do not** port bulk `taskFormBundles` / board-wire; rework Commit wire is `next` + optional `nextTaskFormBundle`.

---

## Tracker (source of truth)

| | |
| --- | --- |
| Ledger | `.scratch/living-groups-rework/COORDINATOR.md` |
| Tickets | `.scratch/living-groups-rework/issues/01` … `08` |
| Related HttpPort board | `.scratch/living-http/` (Sidecar already on rework; **08** proves groups path) |

### Ticket order

```text
01 → 02 → ┬→ 03 → 04 ┐
          ├→ 06      ├→ 05 → 07
          └→ 08 ─────┘
```

- **01** Domain + unlock + viewer Worklist groups  
- **02** `spawningNew` Commit + Close HTTP + nested task-form + `next` while group open  
- **03** Overview Spawn → nested Task route → first-leaf form  
- **04** Overview Close (confirm) → advance  
- **05** Stay on form when links; Start another / Close; any Commit with downloads  
- **06** Home nested Available (parallel after 02)  
- **08** Real Sidecar labels on group leaf Commit (parallel after 02; blocks 07)  
- **07** Plant path check (lots + line weigh)

---

## Pipeline (per ticket)

```text
ticket ready-for-agent
  → implementer subagent (one ticket; port from donor, adapt to rework contracts)
  → code review subagent (Standards + Spec vs ticket + grill/ADR)
      → APPROVE → mark done in COORDINATOR; next frontier ticket
      → CHANGE REQUESTS → fixer subagent → re-review (loop until APPROVE or hard call)
```

**Patterns:** same as living-http / flat-port smoke coordination — sparse briefs, pointers to ticket + COORDINATOR + donor paths; one concern per subagent. Use repo code-review skill / parallel Standards+Spec agents when available.

**After a slice of tickets (human call):** smoke on Compose with `LIVING_HTTP_URL` / token; then squash/merge feature branch → `feat/attesta-rework` when human OK.

---

## Hard calls already settled (do not re-litigate)

See COORDINATOR “Settled” + ADR-0006 Task groups amend + GLOSSARY. Especially:

- Port onto rework; JobPanel dead; Sidecar product HttpPort  
- Spawn = first-leaf mint; Close on overview (+ loop CTAs after labeled submit)  
- Open group + no links → overview `next`; links → read-only form + Start another / Close; after Close → usual next  
- Nested URL includes group coordinates; home Available may list nested ready Tasks; no home Spawn band  

**Escalate to human** only for new product forks, destructive seed/reset, push/squash timing, or blocker conflicts with other rework work.

---

## Donor vs rework contract traps

| Take from donor | Keep from rework |
| --- | --- |
| `groups.go`, unlock path deepen, `spawningNew`, Close lifecycle | `Load(org, op)`, empty home bundles, `GET task-form`, `CommitNext` |
| Worklist `Groups` chrome | Viewer `viewerAvailable` / role scoping |
| Nested TaskForm / mint SPA behavior | `worklist-home` / overview / task routes, `nav-place` |
| — | `HttpPortFromEnv` / Sidecar / artifact proxy — **never** donor `NewFakeHttpPort()` product default |

Close / spawn Commit responses must stay **rework-shaped** (Worklist + `next` / optional bundle), not bulk board-wire.

---

## Verify (typical)

```bash
cd server && go test ./internal/living/... ./cmd/server -count=1
cd app && bun run check && bun run test:unit -- --run src/lib/living
# 08 / 07: Compose living-http + LIVING_HTTP_* ; spawn+Commit lot → label links
```

---

## Out of scope

- Rebuilding living-http / HttpPort tranche B  
- Re-opening parked JobPanel deepen / task-groups architecture G leftovers as this board  
- Push / force / seed `--replace` without human  
- Re-litigating mint vs spawn HTTP (spawn HTTP already removed on donor)

---

## First actions for coordinator

1. Confirm with human: feature branch name + worktree vs primary rework checkout.  
2. Commit uncommitted glossary / ADR-0006 / `.scratch/living-groups-rework/` onto that line.  
3. Spawn implementer for **01**; update COORDINATOR Active/Done as you go.  
4. After each implement: review → fixer loop before starting the next blocked ticket (03∥06∥08 after 02 is OK in parallel **only** if separate worktrees/branches—or serialize on one branch).

---

## Suggested skills

Call the Skill tool for:

1. **code-review** (or project equivalent) — Standards + Spec after each ticket implement  
2. **chief-of-staff** / coordinator patterns if present — subagent loop, sparse briefs  
3. **handoff** — when pausing again  
4. **writing-for-agents** — if editing COORDINATOR / ticket acceptance  
5. Domain: read `AGENTS.md` → `docs/agents/domain.md`, `GLOSSARY.md`, ADR-0005/0006 before briefs  

Svelte UI tickets (**03–06**, **05**): use **svelte-file-editor** / Svelte MCP per workspace rules.
