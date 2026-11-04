# Handoff — living-groups-rework parallel fixes (de-Feragame + leftovers)

**Date:** 2026-11-03  
**Focus for next session:** finish **de-Feragame engine** work in parallel (ticket 11 commit + audit triage → more tickets), then human smoke / squash.  
**Mode:** chief-of-staff DRI; implement / diagnose / review via **subagents**; sparse briefs + path pointers.

---

## Mission remainder

Board **01–10** shipped (commits on tip). Human hard call: **Living Go must not encode Feragame Domain entity kinds.** Ticket **11** implements Config `instanceLabelEntity` (uncommitted at handoff). Branch-wide Feragame-leak **audit** was in flight — consume its table and open parallel fix tickets.

Success for this slice: zero Feragame entity vocabulary in living **engine** production code; ticket 11 dual-APPROVE + committed; audit findings ticketed/fixed or explicitly deferred; branch ready for human squash → `feat/attesta-rework` (not pushed unless asked).

---

## Git map

| Ref | Tip (approx) | Role |
| --- | --- | --- |
| Feature branch | `feat/living-groups-rework` @ **`1ec3f3f`** (+ **uncommitted** ticket 11) | Primary line (cut from `feat/attesta-rework` @ `aedfb5e`) |
| Integration target | `feat/attesta-rework` | Squash/merge **when human OK** |
| Donor | `feat/task-groups` worktree | Read only |

**Not pushed.** Ignore unrelated dirty `.scratch/living-worklist/*`.

### Notable recent commits (newest first)

- `1ec3f3f` — `feat(ui): card-wrap Task groups on Job overview`
- `fb8c144` — `feat(living): project load id as group-instance Worklist label` (**introduced** hardcoded `Kind("load")` — ticket 11 undoes)
- `4e6aed6` — `feat(living): list group instances on Job overview`
- `073f322` — `fix(living): stop double Commit from dual form submit handlers`
- `dcdb859` — provisional first-leaf Available on home (hard call A)
- … board 01–08 through earlier history — `git log aedfb5e..HEAD`

---

## Tracker / law

| | |
| --- | --- |
| Ledger | `.scratch/living-groups-rework/COORDINATOR.md` |
| Issues | `.scratch/living-groups-rework/issues/` (local) |
| ADR | `docs/adr/0006-living-operator-worklist-home.md` |
| Glossary | `GLOSSARY.md` Task group / Spawn |

### Settled (do not undo without hard call)

- Hard call A (home provisional first-leaf Available when `canSpawn`).
- **Hard call de-Feragame (2026-11-03):** living Go must not hardcode Feragame Domain entity kinds; group-instance overview `label` join is **Config-declared** (`instanceLabelEntity`).
- Structural group instance id stays engine `lots-N` (`nextGroupInstanceID`); not Config. Operator-facing identity value stays transform mint; Worklist projects `label`.
- Overview lists group instances in a card + right-flushed Spawn/Close; display `label \|\| id`.

---

## Open / next fixes (priority) — **parallel OK**

### 1. Ticket 11 — Config `instanceLabelEntity` — **uncommitted; finish review + commit**

- Issue: `.scratch/living-groups-rework/issues/11-config-group-instance-label-entity.md`
- Implementer: done ([Implementer](226e8385-ea1b-419f-8d05-822266a20571))
- Spec: APPROVE ([Spec](f6324153-1c03-46a9-a3c7-345764549dc5))
- Standards: REQUEST CHANGES for stale godoc → **fixed in tree** (struct comment + `types.ts` label JSDoc; also repaired accidental tab-indent on `type WorklistGroupInstance`). Re-run Standards glance → commit.
- Diff touchpoints: `server/internal/living/worklist.go`, `worklist_test.go`, `fixtures/feragame-config.yaml` (`lots: instanceLabelEntity: load`), ADR-0006, `app/src/lib/living/types.ts`, issue 11, COORDINATOR.

### 2. Feragame-leak audit — **triage → parallel tickets**

- Agent: [Audit](9c5fdd0e-e3e8-498e-938a-3f4996773ad7) (may still be running or finished after this handoff — **read its final table**; do not re-scout from zero).
- Scope: production `server/internal/living/**/*.go` (exclude `_test.go` unless defaults force plant kinds); SPA projection hardcodes. Fixtures/seed/demo allowed.
- Ticket 11 covers the known branch-new `Kind("load")` in Worklist. Audit may find **pre-existing** or **other branch** leaks — open tickets 12+ and run implementers **in parallel**.

### 3. Soft / human

- Smoke: overview card + load-id labels (after 11); double-Commit already `073f322`.
- Squash/merge → `feat/attesta-rework`; push; seed — **only when asked**.

---

## Pipeline reminder

```text
audit findings → ticket(s) → implementer(s parallel) → Standards+Spec → fixer → commit
ticket 11 → re-APPROVE Standards → commit
human re-smoke → squash call
```

Escalate: product forks (Config group `label` expr / URL≡load id), push/squash, destructive seed.

---

## Verify (typical)

```bash
cd server && go test ./internal/living/... -count=1
cd app && bun run test:unit -- --run src/lib/living src/lib/router
# engine leak smoke (expect no hits in production .go excluding tests):
rg -n --glob '!*_test.go' 'Kind\("load"\)|Kind\("truck"\)|Kind\("pallet"\)' server/internal/living/
```

---

## Suggested skills

Call the Skill tool for:

1. **chief-of-staff** — parallel subagents, sparse briefs, hard calls only  
2. **code-review** — Standards + Spec after each fix  
3. **diagnosing-bugs** — if audit/smoke finds runtime surprises  
4. **handoff** — when pausing again  
5. Domain: `AGENTS.md` → `docs/agents/domain.md`, `GLOSSARY.md`, ADR-0006 before briefs  

---

## First actions for next coordinator

1. Confirm [Audit](9c5fdd0e-e3e8-498e-938a-3f4996773ad7) finished; fold findings into COORDINATOR + issues **12+**; launch parallel implementers for critical/high.  
2. Finish ticket 11: quick Standards re-check → **commit** (Conventional Commits; do not include `.scratch/living-worklist/*`).  
3. Keep Active/Frontier updated; do not push unless asked.  
4. When engine clean + smoke green: ask human about squash onto `feat/attesta-rework`.
