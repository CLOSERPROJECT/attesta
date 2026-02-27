# Closer demo (Gallium Recycling Notarization)

Small end-to-end demo with Go + HTMX + SSE + MongoDB + Cerbos.

## Requirements

You can run the project in one of two ways:
### Manual toolchain
- Go 1.25+
- Node.js 18+ (with npm)
- [Task](https://taskfile.dev)
- Docker + Docker Compose

### Using mise (recommended)
- [mise](https://mise.jdx.dev/installing-mise.html)
- Docker + Docker Compose

## Quick start

You can run the project in one of two ways:
Make sure you have installed **either**:
- The full manual toolchain (Go, Node, Task, Docker), or
- mise + Docker

If you are using the manual toolchain, skip the `mise` commands below.

```bash
# Clone the repository
git clone https://github.com/CLOSERPROJECT/attesta
cd attesta

# If using mise:
mise trust
mise install

# Start the development environment
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
- Uses authenticated users with session cookies.
- Cerbos enforces role + sequence gating.
- Mongo stores process progress + notarizations.
- SSE broadcasts realtime updates to timelines.

## Curl examples
Start a process in a selected workflow (`workflow`):
```bash
curl -X POST http://localhost:3000/w/workflow/process/start -i
```

Login and capture the session cookie:
```bash
curl -X POST http://localhost:3000/login \
  -d 'email=admin@example.com' -d 'password=change-me' -i
```

Complete substep 1.1 (dep1):
```bash
curl -X POST http://localhost:3000/w/workflow/process/PROCESS_ID/substep/1.1/complete \
  -H 'Cookie: attesta_session=SESSION_ID' \
  -d 'value=10&activeRole=dep1'
```

Attempt out-of-sequence (should fail):
```bash
curl -X POST http://localhost:3000/w/workflow/process/PROCESS_ID/substep/2.1/complete \
  -H 'Cookie: attesta_session=SESSION_ID' \
  -d 'value=5&activeRole=dep2'
```

## Notes
- Cerbos PDP is expected at `http://localhost:3592`.
- MongoDB is expected at `mongodb://localhost:27017`.
- Timeline updates pull `/w/:workflow/process/:id/timeline` when SSE events arrive.
- Existing processes without `workflowKey` remain visible under the default `workflow` key and are backfilled on first update.

## Deployment Checklist
1. Set auth/bootstrap env vars:
   - `ADMIN_EMAIL`, `ADMIN_PASSWORD`
   - `ANYONE_CAN_CREATE_ACCOUNT` (recommended `false` in production)
   - `SESSION_TTL_DAYS` and `COOKIE_SECURE=true` behind HTTPS
2. Start services and verify Mongo + Cerbos health.
3. Login as platform admin and create organizations.
4. Create org admin invites, then create org roles/users from org admin pages.
5. Ensure workflow YAML org/role slugs match Mongo entities.
6. Keep DPP route `/01/...` public only if intended; keep authenticated downloads protected unless explicitly opened.

## Org admin edge cases
- `Delete account` is a soft delete: the backend sets `status=deleted`, clears `passwordHash`, and clears `roleSlugs`.
- Invite status is derived from invite timestamps:
  - `accepted` when `usedAt` is present
  - `expired` when `usedAt` is empty and `expiresAt` is in the past
  - `pending` otherwise
- Inviting an email that already belongs to another organization is rejected.

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
