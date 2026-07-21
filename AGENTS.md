# AGENTS.md

## Repository overview
This repo is a small end-to-end demo:

- **Backend**: Go HTTP server (net/http) rendering HTML templates, using **MongoDB** for persistence and **GridFS** for file uploads.
- **Frontend**: Vite-built JS/CSS bundle, plus **HTMX** in templates and **SSE** for live updates.
- **Authorization**: **Cerbos** policy engine for “can complete substep” checks.

See: `README.md`, `QUICKSTART.md`, `DOCKER.md`, `docs/css.md` (main app styling).

## Current auth/org status (2026-07)
- Demo impersonation has been removed from production code paths.
- Session auth is active (`attesta_session` cookie). Regular users store an Appwrite session secret; platform admin uses a separate env-derived session value (`platform-admin:…`).
- Stream dashboard is `/w/:key/` (lists stream instances for one stream). Legacy `/dashboard` and `/w/:key/dashboard` are not registered.
- Admin consoles:
  - Platform admin: `/admin/orgs` (create/edit/delete orgs, upload logos, invite org admins)
  - Org admin: `/org-admin/profile`, `/org-admin/roles`, `/org-admin/members` (legacy `GET /org-admin/users` redirects to profile)
- Platform admin is env-driven (`ADMIN_EMAIL`, `ADMIN_PASSWORD`). On startup the server ensures that account exists in Appwrite (`bootstrapPlatformAdminIdentity`). Cerbos policy `platform_admin_console` gates console access.
- Auth/org state now lives in Appwrite:
  - orgs -> teams
  - role catalog -> team prefs (`roles[].palette` resolved to CSS via `data-role-palette` on templates)
  - accepted roles -> user labels
  - invites -> memberships
  - signup/login/reset -> Appwrite account/session/recovery flows
- Global topbar now renders role-aware admin links on authenticated pages:
  - Platform admin sees `Orgs` (`/admin/orgs`)
  - Org admin with org context sees `My Org` (`/org-admin/profile`)
- Workflow YAML supports `organizations`, `roles`, step-level `organization`, and substep `roles`.
- Slug collisions on org and role creation now surface explicit `... slug already exists` errors in admin UIs.
- Org admin members section (`/org-admin/members`; forms still `POST /org-admin/users`) supports:
  - invites with zero-to-many roles (`roles` multi-select, `intent=invite`)
  - "Invites I sent" with derived statuses (`pending`, `accepted`, `expired`)
  - user role editing (`intent=set_roles`) and soft-delete (`intent=delete_user`) with self-protection checks.

## Agent behavior expectations

When acting as a coding agent in this repository:

- Prefer **minimal, localized changes**
- Do not refactor for style or architecture unless explicitly requested
- Match existing patterns, even if they are imperfect
- Read `docs/css.md` before changing templates or styles in `web/src/styles/`
- Read relevant code before making changes (especially `server/cmd/server/main.go`)
- Ask before making breaking changes to routes, workflow config, or persistence

## Layout
- `server/` — Go module (`server/go.mod`)
  - `server/cmd/server/` — single `package main`; route wiring and most handlers in `main.go`, with domain logic peeled into focused files: `timeline_builder.go`, `substep_views_builder.go`, `stream_instance_detail.go`, `stream_step_summary.go`, `done_by_identity.go`, `dpp.go`, `components.go`, `authorizer.go`, `store.go`, `identity*.go`, `formata_builder.go`, `role_meta.go`, `templates.go`.
  - `server/templates/` — Go `html/template` templates (`layout.html` at root; full screens in `pages/`; reusable partials in `components/` as they are migrated).
  - `server/config/workflow.yaml` — runtime workflow + departments + users.
- `web/` — Vite project
  - `web/src/main.js` — SSE + partial refresh client logic.
  - `web/src/styles/` — layered CSS (`tokens.css`, `role-palette.css`, …; see `docs/css.md`).
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

Backend unit tests with a 90% gate (from repo root):
```bash
task cover
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
Backend environment variables are read in `main()` (`server/cmd/server/main.go` env bootstrap). Common vars:
- `MONGODB_URI` (default `mongodb://localhost:27017`)
- `CERBOS_URL` (default `http://localhost:3592`)
- `APPWRITE_ENDPOINT` (default `http://appwrite/v1`)
- `APPWRITE_PROJECT_ID`
- `APPWRITE_API_KEY`
- `APPWRITE_INVITE_REDIRECT_URL`
- `APPWRITE_RESET_REDIRECT_URL`
- `APPWRITE_ORG_ASSETS_BUCKET` (default `org-assets`)
- `WORKFLOW_CONFIG` (default `config/workflow.yaml`); `WORKFLOW_CONFIG_DIR` overrides the catalog directory
- `ATTACHMENT_MAX_BYTES` (default 25 MiB) — max upload size via `attachmentMaxBytes()`
- `ADMIN_EMAIL`, `ADMIN_PASSWORD` — platform admin credentials; both required to enable the console
- `ANYONE_CAN_CREATE_ACCOUNT`
- `SESSION_TTL_DAYS`, `COOKIE_SECURE`

Example env file: `.env.example`.

Workflow YAML lives under `server/config/` (and optional `WORKFLOW_CONFIG_DIR`). Runtime lookup uses `Server.runtimeConfig()` → `configProvider` (tests) or `workflowByKey()` / catalog reload — not a `getConfig()` helper.

## Backend architecture notes (what to know before changing things)
### HTTP routes
Global routes are registered in `Server.newMux()` (`server/cmd/server/main.go`). Workflow-scoped routes mount at `/w/` via `handleWorkflowRoutes` → `handleProcessRoutes`.

**Global (non-workflow):**
- `GET /` — stream picker (`handleHome`)
- `GET/POST /login`, `GET/POST /signup`, `POST /logout`
- `GET /invite/…`, `GET/POST /reset`, `GET/POST /reset/…`
- `GET/POST /admin/orgs`, `GET/POST /admin/orgs/` (platform admin org console; logo at `/admin/orgs/logo/:id`)
- `GET /org-admin/profile`, `/org-admin/roles`, `/org-admin/members` (org settings sections); `GET /org-admin/users` → profile; `POST /org-admin/users`, `POST /org-admin/roles`; `/org-admin/formata-builder`, …
- `GET /01/…` — public DPP Digital Link
- `GET /events` — legacy SSE mux entry (production UI uses workflow-scoped path below)

**Workflow-scoped (`/w/:key/…`):**
- `GET /w/:key/` — stream dashboard (instance list + timeline preview)
- `POST /w/:key/process/start`
- `GET /w/:key/process/:id` — stream instance detail page
- `GET /w/:key/process/:id/content` — HTMX/SSE content partial (replaces old `/timeline`)
- `GET /w/:key/process/:id/downloads` — downloads partial
- `POST /w/:key/process/:id/terminate`
- `POST /w/:key/process/:id/substep/:substepId/complete`
- `GET/POST /w/:key/process/:id/substep/:substepId/override`
- `GET /w/:key/process/:id/attachment/:attachmentId/file` — attachment download
- Export downloads: `files.zip`, `notarized.json`, `merkle.json` under `/w/:key/process/:id/…`
- `GET /w/:key/events?processId=…` or `?role=…` — workflow-scoped SSE (used by `web/src/main.js`)

Legacy `/dashboard` and `/w/:key/dashboard` return 404 (`workflow_coverage_test`).

### Actor/role identity
Session auth via `attesta_session` cookie:
- Regular users: Appwrite session secret from login/signup/invite flows (`readSession()`, `currentUser()` in `main.go`)
- Platform admin: env-derived session value (`platform-admin:…` via `platformAdminSessionValue()`)
- Request actor for Cerbos/completion: `Actor` built from authenticated user + workflow context (org slug, role slugs, `workflowKey`)

Demo impersonation (`demo_user` cookie, `readActor()`, `handleImpersonate()`) is removed from production code; `demo_user` may still appear in older tests.

### Authorization (Cerbos)
The backend checks whether a substep can be completed via Cerbos:
- client: `CerbosAuthorizer` in `server/cmd/server/authorizer.go`
- policy: `cerbos/policies/substep_policy.yaml`

Cerbos request includes `sequenceOk` and role requirements (`CerbosAuthorizer` in `authorizer.go`).

### Process progress keys (Mongo gotcha)
Substep IDs contain dots (e.g. `1.1`). MongoDB field names cannot contain dots, so progress map keys are encoded:
- encode for storage: `encodeProgressKey()` replaces `.` with `_`
- decode for reads: `normalizeProgressKeys()` replaces `_` with `.`

When touching progress persistence, follow this pattern (see `MongoStore.UpdateProcessProgress()` in `store.go`).

### File uploads / downloads
- Completion payloads are either scalar (`ParseForm`) or file (`ParseMultipartForm`) based on workflow `inputType`.
- File uploads are size-limited with `http.MaxBytesReader` and `ATTACHMENT_MAX_BYTES`.
- Files are stored in **Mongo GridFS** bucket named **`attachments`** (`store.go`).
- Metadata is stored in `attachments.files` (see `LoadAttachmentByID()` in `store.go`).

Download endpoint `handleDownloadProcessAttachment` streams GridFS content and sets `Content-Disposition` with a sanitized filename (`sanitizeAttachmentFilename()` in `main.go`).

### SSE (server) + partial refresh (web)
- SSE hub is `SSEHub` (`main.go`).
- Backend emits:
  - `event: process-updated` for process streams
  - `event: role-updated` for role dashboards
  (see `handleEvents()` in `main.go`; workflow-scoped at `/w/:key/events`).
- Frontend listens via `EventSource` and refreshes partial HTML via `fetch()` (`web/src/main.js`).

### DPP / GS1 Digital Link
- Workflow YAML supports optional `dpp:` config (`enabled`, `gtin`, `lotInputKey`, `lotDefault`, `serialInputKey`, `serialStrategy`, plus presentation fields).
- `gtin` is normalized/validated at config load (must resolve to 14 digits when enabled).
- On first transition to process `done`, backend stores `process.dpp` (`gtin`, `lot`, `serial`, `generatedAt`) and keeps identifiers stable on repeated completion calls.
- Public Digital Link route is `GET /01/{gtin}/10/{lot}/21/{serial}`:
  - HTML landing page (template: `server/templates/pages/dpp.html`)
  - JSON (`Accept: application/json` or `?format=json`)
- DPP HTML traceability now renders user-entered values and file download links inline per substep (no separate Documents section).
- Process page downloads panel now shows a DPP link when `process.DPP` exists.

## Templates and static assets
- Templates load from `server/templates/*.html`, `server/templates/pages/*.html`, and `server/templates/components/*.html` via `parseTemplates()` in `server/cmd/server/templates.go`. Custom funcs in `templateFuncs()` include `dict` for inline map literals and typed wrappers such as `streamTimelineStep` / `streamTimelineSubstep` (e.g. `{{ template "stream_timeline_step" (streamTimelineStep . $.HideStatus) }}`).
- **Template define names** match the file stem (no extension): e.g. `components/stream_card.html` → `{{ define "stream_card" }}`. Page wrappers and body blocks still use legacy `*.html` / `*_body` defines until migrated.
- **Shared view structs** for reusable components live in `server/cmd/server/components.go` (`SubstepBodyView`, `StreamInstanceDetailView`, `StreamCardView`, …). Use struct literals at call sites — no fluent `With*` builders unless there is real logic. Page/view assembly is partially peeled (`stream_instance_detail.go`, `substep_views_builder.go`, `timeline_builder.go`); remaining handlers stay in `main.go`.
- **Component tiers:** full template components (`templates/components/` + view struct in `components.go`); **CSS-only components** (see `docs/css.md`) — reused markup with a dedicated CSS module and inline HTML, no full template partial (examples: page-header in `page-header.css`, panel in `panel.css`, dialog in `dialog.css`, list-row in `list-row.css`); primitives in `shared.css`. Full component example for trails: `breadcrumbs` (`breadcrumbs.html` + `BreadcrumbsView`). Migrate one at a time.
- Substep bodies (`server/templates/components/substep_body.html`) dispatch on explicit **`Mode`** (`preview`|`actionable`|`result`|`message`) via `effectiveSubstepBodyMode`; builders set `SubstepBodyView.Mode` (`resolveSubstepBodyMode` in `components.go`).
- Stream timeline (`server/templates/components/stream_timeline.html`) renders the step/substep accordion tree on stream instance detail and stream dashboard preview; inner define `stream_timeline_step` calls `substep_shell` via `(streamTimelineSubstep . $.HideStatus)`; `substep_shell` dispatches to `substep_body` with `TimelineSubstep.Body` (`*SubstepBodyView`).
- Stream instance detail partial (`stream_instance_detail_content`) is built by `buildStreamInstanceDetailView` in `stream_instance_detail.go` and exposed on `ProcessPageView.Detail` (`StreamInstanceDetailView`).
- Locked Formata substeps render `.js-formata-host[data-formata-disabled="true"]` in preview mode; when disabled, the builder link is replaced by “Locked: complete previous steps first.”
- Static assets are served from `../web/dist` under `/static/` (`newMux()` in `main.go`).
- Layout template includes HTMX via an external script tag (`server/templates/layout.html`).

## Testing patterns
- Unit tests live next to code in `server/cmd/server/*_test.go`.
- Most handler tests use `httptest.NewRequest`/`httptest.NewRecorder` and a `MemoryStore` (`store.go`).
- Integration tests are behind build tag `integration` (`server/cmd/server/integration_complete_test.go`) and skip if dependencies are unavailable.

## Deployment files
Observed Dockerfiles:
- `deployment/Dockerfile.coolify` — builds web bundle + Go binary into a small runtime image.
- `deployment/Dockerfile.ephemeral` — same as coolify, but with an entrypoint that optionally runs a seed script (`deployment/ephemeral-entrypoint.sh`).
- `deployment/Dockerfile.cerbos` — builds Cerbos image with repo config/policies.

## Known gotchas / inconsistencies (as checked in this repo)
- `DOCKER.md` lists Cerbos image `ghcr.io/cerbos/cerbos:0.39.0`, but `deployment/docker-compose.local.yaml` and `deployment/Dockerfile.cerbos` use **0.50.0**.
- `Taskfile.yml` includes a `cerbos-health` task that runs `curl`.
