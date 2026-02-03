# Closer demo (Gallium Recycling Notarization)

Small end-to-end demo with Go + HTMX + SSE + MongoDB + Cerbos.

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

## What it does
- Seeds a workflow definition on first run.
- Starts a process instance with sequential substeps.
- Backoffice impersonates roles (dep1, dep2, dep3) and completes actions.
- Cerbos enforces role + sequence gating.
- Mongo stores process progress + notarizations.
- SSE broadcasts realtime updates to timelines.

## Curl examples
Start a process:
```bash
curl -X POST http://localhost:3000/process/start -i
```

Impersonate dep1:
```bash
curl -X POST http://localhost:3000/impersonate \
  -d 'userId=u1' -d 'role=dep1' -i
```

Complete substep 1.1 (dep1):
```bash
curl -X POST http://localhost:3000/process/PROCESS_ID/substep/1.1/complete \
  -H 'Cookie: demo_user=u1|dep1' \
  -d 'value=10'
```

Attempt out-of-sequence (should fail):
```bash
curl -X POST http://localhost:3000/process/PROCESS_ID/substep/2.1/complete \
  -H 'Cookie: demo_user=u2|dep2' \
  -d 'value=5'
```

## Notes
- Cerbos PDP is expected at `http://localhost:3592`.
- MongoDB is expected at `mongodb://localhost:27017`.
- Timeline updates pull `/process/:id/timeline` when SSE events arrive.

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

Coverage with 70% gate:
```bash
task cover
# or: cd server && go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out
```

Optional integration test command (Docker-backed):
```bash
docker compose -f deployment/docker-compose.local.yaml up -d
cd server
go test -tags=integration ./...
```
