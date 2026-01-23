# Quick Start Guide

This guide will help you get started with the Attesta Docker Compose setup.

## Prerequisites

- Docker installed (version 20.10 or higher)
- Docker Compose installed (version 2.0 or higher)

## Starting the Services

1. Clone the repository (if you haven't already):
   ```bash
   git clone https://github.com/CLOSERPROJECT/attesta.git
   cd attesta
   ```

2. Configure environment variables:
   ```bash
   cp .env.example .env
   # Edit .env and set secure passwords
   ```

3. Start all services:
   ```bash
   docker compose up -d
   ```

3. Verify services are running:
   ```bash
   docker compose ps
   ```
   
   You should see both `attesta-mongodb` and `attesta-cerbos` with status "Up" and "healthy".

## Verifying the Setup

### Test MongoDB Connection

```bash
# Using mongosh from the MongoDB container
docker exec attesta-mongodb mongosh --eval "db.adminCommand('ping')"

# Or connect with credentials (use your .env values)
docker exec attesta-mongodb mongosh -u admin -p change-this-password --authenticationDatabase admin
```

### Test Cerbos API

```bash
# Check Cerbos health
curl http://localhost:3592/_cerbos/health

# List policies (requires authentication)
curl -u cerbos:cerbosAdmin http://localhost:3592/_cerbos/admin/policy
```

## Example Usage

### Checking Permissions with Cerbos

Here's an example of checking if a user can perform an action:

```bash
curl -X POST http://localhost:3592/api/check/resources \
  -H "Content-Type: application/json" \
  -d '{
    "principal": {
      "id": "user123",
      "roles": ["editor"]
    },
    "resource": {
      "kind": "document",
      "id": "doc1"
    },
    "actions": ["read", "update", "delete"]
  }'
```

## Stopping the Services

```bash
# Stop services but keep data
docker compose down

# Stop services and remove all data
docker compose down -v
```

## Customizing the Setup

### Changing MongoDB Credentials

1. Edit the `.env` file and update:
   ```
   MONGO_USERNAME=your_username
   MONGO_PASSWORD=your_secure_password
   MONGO_DATABASE=your_database
   ```

2. Restart services:
   ```bash
   docker compose down
   docker compose up -d
   ```

### Adding Cerbos Policies

1. Create a new policy file in `cerbos/policies/`
2. Follow the Cerbos policy format (see `document_policy.yaml` for an example)
3. Restart Cerbos: `docker compose restart cerbos`

## Troubleshooting

View logs for debugging:
```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f mongodb
docker compose logs -f cerbos
```

## Next Steps

- Read the full documentation in [DOCKER.md](DOCKER.md)
- Learn more about [Cerbos policies](https://docs.cerbos.dev/cerbos/latest/policies/)
- Explore [MongoDB documentation](https://docs.mongodb.com/)
