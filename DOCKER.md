# Docker Compose Setup

This demo uses Docker Compose to run MongoDB and Cerbos (plus an optional Mongo Express UI).

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

## Quick Start
```bash
docker compose up -d
```

## Verifying the Setup
```bash
curl http://localhost:3592/_cerbos/health
```

## Stopping the Services
```bash
# Stop services but keep data
docker compose down

# Stop services and remove data
docker compose down -v
```
