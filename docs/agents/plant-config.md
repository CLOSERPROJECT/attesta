# Plant Org Config

Checklist when changing the Feragame plant fixture
[`server/internal/living/fixtures/feragame-config.yaml`](../../server/internal/living/fixtures/feragame-config.yaml)
(embedded Active Config + SPA `$living-fixtures`).

## Rules

- **Dumb executor:** plant law lives in Config (`domain.entities`, transforms,
  `readyWhen`). Go and `/app` only evaluate; no Feragame- or plant-named
  branches in executors. See [`GLOSSARY.md`](../../GLOSSARY.md).
- **Domain gates:** exclusivity and open/close use Task **`readyWhen`** over
  Domain (and `$domain.submissions` when bind-once matters). Do not encode
  entity open/closed law in **`startsWhen`**.
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
8. **Local stacks** — Active Config + Domain need
   `task seed -- --replace` (human-gated wipe) after fixture shape changes.

## Done when

- Fixture parses and seed builds a coherent Domain.
- Followers in steps 2–6 are updated or explicitly unchanged with reason.
- Verify commands in step 7 are green.
- No new plant-named branches in Go or SPA.
