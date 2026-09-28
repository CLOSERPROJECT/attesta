# Docker Compose Setup

This demo uses one Docker Compose project per checkout to run MongoDB, Cerbos,
Appwrite, and optionally the containerized Attesta app.
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

`scripts/worktree-compose.sh` loads the generated or manually edited ports from
`.env.worktree` and sets a Compose project name derived from the checkout's
canonical path. Compose therefore scopes container names, networks, and
volumes to that worktree, including checkouts with the same directory name.
`task start` starts infrastructure only; `task dev` runs Attesta and Vite on
the host.

The local Compose path applies
`deployment/appwrite/docker-compose.worktree.yaml` as a local-only override to
the vendored Appwrite baseline. That override owns the worktree-specific ports,
resource labels, scoped names, and fresh-stack seed hook. The standalone
operator baseline in `deployment/appwrite/docker-compose.appwrite.yaml` and the
Coolify paths in `deployment/docker-compose.coolify.yaml` and
`deployment/Dockerfile.coolify` remain unchanged and do not consume the local
override.

Stacks created before canonical-path project names used the directory basename
as their Compose project. To reattach those existing volumes, run
`task worktree:bootstrap`, set the legacy name (for example
`COMPOSE_PROJECT_NAME=attesta`) in `.env.worktree`, and then run `task start`.

For a parallel worktree, bootstrap temporarily quiesces the primary Compose
project and captures one all-or-nothing bundle containing MongoDB, Appwrite
MariaDB, and Appwrite's durable file volumes. It then restores the primary's
previous running state. The bundle is restored only before an uninitialized
worktree stack starts; subsequent starts preserve the worktree's own data.
Partial or corrupt bundles stop startup instead of mixing databases from
different points in time. If the primary has no durable volumes yet, bootstrap
skips the bundle and the new stack initializes from the checked-in seed.

Open:
- Run `bash scripts/worktree-env.sh print` to see the current checkout's URLs.
- `task dev` serves the app on `PORT` and Vite on `VITE_PORT`.

After the primary checkout's first boot:
1. Create the first Appwrite console account.
2. Create the Attesta Appwrite project.
3. Create an API key for Attesta.
4. Open Mailpit on `http://localhost:8025` to inspect invite, recovery, and affiliation emails.
5. Create the `org-assets` bucket.
6. Set `APPWRITE_PROJECT_ID` and `APPWRITE_API_KEY` for the Attesta service, then restart Attesta.

Fresh stacks without a copied data bundle restore the checked-in local seed on
first initialization. Copied bundles retain the matching project, users,
teams, memberships, API keys, and assets while discarding host-specific
sessions, certificates, caches, and routing state.

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

# Also discard the saved coordinated snapshot bundle
task purge

# Preview orphaned Attesta Docker resources left by deleted worktrees
task worktree:gc

# Remove only the resources reported as orphaned
task worktree:gc:apply
```

## Coolify
Use `deployment/Dockerfile.coolify` with the Coolify proxy (no `ports:` in
`deployment/docker-compose.coolify.yaml`). The Coolify stack will include the
vendored Appwrite module instead of assuming an external identity deployment.

Coolify-specific notes:
- `traefik` from the vendored Appwrite stack is still included, but only as an internal router for `/`, `/console`, and `/v1/realtime`.
- Set `SERVICE_FQDN_APPWRITE_80` on the `traefik` service if you want a public Appwrite hostname in Coolify.
- Set `APPWRITE_PROJECT_ID`, `APPWRITE_API_KEY`, `APPWRITE_INVITE_REDIRECT_URL`, and `APPWRITE_RESET_REDIRECT_URL` on the `attesta` service before testing auth flows.
- Attesta reaches Appwrite internally through `http://${SERVICE_NAME_APPWRITE:-appwrite}:80/v1`.
- To restore `deployment/appwrite/appwrite-seed.sql` only in preview deployments, set `APPWRITE_RESTORE_SEED_SQL=true` in Coolify's Preview Deployment environment variables and leave it unset in production. The SQL and init hook are baked into the custom MariaDB image, the import runs only on first MariaDB initialization, and the init hook clears Appwrite runtime tables such as sessions, certificates, and domain rules so each preview can recreate host-specific state cleanly.

## Ephemeral previews (PRs)
Build with `deployment/Dockerfile.ephemeral`. On startup it will run a seed
script if provided via `EPHEMERAL_SEED_SCRIPT` (path inside the container) or
`/app/seed/seed.sh`. This is intended for anonymized copy/restore flows.
