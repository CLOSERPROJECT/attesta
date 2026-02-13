# Quick Start Guide

This demo runs MongoDB + Cerbos with Docker Compose, a Go server, and a Vite-built asset bundle.

## Prerequisites
- Docker + Docker Compose
- Go 1.25+
- Node.js 18+

## Start services
```bash
docker compose -f deployment/docker-compose.local.yaml up -d
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

The entry pages are workflow pickers. Business routes are workflow-scoped under `/w/{workflowKey}/...`.

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
