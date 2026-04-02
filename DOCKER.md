# Docker Compose Setup

This demo uses Docker Compose to run MongoDB, Cerbos, Appwrite, and Attesta.
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
- **Port**: 3000
- **Mongo**: `mongodb://mongodb:27017`
- **Cerbos**: `http://cerbos:3592`

## Quick Start
```bash
docker compose -f deployment/docker-compose.local.yaml up -d
```

Open:
- Appwrite Console: http://localhost
- App: http://localhost:3030
- Mailpit: http://localhost:8025

After first boot:
1. Create the first Appwrite console account.
2. Create the Attesta Appwrite project.
3. Create an API key for Attesta.
4. Open Mailpit on `http://localhost:8025` to inspect invite and recovery emails.
5. Create the `org-assets` bucket.
6. Set `APPWRITE_PROJECT_ID` and `APPWRITE_API_KEY` for the Attesta service, then restart Attesta.

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
