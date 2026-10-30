# Plan: Commit mint intent (kill `_new`)

**Status:** implemented + reviewer APPROVE — uncommitted; human smoke / commit when ready  
**Branch:** `fix/smoke-issues`  
**Owner:** chief-of-staff session

## Locked product

- First Task Commit mints the Job instance (config `instanceId`/`label` after transform).
- Nuovo camion / Nuovo turno = provisional UI only (no HTTP until first Task Commit).
- Start privilege = first Task `rolesAllowed`. Cerbos stays single action `commit` (no `start` action).
- `_new` on the wire was a mistake — remove; do not deepen.

## Wire contract

| | `startingNew` | `none` (omit `mint` → `none`) |
|---|---|---|
| `taskPath` | required | required |
| `jobInstanceId` | omit / `""`; **ignore** any client id | required, real id |
| `mint` | `{ "kind": "startingNew" }` | `{ "kind": "none" }` |
| `value` | sealed | sealed |

- Receive `_new` as `jobInstanceId` → hard prep error.
- Defer `mintPending`.
- Draft key may keep local `_new`; never send on wire.
- Response `jobInstanceId` = minted/existing; client `onCommitted` uses **response** id.

## Server mint rules

1. `startingNew` → Task must be first in authored order; evaluate exprs after transform; uniqueness vs **evaluated** id (not client id). If instance already exists → deny (prep), do not append duplicate.
2. `none` → require non-empty id; never mint.
3. Harden Mutate: reject duplicate `(jobId, id)` JobInstance append.
4. Authorize before mint with empty instance id (Unlock already allows empty on first Task).

## Client

- Provisional TaskForm (`jobInstanceId === ''`): `mint.startingNew`, omit id.
- Instance-bound: `mint.none` + real id.
- Clear draft using provisional key; `onCommitted(result.jobInstanceId)`.

## Tests to absorb / rewrite

- `server/internal/living/commit_new_repro_test.go`
- `server/internal/living/line_shift_new_dup_repro_test.go` → second `startingNew` + same `shift:1st` must not duplicate Domain rows.

## Acceptance

1. Register provisional Commit mints NIR id; no `_new` in Domain.
2. Nuovo turno `1st` twice → one `line`/`1st`; no `each_key_duplicate`.
3. Mid-Job Commit uses `none` + real id.
4. Do not “fix” only by changing JobPanel each-key.

## Context pointers

- `.scratch/living-flat-port/HANDOFF-smoke-instanceId-2026-10-30.md`
- `.scratch/living-flat-port/HANDOFF-new-sentinel-turno-dup-2026-10-30.md`
- Experiment: `attesta-experiment/src/lib/substep-submit-mint.ts`, `commit-http-dto.ts`
- Key files: `server/internal/living/commit.go`, `instance_exprs.go`, `mutate.go`; `app/src/lib/living/TaskForm.svelte`, `commit-port.ts`, `http-commit-port.ts`, `session.svelte.ts`

## Out of scope

- Separate create HTTP route; ticket 13 rename; Task groups; OrderedObject rewrite unless red test forces it after mint fix.
