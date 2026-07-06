#!/bin/sh
set -e

ROOT_DIR="$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deployment/docker-compose.local.yaml"
SEED_SQL="$ROOT_DIR/deployment/appwrite/appwrite-seed.sql"
INIT_SCRIPT="$ROOT_DIR/deployment/appwrite/mariadb-init-preview-seed.sh"

DB_ROOT_PASSWORD="${_APP_DB_ROOT_PASS:-rootsecretpassword}"
DB_NAME="${_APP_DB_SCHEMA:-appwrite}"
MARIADB_CONTAINER="${MARIADB_CONTAINER:-appwrite-mariadb}"

if [ ! -f "$SEED_SQL" ]; then
  echo "seed SQL not found: $SEED_SQL" >&2
  exit 1
fi

echo "starting Appwrite MariaDB (other services may already be running)"
docker compose -f "$COMPOSE_FILE" up -d mariadb

echo "waiting for MariaDB to accept connections"
for _ in $(seq 1 60); do
  if docker exec "$MARIADB_CONTAINER" healthcheck.sh --connect --innodb_initialized >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

if ! docker exec "$MARIADB_CONTAINER" healthcheck.sh --connect --innodb_initialized >/dev/null 2>&1; then
  echo "MariaDB did not become ready in time" >&2
  exit 1
fi

echo "restoring preview seed into MariaDB (this replaces the Appwrite database)"
docker exec -i "$MARIADB_CONTAINER" mariadb \
  --user=root \
  --password="$DB_ROOT_PASSWORD" \
  "$DB_NAME" < "$SEED_SQL"

echo "cleaning Appwrite preview runtime state"
docker exec -i "$MARIADB_CONTAINER" mariadb \
  --user=root \
  --password="$DB_ROOT_PASSWORD" \
  "$DB_NAME" <<'SQL'
DELETE FROM _console_certificates;
DELETE FROM _console_rules;
DELETE FROM _console_sessions;
DELETE FROM _1_sessions;
SQL

echo "restarting Appwrite services to pick up restored data"
docker compose -f "$COMPOSE_FILE" restart appwrite appwrite-worker-mails appwrite-worker-messaging appwrite-realtime 2>/dev/null \
  || docker compose -f "$COMPOSE_FILE" restart appwrite

echo "seed restore complete"
