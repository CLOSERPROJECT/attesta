# Overview accordion slice (tickets 14–17)

**Law:** `GLOSSARY.md` (child Task, Worklist four-value status, Job instance overview, Submission links); ADR-0005/0006 amends (2026-11-03 grill).  
**Board:** `issues/14`–`17`; `COORDINATOR.md`.  
**Loop:** implement **all** → **one** Standards+Spec review at end (human). Do not push.

## Settled (do not re-litigate)

- Status wire: `locked | available | actionable | done`; retire `viewerAvailable[]`.
- Aggregate group occurrence: any `actionable` → else any `available` → else all `done` → else `locked`.
- UI tags: Bloccato / Disponibile / Per te / Fatto.
- Links: Domain Put → HttpPort → **separate** side-record persist (no Domain rollback on store fail; no HTTP replay).
- Recap: lazy GET facts + stored links; Load-level auth.
- Overview: shadcn Accordion; numbers on Job-level Task + Task group cards only; group accordion → occurrences → child Tasks; `actionable` → form; `done` → recap.

## Verify

```bash
cd server && go test ./internal/living/... -count=1
cd app && bun run test:unit -- --run src/lib/living src/lib/router
```

## Pipeline status

| Ticket | Status |
| --- | --- |
| 14–17 | implemented (uncommitted vs `83a87f6`) |
| End review | done |
| Fixer cleanup | done (comments, parseHttpLinkItem, Job-level chevron, TaskStatusTag) |
| Judgement calls | done (`ActionableTaskRef`, overview snippet dedupe, nested child chrome titles) |
