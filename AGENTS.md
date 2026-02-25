# AGENTS.md

## Repository overview
This repo is a small end-to-end demo:

- **Backend**: Go HTTP server (net/http) rendering HTML templates, using **MongoDB** for persistence and **GridFS** for file uploads.
- **Frontend**: Vite-built JS/CSS bundle, plus **HTMX** in templates and **SSE** for live updates.
- **Authorization**: **Cerbos** policy engine for “can complete substep” checks.

See: `README.md`, `QUICKSTART.md`, `DOCKER.md`.

## Agent behavior expectations

When acting as a coding agent in this repository:

- Prefer **minimal, localized changes**
- Do not refactor for style or architecture unless explicitly requested
- Match existing patterns, even if they are imperfect
- Read relevant code before making changes (especially `server/cmd/server/main.go`)
- Ask before making breaking changes to routes, workflow config, or persistence

## Layout
- `server/` — Go module (`server/go.mod`)
  - `server/cmd/server/` — all backend code (single `package main`) and most logic.
  - `server/templates/` — Go `html/template` templates.
  - `server/config/workflow.yaml` — runtime workflow + departments + users.
- `web/` — Vite project
  - `web/src/main.js` — SSE + partial refresh client logic.
  - `web/dist/` — build output (served by backend as static assets).
- `cerbos/` — Cerbos configuration and policies
  - `cerbos/config/config.yaml`
  - `cerbos/policies/substep_policy.yaml`
- `deployment/` — Dockerfiles + Compose for local/Coolify/ephemeral builds
- `Taskfile.yml` — common developer commands.

## Tooling / versions
Observed requirements:
- Go **1.25.x** (CI pins 1.25.5 via mise; see `mise.toml`, `.github/workflows/tests.yml`)
- Node.js **18+** (see `README.md`, `web/package.json`, `deployment/Dockerfile.*`)
- Docker + Docker Compose (see `README.md`, `QUICKSTART.md`, `DOCKER.md`)
- `task` (Taskfile runner; used in CI)

## Essential commands
### Backend (Go)
Run the server (from repo root):
```bash
cd server
go mod tidy
go run ./cmd/server
```

Backend unit tests (from repo root):
```bash
task test
# or:
cd server
go test ./...
```

Coverage with a 90% gate (this is what CI runs):
```bash
task cover
```

Integration tests (Docker-backed; tests will `Skip` if Mongo/Cerbos are unavailable):
```bash
cd server
go test -tags=integration ./...
```

### Frontend (Vite)
Build bundle (backend expects `../web/dist` to exist):
```bash
cd web
npm install
npm run build
```

Dev server (if you’re working on the web bundle):
```bash
cd web
npm run dev
```

### Docker / infra
Docker Compose files are under `deployment/`.

Bring up stack:
```bash
docker compose -f deployment/docker-compose.local.yaml up -d
```

Taskfile shortcut:
```bash
task start
```

## Runtime configuration
Backend environment variables (observed):
- `MONGODB_URI` (default `mongodb://localhost:27017`) — used in `server/cmd/server/main.go:248`
- `CERBOS_URL` (default `http://localhost:3592`) — used in `server/cmd/server/main.go:267`
- `WORKFLOW_CONFIG` (default `config/workflow.yaml`) — used in `server/cmd/server/main.go:271`
- `ATTACHMENT_MAX_BYTES` (default 25 MiB) — max upload size; used in `server/cmd/server/main.go:298-309`

Example env file: `.env.example`.

Workflow config structure lives in `server/config/workflow.yaml` and is loaded/reloaded by `Server.getConfig()` (`server/cmd/server/main.go:1033+`).

## Backend architecture notes (what to know before changing things)
### HTTP routes
Routes are registered in `server/cmd/server/main.go:274-283`.
Key endpoints:
- `POST /process/start`
- `GET /process/:id`
- `GET /process/:id/timeline` (partial HTML)
- `POST /process/:id/substep/:substepId/complete`
- `GET /process/:id/substep/:substepId/file` (download)
- `GET /backoffice` and `/backoffice/:role` (dashboard)
- `POST /impersonate` (sets a cookie)
- `GET /events` (SSE)

### Actor/role identity
Impersonation is cookie-based:
- cookie name: `demo_user`
- cookie value format: `userId|role`

See `readActor()` in `server/cmd/server/main.go:1175-1185` and handler `handleImpersonate()` in `server/cmd/server/main.go:534-560`.

### Authorization (Cerbos)
The backend checks whether a substep can be completed via Cerbos:
- client: `CerbosAuthorizer` in `server/cmd/server/authorizer.go`
- policy: `cerbos/policies/substep_policy.yaml`

Cerbos request includes `sequenceOk` and role requirements (`server/cmd/server/authorizer.go:33-56`).

### Process progress keys (Mongo gotcha)
Substep IDs contain dots (e.g. `1.1`). MongoDB field names cannot contain dots, so progress map keys are encoded:
- encode for storage: `encodeProgressKey()` replaces `.` with `_` (`server/cmd/server/main.go:1400-1402`)
- decode for reads: `normalizeProgressKeys()` replaces `_` with `.` (`server/cmd/server/main.go:1404-1414`)

When touching progress persistence, follow this pattern (see `MongoStore.UpdateProcessProgress()` in `server/cmd/server/store.go:111-118`).

### File uploads / downloads
- Completion payloads are either scalar (`ParseForm`) or file (`ParseMultipartForm`) based on workflow `inputType`.
- File uploads are size-limited with `http.MaxBytesReader` and `ATTACHMENT_MAX_BYTES`.
- Files are stored in **Mongo GridFS** bucket named **`attachments`** (`server/cmd/server/store.go:229-231`).
- Metadata is stored in `attachments.files` (see `LoadAttachmentByID()` in `server/cmd/server/store.go:187+`).

Download endpoint streams GridFS content and sets `Content-Disposition` with a sanitized filename (`server/cmd/server/main.go:437-489`, `sanitizeAttachmentFilename()` at `server/cmd/server/main.go:842-860`).

### SSE (server) + partial refresh (web)
- SSE hub is `SSEHub` (`server/cmd/server/main.go:109-1640`).
- Backend emits:
  - `event: process-updated` for process streams
  - `event: role-updated` for role dashboards
  (see `handleEvents()` in `server/cmd/server/main.go:862-909`).
- Frontend listens via `EventSource` and refreshes partial HTML via `fetch()` (`web/src/main.js`).

### DPP / GS1 Digital Link
- Workflow YAML supports optional `dpp:` config (`enabled`, `gtin`, `lotInputKey`, `lotDefault`, `serialInputKey`, `serialStrategy`, plus presentation fields).
- `gtin` is normalized/validated at config load (must resolve to 14 digits when enabled).
- On first transition to process `done`, backend stores `process.dpp` (`gtin`, `lot`, `serial`, `generatedAt`) and keeps identifiers stable on repeated completion calls.
- Public Digital Link route is `GET /01/{gtin}/10/{lot}/21/{serial}`:
  - HTML landing page (template: `server/templates/dpp.html`)
  - JSON (`Accept: application/json` or `?format=json`)
- DPP HTML traceability now renders user-entered values and file download links inline per substep (no separate Documents section).
- Process page downloads panel now shows a DPP link when `process.DPP` exists.

## Templates and static assets
- Templates are parsed from `server/templates/*.html` (`server/cmd/server/main.go:258-261`).
- Backoffice department/process headers now show workflow name plus a computed step title:
  - Todo actions use parent step title from the todo substep.
  - Active streams use the step title of the role-scoped next available substep.
  - Done streams/process headers fall back to the last workflow step title.
- Backoffice action cards (`server/templates/action_list.html`) render editable forms only for non-`done` actions; `done` actions render a read-only Submitted block with flattened values and attachment download links.
- Locked Formata actions render `action-card action-locked` and `.js-formata-host[data-formata-disabled="true"]`; when disabled, the builder link is replaced by “Locked: complete previous steps first.”
- Static assets are served from `../web/dist` under `/static/` (`server/cmd/server/main.go:275`).
- Layout template includes HTMX via an external script tag (`server/templates/layout.html:9-13`).

## Testing patterns
- Unit tests live next to code in `server/cmd/server/*_test.go`.
- Most handler tests use `httptest.NewRequest`/`httptest.NewRecorder` and a `MemoryStore` (`server/cmd/server/store.go:233+`).
- `server/cmd/server/backoffice_templates_render_test.go` parses real templates (`server/templates/*.html`) to assert workflow + step labels render in `/backoffice/:role` and `/backoffice/:role/process/:id`.
- Integration tests are behind build tag `integration` (`server/cmd/server/integration_complete_test.go`) and skip if dependencies are unavailable.

## Deployment files
Observed Dockerfiles:
- `deployment/Dockerfile.coolify` — builds web bundle + Go binary into a small runtime image.
- `deployment/Dockerfile.ephemeral` — same as coolify, but with an entrypoint that optionally runs a seed script (`deployment/ephemeral-entrypoint.sh`).
- `deployment/Dockerfile.cerbos` — builds Cerbos image with repo config/policies.

## Known gotchas / inconsistencies (as checked in this repo)
- `DOCKER.md` lists Cerbos image `ghcr.io/cerbos/cerbos:0.39.0`, but `deployment/docker-compose.local.yaml` and `deployment/Dockerfile.cerbos` use **0.50.0**.
- `Taskfile.yml` includes a `cerbos-health` task that runs `curl`.
