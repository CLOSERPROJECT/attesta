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
| 14 | — | queued | FormPlan rendering — next after 18 squash |
| 15 | — | revised | gaps inventory (not implement) |
| 19 | — | queued later | durable Mongo Domain/OrgConfig stores |

## Done

| Ticket | Squash on rework | Notes |
| --- | --- | --- |
| 01–12 | through `dea2e04` | flat port implementable; rename deferred |
| 16 | d03f0ba | config-driven Commit (ELT); reviewer APPROVE; human smoke |
| 17 | 9ca1870 | readyWhenOk Cerbos attr; reviewer APPROVE; human smoke |
| 18 | pending squash SHA | HttpPort after mutate + FakeHttpPort; reviewer APPROVE; human smoke |

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
- **UI gap:** FormPlan/derived → ticket **14** (next)
- Follow-on: **19** durable stores
