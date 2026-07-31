# Dev catalog streams seed

Date: 2026-07-31  
Scope: re-runnable Mongo CLI that full-replaces Formata stream definitions so every taxonomy leaf has 1–3 categorized catalog streams (random per leaf), for local/dev public-home browsing.  
Out of scope: Writing `server/config/*.yaml` on disk; Appwrite/user/org seeding; taxonomy seeding (use `seed-categories`); process/instance seeding (use `seed-instances`); auto-run on `task start` / `task dev`; changing catalog merge rules.

## Goal

Developers can run one command and get a Formata catalog where **every** Category → Sub-category leaf has between **one and three** streams with distinct names and valid `(categorySlug, subCategorySlug)` pairs—enough for the public-home category sidebar to show cards on every leaf—without hand-editing Formata docs.

## Decisions

1. **Approach:** Real Mongo seed CLI in the same family as `seed-categories` / `seed-instances` (`seed-catalog-streams`), not a scratch mongosh/Python script.
2. **Replace:** Full wipe of `formata_builder_streams`, then insert the new set (idempotent demo catalog).
3. **Counts:** Per leaf, choose `n` uniformly from `{1, 2, 3}` each run.
4. **Templates:** Same catalog identity rule as the running app. Snapshot keys via `workflowCatalog()` for cleanup. Template YAML bodies: Formata `Stream` strings when any Formata streams exist; otherwise raw workflow YAML files from the config dir. Do **not** round-trip `RuntimeConfig` through YAML marshal (lossy / wrong Formata document shape).
5. **Process cleanup:** Before wipe, for each current catalog key call `DeleteWorkflowData(key)` so old ObjectID-keyed instances do not linger. Do **not** create new processes—that remains `seed-instances`.
6. **Taxonomy prerequisite:** Leaves come from the live taxonomy store. Empty taxonomy → error (run `seed-categories` / empty-store bootstrap first).
7. **Naming:** `{SubCategory display name} — {Pilot|Standard|Extended}` for slots 1..n.

## Run order

1. `task seed:categories` (if taxonomy empty / needs full replace)  
2. **`task seed:catalog-streams`**  
3. `task seed:instances`

## CLI & DX

| Piece | Behavior |
|-------|----------|
| Binary entry | `go run ./cmd/server seed-catalog-streams` (args after the subcommand reserved; none required for v1) |
| Task | `task seed:catalog-streams` → runs the command from `server/` |
| Mongo | `MONGODB_URI` (default `mongodb://localhost:27017`), `MONGODB_DATABASE` (default `closer_demo`) — same as other seeds |
| Config dir | Same env as server (`WORKFLOW_CONFIG` / `WORKFLOW_CONFIG_DIR`) so file-catalog templates match the running app |
| Side effects | Rewrites Formata stream definitions; deletes process/attachment data for prior catalog keys via `DeleteWorkflowData`. Does **not** modify taxonomy, Appwrite, or on-disk config YAML |

## Algorithm

1. Open Mongo store (same opener pattern as `seed-categories`).
2. Construct a `Server` with that store + config dir.
3. Load taxonomy leaves (category slug + sub-category slug + display name). If none → error.
4. Load template YAML bodies:
   - If `ListFormataBuilderStreams` is non-empty → use each document’s `Stream` field.
   - Else → read config-dir `*.yaml` / `*.yml` files that pass `isWorkflowCatalogConfigFile` (skip `categories.yaml`).
   - If still empty → error.
5. Load `workflowCatalog()` keys (may be empty on a fresh DB). For each key, `DeleteWorkflowData(key)`.
6. Delete all Formata builder streams (list + `DeleteFormataBuilderStream`, or equivalent wipe).
7. For each leaf:
   - `n := random in {1,2,3}`
   - For `i in 0..n-1`: pick a random template body; strip all `categorySlug` / `subCategorySlug` lines (including a final line without trailing newline); insert one pair under `workflow:`; set workflow `name` to `{SubName} — {Pilot|Standard|Extended}[i]`; `yaml.Unmarshal` to validate; `SaveFormataBuilderStream` with a new ObjectID.
8. Log leaf count, streams inserted, and reminder to run `seed-instances`.

## YAML surgery requirements

- Must not leave duplicate `categorySlug` / `subCategorySlug` keys (regression from the ad-hoc one-shot that missed a trailing line without `\n`).
- Must not confuse role-level `name:` fields with workflow `name:` (Formata docs often list roles before `workflow:`).
- Inserted documents must parse with the same YAML loader the catalog uses.

## Code shape

| File | Role |
|------|------|
| `catalog_streams_seed.go` | Pure helpers: target counts, strip/set category pair, rename workflow, validate YAML |
| `catalog_streams_seed_cmd.go` | CLI: open Mongo, load taxonomy/templates/catalog keys, delete workflow data, wipe Formata, insert clones, logging |
| `catalog_streams_seed*_test.go` | `MemoryStore` unit tests |
| `main.go` | Dispatch `seed-catalog-streams` next to other seed subcommands |
| `Taskfile.yml` | `seed:catalog-streams` task |

## Error handling

- Mongo connect/ping failure → non-zero exit.
- Empty taxonomy or empty templates → error.
- Empty catalog keys at step 5 → skip deletes (not an error).
- Failure on delete, wipe, YAML build/parse, or insert → abort; log the failing leaf or key. No silent partial success.
- Successful run logs how many leaves and streams were seeded.

## Testing

**Unit (`MemoryStore`):**
- Every leaf ends with 1–3 streams; names match `{SubName} — {variant}`.
- Each stream’s category pair matches its leaf and appears exactly once in the YAML.
- Second seed replaces the Formata set (no leftover IDs from the first run).
- Processes under old catalog keys are removed via `DeleteWorkflowData`.
- Trailing duplicate `subCategorySlug` without final newline is stripped.
- When Formata is empty, templates load from config-dir workflow YAML files.

**Manual:**
1. Stack up (`task start` / `task dev`).
2. `task seed:categories` (if needed) → `task seed:catalog-streams` → `task seed:instances`.
3. Public home: every subcategory shows at least one card (UI still caps at 6).
4. Server logs show no `yaml: unmarshal` / duplicate mapping-key errors for catalog streams.

## Catalog caveat

`workflowCatalog()` returns **only Formata streams when any exist**; otherwise YAML under the config directory. After this seed, the live catalog is always Formata-backed. File YAML on disk is template input only when Mongo Formata is empty at the start of the run; this command does not update those files.
