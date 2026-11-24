# Docker Compose and Coolify

Compose and Coolify/deploy reference for Attesta. For day-1 local setup (clone, `task start`, Appwrite prepare/seed, host Go + Vite), use [QUICKSTART.md](QUICKSTART.md). Product overview and host env vars live in [README.md](README.md) and `.env.example`. Worktree install hooks (`wt`, Codex, Cursor) are summarized in QUICKSTART/README; this file covers Docker-scoped mechanics only.

All Docker-related files live under `deployment/`. Scripts under `scripts/` remain the source of truth; this doc caches layout and gotchas.

## Services

Local stack: `deployment/docker-compose.local.yaml` (includes the Appwrite baseline + worktree override). Host ports come from `.env.worktree` (defaults shown below).

### MongoDB

- **Image**: `mongo:7.0`
- **Host port**: `${MONGODB_PORT:-27017}` → 27017
- **Auth**: disabled for local demo

### Cerbos

- **Image**: `ghcr.io/cerbos/cerbos:0.50.0`
- **Host port**: `${CERBOS_PORT:-3592}` → 3592
- **Config**: `cerbos/config/config.yaml`
- **Policies**: `cerbos/policies/*.yaml`

### Appwrite

- **Baseline**: `deployment/appwrite/docker-compose.appwrite.yaml`
- **Local override**: `deployment/appwrite/docker-compose.worktree.yaml`
- **Env template**: `deployment/appwrite/.env.appwrite.example`
- **Purpose**: identity, teams, memberships, labels, and org asset storage

### Mongo Express (optional)

- **Image**: `mongo-express:1.0.2`
- **Host port**: `${MONGO_EXPRESS_PORT:-8081}` → 8081

### Attesta app

- **Local image**: build from `deployment/Dockerfile.local`
- **Container port**: 3000 (`${DOCKER_APP_PORT:-3030}` on the host when run via `task start:docker`)
- **Mongo**: `mongodb://mongodb:27017`
- **Cerbos**: `http://cerbos:3592`

Also in the local compose file (not the Coolify focus of this doc): Mailpit (`axllent/mailpit:v1.29.3`) and the plant `org-http` peer (`deployment/Dockerfile.org-http`).

## Layout under `deployment/`

| Path | Role |
| --- | --- |
| `docker-compose.local.yaml` | Local Compose entry (infra + optional Attesta) |
| `appwrite/docker-compose.appwrite.yaml` | Vendored Appwrite baseline |
| `appwrite/docker-compose.worktree.yaml` | Local-only override: worktree ports, resource labels, scoped names |
| `appwrite/.env.appwrite.example` | Appwrite env template for local include |
| `Dockerfile.local` | Local Attesta image |
| `Dockerfile.org-http` | Plant HttpPort peer |
| `Dockerfile.cerbos` | Cerbos image used by Coolify |
| `docker-compose.coolify.yaml` | Coolify stack (no local Appwrite override; no Traefik `ports:`) |
| `Dockerfile.coolify` | Coolify Attesta image |
| `coolify-entrypoint.sh` | Coolify entrypoint (`prepare` then server when seeding) |

## Worktree Compose

`scripts/worktree-compose.sh` is the Compose entrypoint for one isolated stack per checkout. Task targets (`task start`, `start:docker`, `stop`, `reset`, …) call it.

Mechanics:

- Requires Docker Compose **2.24.4+** (local Appwrite override uses `!override` / `!reset`).
- Loads ports from `.env.worktree` (generated once; later manual edits are preserved) via `scripts/worktree-env.sh`.
- Sets `COMPOSE_PROJECT_NAME` from the checkout’s canonical path so containers, networks, and volumes never collide across primary, linked, or independent clones—even when directory names match.
- Local path applies `docker-compose.worktree.yaml` on top of the Appwrite baseline. Coolify (`docker-compose.coolify.yaml` / `Dockerfile.coolify`) does **not** consume that override.
- `task start` / `worktree-compose.sh up` starts infrastructure only; `task start:docker` / `up-app` also runs the containerized Attesta service. Host development uses `task dev` (see QUICKSTART).

Demo data is not a volume copy between worktrees. Empty stacks get demo fixtures from the Go prepare/seed path documented in [QUICKSTART.md](QUICKSTART.md).

## Verify setup

```bash
source <(bash scripts/worktree-env.sh export)
curl "http://localhost:${CERBOS_PORT}/_cerbos/health"
```

`bash scripts/worktree-env.sh print` lists the current checkout’s ports and URLs.

## Lifecycle

```bash
# Remove only this worktree's containers and networks; keep its data
task stop

# Remove only this worktree's containers, networks, and volumes
task reset

# Same as reset after an interactive confirmation
task purge

# Preview orphaned Attesta Docker resources left by deleted worktrees
task worktree:gc

# Remove only the resources reported as orphaned
task worktree:gc:apply
```

Cleanup only considers resources carrying this clone’s repository identity and the exact worktree owner. Foreign-clone and unlabeled resources are skipped. `task reset` and `task purge` inspect every volume in the current Compose project and fail closed if any ownership label is missing or different.

## Coolify

Use `deployment/Dockerfile.coolify` with the Coolify proxy (no `ports:` in `deployment/docker-compose.coolify.yaml`). The Coolify stack includes the vendored Appwrite module instead of assuming an external identity deployment.

Coolify-specific notes:

- Coolify owns public routing (no Traefik in this compose). Expose Appwrite with Coolify `SERVICE_FQDN_APPWRITE` / `SERVICE_URL_*` on the Appwrite services.
- Attesta’s compose default reaches Appwrite at `https://${SERVICE_FQDN_APPWRITE}/v1` (public FQDN). Prefer that when the browser and server must share the same Appwrite origin.
- The `appwrite` healthcheck sends `Host: $_APP_DOMAIN` so preview stacks (where bare `localhost` returns 404) still report healthy.
- MariaDB uses stock `image: mariadb:10.11` (same as the local Appwrite baseline). Demo identity data comes from the Go seed after console bootstrap.
- `/api/org/events` is a long-lived Server-Sent Events response for Org Domain revision invalidation ([ADR-0015](docs/adr/0015-org-domain-change-feed.md)): any proxy in front of Attesta must not buffer it (the server sends `X-Accel-Buffering: no` and a keepalive comment every 25 seconds). Live board refresh assumes a single Attesta container.

### Coolify Preview env checklist

Set these on the `attesta` service for automatic Appwrite project + demo fixture:

| Variable | Preview | Notes |
| --- | --- | --- |
| `ATTESTA_SEED` | `true` | Leave unset/`false` in production. Org Domain + Active Config fixture come only from this Go seed (`attesta seed` / `task seed`), not product boot |
| `ATTESTA_ROOT_DIR` | `/app` | App root for `.env.appwrite` (Dockerfile/entrypoint set this; matches image app root, not `WORKDIR /app/server`) |
| `APPWRITE_PROJECT_ID` | `attesta` | Required; bootstrap creates this project id. Do **not** reuse production’s id on Preview |
| `APPWRITE_API_KEY` | optional / empty | Stale Coolify keys are fine — `attesta prepare` bootstraps and writes `$ATTESTA_ROOT_DIR/.env.appwrite`; `main` loads it at startup |
| `APPWRITE_CONSOLE_EMAIL` | optional | Go default `admin@appwrite.local` when unset |
| `APPWRITE_CONSOLE_PASSWORD` | optional | Go default `admin@appwrite.local` when unset |
| `APPWRITE_ORG_ASSETS_BUCKET` | optional | Go default `org-assets` when unset |
| `TURNSTILE_SITE_KEY` / `TURNSTILE_SECRET_KEY` | required | Set in Coolify (Preview + Production as appropriate) |
| `PUBLIC_BASE_URL` | `https://${SERVICE_FQDN_ATTESTA_3000}` | Compose default; invite/reset/verify links derive from this origin + fixed paths |

Attesta service env vars that must differ per Preview use bare `${VAR}` in compose (no `:-` / `:?`). Coolify inlines self-referential `${VAR:-default}` from Production into deployable compose, so Preview UI values never win. After compose/env changes: **Reload Compose**, confirm deployable still shows `${VAR}` (not literals), then redeploy.

When `ATTESTA_SEED` is truthy, the Coolify entrypoint runs `./attesta prepare` (default `--policy=coolify`: hard-fail wait for Appwrite `/v1/health` guest 401 or 200 → bootstrap → empty-only seed by value) then `exec`s the server. Prepare writes `$ATTESTA_ROOT_DIR/.env.appwrite`; the server loads it at startup so bootstrap credentials win over stale injected `APPWRITE_API_KEY`. When `ATTESTA_SEED` is unset/false, prepare is a no-op. Manual re-seed is `attesta seed` / `task seed` (optionally `--replace`). Local prepare/seed sequencing for worktrees is in [QUICKSTART.md](QUICKSTART.md).
