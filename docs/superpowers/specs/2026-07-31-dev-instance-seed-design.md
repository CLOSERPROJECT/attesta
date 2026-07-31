# Dev stream instance seed

Date: 2026-07-31  
Scope: re-runnable Mongo CLI that seeds realistic process instances for every stream in the runtime catalog, so public and authenticated UIs look populated in local/dev.  
Out of scope: HTTP mocks / fixture store wrappers; Appwrite/user/org seeding; taxonomy seeding (already `seed-categories`); auto-run on `task start` / `task dev`; changing catalog merge rules.

## Goal

Developers can run one command and see each catalog stream with multiple instances in various statuses—enough for public home metrics, `/streams/:key` recent completed runs, and `/my/streams/:key/` status groups—without manually completing workflows.

## Decisions

1. **Approach:** Real Mongo seed CLI (same family as `seed-categories`), not in-process mocks or hand-authored per-stream process YAML.
2. **Surfaces:** One data set feeds both public pages and authenticated dashboards.
3. **Fidelity:** Realistic runs—partial/full progress, termination metadata, DPP on some completed instances when the stream enables it, and real GridFS blobs for `inputType: file` substeps.
4. **Replace:** For each catalog key, `DeleteWorkflowData(key)` then insert the fixture set (idempotent demo state).
5. **Catalog:** Use the same `workflowCatalog()` rules the server uses (Formata streams if any exist, otherwise YAML under the config dir). Do not invent a second merge of YAML + Formata.
6. **v1 counts:** Fixed mix per stream (8 instances); algorithmic progress from each workflow’s ordered substeps.

## CLI & DX

| Piece | Behavior |
|-------|----------|
| Binary entry | `go run ./cmd/server seed-instances` (args after the subcommand reserved; none required for v1) |
| Task | `task seed:instances` → runs the command from `server/` |
| Mongo | `MONGODB_URI` (default `mongodb://localhost:27017`), `MONGODB_DATABASE` (default `closer_demo`) — same as categories seed |
| Config dir | Same env as server (`WORKFLOW_CONFIG` / `WORKFLOW_CONFIG_DIR`) so YAML catalog matches the running app |
| Side effects | Does **not** modify taxonomy, Appwrite, or Formata stream definitions—only processes and related attachments cleaned/created via existing store APIs |

## Per-stream instance mix

| Kind | Count | Status | Progress | Extra |
|------|-------|--------|----------|--------|
| Fresh active | 1 | `active` | empty | Name like `{Stream} — just started` |
| Mid active | 2 | `active` | first ~40% of ordered substeps `done` | Staggered `CreatedAt` |
| Completed | 3 | `done` | all substeps `done` | Prefer DPP on 2 when `dpp.enabled`; 1 without DPP when possible; if DPP required for a fixture slot and build fails, abort that stream |
| Terminated | 2 | `terminated` | partial progress + `Termination{Reason, EndedAt, Actor}` | Distinct reasons |

Totals: **8 instances per catalog stream**.

Names and timestamps are offset so lists are distinguishable.

## Progress & payload generation

1. Walk `orderedSubsteps(def)` for the stream’s `RuntimeConfig.Workflow`.
2. Mark a prefix of substeps `done` with `DoneAt` and a synthetic `DoneBy` actor.
3. Persist progress using the same key encoding as production (`encodeProgressKey` / store insert path).
4. Synthesize `Data` by `inputType`:
   - scalars (`string`, `number`, etc.): simple placeholder values
   - object / `formata`: small plausible object (enough for timeline/result UI)
   - **`file`:** upload a tiny real GridFS attachment via `SaveAttachment`, then store the real `attachmentId` / filename / sha metadata in progress `Data` in the same shape completion uses. Fake ids are not allowed—downloads would 404.
5. For `done` instances on DPP-enabled streams: after insert (process ID known), set `ProcessDPP` using the same helpers/fields as completion (`buildProcessDPP` / gtin, lot from progress or `lotDefault`, serial from strategy).
6. `DeleteWorkflowData` already removes attachments for wiped processes; re-seed stays consistent.

`isProcessDone` only checks substep `state == done`; seeding still attaches real files so instance detail and download panels work when file steps exist.

## Code shape

| File | Role |
|------|------|
| `instances_seed.go` | Pure helpers: build fixture set for one config, synthesize progress/Data, DPP, file blobs |
| `instances_seed_cmd.go` | CLI: open Mongo store, load catalog via `workflowCatalog()`, per-key delete + insert, logging |
| `instances_seed*_test.go` | `MemoryStore` unit tests |
| `main.go` | Dispatch `seed-instances` next to `seed-categories` |
| `Taskfile.yml` | `seed:instances` task |

## Error handling

- Mongo connect/ping failure → non-zero exit.
- Empty catalog → error.
- Per-stream failure on delete, insert, DPP (when required for a slot), or file upload → abort; log the failing workflow key. No silent “partial catalog” success.
- Successful run logs how many streams and instances were seeded.

## Testing

**Unit (`MemoryStore`):**
- Fixture mix counts and statuses per workflow.
- Replace: second seed for a key does not leave prior processes.
- DPP attached only when enabled / as specified by mix rules.
- File substeps produce `SaveAttachment` + real id in progress data.

**Manual:**
1. Stack up (`task start` / `task dev`).
2. `task seed:instances`.
3. Public home: run counts / active-now chips populated.
4. `/streams/:key`: recent completed runs (DPP links when present).
5. `/my/streams/:key/`: active / done / terminated groups non-empty.
6. Open a done instance that includes a file substep; download succeeds.

## Catalog caveat

`workflowCatalog()` returns **only Formata streams when any exist**; otherwise YAML files under the config directory. The seed must follow that rule so it populates what `/` and `/my` actually show. Fixing YAML+Formata merge is out of scope.
