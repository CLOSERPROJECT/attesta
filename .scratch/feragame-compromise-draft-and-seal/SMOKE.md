# Smoke checklist — draft-and-seal A/B

App (this worktree): `http://localhost:13622` (`PORT` in `.env.worktree`). `task dev` should already be running.

## Pilot Stream seed

| | |
| --- | --- |
| File | `server/config/feragame-pilot.yaml` |
| Display name | Feragame production day (pilot) |
| Pack status | Excluded from `openCatalogTemplatePack` via `isCatalogTemplateFile` |
| Seed path | `task seed` / `attesta seed` (or prepare when `ATTESTA_SEED`); **not** `seed:catalog-streams` alone |
| Stable URL key | `ef8d2f15c968fb8d879d56aa` (`stableObjectID("named-pilot-stream:feragame-pilot")`) |
| Org | `feragame` (demo fixture) |
| Users | `admin@feragame.com` / `admin@feragame.com` (chemist + Org admin); platform admin also org-admin on `feragame` |
| Role on Substeps | `chemist` |
| Strategy | `completionStrategy: draft-and-seal` |

**Start path (after seed):** `/my/streams/ef8d2f15c968fb8d879d56aa/`

Named pilot is upserted after catalog clones + demo instances, with a fixed ObjectID and no seed instances (empty for smoke). Find it in the UI under **Feragame production day (pilot)** or by the URL key above.

### Reload / restart

- Prefer `task seed -- --replace` (or empty-stack prepare) so identity + named pilot stay aligned.
- `task seed:catalog-streams` only multiplies pack templates — it does **not** load `feragame-pilot.yaml` (pack deny-list) and wipes FormataBuilderStreams first; run full `task seed -- --replace` afterward to restore the named pilot.
- Catalog cache reloads from Mongo when stream ids/modtimes change.

## Automated

```bash
cd server/cmd/server && go test -count=1 -timeout 120s . \
  -run 'TestCompleteSubstep|TestEnsureCompletion|TestProcessCompletion|TestComputeAvailability|TestSequenceAndDone|TestNextAvailable|TestNormalizeCompletion|TestWorkflowCompletion|TestClassicStrategy|TestDraftAndSeal|TestClassicSaveDraft|TestParseRuntimeConfigCompletion|TestWorkflowDefCompletion|TestBuildSubstepViewsDraftAndSeal|TestBuildStreamInstanceDetailViewDraftAndSeal|TestBuildStreamInstanceDetailViewClassic|TestHandleCompleteSubstepDraftAndSeal|TestHandleSealStep|TestHandleProcessRoutesDispatchesSeal|TestIsCatalogTemplateFile|TestDefaultDemoFixtureIncludesFeragame|TestSeedNamedPilotStreams'
```

## Browser (pilot Stream)

**Known bug (fixed in `web/src/main.js`):** Formata submit used FormData flat keys (`batches[][loadId]`), so Mongo stored drafts that Formata could not reload. Prefer `getValue()` nested JSON. Hard refresh after Vite HMR if needed.

1. Log in as `admin@feragame.com` / `admin@feragame.com` (chemist + Org admin for seal).
2. Open **Feragame production day (pilot)** → start a Stream instance (`/my/streams/ef8d2f15c968fb8d879d56aa/`).
3. Confirm both kind Substeps show **drafting** / available (not Locked by sequence).
4. Save draft rows on Batch loads and UdC; reload — still editable; no Complete.
5. Confirm a non–org-admin cannot seal (if you have such a user).
6. As Org admin: **Seal Step** → confirm → evidence frozen, instance done.
7. Control: open a classic Stream instance — still sequential Complete.

## Pointers

- Spec/tickets: `.scratch/feragame-compromise-draft-and-seal/`
- Test blueprint shape: `feragameDraftAndSealDef` in `process_completion_draft_and_seal_test.go`
