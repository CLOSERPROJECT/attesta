# Docker Compose Setup

This demo uses Docker Compose to run MongoDB and Cerbos (plus an optional Mongo Express UI).
All Docker-related files live under `deployment/`.

## Services

### MongoDB Community Edition
- **Image**: `mongo:7.0`
- **Port**: 27017
- **Auth**: Disabled for local demo simplicity

### Cerbos Policy Engine
- **Image**: `ghcr.io/cerbos/cerbos:0.39.0`
- **HTTP Port**: 3592
- **Config**: `cerbos/config/config.yaml`
- **Policies**: `cerbos/policies/*.yaml`

### Mongo Express (optional)
- **Image**: `mongo-express:1.0.2`
- **Port**: 8081

### Attesta app
- **Image**: Built from local `deployment/Dockerfile.local`
- **Port**: 3000
- **Mongo**: `mongodb://mongodb:27017`
- **Cerbos**: `http://cerbos:3592`

## Quick Start
```bash
docker compose -f deployment/docker-compose.local.yaml up -d
```

Open:
- App: http://localhost:3030

## Verifying the Setup
```bash
curl http://localhost:3592/_cerbos/health
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
