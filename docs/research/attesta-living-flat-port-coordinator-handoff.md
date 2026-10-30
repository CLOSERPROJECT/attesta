# Handoff: coordinate living flat-port implementation

**Date:** 2026-10-29  
**Repo:** `forkbomb/attesta`  
**Branch:** `feat/attesta-rework`  
**Job:** coordinate (and optionally drive) the flat truck-loop port into `/app` + Go using the ticket queue. Do **not** implement Task groups / lot intake.

## Read first (authority order)

1. [`docs/research/attesta-app-living-product-decisions.md`](attesta-app-living-product-decisions.md) — locked domain, port cut, placeholders, **copy/rewrite policy**, Cerbos, persistence  
2. [`GLOSSARY.md`](../../GLOSSARY.md) + [`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md) — living nouns (Job/Task/Domain/…); legacy Stream stays in `CONTEXT.md`  
3. Tickets: [`.scratch/living-flat-port/issues/`](../../.scratch/living-flat-port/issues/) **01–13**  
4. Source behaviour: `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta-experiment` (fixture `src/lib/fixtures/truck-loop-steps.yaml`, pure `src/lib/*`)  
5. Existing SPA mount: `server/cmd/server/app_spa.go` (`/app`, verified session)

Optional: scenario challenge notes in `attesta-glossary-scenario-challenge-handoff.md` (pressure-test context; nesting deferred).

## Goal

Ship a **demoable flat** living product: org Domain + versioned Org Config fixture, Load/Commit over placeholder HTTP, Cerbos on Commit, `/app` UI for intake + line — **without** whole-config-execution / run-id identity.

## Ticket frontier

| # | Title | Blocked by |
| --- | --- | --- |
| **01** | Living Load — empty Domain + fixture Org Config | — **start here** |
| 02 | Commit skeleton — register (test Authorizer) | 01 |
| 03 | Cerbos on living Commit | 02 |
| 04 | Intake Unlock order (simple Tasks) | 03 |
| 05 | Intake Leftovers `all` (palletWeight) | 04 |
| 06 | Line `startsWhen` opens Job | 05 |
| 07 | Line Job instances + Leftovers `some` | 06 |
| 08 | Line remaining Tasks | 07 |
| 09 | `/app` Load shell + HTTP ports | 03 (parallel after Cerbos) |
| 10 | `/app` intake Commit UI | 05, 09 |
| 11 | `/app` line Commit UI | 08, 10 |
| 12 | Demo seed / Domain wipe + all-roles user | 08 |
| 13 | Rename placeholders (discussion + code) | 11, 12 |

Work **one ticket at a time** when implementing; as coordinator, keep the queue and blockers honest (mark done in the ticket file Status / checkboxes).

## Code seam placeholders (until 13)

| Placeholder | Role |
| --- | --- |
| `server/internal/living` | Go package |
| `DomainStore` / `ConfigStore` | Persistence Interfaces |
| `GET /api/living/domain` | Load |
| `POST /api/living/commit` | Commit (Task path + Job instance id + sealed value) |
| Cerbos kind `living` | Draft Commit resource |

Rejected even as placeholders: `/api/runs`, `RunStore`, `LoadRunPort`.

## Process

- Slice branches off `feat/attesta-rework`; **no PR required**; squash into rework after human smoke.  
- Conventional Commits for commits.  
- Ask before breaking existing routes / workflow config / progress-key encoding.  
- Additive only — do not replace HTMX `/my` or merge UI into `web/` templates.

## Port policy (summary)

- **Copy + adapt:** pure TS Modules/tests, fixture, UI → `app/`; rename Step→Job etc. on the Commit path.  
- **Rewrite:** Kit `+server` / SQLite / run-id HTTP → Go living + stores; Go Commit accept path; no harness PUT.  
- **Don’t copy:** spikes, Task group/lot work, harness product UI.

Full text in the decisions dump.

## Do / don’t

**Do:** drive 01→…; keep glossary nouns in code/docs you touch; wire session like `/app`; Cerbos sole allow/deny on product Commit after ticket 03; path-shaped Commit for later nesting.

**Don’t:** implement Task group / Feragame lot reshape; revive run-id Domain; fake lots with Leftovers `some`; big-bang `#249` extract; rename placeholders before 13; rewrite decisions/glossary unless the human asks.

## Suggested agent skills

- `domain-modeling` — if a new product noun appears  
- Svelte MCP / `svelte-file-editor` — any `.svelte` under `app/`  
- `attesta-ui-components` — only if bridging Go templates/CSS (unlikely for `/app`)  
- `diagnosing-bugs` — proxy / SPA / session regressions  
- codebase-memory — Go mux / session / Cerbos call graph when wiring

## Verify when a slice lands

Per ticket acceptance checkboxes, plus as the stack grows:

```bash
cd server && go test ./internal/living/... ./cmd/server -count=1
# after app UI tickets:
cd app && bun run check && bun run build
# smoke: task:dev → login → /app → Load; later Commit intake/line
```

## When flat port is done

Ticket **13** naming discussion + rename; then human can freeze experiment edit target and plan nesting / lot intake as a separate queue.