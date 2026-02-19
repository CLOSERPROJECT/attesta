# Closer demo (Gallium Recycling Notarization)

Small end-to-end demo with Go + HTMX + SSE + MongoDB + Cerbos, with optional Appwrite team membership checks.

## Requirements
- Go 1.25+
- Node.js 18+
- Docker + Docker Compose

## Quick start
```bash
# 1) Infra
docker compose -f deployment/docker-compose.local.yaml up -d

# 2) Build frontend assets
cd web
npm install
npm run build

# 3) Run Go server
cd ../server
go mod tidy
go run ./cmd/server
```

Open:
- Home: http://localhost:3000
- Backoffice: http://localhost:3000/backoffice
- Mongo Express (optional): http://localhost:8081
- Mailpit (optional): http://localhost:18025

To expose Appwrite Console on host (optional):
```bash
task start:appwrite-ui
# then open http://localhost:19080/console
```

## What it does
- Seeds a workflow definition on first run.
- Starts a process instance with sequential substeps.
- Backoffice impersonates roles (dep1, dep2, dep3) and completes actions.
- Cerbos enforces role + sequence gating.
- Cerbos can also enforce Appwrite team membership per substep (`appwriteTeamIds`).
- Mongo stores process progress + notarizations.
- SSE broadcasts realtime updates to timelines.

## Curl examples
Start a process in a selected workflow (`workflow`):
```bash
curl -X POST http://localhost:3000/w/workflow/process/start -i
```

Impersonate dep1:
```bash
curl -X POST http://localhost:3000/w/workflow/impersonate \
  -d 'userId=u1' -d 'role=dep1' -i
```

Complete substep 1.1 (dep1):
```bash
curl -X POST http://localhost:3000/w/workflow/process/PROCESS_ID/substep/1.1/complete \
  -H 'Cookie: demo_user=u1|dep1|workflow' \
  -d 'value=10'
```

Attempt out-of-sequence (should fail):
```bash
curl -X POST http://localhost:3000/w/workflow/process/PROCESS_ID/substep/2.1/complete \
  -H 'Cookie: demo_user=u2|dep2|workflow' \
  -d 'value=5'
```

## Notes
- Cerbos PDP is expected at `http://localhost:3592`.
- MongoDB is expected at `mongodb://localhost:27017`.
- Appwrite team resolution is enabled when `APPWRITE_ENDPOINT`, `APPWRITE_PROJECT_ID`, and `APPWRITE_API_KEY` are configured.
- Timeline updates pull `/w/:workflow/process/:id/timeline` when SSE events arrive.
- Existing processes without `workflowKey` remain visible under the default `workflow` key and are backfilled on first update.

## Appwrite teams in policy
You can require Appwrite team membership for specific substeps in workflow YAML:

```yaml
substeps:
  - id: "2.1"
    role: "dep2"
    appwriteTeamIds: ["team-refinery", "team-qa"]
```

`appwriteTeamIds` is optional. If omitted (or empty), only the role + sequence checks apply.
When Appwrite integration is not configured (or when lookup fails and `APPWRITE_TEAMS_STRICT=false`), team checks are bypassed.

The default `server/config/workflow.yaml` already includes concrete team IDs:
- `team-intake` for intake substeps
- `team-refinery` for refinement substeps
- `team-qa` for QA/notarization substeps

Debug current actor team resolution:
```bash
curl http://localhost:3000/w/workflow/debug/teams
# or legacy route:
curl "http://localhost:3000/debug/teams?workflow=workflow"
```

## Provision Appwrite From Workflow
Workflow files can be used as source of truth for Appwrite provisioning.

Required in YAML for user provisioning:
- `users[].email` (Appwrite user creation needs an email)
- optional `departments[].appwriteTeamId` and `users[].appwriteTeamIds`

Enable sync on startup:
```bash
export APPWRITE_SYNC_FROM_WORKFLOW=true
export APPWRITE_SYNC_DEFAULT_PASSWORD='TempPassw0rd!'
export APPWRITE_SYNC_MEMBERSHIP_URL='http://localhost:3030'
```

For Docker Compose, set envs and recreate `attesta`:
```bash
docker compose -f deployment/docker-compose.local.yaml up -d --force-recreate attesta
```

## DPP Digital Link configuration
Configure GS1 Digital Link generation per workflow YAML (`server/config/*.yaml`):

```yaml
dpp:
  enabled: true
  gtin: "09506000134352"
  lotInputKey: "batchId"
  lotDefault: "defaultProduct"
  serialInputKey: "serialCode"
  serialStrategy: "process_id_hex"
```

- Generated links follow `/01/{GTIN}/10/{LOT}/21/{SERIAL}` and resolve to a public DPP page.
- DPP identifiers are generated only when a process first reaches `done`.
- Default rollout behavior is minimal: already-completed processes are not automatically backfilled.
- Recommended backfill approach: add a small admin-only CLI/endpoint that scans `status=done` + missing `dpp`, computes identifiers, and updates `process.dpp`.

## File inputs
```yaml
inputKey: "Gallium certification"
inputType: "file"
```
- File steps use multipart upload from backoffice and expose a process/substep download URL.
- Upload size is controlled by `ATTACHMENT_MAX_BYTES` (default `26214400`, i.e. 25 MiB).
- Files are stored in Mongo GridFS bucket `attachments` (`attachments.files` + `attachments.chunks`).

## Tests
Unit tests:
```bash
task test
# or: cd server && go test ./...
```

Coverage with 90% unit-test gate:
```bash
task cover
# or: cd server && go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out
```
`task cover` only runs the unit suite (`go test ./...`) and enforces `>= 90.0%` total coverage.

Optional integration test command (Docker-backed):
```bash
docker compose -f deployment/docker-compose.local.yaml up -d
cd server
go test -tags=integration ./...
```

## License
© 2025-2026 Forkbomb bv (forkbomb.eu) — The Forkbomb Company.

Licensed under the GNU AGPLv3 (see `LICENSE`).

## Funding
CLOSER (Circular raw materiaLs for european Open Strategic autonomy on chips and microElectronics pRoduction, Project No. 101161109) is funded by the European Union under the Interregional Innovation Investments (I3) Instrument of the European Regional Development Fund, managed by the European Innovation Council and SMEs Executive Agency (EISMEA).

This repository/website is part of the CLOSER project and has received funding from the European Union. Views and opinions expressed are those of the author(s) only and do not necessarily reflect those of the European Union or EISMEA. Neither the European Union nor the granting authority can be held responsible for them.

## Troubleshooting
- `open Dockerfile.local`: you’re on an old checkout — `deployment/Dockerfile.local` is required by `deployment/docker-compose.local.yaml`.
