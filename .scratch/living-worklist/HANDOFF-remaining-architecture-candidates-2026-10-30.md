# Handoff: remaining architecture candidates (post–dataReport)

**Date:** 2026-10-30  
**Repo:** `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta`  
**Checkout:** `feat/attesta-rework` @ `773113d`  
**Focus for next session:** pick / grill / ticket the **remaining** thin-SPA and related architecture candidates — **not** re-do Worklist 01–05 or dataReport chrome (done).

Durable twin of this file (if committed later):  
`.scratch/living-worklist/HANDOFF-remaining-architecture-candidates-2026-10-30.md`

---

## Where things stand

| Arc | Status | Pointers |
| --- | --- | --- |
| Worklist 01–05 | **Done** (squash `70298a3`) | `.scratch/living-worklist/issues/01`–`05`, ADR-0005 |
| Thin-SPA cleanup #6→4→5→2 | **Done** (`fc0d1e2`) | FormPlan residual + orphan Commit-prep twins deleted |
| Chrome Domain projectors (#1) | **Done** (+ weightCheck unify) | `5354984` dataReport; `773113d` weightCheck + dead **recap** purge |
| Local tickets 06–07 | **Done** | `.scratch/living-worklist/issues/06`–`07` |

**Settled law (do not re-grill):** Domain-only aggregate chrome → authored JSONata `kind: dataReport` → Go eval on Load/Commit → `bindings.chromeDisplay` → SPA paints. Live client chrome remains only where draft/operator is required (`readonly`). `valuePrint` removed. Plant math/copy stays in YAML.

**Smoke helper (uncommitted policy was earlier; now in tree):** `server/cmd/seed-loaded-extracted/` — wipe + seed loadedExtracted + weightCheck-available intake.

---

## Sources gathered (this handoff)

| Source | Path | What it contributed |
| --- | --- | --- |
| Thin-SPA next candidates (scratch) | `.scratch/living-worklist/NEXT-thin-spa-candidates-2026-10-30.md` | Ranked #1–7 after Worklist |
| Thin-SPA next handoff (TMP) | `$TMPDIR/handoff-living-thin-spa-next-candidates-2026-10-30.md` | Same set + session flow |
| Architecture review (thin-SPA set) | `$TMPDIR/architecture-review-20261030-133840.html` | Cards for #1–7 |
| Architecture review (broader DRY) | `$TMPDIR/architecture-review-20261030-130731.html` | Unlock twin, fixture DRY, JobChrome, Leftovers plant, shared vectors |
| dataReport grill + implement | `.scratch/living-worklist/GRILL-dataReport-…`, `NOTES-dataReport-…`, `$TMPDIR/handoff-implement-dataReport-…` | Candidate #1 settled + shipped |
| weightCheck unify | `.scratch/living-worklist/NOTES-unify-…`, `issues/07-…` | Domain recap → dataReport; recap deleted |
| Worklist supervisor | `.scratch/living-worklist/HANDOFF-supervise-worklist-branch-…` | Implement style; out-of-scope reminders |
| Flat-port gap scout | `.scratch/living-flat-port/issues/15-experiment-gaps-…` | Different queue (ELT/rename/e2e) — see “Adjacent” |
| Flat-port COORDINATOR | `.scratch/living-flat-port/COORDINATOR.md` | 13–19 status; rename paused |
| Product decisions | `docs/research/attesta-app-living-product-decisions.md` | No SPA Unlock/startsWhen |

---

## Candidate board (updated)

### Thin-SPA / server-projection class (133840 + NEXT)

| # | Candidate | Was | **Now** | Next action |
| --- | --- | --- | --- | --- |
| **1** | Chrome Domain projectors → bindings | Strong / recommended first | **Done** — `dataReport` + `chromeDisplay`; weightCheck + loadedExtracted; SPA projector + **recap** gone | Only if new Domain-only chrome appears → same seam |
| **2** | Harden FormPlan residual | Strong | **Done** (`fc0d1e2`) | Guardrails / lint if re-growth feared |
| **6** | Orphan Commit-prep twins | Explore / delete | **Done** (`fc0d1e2`) | — |
| **4** | Authored-shape classifiers | Explore | **Done** with cleanup (`fc0d1e2`) | Re-scout if SPA still classifies authored shape for prod |
| **5** | Domain→Draft re-seed | Explore | **Done** with cleanup (`fc0d1e2`) | Runtime seed = bundle `initialData` |
| **3** | Seal / progress omit | Explore S–M | **Still open** — client `sealSubmission` + progress normalize twin Go prep | Grill: peel seal into Go / declare recipe on bundle vs keep twin-by-design |
| **7** | Roles / readyWhen chrome | Speculative | **Still open / deferred** — ADR-0005 | Don’t grow SPA gates until product asks |

### Broader DRY review (130731) — status relative to Worklist/dataReport

| Item (130731) | Status | Notes |
| --- | --- | --- |
| Deep Unlock + startsWhen TS+Go twin | **Superseded** | Worklist owns chrome gates; SPA Unlock/startsWhen must not return (ADR-0005) |
| Twin truck-loop YAML fixtures | **Done** | Single `server/.../feragame-config.yaml`; SPA via `$living-fixtures` |
| Delete JobChrome pass-through | **Paused** | Still mounts via `LivingShell` → `JobChrome` → `JobPanel` |
| Config-driven Leftovers for Job visibility | **Mostly absorbed / revisit** | Commit/Worklist path; plant leftover hardcodes may remain in SPA UI helpers — scout before ticket |
| Commit-prep filter/flatten twins | **Done** | Same as thin-SPA #6 |
| Shared gate / Leftovers golden vectors | **Open / explore** | Still multi-suite hand graphs; optional leverage |

### Live chrome leftovers (post–recap purge)

| Kind | Authored in truck-loop? | Mechanism |
| --- | --- | --- |
| `dataReport` | Yes (weightCheck, loadedExtracted) | Server `chromeDisplay` |
| `readonly` | Yes (e.g. `currentOperator`, register) | Client live eval |
| `valuePrint` | No | **Deleted** — no authored rows; widget + compile path gone |
| `recap` | No | **Deleted** (`773113d`) |

---

## Recommended remaining work (ordered)

1. **Grill #3 Seal / progress omit** — only live “twin by design” left in the thin-SPA ranked set; decide peel vs contract.  
2. ~~**Scout + optional delete `valuePrint`**~~ **Done** — removed.  
3. **Optional shallow wins from 130731:** JobChrome delete (still paused); twin fixture DRY → single `feragame-config` **done**.  
4. **Shared golden vectors** — explore after #3, not blocking.  
5. **#7 Roles/readyWhen chrome** — speculative; product-gated.  
6. **Adjacent (not thin-SPA):** flat-port **13** rename when names agreed; plant e2e / draft persistence from issue 15 — separate queues.

Do **not** re-open: SPA Unlock/startsWhen plant; Feragame projector Modules; putting materials/tare tables into Go/SPA living packages.

---

## Suggested next session flow

1. Confirm tip ≥ `773113d`; human smoke weightCheck + loadedExtracted if not done.  
2. `/grilling` (or improve-codebase-architecture) on **#3 seal/progress** — or user’s pick from board.  
3. `/to-tickets` → local `.scratch/living-worklist/issues/08-…` (prefer local).  
4. Implement / review; conventional commit on `feat/attesta-rework`.  
5. Refresh this board + `NEXT-thin-spa-candidates-…` status table.

---

## Suggested skills

| Skill | When |
| --- | --- |
| **grilling** / **improve-codebase-architecture** | Deepen #3 or refresh HTML report |
| **codebase-design** | Seal recipe seam / bundle interface |
| **domain-modeling** | Glossary if seal/progress nouns change |
| **to-tickets** | After grill — local `.scratch/…/issues/` |
| **tdd** | Seal contract JSON tests |
| **handoff** | End of session |
| **chief-of-staff** | Multi-agent implement loop |

Repo: `AGENTS.md`, `docs/agents/domain.md`, ADR-0005, `GLOSSARY.md` (Chrome / Task form bundle / Load).

---

## Hard calls (escalate)

- Reintroducing SPA Unlock / startsWhen / Domain aggregate projectors  
- Deleting `valuePrint` without checking default `stringFieldForChrome` callers  
- Unpausing JobChrome delete without human OK  

- Push / force without human OK  

## Out of scope for “remaining thin-SPA candidates”

- Re-implementing dataReport / Worklist 01–05  
- Flat-port ELT ticket 16+ (already landed on rework historically — don’t reopen as “candidates” here)  
- Product rename ticket **13** until names agreed  
