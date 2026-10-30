# Handoff: coordinate whole **`fix/smoke-issues`** branch

**Date:** 2026-10-30  
**Repo:** `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta`  
**Your job:** chief-of-staff DRI for the **entire** smoke hotfix branch until it is reviewed, human-smoked, and **squash-merged into `feat/attesta-rework`** (or further issues are filed/fixed on the same branch).

**Mode:** all implement/review/fix/diagnose via **one subagent at a time**; coordinator keeps ledger + pipeline; surface **hard calls only**.

---

## Mission

Unblock living `/app` truck-loop smoke after flat-port follow-ons (14–19). Humans hit a **cascade** of Commit denials; this branch stacks the fixes. Your success = Nuovo camion → register → continue intake works on a stack running this branch, then squash onto rework.

---

## Git map

| Ref | SHA / tip | Role |
| --- | --- | --- |
| `feat/attesta-rework` | `fb82445` | Base; tickets 14–19 already squashed here; **do not** leave hotfixes only on rework without branch |
| **`fix/smoke-issues`** | `7f0fee7` | **Active work branch** |

```text
fb82445 (rework)
  └─ 27e30d7  fix(app): wire provisional Commit with _new instance id
       └─ 4999615  fix(living): coerce JSON-decoded Jobs with AsJob
            └─ 7f0fee7  fix(living): preserve Job Task order after Mongo JSON
```

**Policy:** keep smoke fixes on `fix/smoke-issues` until APPROVE + human smoke; then squash into rework (same pipeline as living tickets). Do not push unless human asks.

Scratch uncommitted: `.scratch/living-flat-port/COORDINATOR.md`, handoff md files — commit as docs when convenient.

---

## Cascade (full branch story)

| # | Error | Root cause | Commit | Review |
| --- | --- | --- | --- | --- |
| 1 | `Missing Job instance id for Commit` | Client plant mint needed sealed defaults FormPlan often omits | `27e30d7` provisional `_new` | APPROVE ([hotfix review](60fa6fd2-9ae7-40d7-b07b-29aba836c17f)) |
| 2 | `invalid Job "intake"` | Mongo JSON → Jobs as `map`; `.(Job)` fail | `4999615` AsJob + normalize | same APPROVE |
| 3 | `instanceId must evaluate to a non-empty string` | `AsJob(map)` alphabetized Tasks → first=`codiceFir` → no `$load` | `7f0fee7` RawMessage Job decode | **APPROVE** ([Review Task order fix](a49181b2-cfe4-4889-8b27-9ebd4df457ac)) |

**Issue #3 specialist handoff** (delegate, don’t duplicate):  
`.scratch/living-flat-port/HANDOFF-supervise-task-order-fix-2026-10-30.md`

**Deep diagnosis notes:**  
`.scratch/living-flat-port/HANDOFF-smoke-instanceId-2026-10-30.md` (tip SHAs may lag; trust `git log`)

**Diagnoses / agents:**  
- #1 [bc272911…](bc272911-75eb-4707-b6dc-edd33c1ab547)  
- #2 fixer [26b6ebb0…](26b6ebb0-3d7c-4f98-b281-1cf393db08d9)  
- #3 [8edde440…](8edde440-8a09-449f-be58-495d4090a621) → fixer [ddc6d54a…](ddc6d54a-e4d7-4e44-a749-534aa71a2137)

---

## Your pipeline (branch-level)

```text
per issue: diagnose? → fix on fix/smoke-issues → one reviewer → fixer → …
branch: human smoke full path → squash merge into feat/attesta-rework → update COORDINATOR
```

1. **Close #3** — ensure Task-order fix is APPROVE + green tests; specialist agent can own that slice.
2. **Human smoke** whole branch (not just register mint):
   - Nuovo camion → **register** first → Commit → minted NIR id  
   - `codiceFir` unlocks → continue at least one more intake Task if time  
   - No: missing id / invalid Job / instance type errors  
3. **If new smoke errors** appear → diagnose subagent → fix on **same branch** → review (extend cascade table).
4. **Squash** into `feat/attesta-rework` when human OK (one conventional commit summarizing smoke unblock, or keep series if human prefers — default: squash like tickets).
5. Update `.scratch/living-flat-port/COORDINATOR.md` Diagnostics + Done notes.

---

## Product / first-step law (don’t rediscover)

| UI | Meaning |
| --- | --- |
| **Nuovo camion** | `startLabel` — provisional UI only |
| First Task | must be **`intake/register`** (authored order) |
| Instance id | fixture `instanceId: $load.id` after register mint `as: load` — **server authority** |

Demo: `living@attesta.com` / org `feragame`; wipe `ATTESTA_LIVING_SEED`. HC-seed-replace (`task seed -- --replace`) still open — ask human before destructive seed.

---

## Verify (branch)

```bash
cd server && go test ./internal/living/... ./cmd/server -count=1
cd app && bun run check && bun run test:unit -- --run src/lib/living
```

Living stack must load **Mongo** Org Config (product path) — Memory-only unit tests hid #2/#3.

---

## Out of scope for this branch

- Ticket **13** rename (paused)  
- FormPlan accordion / HttpLinkStrip polish  
- Task groups / lot intake  
- Merging into HTMX `web/`  
- Rewriting decisions dump / glossary unless a new product noun appears  

Flat-port follow-ons already on rework: **16** ELT, **17** readyWhen, **18** HttpPort, **14** FormPlan, **19** Mongo stores. Ledger: `.scratch/living-flat-port/COORDINATOR.md`. Decisions: `docs/research/attesta-app-living-product-decisions.md`. Experiment: `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment`.

---

## Environment improvements (strategic)

- Prefer fail-closed tests that **JSON/Mongo-roundtrip** Active Config before Commit (unit Memory alone is insufficient).
- Keep OrderedObject / Job decode as the single order authority — no plant switches.
- After squash, consider deleting obsolete `AsJob(map)` paths if any remain.

---

## Immediate next actions

1. ~~Reviewer~~ — **APPROVE** on `7f0fee7`.  
2. **Ask human to smoke** on `fix/smoke-issues` (Nuovo camion → register first → NIR mint).  
3. On green smoke → squash whole branch into `feat/attesta-rework`; on new error → diagnose on same branch.  
4. Specialist handoff still useful for any #3 follow-up: `HANDOFF-supervise-task-order-fix-2026-10-30.md`.
