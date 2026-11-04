# Living flat-port coordinator ledger

**Base:** `feat/attesta-rework`  
**Role:** chief-of-staff DRI — **all implement/review/fix/diagnose work in subagents** (protect coordinator context); sparse briefs via pointers  
**Pipeline (locked):** ticket → branch (here) → implementer → code reviewer → fixer → squash merge (here)  
**Parallelization:** one subagent at a time  
**Surface to human:** hard calls only  
**Squash policy:** after reviewer APPROVE (or fixer green); human smokes on rework

## Active

| Ticket | Branch | Stage | Notes |
| --- | --- | --- | --- |
| 13 | — | **paused** | discussion notes: `13-rename-placeholders-discussion.md` |
| 15 | — | revised | gaps inventory (not implement) |

## Done

| Ticket | Squash on rework | Notes |
| --- | --- | --- |
| 01–12 | through `dea2e04` | flat port implementable; rename deferred |
| 16 | d03f0ba | config-driven Commit (ELT); reviewer APPROVE; human smoke |
| 17 | 9ca1870 | readyWhenOk Cerbos attr; reviewer APPROVE; human smoke |
| 18 | ce9974c | HttpPort after mutate + FakeHttpPort; reviewer APPROVE; human smoke |
| 14 | 4789ecd | FormPlan UI; reviewer APPROVE; human smoke |
| 19 | c815ebc | durable Mongo Domain/OrgConfig; reviewer APPROVE; human smoke |
| smoke-issues | 87714c2 | truck-loop smoke cascade: mint.startingNew, AsJob/Task order/tasks-array, lineLoad progress, schema defaults + `$today`; human smoke green |

## Diagnostics

| Issue | Result |
| --- | --- |
| living@attesta.com login | **seed miss** — empty-only no-op; need `task seed -- --replace` ([Diagnose living demo login](ab9bcd49-6d63-4305-b809-37ee2f18490c)) |
| Missing Job instance id / `_new` | **fixed** — `mint.startingNew` (no `_new`); uniqueness vs evaluated id. Plan: `PLAN-commit-mint-intent-2026-10-30.md` |
| invalid Job `"intake"` | **fixed** — `AsJob` + normalize after Mongo JSON |
| instanceId empty / Task order | **fixed** — RawMessage Job decode + tasks ordered array |
| lineLoad progress / Completo one-of-many | **fixed** — null anyOf, omit untouched, gate mapTable seed, submit error UX |
| register NIR / date defaults | **fixed** — schema default merge + authored `$today` |

### HC-seed-replace

**Done** 2026-10-30 — `task seed -- --replace` (+ CLI wires living Mongo SeedDemo). Re-seed after fixture YAML changes (`$today`).

## Hard calls (open)

### HC-13 naming (paused)

See `.scratch/living-flat-port/issues/13-rename-placeholders-discussion.md`.  
Settled: drop `living`; do not name package bare `domain` (clashes with Org Config `domain` section).  
Open: package name, HTTP paths, Cerbos kind, store renames, env flag, order vs FormPlan (14 done).

## Env improvements

- Demo: `living@attesta.com` / org `feragame`; wipe via `ATTESTA_LIVING_SEED`
- Prefer fail-closed JSON/Mongo-roundtrip tests before Commit (Memory alone hid smoke #2/#3)
- Optional polish: accordion chrome / HttpLinkStrip depth
- Implementable follow-on queue empty except **13** when names agreed
