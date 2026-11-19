# Export the legacy Stream catalog (handoff guide)

This is the human half of [ticket 02](../issues/02-export-legacy-stream-catalog.md). You export every legacy Stream from one deployed Attesta environment, with instance and DPP counts per Stream, into this folder. Everything is read-only against the deployed database, and nothing is installed or left running on the server.

## What you get

| File | Content |
| --- | --- |
| `raw-export.json` | Direct output of `export.js`. Contains the same YAML as the Stream files, so treat it as unscrubbed and delete it at the end. |
| `<_id>.yaml` | One per document in `formata_builder_streams`: the raw `stream` field, byte for byte. The `_id` hex is the `workflowKey` that `processes` rows point at. |
| `index.json` | One entry per Stream: the other document fields (`updatedAt`, `createdByUserId`, `updatedByUserId`), name, category slugs, steps with their organization slugs, instance counts by status, DPP count, origin (seed copy or hand-authored), YAML size and sha256. Also lists `processes` groups whose `workflowKey` matches no Stream. |
| `summary.md` | The table the ticket asks for: Stream id, name, steps → organization slugs, instance count by status, DPP count, origin. |
| `scrub-report.txt` | Lines in the YAML that look like credentials, URLs or emails. Opaque values are masked. |

## Ground rules

- `export.js` only calls `listCollections`, `find`, `aggregate` and `countDocuments`. Do not edit it to add writes, and do not run ad hoc `update`, `delete`, `drop` or `insert` commands against this database.
- The Coolify stack in `deployment/docker-compose.coolify.yaml` runs `mongo:7.0` with no auth, no `ports:` mapping, and the app connects with `mongodb://mongodb:27017` and no credentials. That means a read-only Mongo user cannot be enforced there. The protection is the script above plus discipline. Do not publish the Mongo port or enable Mongo Express to get at the data more conveniently.
- If your environment was hardened with Mongo auth, use a user with only the `read` role on the Attesta database (see [Optional: read-only user](#optional-read-only-user)).

## Assumptions to confirm

These come from reading the repo. Steps 1 and 2 confirm each one against the real environment.

- Mongo URI env var is `MONGODB_URI` (default `mongodb://localhost:27017`), and the database name env var is `MONGODB_DATABASE` (default `closer_demo`). Both are read in `server/cmd/server/seed_session.go`, which `main` uses at startup. The Coolify compose file passes `MONGODB_URI` to the `attesta` service but does not set `MONGODB_DATABASE`, so a stock deployment uses `closer_demo`.
- Collections are `formata_builder_streams` (type `FormataBuilderStream` in `server/cmd/server/store.go`: `_id`, `stream`, `updatedAt`, `createdByUserId`, `updatedByUserId`) and `processes` (type `Process` in `server/cmd/server/main.go`: `workflowKey`, `status`, and an optional `dpp` subdocument with `gtin`, `lot`, `serial`, `generatedAt`).
- `status` is `active`, `done` or `terminated`. An instance has an issued DPP when its `dpp` field is present, because `UpdateProcessDPP` is the only writer and the field is `omitempty`.
- You can reach the Docker host that runs the Coolify stack (SSH, or Coolify's server terminal) and run `docker exec` there.

## Prerequisites

- Shell access to the Docker host of the target environment, and permission to run `docker exec` against its Mongo container.
- On your laptop, in this repo checkout at the root: `bun` for the post-processing step. `mise.toml` pins `bun = "1.4.2"`, so `mise install` provides it. `jq` is used for sanity checks only.
- `mongosh` and `mongoexport` are not provided by `mise.toml` or any `task` target. You do not need them locally, because the export runs inside the Mongo container (`mongo:7.0` ships `mongosh`; the repo itself uses `docker compose exec mongodb mongosh` in `scripts/worktree-compose.sh mongo-ping`).

## Step 0: pick the environment

Decide which one matters (ticket checklist item 1) and use one environment name consistently below: `staging` or `production`. The commands use these placeholders, so set them once in your shell:

```bash
export ENV_NAME=staging            # or production
export DOCKER_HOST_SSH=user@host   # the Docker host running that Coolify stack
```

## Step 1: find the Mongo container and confirm the database name

Preview stacks, staging and production can share one Docker host, so identify the right container before using it. Coolify names containers `<service>-<resource-uuid>`; the uuid is the one shown in the Coolify resource URL for that environment.

```bash
ssh "$DOCKER_HOST_SSH" "docker ps --filter ancestor=mongo:7.0 --format '{{.Names}}\t{{.Status}}'"
```

`mongo-express` containers use a different image and will not appear. If several Mongo containers show up, pick the one whose uuid suffix belongs to the target resource, then set:

```bash
export MONGO_CONTAINER=mongodb-xxxxxxxxxxxx   # name from the previous command
```

Confirm which database the app uses and that the collections are there. The first command should print nothing for `MONGODB_DATABASE` on a stock deployment (the default `closer_demo` applies); if it prints a value, use that name below.

```bash
ssh "$DOCKER_HOST_SSH" "docker ps --format '{{.Names}}' | grep -i attesta"
ssh "$DOCKER_HOST_SSH" "docker exec <attesta-container> printenv MONGODB_URI MONGODB_DATABASE"
ssh "$DOCKER_HOST_SSH" "docker exec $MONGO_CONTAINER mongosh --quiet --eval 'db.adminCommand({listDatabases:1,nameOnly:true}).databases.map(d => d.name)'"
```

```bash
export MONGO_DB=closer_demo   # or the MONGODB_DATABASE value found above
```

The script refuses to run if either collection is missing from `$MONGO_DB`, so a wrong name fails loudly instead of exporting nothing.

## Step 2: run the export

Run from the repo root. The script travels over stdin, so nothing is copied onto the server, and the JSON lands in this folder.

```bash
ssh "$DOCKER_HOST_SSH" "docker exec -i $MONGO_CONTAINER mongosh --quiet \"mongodb://localhost:27017/$MONGO_DB\" --eval \"\$(cat)\"" < .scratch/passport-chains/legacy-streams/export.js > .scratch/passport-chains/legacy-streams/raw-export.json
```

If you are already on the Docker host (Coolify server terminal), copy `export.js` there and run the same thing without `ssh`. The redirect must not be on the same command as `$(cat)`, so read the file by name:

```bash
docker exec -i "$MONGO_CONTAINER" mongosh --quiet "mongodb://localhost:27017/$MONGO_DB" --eval "$(cat export.js)" > raw-export.json
```

If quoting causes trouble, the fallback copies the script into the container's `/tmp` (a file, not a database write) and removes it afterwards:

```bash
docker cp export.js "$MONGO_CONTAINER":/tmp/export.js
docker exec "$MONGO_CONTAINER" mongosh --quiet "mongodb://localhost:27017/$MONGO_DB" --file /tmp/export.js > raw-export.json
docker exec "$MONGO_CONTAINER" rm /tmp/export.js
```

Check that the output is a single JSON document and look at the counts:

```bash
jq '.counts' .scratch/passport-chains/legacy-streams/raw-export.json
```

`streamDocuments` must equal `streamsExported`. If `jq` fails, open the file: an error message at the top means the script threw (wrong database name, or no access).

## Step 3: split into files and build the summary

```bash
mise exec -- bun .scratch/passport-chains/legacy-streams/postprocess.ts --env "$ENV_NAME"
```

It writes the `<_id>.yaml` files, `index.json`, `summary.md` and `scrub-report.txt` next to `raw-export.json`, then prints PASS or FAIL for these checks and exits non-zero on any FAIL:

- streams exported equals the stream document count
- YAML files written equals streams exported
- no leftover `<_id>.yaml` from an earlier run (delete stale files if this fails)
- grouped process counts add up to the total number of `processes`
- grouped DPP counts add up to the number of processes with a `dpp` field
- every Stream YAML parses

## Step 4: verify

Expected layout, with N equal to the number of Streams:

```
.scratch/passport-chains/legacy-streams/
  EXPORT-GUIDE.md  export.js  postprocess.ts      (already here)
  raw-export.json  index.json  summary.md  scrub-report.txt
  <_id>.yaml × N
```

Sanity checks:

```bash
cd .scratch/passport-chains/legacy-streams
ls [0-9a-f]*.yaml | wc -l
jq '.counts.streamDocuments, (.streams | length)' index.json
```

All three numbers must match. Then get one count that does not come from the export script and compare it:

```bash
ssh "$DOCKER_HOST_SSH" "docker exec $MONGO_CONTAINER mongosh --quiet \"mongodb://localhost:27017/$MONGO_DB\" --eval 'print(db.formata_builder_streams.countDocuments({}), db.processes.countDocuments({}))'"
```

The two numbers are the Stream count and the total process count; they should equal `streamDocuments` and `processesTotal` (they can differ by a handful only if people created instances between the two runs).

Also check by eye:

- `summary.md` has one row per Stream, and every Stream in `index.json` has a non-empty `steps` list with organization slugs. A Stream with empty steps usually means its YAML does not parse; the post-processing step warns about it.
- The `Origin` column: `seed-copy:<file>` means the YAML is identical to a disk template apart from `workflow.name`, `categorySlug` and `subCategorySlug` (the three fields the seeder rewrites in `server/cmd/server/seed_catalog.go`). `seed-copy-edited` means the name and creator look seeded but the body was changed afterwards. `hand-authored` is everything else.
- Rows under "Process groups whose `workflowKey` matches no exported Stream" are instances whose Stream document no longer exists, or a legacy key such as `workflow`. Instances with no `workflowKey` field at all show as `(missing)`; server code treats them as the on-disk `workflow` key. Write these down in the Answer rather than ignoring them.

## Step 5: scrub secrets

Can the YAML contain credentials? By design it should not. The Stream body is parsed into `RuntimeConfig` (`server/cmd/server/main.go`): workflow steps with JSON schemas, organizations, roles, departments, users (only `id`, `name`, `departmentId`), and a `dpp` block (GTIN, product and owner names). None of those fields carries a secret, and the DPP content is public by design. The catch is that the builder stores arbitrary YAML as free text and the parser does not reject unknown keys, so someone could have pasted a URL, token or note into a description, schema `default`, or an extra key. Metadata can also contain personal data: `createdByUserId` and `updatedByUserId` hold identity user ids, or `platform-admin:<email>` for seeded and admin-created Streams.

1. Open `scrub-report.txt` and review `url-credentials`, `private-key` and `secret-keyword` lines first, then the `url`, `opaque-string` and `email` lines. A hit is `<stream id>  <file>:<line>  <kind>  <match>`.
2. For each real secret, edit that line in `<_id>.yaml` and replace the value with `REDACTED`. Keep the key so the YAML shape survives.
3. Do not re-run `postprocess.ts` after redacting, because it rewrites the YAML files from `raw-export.json`. The `yamlSha256` and `yamlBytes` in `index.json` describe the unredacted YAML, which is fine; mention it in the Answer.
4. If the report only shows seeded demo content (nothing from step 1), say so in the Answer and move on.
5. `.scratch/` is gitignored, so nothing here is committed by accident. If you plan to share the folder, also decide what to do about the `platform-admin:<email>` values in `index.json`.
6. Rotate any real credential you found, because it has been in a Stream definition that other people with database access could read.

Finally delete the unscrubbed copy:

```bash
rm .scratch/passport-chains/legacy-streams/raw-export.json
```

## Step 6: fill in the ticket's Answer

Open [the ticket](../issues/02-export-legacy-stream-catalog.md) and add an `## Answer` section above `## Comments`, filled in from `summary.md` and `index.json`. The ticket says it is resolved when the export is in place and the answer records the environment, the Stream count and the table. Set the ticket's `Status` line to `resolved` yourself once this is done.

````markdown
## Answer

- Environment: staging | production (database `closer_demo`, exported YYYY-MM-DD, Mongo server 7.0.x)
- Streams: N documents in `formata_builder_streams`, N files in `legacy-streams/`
- Instances: P total in `processes` (A active, D done, T terminated), X with an issued DPP
- Origin (checklist item 4): S seed copies of the disk templates (`workflow.yaml` a, `stream.yaml` b, `secondary.yaml` c), E seed copies edited later, H hand-authored. Hand-authored Streams: <ids and names, or "none">
- Process groups with no Stream document: <list from `index.json` `orphanProcessGroups`, or "none">
- Scrub: <"no secrets found" or what was redacted and rotated>. `yamlSha256` in `index.json` is pre-scrub.

<paste the table from summary.md here>
````

Check that the steps → organization slugs column reads correctly for the hand-authored rows, since the conversion tickets build on it.

## Caveats

- Seed-copy detection compares against `server/config/*.yaml` in your checkout. If the deployed image carries older templates, an unedited seed copy is labelled `seed-copy-edited` or `hand-authored`. Check the Stream names: seeded ones look like `<Sub-category> — Pilot|Standard|Extended`.
- Instance counts include `terminated` and `done` instances, as asked. `index.json` keeps the per-status split if you want to exclude them.
- The DPP count is instances with a `dpp` subdocument. `dppEnabled` in `index.json` is the Stream YAML's `dpp.enabled` flag, which says whether the Stream can issue one.

## Optional: read-only user

Only relevant if the environment runs Mongo with auth. A database admin creates the user (this is the one write in the whole procedure, and it is not part of the export), then you add `--username`, `--password` and `--authenticationDatabase` to the `mongosh` call in step 2 and drop the user afterwards.

```javascript
db.getSiblingDB("closer_demo").createUser({ user: "catalog_export", pwd: passwordPrompt(), roles: [{ role: "read", db: "closer_demo" }] })
```

## Optional: take a dump instead

If you would rather not run any query on the live database beyond a copy, dump the two collections, restore them into a local scratch Mongo, and run steps 2 to 6 against that one. This needs `mongodump` inside the container (not verified for the `mongo:7.0` image) or on the host.

```bash
ssh "$DOCKER_HOST_SSH" "docker exec $MONGO_CONTAINER mongodump --db $MONGO_DB --collection formata_builder_streams --archive" > streams.archive
ssh "$DOCKER_HOST_SSH" "docker exec $MONGO_CONTAINER mongodump --db $MONGO_DB --collection processes --archive" > processes.archive
```

## Rehearse locally first (optional)

The same script works against the repo's local stack. `task start` brings up Mongo on `MONGODB_PORT` from `.env.worktree`; Mongo there has no auth and the database is `closer_demo`. With a host `mongosh` (not provided by `mise.toml`; `brew install mongosh`):

```bash
source <(bash scripts/worktree-env.sh export)
mongosh --quiet "mongodb://localhost:${MONGODB_PORT}/closer_demo" --file .scratch/passport-chains/legacy-streams/export.js > /tmp/raw-export.json
mise exec -- bun .scratch/passport-chains/legacy-streams/postprocess.ts --env local --in /tmp/raw-export.json
```

Writing to `/tmp` keeps rehearsal output out of this folder, so it cannot be mistaken for the real export.
