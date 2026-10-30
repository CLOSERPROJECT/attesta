# Handoff: supervise **Worklist / Task form bundle / wire** branch

**Date:** 2026-10-30  
**Repo:** `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta`  
**Your job:** chief-of-staff DRI for **`feat/living-worklist`** until tickets **01–05** are each implemented, fixed, reviewed, and **committed once per ticket** on this branch (then human smoke / squash into `feat/attesta-rework` when asked).

**Mode:** all implement / fix / review via **one subagent at a time**; keep a ledger; surface **hard calls only**. Do not open GitHub issues for these tickets — local tracker only.

---

## Mission

Move living `/app` chrome off SPA Unlock/startsWhen plants and (later) off full Domain + Org Config on the wire:

1. Server projects **Worklist** (gates + leftover-open → statuses).  
2. Server ships **Task form bundle**s (minimum form mount payload).  
3. Load/Commit stop shipping full Domain/Config to the SPA.

Success = truck-loop Jobs chrome works from Worklist (+ bundles as tickets land); living code stays **generic** (plant names only in YAML / fixtures / seed).

---

## Git map

| Ref | Tip (at handoff) | Role |
| --- | --- | --- |
| `feat/attesta-rework` | `d61be2d` (parent of branch) | Base; do not land product commits here until squash |
| **`feat/living-worklist`** | `1b1886c` | **Active work branch** |

```text
d61be2d  feat/attesta-rework (smoke-issues squash recorded)
  └─ 1b1886c  docs(living): ADR Worklist server projection
              (+ uncommitted .scratch/living-worklist/ — commit as docs when convenient)
```

**Commit policy (locked):**

- **One conventional commit per ticket** on `feat/living-worklist` after that ticket’s reviewer **APPROVE** (or fixer green + re-APPROVE).  
- Do **not** squash tickets together on this branch.  
- Do **not** push unless the human asks.  
- Squash-merge the **whole branch** into `feat/attesta-rework` only when human OK (separate step).

Suggested commit titles:

| Ticket | Suggested message |
| --- | --- |
| 01 | `feat(living): project Worklist on Load` |
| 02 | `feat(living): return Worklist on Commit and drop SPA gates` |
| 03 | `feat(living): Task form bundle for first available Task` |
| 04 | `feat(living): Task form bundles for all available and provisional` |
| 05 | `feat(living): Load and Commit without full Domain and Org Config` |

Docs-only (already landed): `1b1886c` ADR + glossary + decisions amend. Fold `.scratch/living-worklist/` into a `docs(living): …` commit when useful; do not block product tickets.

---

## Local tickets (canonical)

Directory: `.scratch/living-worklist/issues/`

| # | File | Blocked by | Frontier |
| --- | --- | --- | --- |
| **01** | `01-worklist-on-load.md` | — | **yes — start here** |
| 02 | `02-worklist-on-commit-remove-spa-gates.md` | 01 | after 01 done |
| 03 | `03-task-form-bundle-first-available.md` | 01 | parallel with 02 after 01 |
| 04 | `04-task-form-bundles-all-available-provisional.md` | 03 | after 03 |
| 05 | `05-wire-without-full-domain-config.md` | 02 + 04 | last |

**Ledger:** `.scratch/living-worklist/COORDINATOR.md` (update after each ticket stage).

**Closed by mistake (ignore):** GitHub #278–#282 — tracking is local only.

---

## Settled product law (do not re-grill)

Authority: ADR-0005, `GLOSSARY.md` (**Worklist**, **Task form bundle**, **Load**), decisions dump amend (no production TS Unlock/startsWhen).

| Decision | Rule |
| --- | --- |
| Gate authority | Go only; SPA reads Worklist statuses |
| Worklist shape | `jobs[]` → `jobId`, `canStart`, `instances[]` → `{ id, label, tasks: { [taskId]: locked\|available\|done } }` |
| Leftover clear | status `done` (no fourth status) |
| startsWhen blocked copy | UI string for now (`!canStart`) |
| readyWhen | Commit-only; not in Worklist v1 |
| Roles | not in Worklist v1 |
| Wire (01–04) | Domain + Config may remain until **05** |
| Task form bundle | schema/uiSchema, initial data, leftover ids/columns, Chrome/Derived **rules**, seal ids, title, **bindings bag**; live Chrome/Derived stay client-side |
| Provisional | empty `jobInstanceId` bundle when `canStart` (ticket 04) |
| Genericity | no plant Job/Task ids in projector/UI conditionals — only YAML/fixture/seed |

---

## Pipeline (per ticket)

```text
implementer → reviewer → (fixer → reviewer)* → one commit on feat/living-worklist → mark ticket done in COORDINATOR
```

1. **Claim frontier ticket** (update COORDINATOR Stage).  
2. **Implementer** subagent: read ticket + ADR-0005 + GLOSSARY; implement vertical slice; run tests below; leave tree ready to commit but **supervisor commits** (or instruct implementer to commit only if you want them to — default: supervisor creates the one commit after APPROVE).  
3. **Reviewer** subagent: Standards + Spec against the ticket acceptance criteria and ADR; verdict APPROVE or CHANGES_REQUESTED.  
4. If CHANGES_REQUESTED → **fixer** → re-review until APPROVE.  
5. **One commit** for that ticket (conventional message).  
6. Mark ticket **done** in COORDINATOR; open next frontier (01 unlocks 02 and 03; 05 waits for both 02 and 04).

**Parallelization:** one subagent at a time. After 01, prefer finish **02** then **03** (or 03 then 02) — do not interleave two implementers. 03 does not need 02.

**Human smoke:** after **02** (Worklist chrome) and again after **05** (thin wire). Ask human; do not invent stack credentials.

---

## Verify (every ticket)

```bash
cd server && go test ./internal/living/... ./cmd/server -count=1
cd app && bun run check && bun run test:unit -- --run src/lib/living
```

Prefer tests that exercise **JSON Load/Commit shapes** (Worklist / bundles on the wire), not only Memory unit helpers. Product Load path may use Mongo Active Config — Memory-only tests hid past smoke bugs.

---

## Out of scope (pause)

Not on this branch unless a ticket acceptance criterion forces a tiny touch:

- Twin truck-loop fixture DRY  
- Delete JobChrome pass-through  
- Rename `feragame*` chrome widget registry  
- Task groups / lot intake / Cerbos on Load  
- Ticket 13 rename (flat-port)  
- GitHub issue publish  
- Unrelated architecture-review candidates  

---

## Hard calls (escalate to human)

- Changing Worklist status vocabulary or shipping full Domain forever  
- Keeping production TS Unlock/startsWhen after 02  
- Putting Feragame/intake/line Task path switches in Go/app living packages  
- Merging multiple tickets into one commit  
- Push / squash to `feat/attesta-rework` without human OK  

---

## Immediate next actions

1. Commit or leave scratch ledger as needed; stay on `feat/living-worklist`.  
2. Spawn **implementer** for **01** (`01-worklist-on-load.md`).  
3. Review → fix loop → **one commit** → mark 01 done.  
4. Continue 02 ∥ 03 frontier per ledger until 05 complete.  
5. Ask human for smoke; only then discuss squash into `feat/attesta-rework`.
