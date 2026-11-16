# Implementation plan — submit cardinality + lot session

Depends on [`LOCKED.md`](./LOCKED.md) and
[`docs/adr/0008-living-task-submit-cardinality.md`](../../docs/adr/0008-living-task-submit-cardinality.md).
Follow [`docs/agents/plant-config.md`](../../docs/agents/plant-config.md) for
fixture followers. **Dumb executor:** no Feragame-named branches in Go/SPA.

Do not start until explicitly asked. Prefer small PRs in this order.

## Phase 0 — Docs already landed

- [x] LOCKED + ADR-0008 + glossary terms (this board)
- [ ] Optional: GitHub wayfinder map / child issues mirroring phases below

## Phase 1 — Engine: `submit: once | many`

**Goal:** generic cardinality without plant ids.

1. Org Config parse/validate: optional Task key `submit: once | many` (default
   `once`). Reject on Task groups if groups still parse.
2. Worklist projection: for `many`, after Unlock + readyWhen + role split, never
   map a Submission to `done`; stay `actionable`/`available`/`unavailable`.
3. Commit / Cerbos: allow repeat Commit on `many` while readyWhen holds; still
   append Submission. Once Tasks unchanged (second Commit denied / already done).
4. Job “finished” / In progress / Completed bands: redefine completion for Jobs
   that include `many` Tasks (e.g. finished when all **once** bookends that
   matter are done and no `many` Task is still `actionable`/`unavailable` with
   open Domain work — lock exact rule in ticket; likely: instance complete when
   `closeTurno`/`weightCheck`/passport once leaves done and no open/parked lot
   requiring line work — Config-driven via readyWhen emptiness, not special
   cases).
5. Tests in `internal/living` for Worklist + Commit many vs once (synthetic
   mini-config, not Feragame-named branches).
6. Amend ADR-0005 wire comments in code docs if any duplicate the old done law.

**Done when:** synthetic config with one `many` Task stays actionable across
two Commits; `once` still flips to `done`.

## Phase 2 — Domain rename `load` → `lot` + status enum

**Goal:** entity shape without yet flattening all groups.

1. Fixture `domain.entities`: rename `load` → `lot`; fields include status
   `open | parked | fullyProcessed` (replace boolean `fullyProcessed`).
2. Update all Config exprs, passport subject, transforms, reads, seed,
   demo_commits, living tests, SPA fixture parsers.
3. Glossary already names **lot**; keep seed ids stable or migrate deliberately
   in one replace seed (`task seed:feragame-config` / full).

**Done when:** `go test ./internal/living/`, Living HTTP tests, and
`bun run test:unit` living fixtures green on rename alone (groups may still
exist temporarily).

## Phase 3 — Feragame intake flatten

**Goal:** drop intake `lots` group; many `register` + once `weightCheck`.

1. Flatten `lots` group to leaf `register` with `submit: many`,
   `onCommit.navigation: stay` as needed.
2. `weightCheck` readyWhen: ≥1 lot for truck; after weightCheck Submission,
   register readyWhen false.
3. Remove Spawn/Close / `closeRequiresChildSubmission` for intake.
4. Seed + tests for multi-lot truck without group instances.

**Done when:** plant-config checklist green for intake path; no intake
`kind: group`.

## Phase 4 — Feragame line: lot session + peer many-submit

**Goal:** locked line law.

1. Add `startLot` / `closeLot` (`submit: many`) with readyWhen on lot status and
   pallet completeness (close → set status `fullyProcessed`).
2. Flatten `palletWeight`, `lineLoad`, `containerReg` groups to leaf many-submit
   Tasks; scope reads to open lot; Domain weigh-before-progress on
   `markProgress` (`$exists(weight)` in readyWhen/`reads.where` — see issue
   `01-gate-bancale-progress-on-weight`); **no** progress gate on containerReg;
   Tasks remain peers (no Unlock pipeline).
3. `closeTurno` transform: `open` → `parked`; `openTurno` transform: `parked` →
   `open`.
4. Exclusivity: at most one lot in `open`|`parked`.
5. Remove all Feragame `unlockFrom` and `kind: group` from line.
6. containerReg must **not** set `fullyProcessed`; only `closeLot` does.
7. Seed scenarios: multi-lot shift; park across close/open turno; closeLot
   denied until all pallets weighed + progress done.
8. Update fixture_test locked expectations (delete weigh-gates-load asserts).

**Done when:** plant-config verify commands green; line has no groups/unlockFrom.

## Phase 5 — SPA / overview chrome

**Goal:** Feragame UX without depending on group Spawn/Close.

1. Overview: leaf Task rows only for flattened Jobs; keep group UI code paths
   only if engine still serves other configs (or gate behind Worklist group
   presence).
2. Home Available: no provisional group `new` spawn rows for Feragame Tasks;
   many-submit stays one actionable row while readyWhen.
3. Stay + Continue for many-submit register/weigh loops (Commit navigation
   already exists).
4. Copy/i18n for start/close lot, parked (Config titles first).

**Done when:** manual smoke on `/app` intake + line; unit tests for worklist
rendering without groups.

## Phase 6 — Passport + leftovers cleanup

1. Passport eligibility: lot status `fullyProcessed` ∧ not locked.
2. Grep for `load`, `fullyProcessed: true` boolean, `kind: group`, plant
   `unlockFrom`; fix stragglers.
3. Supersede note already on lineload-ready LOCKED; close or comment related
   open issues that encode old gates.

## Phase 7 — Later (explicit ask)

- Remove Task-group engine + SPA chrome entirely if no remaining configs need
  them.
- Remove `unlockFrom` from engine if unused repo-wide.

## Suggested PR slice order

1. Phase 1 (engine only, synthetic tests)
2. Phase 2 (rename + enum, mechanical)
3. Phase 3 (intake)
4. Phase 4 (line) — largest plant PR; keep followers in-commit per plant-config
5. Phase 5 (SPA polish) can pair with 3–4 if small
6. Phase 6 cleanup

## Verify (each plant-touching PR)

Per `docs/agents/plant-config.md`:

- `go test ./internal/living/`
- `go test ./cmd/server/ -run Living`
- `bun run test:unit -- --run src/lib/living` from `app/`
- After YAML shape changes: `task seed:feragame-config` (or full as needed)
