# Handoff: supervise **Task-order / `$load.id`** smoke fix only

**Date:** 2026-10-30  
**Repo:** `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta`  
**Your job:** DRI for **issue #3 only** on `fix/smoke-issues` — the `instanceId must evaluate to a non-empty string` / Task-order decode fix. Do **not** own the whole branch or earlier hotfixes unless review forces a touch.

**Mode:** chief-of-staff style — implement/review/fix via **one subagent at a time**; sparse briefs; surface hard calls only.

---

## Scope (in)

- Commit **`7f0fee7`** — `fix(living): preserve Job Task order after Mongo JSON`
- Related tests: `commit_new_repro_test.go`, `config_json_roundtrip_test.go`, any follow-up on `OrgConfig.UnmarshalJSON` / `AsJob`
- Pipeline for **this** fix: if reviewer pending → wait; APPROVE → human smoke; CHANGES_REQUESTED → fixer → re-review
- Update this issue’s notes in COORDINATOR Diagnostics when done

## Scope (out)

- Owning `27e30d7` (`_new`) or `4999615` (AsJob coerce) as new work — already landed; only touch if your fix regresses them
- Squash-merge whole branch into `feat/attesta-rework` — **branch coordinator** owns that
- Ticket 13 rename, FormPlan polish, HC-seed-replace, Task groups

---

## Symptom → cause → fix (locked)

| | |
| --- | --- |
| **UI** | `Commit denied: living: instance type: instanceId must evaluate to a non-empty string` after Nuovo camion |
| **Cause** | Mongo/JSON Job decode used `AsJob(map)` → nested `tasks` remarshaled via **unordered** map → alphabetical order → first Task **`codiceFir`** not `register` → mint gate without `$load` → `$load.id` undefined |
| **Fix landed** | `OrgConfig.UnmarshalJSON` decodes each Job from `json.RawMessage` so `Tasks` OrderedObject keeps authored order; `AsJob` no longer map-remarshals |
| **Do not** | Change fixture `instanceId: $load.id` |

**Diagnoser (authority):** [Diagnose empty instanceId expr](8edde440-8a09-449f-be58-495d4090a621)  
**Fixer:** [Fix Job Task order decode](ddc6d54a-e4d7-4e44-a749-534aa71a2137) → `7f0fee7`  
**Reviewer:** [Review Task order fix](a49181b2-cfe4-4889-8b27-9ebd4df457ac) — **VERDICT: APPROVE** (2026-10-30). Next: human smoke; branch coordinator owns squash.

Longer narrative (cascade #1–#2): `.scratch/living-flat-port/HANDOFF-smoke-instanceId-2026-10-30.md` (partially stale tip SHAs — prefer this file + `git log`).

---

## Git

```text
Branch: fix/smoke-issues
Base:   feat/attesta-rework @ fb82445 (== origin when branched)
Tip:    7f0fee7  (ahead: 27e30d7 → 4999615 → 7f0fee7)
```

Uncommitted scratch may exist: `COORDINATOR.md`, `HANDOFF-smoke-instanceId-*.md` — fold into docs commit if useful; do not block product fix.

---

## Your routine

1. Confirm reviewer verdict for `7f0fee7` (or resume/spawn reviewer if missing).
2. If CHANGES_REQUESTED → one fixer → re-review.
3. Ask human to smoke **this** path only (below).
4. Report to **branch coordinator**: APPROVE + smoke green/red; do not squash alone unless they ask.

## Verify

```bash
cd server && go test ./internal/living/... ./cmd/server -count=1
# especially:
go test ./internal/living/ -run 'TestSmokeNuovoCamion|TestCommitRegisterNewAfterMongoJSON|JSONRoundtrip' -count=1 -v
```

Acceptance: after Mongo/JSON config path, `OrderedTaskIDs(intake)[0] == "register"`; `_new` register Commit mints `NIR-261019-NO-1` (fixture grammar).

## Human smoke (this fix)

1. Checkout `fix/smoke-issues`; restart `task:dev` if needed (Mongo Active Config re-read).
2. `/app` → **Nuovo camion** → first form must be **register** (not Codice FIR).
3. Submit register → minted `NIR-…`; no instance-type / invalid Job / missing id errors; `codiceFir` unlocks.

## Hard calls

None expected. If someone proposes changing fixture exprs instead of decoder fidelity → **reject** (config is authority; Feragame stays in YAML).
