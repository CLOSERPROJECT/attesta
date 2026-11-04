# Coordinator — Task groups on rework

**Integration / feature branch:** `feat/living-groups-rework` (cut from `feat/attesta-rework` @ aedfb5e).  
**Do not** implement on the `feat/task-groups` worktree as the primary line — it lacks the new GUI and still defaults product Commit to FakeHttp. Treat that worktree as a **donor** (cherry-pick / port source) only.  
**Tracker:** `.scratch/living-groups-rework/issues/` (local; not GitHub)  
**Grill:** integration session 2026-11-03 (glossary + ADR-0006 amended on rework).  
**Related:** `.scratch/living-http/` — Sidecar / living-http already on rework; ticket **08** proves groups Commits use it for real labels.  
**Coordinator handoff:** `.scratch/living-groups-rework/HANDOFF-coordinator-2026-11-03.md` (also `$TMPDIR/handoff-living-groups-rework-coordinator-20261103.md`).  
**Pipeline:** ticket → implementer → code review (Standards+Spec) → fixer → …; human smoke / squash onto `feat/attesta-rework`.

## Settled (do not re-litigate)

- **Hard call 2026-11-03:** Living Go must not encode Feragame Domain entity kinds (e.g. `"load"` in Worklist projection). Group-instance overview `label` join is Config-declared (ticket 11).

- Port onto rework spine (lazy task-form, Commit `next`, Sidecar HttpPort); JobPanel stays dead.
- Operator Spawn = first-leaf Commit (`spawningNew`); no product empty-open spawn HTTP.
- Spawn/Close chrome on **Job instance overview**. Close stays overview-only.
- Home Available: minted nested ready Tasks (ticket 06) **and**, when `canSpawn`, the **provisional first leaf** (`groupInstanceId` `new` → Commit `spawningNew`) — Available chrome that performs Spawn-via-first-leaf; **no** separate home Spawn band. Hard call **2026-11-03** (reopened Settled for A; prior law was overview-only Spawn).
- Nested task URL includes job / group / group instance (`new` when provisional) / task.
- Open group + child Commit without download links → overview; with download links → read-only form + links + Start another / Close (confirm); after Close → usual next task / overview / home.
- Stay-for-links applies to any Commit that returns downloads, not only group children.

## Frontier

**14–17 overview accordion** (status enum → link store → recap → shadcn overview). Parallel OK: **14** ‖ **15**; **16** after 15; **17** after 14+16.  
Loop this slice: **implement all, review once at end** (human 2026-11-03). Tip `feat/living-groups-rework` (not pushed). Do not push/seed `--replace` without ask.

## Active

_(clear)_ — board through **19** at tip. Pointer: `OVERVIEW-ACCORDION.md`.  
Human next: smoke overview accordion / labels / Close dialog; squash → `feat/attesta-rework` when OK. Do not push unless asked.

**Audit done ([Audit](9c5fdd0e-e3e8-498e-938a-3f4996773ad7)):**
| Sev | Leak | Action |
| --- | --- | --- |
| critical | Worklist `Kind("load")` | **11** `9bbc580` |
| high | Close `group.ID()=="lots"` | **12** `d77ca91` |
| medium | SPA dataReport `loadId` DTO | **13** `4e4d7e8` |
| medium | FakeHttpPort plant CSV cols | **not product** — FakeHttp removed from Commit (`ea16276`); leftover test double / sidecar drift only ([Explore](20fb3be1-a282-4797-a698-430e2c226655)) |

## Settled note (2026-11-03)

Human reopened ADR deferred labels-under-lot **narrowly**: Worklist group instances may carry optional `label` = Domain load id; overview shows `label || id`. Not a plant-only `loadId` JSON key.

**Group instance id vs Config ([Explore](e408c4c1-afb9-4e52-a466-86bc7b5886df)):** structural id (`lots-N`) is engine-minted on `spawningNew` (`nextGroupInstanceID`), not Config. Operator-facing load id is Config transform + Worklist `label`. Do **not** add Job-style group `instanceId` unless product wants URL ≡ load id (Spawn timing/uniqueness fork).

**Config group-instance `label` expr? ([Explore](bbd33c15-49da-41af-bf51-5a41a6f8ee7c)):** **Later.** Transform mint already authors operator identity; Worklist projects it. Job-twin durable group `label` would reopen ADR free-form labels. Risk of forever-`load` join: non-load groups blank/wrong — genericize join only when a second group needs identity.

**Clarify (human 2026-11-03):** question is not Task title — it is whether **“Worklist group-instance label = Domain load id”** should be **Config-declared** vs Go `Kind("load")` heuristic (ticket 10). Recommendation below: value stays mint; join rule should become Config when we touch it again (or now if hard-called).

## Done

01, 02, 03, 04, 05, 06, 07, 08 — all Standards+Spec APPROVE (with fixer loops where needed)  
Hard call A (2026-11-03) — home Available provisional first-leaf when `canSpawn`  
`073f322` — double-Commit fix (human smoke OK)  
09–10 — overview instances + card + load-id label (`4e6aed6`…`1ec3f3f` / `fb8c144`)  
11 — `instanceLabelEntity` (`9bbc580`)  
12 — Close Config-only, no `"lots"` fallback (`d77ca91`)  
13 — dataReport section chrome `id`/`secondaryIds` (`4e4d7e8`)  
14–17 — overview accordion slice (`3603bbd`)  
18 — single-child group occurrence flatten  
19 — overview accordion polish (`adf9516`)

## Tickets

| # | File | Blocked by |
| --- | --- | --- |
| 01 | `issues/01-server-groups-domain-worklist.md` | — |
| 02 | `issues/02-server-spawn-close-next.md` | 01 |
| 03 | `issues/03-overview-spawn-first-form.md` | 02 |
| 04 | `issues/04-overview-close-advance.md` | 03 |
| 05 | `issues/05-post-submit-links-loop.md` | 03, 04 |
| 06 | `issues/06-home-nested-available.md` | 02 |
| 07 | `issues/07-plant-path-check.md` | 05, 06, 08 |
| 08 | `issues/08-sidecar-labels-group-commit.md` | 02 |
| 09 | `issues/09-overview-group-instances.md` | — |
| 10 | `issues/10-overview-group-instance-label.md` | 09 |
| 11 | `issues/11-config-group-instance-label-entity.md` | 10 |
| 12 | `issues/12-close-gate-no-lots-fallback.md` | — |
| 13 | `issues/13-spa-datareport-section-dto.md` | — |
| 14 | `issues/14-worklist-actionable-status.md` | — |
| 15 | `issues/15-persist-commit-links.md` | — |
| 16 | `issues/16-lazy-task-recap.md` | 15 |
| 17 | `issues/17-overview-accordion-chrome.md` | 14, 16 |
| 18 | `issues/18-single-child-group-instance-flatten.md` | 17 |
| 19 | `issues/19-overview-accordion-polish.md` | 17 |
