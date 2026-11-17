# Plant Org Config

Checklist when changing the Feragame plant fixture
[`server/internal/org/fixtures/feragame-config.yaml`](../../server/internal/org/fixtures/feragame-config.yaml)
(embedded Active Config + SPA `$org-fixtures`).

Plant nouns (lot session, Jobs, roles):
[`docs/feragame/GLOSSARY.md`](../feragame/GLOSSARY.md). Platform nouns:
[`GLOSSARY.md`](../../GLOSSARY.md). Map:
[`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md).

## Rules

- **Dumb executor:** plant law lives in Config (`domain.entities`, transforms,
  `readyWhen`, **`onCommit`** navigation). Go and `/app` only evaluate;
  no Feragame- or plant-named branches in executors. See
  [`GLOSSARY.md`](../../GLOSSARY.md) (**Dumb executor**).
- **Domain gates:** exclusivity and open/close use Task **`readyWhen`** over
  Domain (and `$domain.submissions` when bind-once matters). Do not encode
  entity open/closed law in **`startsWhen`**. Do not author plant **`unlockFrom`**
  when Domain status already expresses the prerequisite (engine may still
  support Unlock).
- **Submit cardinality / lot session:** multi-pass plant work uses leaf Tasks
  with `submit: many` (not repeatable Task groups). Lot entity + session status
  `open | fullyProcessed` only (no `parked`; no turn bookends; no working day /
  shift Domain). Platform law:
  [`docs/adr/0008-org-task-submit-cardinality.md`](../adr/0008-org-task-submit-cardinality.md).
  Lot-session tables and plant entity renames:
  [`docs/feragame/GLOSSARY.md`](../feragame/GLOSSARY.md)—not Attesta ADRs.
- **JSONata exprs:** chrome / transform / QR / HttpPort bodies use jsonata-go —
  see [`jsonata-org.md`](jsonata-org.md) before blaming the executor.
- **`/app` i18n:** SPA chrome vs Config copy vs English platform errors —
  [`docs/adr/0013-org-app-locale-tracks.md`](../adr/0013-org-app-locale-tracks.md)
  and [`app-i18n.md`](app-i18n.md).
- **Same change:** every follower below lands in the same commit/PR as the YAML.

## Steps

1. **Author plant law in the fixture** — entities, Jobs/Tasks, transforms,
   `readyWhen`, catalogs. Prefer existing transform ops (`mint` / `mintMany` /
   `patch` / `bind`); keep Italian plant labels as Config content.
2. **Glossary** — if you add or rename a plant Domain concept, update
   [`docs/feragame/GLOSSARY.md`](../feragame/GLOSSARY.md) in the same change.
   Platform terms stay in root [`GLOSSARY.md`](../../GLOSSARY.md). (Working day
   / shift are not Domain entities on this plant path.)
3. **Seed** — update [`seed.go`](../../server/internal/org/seed.go) and
   [`demo_commits.go`](../../server/internal/org/demo_commits.go) Commit
   sequence and id constants (truck / lot grammars; no day/turno Commits).
4. **Org package tests** — fix callers of the old sequence; add focused
   Commit / Worklist cases for new `readyWhen` laws (intake incompleteness and
   one-open-lot exclusivity).
5. **Org HTTP tests** — update
   [`server/cmd/server/org_test.go`](../../server/cmd/server/org_test.go)
   (and any other `cmd/server` org API paths) for the new Commit payloads and
   job counts (`intake` / `lot_processing` / `passport`).
6. **SPA fixture importers** — update `app/src/lib/org` tests that parse
   `$org-fixtures` or hardcode plant task ids / schemas.
7. **Verify** — `go test ./internal/org/`, `go test ./cmd/server/ -run Living`,
   and `bun run test:unit -- --run src/lib/org` from `app/`.
8. **Local stacks** — after fixture YAML shape changes, run
   `task seed:feragame-config` (Active Config + empty Domain; no Appwrite).
   Soft Config refresh without Domain wipe:
   `task seed:feragame-config -- --keep-domain`. Staged demo Domain:
   `task seed:feragame-domain`. With Feragame Appwrite operators:
   `task seed:feragame-identity` (empty Domain) or `task seed:feragame-full`
   (staged); identity also accepts `--keep-domain`. Full-stack rewrite remains
   `task seed -- --replace` (orgs 1–3 + Formata + Feragame).

## Commit navigation (`onCommit`)

Per Task, optional `onCommit` keys control **server-side** post-Commit routing
([ADR-0006](../adr/0006-org-operator-worklist-home.md); glossary
**Commit navigation**).

| Key | Values | Default |
|-----|--------|---------|
| `navigation` | `home` \| `overview` \| `advance` \| `stay` | `advance` |
| `advanceTarget` | bool | `true` |

- **`navigation`:** `advance` keeps role-eligible auto-advance on the Job
  instance; `home` / `overview` force that destination (`overview` maps to
  **home** on the wire); `stay` parks on the submitted Task (download links;
  many-submit loops typically stay — ADR-0008) and Commit ships deferred
  **Continue** via `continueNavigation` on the wire.
- **`advanceTarget: false`:** never auto-advance to the next Task after this
  Commit. When nothing else is authored, land on **Worklist** home.

### Feragame examples (fixture)

**HTTP link leave → stay** (intake `register` and any Task with `onSubmit.http`
`action: link`):

```yaml
onCommit:
  navigation: stay
```

**Multi-pass leaf → stay** (lot_processing `weigh` / `markProgress` /
`containerReg` / `closeLot`):

```yaml
onCommit:
  navigation: stay
```

Omit both keys on other Tasks (e.g. Job-start `startLot`, passport `select` /
`lock` — default `advance`). Go maps these to Commit `navigation` on the wire;
the SPA must not infer stay from links alone.

## Done when

- Fixture parses and seed builds a coherent Domain.
- Followers in steps 2–6 are updated or explicitly unchanged with reason.
- Verify commands in step 7 are green.
- No new plant-named branches in Go or SPA.