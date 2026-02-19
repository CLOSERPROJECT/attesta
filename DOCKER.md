# Docker Compose Setup

This demo uses Docker Compose to run MongoDB, Cerbos, Appwrite, and Mailpit (plus an optional Mongo Express UI).
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

### Appwrite (optional for team-based policy input)
- **Image**: `appwrite/appwrite:1.8.0`
- **HTTP Port**: not published by default in local compose
- **Dependencies**: MariaDB + Redis
- **Mail**: routed to Mailpit (`mailpit:1025`)

### Mailpit (optional)
- **Image**: `axllent/mailpit:latest`
- **SMTP Port (internal)**: 1025
- **Web UI**: `${MAILPIT_UI_PORT:-18025}`

### Mongo Express (optional)
- **Image**: `mongo-express:1.0.2`
- **Port**: 8081

### Attesta app
- **Image**: Built from local `deployment/Dockerfile.local`
- **Port**: 3000
- **Mongo**: `mongodb://mongodb:27017`
- **Cerbos**: `http://cerbos:3592`
- **Appwrite (optional)**: `APPWRITE_ENDPOINT`, `APPWRITE_PROJECT_ID`, `APPWRITE_API_KEY`

## Quick Start
```bash
docker compose -f deployment/docker-compose.local.yaml up -d
```

Optional: expose Appwrite console on host:
```bash
docker compose -f deployment/docker-compose.local.yaml -f deployment/docker-compose.local.appwrite-exposed.yaml up -d
# http://localhost:19080/console
```

Open:
- App: http://localhost:3030
- Mailpit: http://localhost:18025

## Verifying the Setup
```bash
curl http://localhost:3592/_cerbos/health
```

If you want Appwrite team membership included in Cerbos checks, configure:
- `APPWRITE_ENDPOINT` (for local Compose use `http://appwrite/v1` inside `attesta`)
- `APPWRITE_PROJECT_ID`
- `APPWRITE_API_KEY` (server key with permission to read user memberships)
- `APPWRITE_SYNC_FROM_WORKFLOW=true` (optional startup sync from workflow YAML)
- `APPWRITE_SYNC_DEFAULT_PASSWORD` (used when creating users)
- `APPWRITE_SYNC_MEMBERSHIP_URL` (callback URL used when creating memberships)

Debug endpoint for effective actor/team payload sent to Cerbos:
```bash
curl http://localhost:3030/w/workflow/debug/teams
```

## Stopping the Services
```bash
# Stop services but keep data
docker compose -f deployment/docker-compose.local.yaml down

# Stop services and remove data
docker compose -f deployment/docker-compose.local.yaml down -v
```

## Coolify
Use `deployment/Dockerfile.coolify` with the Coolify proxy (no `ports:` in
`deployment/docker-compose.coolify.yaml`).

## Ephemeral previews (PRs)
Build with `deployment/Dockerfile.ephemeral`. On startup it will run a seed
script if provided via `EPHEMERAL_SEED_SCRIPT` (path inside the container) or
`/app/seed/seed.sh`. This is intended for anonymized copy/restore flows.
