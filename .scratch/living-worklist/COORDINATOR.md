# Living Worklist coordinator ledger

**Base:** `feat/attesta-rework` @ `d61be2d`  
**Branch:** `feat/living-worklist` @ `93d9a71` (+ docs commit pending squash)  
**Role:** supervisor DRI — tickets 01–05 done; docs fold; squash into `feat/attesta-rework` next  
**Tracker:** `.scratch/living-worklist/issues/` (not GitHub)  
**Handoff:** `HANDOFF-supervise-worklist-branch-2026-10-30.md`

## Active

| Ticket | Stage | Notes |
| --- | --- | --- |
| — | — | Product tickets complete; awaiting human smoke + squash SHA record |

## Done

| Ticket | Commit on branch | Notes |
| --- | --- | --- |
| docs ADR | `1b1886c` | ADR-0005 + glossary + decisions amend |
| 01 | `ad681ec` | Worklist on Load |
| 02 | `e90ede2` | Worklist on Commit; SPA gates removed |
| 03 | `37f46b7` | First available Task form bundle on Load |
| 04 | `a7c628b` | All available + provisional bundles |
| 05 | `93d9a71` | Thin Load/Commit wire; bindings DomainSnapshot |

## Squash

| Field | Value |
| --- | --- |
| Into | `feat/attesta-rework` |
| Squash SHA | _(pending)_ |

## Hard calls

- Worklist chrome fields (`taskIds` / `startLabel` / `repeatable`) documented in ADR-0005 + GLOSSARY.
