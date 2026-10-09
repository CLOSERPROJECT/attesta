# Smoke checklist — draft-and-seal A/B

App (this worktree): `http://localhost:13622` (`PORT` in `.env.worktree`). `task dev` should already be running.

## Pilot Stream seed

| | |
| --- | --- |
| File | `server/config/feragame-pilot.yaml` |
| Display name | Feragame production day (pilot) |
| File-backed workflow key | `feragame-pilot` |
| Live Mongo Stream URL key | `6ac90e24ebf16384b69efab0` |
| Org | `organization-1` (Appwrite local seed; `org1@admin.org` has chemist + Org admin) |
| Role on Substeps | `chemist` |
| Strategy | `completionStrategy: draft-and-seal` |

**Start path (this worktree, Mongo catalog):** `/my/streams/6ac90e24ebf16384b69efab0/`

File-backed key `feragame-pilot` applies only when FormataBuilderStreams is empty. This worktree already had 46 catalog docs; the pilot YAML was upserted as ObjectID `6ac90e24ebf16384b69efab0` (collection now 47).

### Reload / restart

- Air watches `server/**/*.yaml` and rebuilds on save; a plain process restart alone does **not** add a new YAML if Mongo FormataBuilderStreams is non-empty (`bootstrapFormataBuilderStreams` only seeds when that collection is empty).
- `task seed:catalog-streams` replaces the whole catalog — do not use it for a one-off pilot load.
- Non-destructive load used here: insert/update only the feragame-pilot document in `closer_demo.formata_builder_streams` on `MONGODB_PORT` (no `task reset` / volume purge). Catalog cache reloads from Mongo on the next request when stream ids/modtimes change — no app restart required.

## Automated

```bash
cd server/cmd/server && go test -count=1 -timeout 120s . \
  -run 'TestCompleteSubstep|TestEnsureCompletion|TestProcessCompletion|TestComputeAvailability|TestSequenceAndDone|TestNextAvailable|TestNormalizeCompletion|TestWorkflowCompletion|TestClassicStrategy|TestDraftAndSeal|TestClassicSaveDraft|TestParseRuntimeConfigCompletion|TestWorkflowDefCompletion|TestBuildSubstepViewsDraftAndSeal|TestBuildStreamInstanceDetailViewDraftAndSeal|TestBuildStreamInstanceDetailViewClassic|TestHandleCompleteSubstepDraftAndSeal|TestHandleSealStep|TestHandleProcessRoutesDispatchesSeal'
```

## Browser (pilot Stream)

**Known bug (fixed in `web/src/main.js`):** Formata submit used FormData flat keys (`batches[][loadId]`), so Mongo stored drafts that Formata could not reload. Prefer `getValue()` nested JSON. Hard refresh after Vite HMR if needed.

1. Log in as a user on `organization-1` who can start the Stream and has Org admin for seal (e.g. `org1@admin.org`).
2. Open **Feragame production day (pilot)** → start a Stream instance (`/my/streams/6ac90e24ebf16384b69efab0/`).
3. Confirm both kind Substeps show **drafting** / available (not Locked by sequence).
4. Save draft rows on Batch loads and UdC; reload — still editable; no Complete.
5. Confirm a non–org-admin cannot seal (if you have such a user).
6. As Org admin: **Seal Step** → confirm → evidence frozen, instance done.
7. Control: open a classic Stream instance — still sequential Complete.

## Pointers

- Spec/tickets: `.scratch/feragame-compromise-draft-and-seal/`
- Test blueprint shape: `feragameDraftAndSealDef` in `process_completion_draft_and_seal_test.go`
