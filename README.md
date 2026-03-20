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

# create .env
cp .env.example .env

# Start the backend services
task start
```

Now visit http://localhost/console/register and create the admin account, once logged in:
* Create a new project and copy the project id inside the .env (APPWRITE_PROJECT_ID)
* Create a new API key with Auth permission (all) and Storage permissions (files.read and files,write) and copy it inside the .env (APPWRITE_API_KEY)
* Visit storage page and create a new bucket named `org-assets`

Now beck to the terminal run:

```bash
task dev
```

Open http://localhost:3000 to your local running attesta software.

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
1. Set Attesta auth env vars:
   - `ANYONE_CAN_CREATE_ACCOUNT` (recommended `false` in production)
   - `SESSION_TTL_DAYS`
   - `COOKIE_SECURE=true` behind HTTPS
2. Set Attesta Appwrite env vars:
   - `APPWRITE_ENDPOINT`
   - `APPWRITE_PROJECT_ID`
   - `APPWRITE_API_KEY`
   - `APPWRITE_INVITE_REDIRECT_URL`
   - `APPWRITE_RESET_REDIRECT_URL`
   - `APPWRITE_ORG_ASSETS_BUCKET`
3. Start services and verify Mongo + Cerbos health.
4. Bootstrap Appwrite from the Console:
   - create the first console account
   - create the Attesta project
   - create the Attesta API key
   - create the org assets storage bucket
5. Create organizations and first org-admin memberships in Appwrite, then manage roles and users from Attesta `/org-admin/*`.
6. Ensure workflow YAML org/role slugs match Appwrite teams and role catalogs.
7. Keep DPP route `/01/...` public only if intended; keep authenticated downloads protected unless explicitly opened.

## Appwrite Cutover
Migration prerequisites:
1. Provision the Appwrite project, platform entries, email templates, and the `org-assets` storage bucket.
2. Export Mongo organizations, roles, and active users into Appwrite teams, team prefs, memberships, and labels.
3. Manually bootstrap the first org-admin owner for each Appwrite team from the Appwrite Console.
4. Deploy Attesta with `APPWRITE_*` env vars configured and the internal `/admin/orgs` flow removed.
5. Invalidate old Mongo-backed sessions so users authenticate again through Appwrite.
6. Expire old pending invite and password-reset tokens and re-issue them through Appwrite.

Rollback note:
- Rollback is only application-safe before Attesta starts writing auth and org mutations into Appwrite.
- After cutover, rolling back Attesta code does not restore Mongo as the source of truth for users, invites, or memberships.

Staging rehearsal:
1. invited user acceptance
2. self-signup when enabled
3. org creation by an unassigned user
4. org-admin invite from Attesta
5. role edit and user removal

## Org admin edge cases
- `Delete user` removes the Appwrite team membership and clears Attesta role labels, but does not delete the global Appwrite account.
- Invite status is derived from Appwrite memberships:
  - `accepted` when the membership is active
  - `expired` when the membership is no longer usable before acceptance
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
