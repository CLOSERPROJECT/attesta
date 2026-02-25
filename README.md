# Closer demo (Gallium Recycling Notarization)

Small end-to-end demo with Go + HTMX + SSE + MongoDB + Cerbos.

## Requirements
- Go 1.25+
- Node.js 18+ (with npm)
- [Task](https://taskfile.dev)
- Docker + Docker Compose
or
- mise + Docker + Docker compose

## Quick start

This quickstart relies on the presence of [mise](https://mise.jdx.dev/installing-mise.html) and Docker + Docker compose on the machine or all the above requirements installed.
In the second case you can skip the mise commands:

```bash
# Clone the repository
git clone https://github.com/CLOSERPROJECT/attesta
cd attesta

# Install requirements if not already present
mise trust && mise install

# Run in a dev environment
task dev
```

Open:
- Home: http://localhost:3000
- Backoffice: http://localhost:3000/backoffice
- Mongo Express (optional): http://localhost:8081

## Add new streams

1. Create a stream using the Attesta Stream Composer: https://closerproject.github.io/formata-arch/#/
2. Click **Export** to download the YAML file.
3. Copy the file into: `./server/config/`
4. Give it a unique name.

Attesta reads this folder live — no restart required.
Your new stream will appear immediately on: http://localhost:3000

## What it does
- Seeds a workflow definition on first run.
- Starts a process instance with sequential substeps.
- Backoffice impersonates roles (dep1, dep2, dep3) and completes actions.
- Cerbos enforces role + sequence gating.
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
- Timeline updates pull `/w/:workflow/process/:id/timeline` when SSE events arrive.
- Existing processes without `workflowKey` remain visible under the default `workflow` key and are backfilled on first update.

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
