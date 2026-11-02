# Next thin-SPA candidates (updated after dataReport)

**Date:** 2026-10-30 (revised)  
**Tip:** `feat/attesta-rework` @ `773113d`  
**Full handoff:** `HANDOFF-remaining-architecture-candidates-2026-10-30.md`  
**Earlier sources:** `$TMPDIR/handoff-living-thin-spa-next-candidates-2026-10-30.md`, `$TMPDIR/architecture-review-20261030-133840.html`, `$TMPDIR/architecture-review-20261030-130731.html`

## Arc status

| Done | SHA / note |
| --- | --- |
| Worklist 01–05 | squash `70298a3` |
| Cleanup #6→4→5→2 | `fc0d1e2` |
| #1 dataReport chrome (+ weightCheck unify, dead recap) | `5354984`, `773113d` |

Do not re-open JobChrome delete or SPA Unlock/startsWhen (ADR-0005). Twin fixture DRY + feragame-config rename: done (single `server/.../feragame-config.yaml`).

## Ranked remaining (thin-SPA class)

| # | Candidate | Strength | Size | Status now |
| --- | --- | --- | --- | --- |
| **3** | Seal / progress omit | Explore | S–M | **Open** — client `sealSubmission` + progress normalize twin Go prep |
| **7** | Roles / readyWhen chrome | Speculative | — | ADR defers; don’t grow SPA gates |
| — | Quarantine unused `valuePrint` | Explore | S | **Done** — widget + compile path removed; unknown kinds skip |
| — | Shared gate/Leftovers golden vectors (130731) | Explore | M | Optional leverage |
| — | Twin fixture DRY → `feragame-config` | Paused→OK | S | **Done** — single YAML + `$living-fixtures` |
| — | JobChrome delete | Paused | S | Unpause only with human OK |

## Done (was ranked)

| # | Candidate | Landed |
| --- | --- | --- |
| **1** | Chrome Domain projectors → bindings | dataReport / chromeDisplay |
| **2** | Harden FormPlan residual | `fc0d1e2` |
| **4** | Authored-shape classifiers | with cleanup |
| **5** | Domain→Draft re-seed | with cleanup |
| **6** | Orphan Commit-prep twins | `fc0d1e2` |

## Recommended next grill

**#3 Seal / progress omit.**
