# Org rename hard-cut — coordinator

**Branch:** `docs/org-rename-lock`  
**Lock:** `.scratch/living-flat-port/issues/13-rename-placeholders-discussion.md`  
**Status:** **DONE** (integration pass 2026-11-16)

## Integration result

Hard-cut seams are in place: `server/internal/org`, `app/src/lib/org`, `/api/org`, Cerbos `task`/`commit` (`cerbos/policies/task_policy.yaml`), `ORG_HTTP_*` / `org-http`, Mongo `org_*`, SPA `OrgSession` / `orgSession`, Feragame `*@feragame.com`.

### Tests (integration)

| Target | Result |
| --- | --- |
| `cd server && go test ./internal/org/... -count=1` | **PASS** |
| `cd services/org-http && go test ./... -count=1` | **PASS** |
| `go test ./cmd/server -count=1 -run 'Org\|DPP\|Feragame\|Seed'` | **SKIP (pre-existing)** — `formata-arch/attesta/assets/*` embed glob missing in this worktree |
| App vitest | **SKIP** — no `app/node_modules` |

### Fixture YAML

- Root Org Config block: `dpp:` (parsed by `ParseFeragameConfigFixture` → `cfg.DPP`).
- Feragame step id `passport` and leaf `onSubmit.passport` are **domain** names (Job / DPP hook), not the retired root YAML key.

### Fixes in integration pass

- `server/templates/pages/org_dpp.html` — removed stale “living Passport” product copy.

## Remaining `living` grep (intentional / non-blocking)

**Hard-cut patterns** (`package living`, `internal/living`, `/api/living`, `LIVING_HTTP_*`, `living-http`, `LivingSession`, `livingResourceKind`, `collectionLiving*`, `living@attesta.com`, `lib/living`, `#lib/living`): **none** in `server/` (prod), `app/`, `deploy/`, `cerbos/`, `services/org-http/`, or `cmd/` (outside comments/test strings).

| Category | Where | Notes |
| --- | --- | --- |
| Archival scratch | `.scratch/living-flat-port/`, `.scratch/living-worklist/`, `.scratch/org-rename-slice1.py`, this file’s history rows | Historical rename notes |
| Research era docs | `docs/research/attesta-app-living-product-decisions.md`, `docs/research/attesta-living-flat-port-coordinator-handoff.md`, etc. | Product “living” glossary era; not runtime seams |
| Glossary prose | `GLOSSARY.md`, `GLOSSARY-MAP.md`, `docs/feragame/GLOSSARY.md`, `docs/agents/plant-config.md` (`jsonata-living.md`) | Domain language, not code paths |
| Go comments / test ids | `server/internal/org/*_test.go`, `server/cmd/server/org_test.go` (`livingCommit`, session ids), `seed_session.go` (`livingScope`), mongo integration db names `living_*_test` | Cosmetic; optional follow-up rename |
| Dev utility | `server/cmd/seed-loaded-extracted/main.go` — `UserID: "living"` | Local extractor only |

## Worker slices

1. **Go engine + host** — DONE  
2. **SPA** — DONE (`69c29ae1`)  
3. **Sidecar/deploy** — DONE (`b6ef43a3`)  
4. **Docs/Cerbos/ADRs** — DONE (`a31675c2`)

## Verify (post-commit)

- `go test ./server/internal/org/...`
- Optional: full `cmd/server` tests after `formata-arch` assets present
- `/api/living` must 404 at runtime (no dual mount)