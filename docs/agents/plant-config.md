# Plant Org Config

Checklist when changing the Feragame plant fixture
[`server/internal/living/fixtures/feragame-config.yaml`](../../server/internal/living/fixtures/feragame-config.yaml)
(embedded Active Config + SPA `$living-fixtures`).

## Rules

- **Dumb executor:** plant law lives in Config (`domain.entities`, transforms,
  `readyWhen`, **`onCommit`** navigation). Go and `/app` only evaluate;
  no Feragame- or plant-named branches in executors. See
  [`GLOSSARY.md`](../../GLOSSARY.md).
- **Domain gates:** exclusivity and open/close use Task **`readyWhen`** over
  Domain (and `$domain.submissions` when bind-once matters). Do not encode
  entity open/closed law in **`startsWhen`**. Do not author plant **`unlockFrom`**
  when Domain status already expresses the prerequisite (engine may still
  support Unlock).
- **Submit cardinality / lot session:** multi-pass plant work uses leaf Tasks
  with `submit: many` (not repeatable Task groups). Lot entity + session status
  `open | parked | fullyProcessed` are locked in
  [`.scratch/living-task-submit-cardinality/LOCKED.md`](../../.scratch/living-task-submit-cardinality/LOCKED.md)
  and [`docs/adr/0008-living-task-submit-cardinality.md`](../adr/0008-living-task-submit-cardinality.md);
  implementation order in that folder’s `IMPLEMENTATION.md`.
- **Same change:** every follower below lands in the same commit/PR as the YAML.

## Steps

1. **Author plant law in the fixture** — entities, Jobs/Tasks, transforms,
   `readyWhen`, catalogs. Prefer existing transform ops (`mint` / `mintMany` /
   `patch` / `bind`); keep Italian plant labels as Config content.
2. **Glossary** — if you add or rename a plant Domain concept (e.g. working day,
   shift), update [`GLOSSARY.md`](../../GLOSSARY.md) in the same change.
3. **Seed** — update [`seed.go`](../../server/internal/living/seed.go) and
   [`demo_commits.go`](../../server/internal/living/demo_commits.go) Commit
   sequence and id constants (truck / shift / day grammars).
4. **Living package tests** — fix callers of the old sequence; add focused
   Commit / Worklist cases for new `readyWhen` laws (see
   `working_day_shift_test.go` as the exclusivity pattern).
5. **Living HTTP tests** — update
   [`server/cmd/server/living_test.go`](../../server/cmd/server/living_test.go)
   (and any other `cmd/server` Living paths) for the new Commit payloads and
   job counts.
6. **SPA fixture importers** — update `app/src/lib/living` tests that parse
   `$living-fixtures` or hardcode plant task ids / schemas.
7. **Verify** — `go test ./internal/living/`, `go test ./cmd/server/ -run Living`,
   and `bun run test:unit -- --run src/lib/living` from `app/`.
8. **Local stacks** — after fixture YAML shape changes, run
   `task seed:feragame-config` (Active Config + empty Domain; no Appwrite).
   Staged demo Domain: `task seed:feragame-domain`. With Feragame Appwrite
   operators: `task seed:feragame-identity` (empty Domain) or
   `task seed:feragame-full` (staged). Full-stack rewrite remains
   `task seed -- --replace` (orgs 1–3 + Formata + Feragame).

## Commit navigation (`onCommit`)

Per Task, optional `onCommit` keys control **server-side** post-Commit routing
(ADR-0006; locked law in
[`.scratch/commit-navigation/LOCKED.md`](../../.scratch/commit-navigation/LOCKED.md)).

| Key | Values | Default |
|-----|--------|---------|
| `navigation` | `home` \| `overview` \| `advance` \| `stay` | `advance` |
| `advanceTarget` | bool | `true` |

- **`navigation`:** `advance` keeps role-eligible auto-advance on the Job
  instance; `home` / `overview` force that destination; `stay` parks on the
  submitted Task (links, Spawn/Close chrome per group law) and Commit ships
  deferred **Continue** via `continueNavigation` on the wire.
- **`advanceTarget: false`:** never auto-advance to the next Task after this
  Commit. When nothing else is authored, land on **Job instance overview**.

### Feragame examples (fixture)

**Open → home** (avoid landing on close sibling):

```yaml
# workingDay / open
onCommit:
  navigation: home

# line / openTurno
onCommit:
  navigation: home
```

**Close → no auto-advance:**

```yaml
# workingDay / close, line / closeTurno
onCommit:
  advanceTarget: false
```

**HTTP link leave → stay** (intake `register` and any Task with `onSubmit.http`
`action: link`):

```yaml
onCommit:
  navigation: stay
```

Omit both keys on other Tasks. Go maps these to Commit `navigation` on the wire;
the SPA must not infer stay from links alone.

## Done when

- Fixture parses and seed builds a coherent Domain.
- Followers in steps 2–6 are updated or explicitly unchanged with reason.
- Verify commands in step 7 are green.
- No new plant-named branches in Go or SPA.
