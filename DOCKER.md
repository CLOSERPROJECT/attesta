# Docker Compose Setup

This directory contains the Docker Compose configuration for the Attesta project.

## Services

### MongoDB Community Edition
- **Image**: `mongo:7.0`
- **Port**: 27017
- **Credentials**: Configured via environment variables (see `.env.example`)
  - Default username: `admin`
  - Default password: `change-this-password`
  - Default database: `attesta`

### Cerbos Policy Engine
- **Image**: `ghcr.io/cerbos/cerbos:0.39.0`
- **HTTP Port**: 3592
- **gRPC Port**: 3593
- **Admin Credentials**:
  - Username: `cerbos`
  - Password: `cerbosAdmin` (configured in `cerbos/config/config.yaml`)

## Quick Start

1. Copy the example environment file and customize it:
   ```bash
   cp .env.example .env
   # Edit .env and set secure passwords
   ```

2. Start all services:
   ```bash
   docker compose up -d
   ```

2. Check service status:
   ```bash
   docker compose ps
   ```

3. View logs:
   ```bash
   docker compose logs -f
   ```

4. Stop all services:
   ```bash
   docker compose down
   ```

5. Stop and remove all data:
   ```bash
   docker compose down -v
   ```

## Configuration

### MongoDB
MongoDB data is persisted in Docker volumes:
- `mongodb_data`: Database files
- `mongodb_config`: Configuration files

### Cerbos
Cerbos configuration and policies are stored in:
- `./cerbos/config/config.yaml`: Main configuration file
- `./cerbos/policies/`: Policy files directory

## Network

Both services are connected via the `attesta-network` bridge network, allowing them to communicate with each other using their service names as hostnames.

## Health Checks

Both services include health checks:
- MongoDB: Checks database connectivity using mongosh
- Cerbos: Checks HTTP health endpoint

## Accessing Services

- **MongoDB**: `mongodb://${MONGO_USERNAME}:${MONGO_PASSWORD}@localhost:27017/${MONGO_DATABASE}`
- **Cerbos HTTP API**: `http://localhost:3592`
- **Cerbos gRPC API**: `localhost:3593`
- **Cerbos Admin API**: `http://localhost:3592/_cerbos/admin`

## Security Notes

- **Never commit `.env` files** to version control
- Change default passwords in `.env` before deploying to production
- Consider using Docker secrets or external secret management for production deployments
- Cerbos admin password hash should be updated in `cerbos/config/config.yaml` or configured via environment variables
