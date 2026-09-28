#!/bin/sh
set -e

ROOT_DIR="$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deployment/docker-compose.local.yaml"
SEED_SQL="$ROOT_DIR/deployment/appwrite/appwrite-seed.sql"

cd "$ROOT_DIR"
if [ -f "$ROOT_DIR/.env" ]; then
  set -a
  # shellcheck disable=SC1091
  . "$ROOT_DIR/.env"
  set +a
fi
eval "$(bash "$ROOT_DIR/scripts/worktree-env.sh" export)"
ATTESTA_WORKTREE_ROOT="$(CDPATH= cd -- "$ROOT_DIR" && pwd -P)"
export COMPOSE_PROJECT_NAME APPWRITE_RESTORE_SEED_SQL ATTESTA_WORKTREE_ROOT

DB_ROOT_PASSWORD="${_APP_DB_ROOT_PASS:-rootsecretpassword}"
DB_NAME="${_APP_DB_SCHEMA:-appwrite}"

compose() {
  docker compose \
    --project-name "$COMPOSE_PROJECT_NAME" \
    -f "$COMPOSE_FILE" \
    "$@"
}

if [ ! -f "$SEED_SQL" ]; then
  echo "seed SQL not found: $SEED_SQL" >&2
  exit 1
fi

echo "starting Appwrite MariaDB (other services may already be running)"
compose up -d mariadb

echo "waiting for MariaDB to accept connections"
for _ in $(seq 1 60); do
  if compose exec -T mariadb healthcheck.sh --connect --innodb_initialized >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

if ! compose exec -T mariadb healthcheck.sh --connect --innodb_initialized >/dev/null 2>&1; then
  echo "MariaDB did not become ready in time" >&2
  exit 1
fi

echo "restoring preview seed into MariaDB (this replaces the Appwrite database)"
compose exec -T mariadb mariadb \
  --user=root \
  --password="$DB_ROOT_PASSWORD" \
  "$DB_NAME" < "$SEED_SQL"

echo "cleaning Appwrite preview runtime state"
compose exec -T mariadb mariadb \
  --user=root \
  --password="$DB_ROOT_PASSWORD" \
  "$DB_NAME" <<'SQL'
DELETE FROM _console_certificates;
DELETE FROM _console_rules;
DELETE FROM _console_sessions;
DELETE FROM _1_sessions;
SQL

echo "restarting Appwrite services to pick up restored data"
compose restart appwrite appwrite-worker-mails appwrite-worker-messaging appwrite-realtime 2>/dev/null \
  || compose restart appwrite

echo "seed restore complete"
