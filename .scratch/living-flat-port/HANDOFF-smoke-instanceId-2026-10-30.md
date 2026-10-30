# Handoff: living `/app` smoke — `instanceId must evaluate to a non-empty string`

**Date:** 2026-10-30  
**Repo:** `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta`  
**For:** next agent fixing smoke Commit after Nuovo camion / register  
**Coordinator mode:** chief-of-staff — prefer implement via subagent; pipeline ticket → branch → implementer → reviewer → fixer → squash

---

## Current symptom (blocking smoke)

After opening **Nuovo camion** and submitting the first Task (**register**):

```text
Commit denied: living: instance type: instanceId must evaluate to a non-empty string
```

Emitted by `LivingShell` prefixing `living.commitError`. Server source:

```text
server/internal/living/instance_exprs.go
  requireInstanceString → InstanceExprError{Kind: "type", Message: name + " must evaluate to a non-empty string"}
  EvaluateJobInstanceExprs → job.InstanceID / job.Label via evaluateExpr
```

Exact wire message shape: `living: instance type: instanceId must evaluate to a non-empty string` (see `InstanceExprError.Error()`).

---

## Branch / git state

| Ref | Tip | Notes |
| --- | --- | --- |
| `feat/attesta-rework` | `fb82445` | matches origin; tickets 14–19 already on this line |
| **`fix/smoke-issues`** | `4999615` | **work here** — two APPROVED hotfixes ahead of rework |

Hotfixes already on `fix/smoke-issues` (do not rediscover):

1. `27e30d7` — `fix(app): wire provisional Commit with _new instance id`  
   Provisional TaskForm Commit sends `jobInstanceId: '_new'` (was client plant mint via `computeIntakeInstanceId` → empty sealed defaults → “Missing Job instance id”).
2. `4999615` — `fix(living): coerce JSON-decoded Jobs with AsJob`  
   Mongo/JSON `OrderedObject` left Jobs as `map[string]any` → `invalid Job "intake"`. `AsJob` + `normalizeOrgConfigJobs` on Mongo Get + JSON round-trip regression.

**Untracked WIP from diagnoser (may exist):**  
`server/internal/living/commit_new_repro_test.go` — provisional `_new` / missing idoneita / JSON-roundtrip probes. Keep, extend, or delete intentionally; do not leave orphaned.

---

## Smoke cascade (already fixed → current)

| # | Error | Cause | Fix |
| --- | --- | --- | --- |
| 1 | `Missing Job instance id for Commit` | Client required plant mint before HTTP; FormPlan sealed value often omitted schema defaults (`idoneitaRiutilizzo`) | `_new` sentinel (`27e30d7`) |
| 2 | `invalid Job "intake"` | Mongo Config JSON decode → Jobs as maps; `.(Job)` failed | `AsJob` / normalize (`4999615`) |
| **3** | **`instanceId must evaluate to a non-empty string`** | **Confirmed** — see Root cause (resolved by diagnoser) | fixer next |

Prior diagnoses: [missing jobInstanceId](bc272911-75eb-4707-b6dc-edd33c1ab547); fixer [invalid Job](26b6ebb0-3d7c-4f98-b281-1cf393db08d9); reviewer APPROVE [hotfix](60fa6fd2-9ae7-40d7-b07b-29aba836c17f). Diagnoser #3: [Diagnose empty instanceId expr](8edde440-8a09-449f-be58-495d4090a621) — **done**.

## Root cause (confirmed)

Mongo/JSON + `AsJob(map)` re-marshals nested `tasks` through an **unordered** map → intake Task order becomes alphabetical (`codiceFir` first). Nuovo camion provisional-Commits **codiceFir** (mint gate on, transform is patch only — **no `$load`**) → `$load.id` undefined → exact UI error.

Do **not** change fixture `instanceId: $load.id`. Fix decoder: decode each Job from `json.RawMessage` into `Job` so `Tasks` OrderedObject keeps authored order. Red/green tests in `commit_new_repro_test.go` (`TestSmokeNuovoCamionInstanceIdAfterMongoJSON` red-capable; `TestCommitRegisterNewAfterMongoJSONMintsLoadID` fix acceptance).

---

## Product law / first step (not a UX mis-order)

| Step | Meaning |
| --- | --- |
| Button **Nuovo camion** | Job `startLabel` — opens **provisional** first Task UI only (no Commit) |
| First Task | **`intake/register`** (fixture order) — this Commit mints the Job instance |
| Chrome title | May show raw `register` (no Italian `schema.title` at top level) — cosmetic |

Fixture intake Job exprs (`server/internal/living/fixtures/truck-loop-steps.yaml`):

```yaml
instanceId: $load.id
label: $load.id
```

Register transform mints load with `as: load` and id grammar  
`{idoneitaRiutilizzo}-{YYMMDD}-{client}-{n}` then Job instance id/label should equal `$load.id`.

Server mint gate: `shouldMintJobInstance` — first Task in authored order + no existing instance for the provisional id → `EvaluateJobInstanceExprs` then `NewJobInstance`.

---

## Hypotheses for #3 (investigate in order)

### H1 — `$load` missing at instance-expr time (best match for exact error)

`evaluateExpr` maps JSONata `ErrUndefined` → `(nil, nil)`; then `requireInstanceString` → **exact user error**.

`$load` is set only if register transform runs `mint` with `as: load` (`transform.go` aliases). If `onSubmit.transform` is missing/empty after config decode, or mint never binds `as`, `$load.id` is undefined → this error.

### H2 — Job / Task order destroyed by JSON ↔ `OrderedObject` (seen in local probe)

Diagnoser probe `TestCommitRegisterJSONRoundtripConfig` (in untracked repro file) logged:

- `InstanceID="$load.id"` (expr string OK after `AsJob`)
- Commit **succeeded** but `jobInstanceId` stayed **`_new`**

That implies `shouldMintJobInstance` returned **false** (likely `OrderedTaskIDs(job.Tasks)[0] != "register"` because nested `tasks` lost key order when Jobs were `map` → `json.Marshal` map → `Job`). Different failure mode than the user’s message, but **same family**: Mongo/JSON OrderedObject fidelity for nested Tasks is fragile. Fix may need ordered decode of Job.tasks (not bare `map` round-trip).

### H3 — Sealed FormPlan value incomplete

Client seal may omit defaults (`idoneitaRiutilizzo`, etc.). Local probes with missing/empty idoneita still returned `err=<nil>` in one run (surprising — may mint weird ids or skip mint). Still verify Network tab sealed body vs `registerValue()` in tests.

### H4 — Not `_new` itself

`TestCommitRegisterWithProvisionalNew` with full `registerValue()` + `_new` **PASS**es and mints `NIR-261019-NO-1` on Memory+fixture path. So `_new` + happy config + full value is fine.

---

## Tight loops to run first

```bash
cd server && go test ./internal/living/ -run 'TestCommitRegisterWithProvisionalNew|TestCommitRegisterJSONRoundtrip|TestLookupTaskAfterJSON' -count=1 -v
```

Then add a red test that fails closed on the **user** error:

1. JSON-roundtrip (or Mongo Get) Active Config  
2. Commit `intake/register` + `_new` + `registerValue()`  
3. Assert minted id matches load grammar **and** `shouldMint` actually ran (not leftover `_new`)  
4. Optionally Commit with FormPlan-like sparse value and assert clear prep/transform error (not silent wrong id)

SPA: Network on failed Commit — request body (`jobInstanceId`, `value` keys) + response error string.

---

## Key files

| Area | Path |
| --- | --- |
| Error | `server/internal/living/instance_exprs.go` |
| Mint after transform | `server/internal/living/commit.go` (`applyConfigCommit`, `shouldMintJobInstance`) |
| `$load` alias | `server/internal/living/transform.go` (`as`, `aliases`) |
| Expr / undefined | `server/internal/living/expr.go` |
| Job coerce | `server/internal/living/config.go` (`AsJob`, `normalizeOrgConfigJobs`) |
| Ordered decode | `server/internal/living/ordered.go` |
| Mongo config | `server/internal/living/store_mongo.go` |
| Fixture | `server/internal/living/fixtures/truck-loop-steps.yaml` |
| Client provisional | `app/src/lib/living/TaskForm.svelte` |
| Job start UI | `app/src/lib/living/JobPanel.svelte` |
| Ledger | `.scratch/living-flat-port/COORDINATOR.md` |
| Gaps inventory | `.scratch/living-flat-port/issues/15-…` |
| Decisions | `docs/research/attesta-app-living-product-decisions.md` |
| Experiment | `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment` (`transform-runner`, step instance exprs) |

---

## Broader context (flat-port follow-ons)

Done on rework (squashed): **16** ELT Commit, **17** readyWhenOk, **18** HttpPort+Fake, **14** FormPlan UI, **19** Mongo Domain/OrgConfig.  
Paused: **13** rename. Inventory-only: **15**.  
Demo login may still need `task seed -- --replace` (HC-seed-replace; destructive). Env wipe: `ATTESTA_LIVING_SEED`.

Out of scope for this smoke fix: Task groups / lot intake; ticket 13 rename; HTMX `/my`.

---

## Suggested fix direction (not locked)

1. **Preserve Task (and Job) key order** across JSON/Mongo decode — nested `tasks` must not go through unordered `map[string]any` marshal when coercing to `Job`.  
2. Ensure after Get/normalize, first Task id is still `register` and `onSubmit.transform` still present with `as: load`.  
3. Keep `_new` provisional client behaviour; server remains mint authority via `$load.id`.  
4. Add regression: round-trip config + `_new` register Commit → real NIR id, never `_new`, never instance type error.

---

## Smoke after fix

1. Checkout `fix/smoke-issues` (+ your commits); restart `task:dev` if needed.  
2. Login living demo / feragame.  
3. `/app` → Nuovo camion → fill register → Commit.  
4. Expect: minted instance id (`NIR-…`), provisional closes, `codiceFir` unlocks; no Missing id / invalid Job / instance type errors.

---

## Hard calls

None locked for #3 yet. If fix requires changing fixture `instanceId` exprs vs decoder fidelity, prefer **decoder/order fidelity** (product law: config is authority; Feragame stays in YAML).
