# Quick Start Guide

This demo runs MongoDB + Cerbos + Appwrite with Docker Compose, a Go server, and a Vite-built asset bundle.

## Prerequisites
- Docker + Docker Compose 2.24.4+
- Go 1.25+
- Node.js 18+
- Task

For parallel worktrees, install the repo-pinned Worktrunk tool with
`mise install`. Worktrunk-created worktrees run `scripts/worktree-bootstrap.sh`
from `.config/wt.toml`. Codex-managed worktrees run it through the checked-in
local environment; for a plain Git worktree, run it explicitly. Like Credimi,
it derives deterministic candidates with Worktrunk's `hash_port`, skips live
listeners and ports reserved by sibling worktrees, and writes the result once
in `.env.worktree`. The flat `KEY=value` file can be edited afterward and is
never overwritten. The same bootstrap saves one coordinated primary MongoDB and
Appwrite bundle under ignored `.worktree-data/`; `task start` restores it only
before an uninitialized worktree stack starts.

Before the first unattended Worktrunk operation, review and approve the shared
hooks with `wt config approvals add`.

## Start services
```bash
task start
```

This starts a Compose project named after the checkout. Its containers,
networks, MongoDB volume, and Appwrite volumes are independent of every other
worktree. Host ports come from `.env.worktree`.

Use `task start:docker` for the full containerized stack, or
`task start:docker:build` to rebuild it. The containerized app is available on
`DOCKER_APP_PORT`; `task start` remains infrastructure-only for host development.

## Bootstrap Appwrite
The primary checkout may be bootstrapped manually after the compose stack is
up:
1. Open the Appwrite console on `http://localhost`.
2. Create the first Appwrite console account.
3. Create the Attesta Appwrite project.
4. Create an API key for Attesta.
5. Open Mailpit on `http://localhost:8025` if you want to inspect invite and recovery emails locally.
6. Create the `org-assets` storage bucket.
7. Export or set:
   - `APPWRITE_PROJECT_ID`
   - `APPWRITE_API_KEY`
8. Run `task start:docker` if those values changed and you use the containerized app.

New parallel worktrees normally restore the primary checkout's coordinated
snapshot, including Appwrite MariaDB and durable file storage. The checked-in
`deployment/appwrite/appwrite-seed.sql` remains the fallback for a genuinely
fresh stack with no copied bundle; setup never mixes that seed with a copied
live MongoDB.

Snapshot capture is serialized across the repository. If the primary has no
durable data, or after `task purge`, the worktree records a persistent
fresh-seed choice instead of reconsidering and copying newer primary data on a
later start.

## Migration Notes
- Existing Mongo-backed sessions are not migrated. After Appwrite credentials are configured, users must log in again through Appwrite.
- Existing Mongo invite and password-reset tokens are not migrated. Re-issue any still-needed invite or reset from the Appwrite-backed Attesta flows.
- If you are cutting over an existing demo dataset, migrate organizations, roles, and active users into Appwrite before relying on `/my/organization/*`.

## Build frontend assets
```bash
cd web
npm install
npm run build
```

## Run the Go server
```bash
cd ../server
go mod tidy
go run ./cmd/server
```

## Open the demo
- Appwrite Console: `http://localhost:${APPWRITE_HTTP_PORT}`
- Public homepage: `http://localhost:${PORT}/`
- Stream picker (after login): `http://localhost:${PORT}/my`
- Platform admin (when enabled): `http://localhost:${PORT}/admin`
- Mailpit: `http://localhost:${MAILPIT_UI_PORT}`

After login, the stream picker is at `/my`. Stream and instance routes live under `/my/streams/{workflowKey}/...` (legacy `/w/` and `/org-admin/` paths return 404).

## Configure DPP Digital Link (optional)
Edit `server/config/workflow.yaml` and add:

```yaml
dpp:
  enabled: true
  gtin: "09506000134352"
```

When enabled, completing a workflow generates a Digital Link at `/01/{GTIN}/10/{LOT}/21/{SERIAL}`.

## Run tests
Unit tests:
```bash
task test
```

Coverage with 90% unit-test gate:
```bash
task cover
```
`task cover` enforces `>= 90.0%` total coverage on the unit suite.

Optional integration command:
```bash
cd server
go test -tags=integration ./...
```
