# Living flat-port coordinator ledger

**Base:** `feat/attesta-rework`  
**Pipeline (locked):** ticket → branch (here) → implementer → code reviewer → fixer → squash merge (here)  
**Parallelization:** one subagent at a time  
**Surface to human:** hard calls only  
**Squash policy:** after reviewer APPROVE (or fixer green); human smokes on rework

## Active

| Ticket | Branch | Stage | Notes |
| --- | --- | --- | --- |
| 13 | — | **paused** | discussion notes: `13-rename-placeholders-discussion.md` |
| 14 | — | queued | FormPlan rendering (after/beside 16) |
| 15 | — | revised | gaps inventory (not implement) |
| 16 | — | ready | **config-driven Commit** — next implement |
| 17 | — | queued later | readyWhen → Cerbos (`blocked by 16`) |
| 18 | — | queued later | HttpPort after mutate (`blocked by 16`) |
| 19 | — | queued later | durable Mongo Domain/OrgConfig stores |

## Done

| Ticket | Squash on rework | Notes |
| --- | --- | --- |
| 01–12 | through `dea2e04` | flat port implementable; rename deferred |

## Diagnostics

| Issue | Result |
| --- | --- |
| living@attesta.com login | **seed miss** — empty-only no-op; need `task seed -- --replace` ([Diagnose living demo login](ab9bcd49-6d63-4305-b809-37ee2f18490c)) |

### HC-seed-replace (open)

`task seed -- --replace` rewrites demo orgs/users (destructive to hand-edited Appwrite fixture). Await human OK before running.

## Hard calls (open)

### HC-13 naming (paused)

See `.scratch/living-flat-port/issues/13-rename-placeholders-discussion.md`.  
Settled: drop `living`; do not name package bare `domain` (clashes with Org Config `domain` section).  
Open: package name, HTTP paths, Cerbos kind, store renames, env flag, order vs ticket 14.

## Env improvements

- Soft: large IntakeJob/LineJob — defer split
- Demo: `living@attesta.com` / org `feragame`; wipe via `ATTESTA_LIVING_SEED`
- **UI gap:** FormPlan/derived → ticket **14**
- **Config-driven gap (pressing):** server Commit is Feragame `apply*Commit` switch, not Org Config ELT → ticket **16**
- Follow-ons filed for later: **17** readyWhen, **18** HttpPort, **19** durable stores
