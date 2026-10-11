# Docker Compose Setup

This demo uses one Docker Compose project per checkout to run MongoDB, Cerbos, Appwrite, and optionally the containerized Attesta app.
All Docker-related files live under `deployment/`.

## Services

### MongoDB Community Edition
- **Image**: `mongo:7.0`
- **Port**: 27017
- **Auth**: Disabled for local demo simplicity

### Cerbos Policy Engine
- **Image**: `ghcr.io/cerbos/cerbos:0.50.0`
- **HTTP Port**: 3592
- **Config**: `cerbos/config/config.yaml`
- **Policies**: `cerbos/policies/*.yaml`

### Appwrite
- **Baseline**: `deployment/appwrite/docker-compose.appwrite.yaml`
- **Env template**: `deployment/appwrite/.env.appwrite.example`
- **HTTP entrypoint**: `http://localhost`
- **Purpose**: identity, teams, memberships, labels, and org asset storage

### Mongo Express (optional)
- **Image**: `mongo-express:1.0.2`
- **Port**: 8081

### Attesta app
- **Image**: Built from local `deployment/Dockerfile.local`
- **Container port**: 3000 (`DOCKER_APP_PORT` on the host when explicitly run)
- **Mongo**: `mongodb://mongodb:27017`
- **Cerbos**: `http://cerbos:3592`

## Quick Start
```bash
task start
```

`scripts/worktree-compose.sh` loads the generated or manually edited ports from `.env.worktree` and sets a Compose project name derived from the checkout's canonical path.
Compose therefore scopes container names, networks, and volumes to that worktree, including checkouts with the same directory name.
`task start` starts infrastructure only; `task dev` runs Attesta and Vite on the host.
`task start:docker` starts the full containerized stack and publishes Attesta on `DOCKER_APP_PORT`; use `task start:docker:build` to rebuild it first.
The wrapper checks Docker Compose 2.24.4+ before loading the local override.

The local Compose path applies `deployment/appwrite/docker-compose.worktree.yaml` as a local-only override to the vendored Appwrite baseline.
That override owns the worktree-specific ports, resource labels, scoped names, and fresh-stack seed hook.
The standalone operator baseline in `deployment/appwrite/docker-compose.appwrite.yaml` and the Coolify paths in `deployment/docker-compose.coolify.yaml` and `deployment/Dockerfile.coolify` remain unchanged and do not consume the local override.

Stacks created by the old direct command used `deployment` as their implicit Compose project.
New checkouts get a path-qualified Compose project name automatically.
To keep pre-worktree `deployment_*` volumes attached, adopt that name once on the primary:

```bash
docker compose --project-name deployment -f deployment/docker-compose.local.yaml down --remove-orphans
COMPOSE_PROJECT_NAME=deployment task worktree:bootstrap
task start
```

Never add `-v` to that migration command.
If `.env.worktree` already exists, add `COMPOSE_PROJECT_NAME=deployment`; persisted worktree identity wins over shell and `.env` values.

Parallel worktrees get isolated Compose projects and empty durable volumes.
Demo data comes from the Go seed after Appwrite is ready and credentials exist (`ATTESTA_SEED=true` / `task seed`).
Adopt `COMPOSE_PROJECT_NAME=deployment` on the primary once if you still need pre-worktree `deployment_*` volumes (see README); the next start relabels unlabeled legacy volumes in place.

Open:
- Run `bash scripts/worktree-env.sh print` to see the current checkout's URLs.
- `task dev` serves the app on `PORT` and Vite on `VITE_PORT`.
- `task start:docker` serves the containerized app on `DOCKER_APP_PORT`.

After the primary checkout's first boot:
1. Create the first Appwrite console account.
2. Create the Attesta Appwrite project.
3. Create an API key for Attesta.
4. Open Mailpit on `http://localhost:8025` to inspect invite, recovery, and affiliation emails.
5. Create the `org-assets` bucket.
6. Set `APPWRITE_PROJECT_ID` and `APPWRITE_API_KEY` for the Attesta service, then run `task start:docker` if using the containerized app.

Fresh stacks start with empty Appwrite MariaDB; demo data comes from the Go seed after credentials exist.

`task start` / `task start:docker` export `ATTESTA_SEED=true` from worktree-env and run `go run ./cmd/server prepare --policy=worktree` (wait → bootstrap → apply env → empty-only seed; soft-skips bootstrap failure and missing credentials). Bootstrap defaults console credentials to `console@attesta.local` / `password` when unset. Use `task seed` (optionally with `--replace`) for a manual run.

## Verifying the Setup
```bash
source <(bash scripts/worktree-env.sh export)
curl "http://localhost:${CERBOS_PORT}/_cerbos/health"
```

## Stopping the Services
```bash
# Remove only this worktree's containers and networks; keep its data
task stop

# Remove only this worktree's containers, networks, and data
task reset

# Same as reset after an interactive confirmation
task purge

# Preview orphaned Attesta Docker resources left by deleted worktrees
task worktree:gc

# Remove only the resources reported as orphaned
task worktree:gc:apply
```

Cleanup only considers resources carrying this clone's repository identity and the exact worktree owner.
Foreign-clone and unlabeled legacy resources are skipped.
`task reset` and `task purge` inspect every volume in the current Compose project and fail closed if any ownership label is missing or different.
That includes preserved volumes from the legacy `deployment` project until the next start copies them onto labeled replacements (Docker cannot add labels to an existing volume).
`task start` / `task dev` run that copy automatically for unlabeled legacy volumes in the current Compose project.
To discard legacy data instead, inspect with `docker volume ls --filter label=com.docker.compose.project=deployment` and remove only the confirmed legacy volume names; the next start creates labeled replacements.

## Coolify
Use `deployment/Dockerfile.coolify` with the Coolify proxy (no `ports:` in
`deployment/docker-compose.coolify.yaml`). The Coolify stack will include the
vendored Appwrite module instead of assuming an external identity deployment.

Coolify-specific notes:
- Coolify owns public routing (no Traefik in this compose). Expose Appwrite with Coolify `SERVICE_FQDN_APPWRITE` / `SERVICE_URL_*` on the Appwrite services.
- Attesta's compose default reaches Appwrite at `https://${SERVICE_FQDN_APPWRITE}/v1` (public FQDN). Prefer that when the browser and server must share the same Appwrite origin.
- The `appwrite` healthcheck sends `Host: $_APP_DOMAIN` so preview stacks (where bare `localhost` returns 404) still report healthy.
- MariaDB is stock `mariadb:10.11` via a passthrough `deployment/Dockerfile.appwrite-mariadb` (Coolify preview build compatibility; no SQL seed). Demo identity data comes from the Go seed after console bootstrap.

### Coolify Preview env checklist
Set these on the `attesta` service for automatic Appwrite project + demo fixture:

| Variable | Preview | Notes |
|---|---|---|
| `ATTESTA_SEED` | `true` | Leave unset/`false` in production |
| `APPWRITE_PROJECT_ID` | e.g. `attesta` | Required; bootstrap creates this project id |
| `APPWRITE_API_KEY` | optional | Coolify may inject a stale key on empty Appwrite; ignore it — `attesta prepare` bootstraps (probes/regenerates when needed) and writes `/tmp/attesta-appwrite.env`; prepare applies it for seed, and the server loads the same file at startup |
| `APPWRITE_CONSOLE_EMAIL` | default `console@attesta.local` | Console login-first; signup only if login fails (409/501 treated as account may exist) |
| `APPWRITE_CONSOLE_PASSWORD` | default `password` | Demo-only; change if you share console access |
| `APPWRITE_ORG_ASSETS_BUCKET` | default `org-assets` | Created when missing |
| `APPWRITE_INVITE_REDIRECT_URL` | preview app URL + `/invite/accept` | Required for invite emails |
| `APPWRITE_RESET_REDIRECT_URL` | preview app URL + `/reset/confirm` | Required for recovery emails |
| `APPWRITE_VERIFY_REDIRECT_URL` | preview app URL + `/verify/confirm` | Required for verification emails |

When `ATTESTA_SEED` is truthy, the Coolify entrypoint runs `./attesta prepare` (default `--policy=coolify`: hard-fail wait for Appwrite `/v1/health` guest 401 or 200 → bootstrap → apply runtime env → empty-only seed) then `exec`s the server. The server loads `/tmp/attesta-appwrite.env` at startup so bootstrap credentials win over stale injected `APPWRITE_API_KEY`. When `ATTESTA_SEED` is unset/false, prepare is a no-op. Manual re-seed is `attesta seed` / `task seed` (optionally `--replace`).

## Ephemeral previews (PRs)
Build with `deployment/Dockerfile.ephemeral`. On startup it will run a seed
script if provided via `EPHEMERAL_SEED_SCRIPT` (path inside the container) or
`/app/seed/seed.sh`. This is intended for anonymized copy/restore flows.
