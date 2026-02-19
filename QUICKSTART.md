# Quick Start Guide

This demo runs MongoDB + Cerbos (+ optional Appwrite and Mailpit) with Docker Compose, a Go server, and a Vite-built asset bundle.

## Prerequisites
- Docker + Docker Compose
- Go 1.25+
- Node.js 18+

## Start services
```bash
docker compose -f deployment/docker-compose.local.yaml up -d
```

Optional: expose Appwrite console on host
```bash
docker compose -f deployment/docker-compose.local.yaml -f deployment/docker-compose.local.appwrite-exposed.yaml up -d
# Appwrite console: http://localhost:19080/console
```

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
- Home: http://localhost:3000
- Backoffice: http://localhost:3000/backoffice
- Mailpit (optional): http://localhost:18025

The entry pages are workflow pickers. Business routes are workflow-scoped under `/w/{workflowKey}/...`.

## Optional Appwrite team checks in Cerbos
Set these env vars before starting `server/cmd/server`:
- `APPWRITE_ENDPOINT` (for local Compose `attesta` container, `http://appwrite/v1`)
- `APPWRITE_PROJECT_ID`
- `APPWRITE_API_KEY`
- `APPWRITE_TEAMS_STRICT` (`false` by default; set `true` to fail closed on Appwrite lookup errors)

Check current actor team resolution (JSON):
```bash
curl http://localhost:3000/w/workflow/debug/teams
```

Optional: provision Appwrite users/teams/memberships from workflow YAML on startup:
```bash
export APPWRITE_SYNC_FROM_WORKFLOW=true
export APPWRITE_SYNC_DEFAULT_PASSWORD='TempPassw0rd!'
export APPWRITE_SYNC_MEMBERSHIP_URL='http://localhost:3030'
```

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
