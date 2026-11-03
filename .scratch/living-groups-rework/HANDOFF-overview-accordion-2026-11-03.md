# Handoff — living-groups-rework after overview accordion

**Date:** 2026-11-03  
**Focus for next session:** human **smoke** of Job instance overview accordion + group labels/Close dialog; then **squash/merge** onto `feat/attesta-rework` when OK. No open implement tickets on this board unless smoke finds gaps.  
**Mode:** chief-of-staff DRI; implement / diagnose / review via **subagents**; sparse briefs + path pointers.

---

## Mission remainder

Board **01–19** shipped on `feat/living-groups-rework` (tip **`adf9516`**). Overview accordion slice (actionable status, LinkStore + task-recap, experiment chrome, single-child flatten) is committed.

Success for next slice: smoke green (load-id labels on lots, Per te nav, done recap + links, Close Dialog, group default-open); human OK to squash → `feat/attesta-rework`; push only if asked. Re-seed / refresh active Org Config if `instanceLabelEntity` labels missing (ticket 11).

---

## Git map

| Ref | Tip (approx) | Role |
| --- | --- | --- |
| Feature branch | `feat/living-groups-rework` @ **`adf9516`** | Primary line (cut from `feat/attesta-rework` @ `aedfb5e`) |
| Integration target | `feat/attesta-rework` @ `aedfb5e` | Squash/merge **when human OK** |
| Donor | `feat/task-groups` worktree / `attesta-experiment` | Read only (chrome donor) |

**Not pushed.** Ignore unrelated dirty `.scratch/living-worklist/*`.

### Notable recent commits (newest first)

- `adf9516` — `feat(ui): polish Job overview accordion to experiment chrome` (ticket **19**)
- `2254455` — `feat(ui): flatten single-child Task group occurrences on overview` (**18**)
- `3603bbd` — `feat(living): overview accordion with actionable status and task recap` (**14–17**)
- `9bbc580` / `d77ca91` / `4e4d7e8` — de-Feragame **11–13**
- … earlier board — `git log aedfb5e..HEAD`

---

## Tracker / law

| | |
| --- | --- |
| Ledger | `.scratch/living-groups-rework/COORDINATOR.md` |
| Issues | `.scratch/living-groups-rework/issues/` (local) |
| Accordion law | `.scratch/living-groups-rework/OVERVIEW-ACCORDION.md` |
| Experiment chrome scout | `.scratch/living-groups-rework/EXPERIMENT-ACCORDION-CHROME.md` |
| ADR | `docs/adr/0005` + `0006` (actionable status + overview accordion amends) |
| Glossary | `GLOSSARY.md` (Worklist four-value status, child Task, Job instance overview) |

### Settled (do not undo without hard call)

- Hard call A (home provisional first-leaf Available when `canSpawn`).
- Hard call de-Feragame: no Feragame entity kinds in living engine; `instanceLabelEntity` Config join.
- Worklist status wire: `locked \| available \| actionable \| done`; no `viewerAvailable[]`.
- Overview: shadcn Accordion; numbers on Job-level Task + Task group only; single-child groups flatten into occurrence; done → lazy recap; links via LinkStore after Commit (no HTTP replay).
- UI tags: Bloccato / Disponibile / Per te / Fatto.

### Deferred (explicit)

- Rename leftover product nouns still saying “Available” in home band copy (wire is `actionable`).
- Durable free-form labels under the lot (beyond Config entity label).
- Squash / push / seed `--replace` — **human only**.

---

## Open / next — soft / human

1. **Smoke** overview: group default-open; load id on occurrence title after register (if missing → refresh active config / seed); Per te → form; done expand → facts+links; Close → Dialog; locked no hover border.
2. Fix only if smoke fails — open ticket **20+**, implementer → one end review if multi-file.
3. When green: squash/merge → `feat/attesta-rework`; push if asked.

---

## Verify (typical)

```bash
cd server && go test ./internal/living/... -count=1
cd app && bun run test:unit -- --run src/lib/living src/lib/router
# label smoke after register: Worklist groups.lots.instances[].label
```

---

## Suggested skills

1. **chief-of-staff** — subagents, sparse briefs  
2. **diagnosing-bugs** — if smoke finds surprises  
3. **code-review** — if a fix ticket lands  
4. Domain: `AGENTS.md` → `docs/agents/domain.md`, `GLOSSARY.md`, ADR-0005/0006, `OVERVIEW-ACCORDION.md`

---

## First actions for next coordinator

1. Read `COORDINATOR.md` + `OVERVIEW-ACCORDION.md`; tip `adf9516`.  
2. Drive or await human smoke; ticket gaps only.  
3. Do not push/squash unless asked.  
4. When smoke OK: ask human about squash onto `feat/attesta-rework`.
