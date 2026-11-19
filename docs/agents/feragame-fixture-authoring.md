# Feragame fixture authoring

Change checklist for the Feragame plant fixture [`server/internal/org/fixtures/feragame-config.yaml`](../../server/internal/org/fixtures/feragame-config.yaml) (embedded Active Config + SPA `$org-fixtures`). Not the platform Config authoring reference—that is [`org-config-authoring.md`](org-config-authoring.md).

Plant nouns: [`docs/feragame/GLOSSARY.md`](../feragame/GLOSSARY.md). Org platform nouns: [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md). Map: [`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md).

## Steps

1. **Author plant law in the fixture** — entities, Jobs/Tasks, transforms, `readyWhen`, catalogs, Italian labels. Prefer existing transform ops (`mint` / `mintMany` / `patch` / `bind`). Follow [`org-config-authoring.md`](org-config-authoring.md) for gates and `onCommit`. Lot session: at most one lot `open`; close sets `fullyProcessed` only when every pallet on that lot has weight and `progress = done`; DPP issue is `lot_processing`’s final once-only `lock` (mirrors intake’s closing `weightCheck`). Intake Job instance id matches truck id.
2. **Glossary** — if you add or rename a plant Domain concept, update [`docs/feragame/GLOSSARY.md`](../feragame/GLOSSARY.md) in the same change. Platform terms stay in [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md).
3. **Seed** — update [`seed.go`](../../server/internal/org/seed.go) and [`demo_commits.go`](../../server/internal/org/demo_commits.go) Commit sequence and id constants (truck / lot grammars; no day/turno Commits).
4. **Org package tests** — fix callers of the old sequence; add focused Commit / Worklist cases for new `readyWhen` laws (intake incompleteness and one-open-lot exclusivity).
5. **Org HTTP tests** — update [`server/cmd/server/org_test.go`](../../server/cmd/server/org_test.go) (and any other `cmd/server` org API paths) for the new Commit payloads and job counts (`intake` / `lot_processing`).
6. **SPA fixture importers** — update `app/src/lib/org` tests that parse `$org-fixtures` or hardcode plant task ids / schemas.
7. **Verify** — `go test ./internal/org/`, `go test ./cmd/server/ -run Living`, and `bun run test:unit -- --run src/lib/org` from `app/`.
8. **Local stacks** — after fixture YAML shape changes, re-seed Feragame scopes as in [`QUICKSTART.md`](../../QUICKSTART.md) (`task seed:feragame-config`, `seed:feragame-domain`, `seed:feragame-identity`, `seed:feragame-full`; `--keep-domain` where noted).

Every follower above lands in the same commit/PR as the YAML.

### Feragame `onCommit` examples

HTTP link leave and multi-pass leaves (`register`, `weigh`, `markProgress`, `containerReg`, `closeLot`) typically stay:

```yaml
onCommit:
  navigation: stay
```

Omit both keys on Job-start and closing once-only Tasks such as `weightCheck` / `lot_processing/lock` (default `advance`). Schema and wire semantics: [`org-config-authoring.md`](org-config-authoring.md).

## Done when

- Fixture parses and seed builds a coherent Domain.
- Followers in steps 2–6 are updated or explicitly unchanged with reason.
- Verify commands in step 7 are green.
- No new plant-named branches in Go or SPA.
