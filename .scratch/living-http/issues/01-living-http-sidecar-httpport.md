# 01: Living-http sidecar — real HttpPort (tranche A)

**What to build:** Move plant `onSubmit.http` fakes out of Go into a Compose service `living-http`. Product Commit uses a real HttpPort client (path + base URL + shared secret). Sidecar produces real-enough PDF/CSV on disk; Attesta proxies artifact GETs for the browser. Failures still do not roll back Domain.

**Blocked by:** nothing (builds on flat-port **18** HttpPort + FakeHttpPort)

**Status:** approved — awaiting human squash + smoke

**Out of scope (tranche B, sketch only):** per-Organization base URLs/credentials; n8n; Coolify wiring; renaming `/fake/*` paths in Org Config.

## Grill decisions (2026-11-02) — do not re-litigate

| # | Decision |
| --- | --- |
| A | Tranche A = real HttpPort + one shared plant sidecar; multi-org peers/auth later |
| B | Sidecar/n8n = HttpPort Adapter target; SPA never calls it; glossary **HttpPort** unchanged |
| C | Org Config URLs stay path-shaped (`/fake/labels`); Go joins `LIVING_HTTP_URL` |
| D | Reject absolute `http(s)://` from Org Config (SSRF lock) |
| E | Sidecar internal Compose network only; shared secret `LIVING_HTTP_TOKEN` |
| F | Response = fetchable artifact URL (no `data:` in Commit); `Links` shape kept |
| G | Browser loads artifacts via **Attesta proxy** (session + org), not sidecar port |
| H | Plant service = small **Go** in-repo; endpoints keep `/fake/labels`, `/fake/register-csv` |
| I | Per-effect timeout (~5–10s); no Commit-path retry; errors → `httpErrors` |
| J | Artifact ids **unguessable**; proxy checks session + org; bind artifact → org at create |
| K | Info logs: method/path/status/duration/org — not bodies |
| L | Product: no `FakeHttpPort`; tests keep recording/fake doubles; missing URL → soft HttpPort errors |
| M | Local Compose + worktree port key; Coolify later |
| N | **Artifacts on local disk** (override: easier than memory for plant smoke/debug) |

### Tranche B sketch (not this ticket)

Each Organization may later configure allowlisted base URL(s) + secrets for their effect peers; plant remains one shared `living-http`. Still behind HttpPort + SSRF rules. n8n replaces the Go sidecar as Adapter target when ready.

## Security (plant)

- **SSRF:** path-only join to env base; never config-chosen host.
- **Caller auth:** Bearer (or equivalent) shared secret Attesta → sidecar.
- **Confused deputy:** secret required even on internal network.
- **Artifact authz:** unguessable id + Attesta session + org match; sidecar not browser-reachable.
- **Secrets:** Compose/env only — not Org Config, not git, not SPA.

## Subagents (sequential)

### 01A — Sidecar + Compose

- [x] New Go service (e.g. `services/living-http` or `server/cmd/living-http`) implementing:
  - `POST /fake/labels` → write PDF under data dir → return plain-text URL path/URL for Attesta to rewrite
  - `POST /fake/register-csv` → write CSV + `Content-Disposition` filename
  - `GET` artifact by unguessable id (token-gated or internal-only; prefer same token)
  - Disk store under configurable data dir; create dir on start; optional simple TTL/mtime cleanup OK
- [x] Dockerfile + `living-http` service in `deployment/docker-compose.local.yaml` (internal port; no host publish by default)
- [x] Worktree env: port key if needed for debug publish optional; `LIVING_HTTP_URL` / `LIVING_HTTP_TOKEN` for `attesta`
- [x] Wire `attesta` service env to reach `http://living-http:<port>` + shared token

### 01B — HttpPort client + artifact proxy

- [x] Real `HttpPort` Adapter (`net/http`): join base + path; send secret; timeout; map response URL/headers → `HttpLinkItem`
- [x] Reject absolute URLs from evaluated `onSubmit.http` url
- [x] Rewrite sidecar artifact URLs into Attesta proxy URLs in `Links` (browser-safe)
- [x] Attesta route: proxy artifact GET with session + org gate; fetch from sidecar with secret
- [x] Product wiring: `livingProductHttpPort()` uses env client, not `NewFakeHttpPort()`
- [x] Remove/stop planting FakeHttpPort in product; keep test doubles
- [x] Tests: success links; HttpPort failure does not rollback Domain; SSRF reject; timeout/soft errors

### 01C — Plant PDF quality + smoke notes

- [x] Labels PDF good enough to open in a browser (real PDF library OK)
- [x] Register Commit on feragame still returns two links; download filename preserved
- [x] Short smoke note in ticket or coordinator when green

### Smoke (01C) — Compose stack in this worktree

Env (Compose defaults in `deployment/docker-compose.local.yaml`):

- `LIVING_HTTP_URL=http://living-http:8090` (attesta → sidecar, internal network)
- `LIVING_HTTP_TOKEN` shared by `attesta` and `living-http` (default `dev-living-http-token`)

Steps:

1. From this worktree: start Compose (`task up` / worktree compose) so `attesta` + `living-http` are up.
2. Sign in as feragame administration; open Living SPA; start intake register; Commit a valid load.
3. Commit response should include two links: **Stampa etichette** (action link) and **Scarica CSV registrazione** (download; filename like `NIR-….csv`).
4. Click labels → browser opens `/api/living/artifacts/<32-hex>` as PDF (session required). CSV download keeps the disposition filename.
5. Automated coverage: `go test` in `services/living-http` + `server/internal/living` + `server/cmd/server` (`-run Living`); register Commit two-link/filename asserts already in `living_test.go` / `commit_http_test.go`.

## Refs

- Glossary: **HttpPort / onSubmit.http**, **Commit**, **Org Config**, **Domain**
- Prior: `.scratch/living-flat-port/issues/18-server-httpport-after-mutate.md`
- Code: `server/internal/living/{http_port,fake_http,submit_http,commit}.go`, `server/cmd/server/living.go`
- Fixture: `server/internal/living/fixtures/feragame-config.yaml` (`intake/register` `onSubmit.http`)
