#!/bin/sh
set -e

if [ -n "${EPHEMERAL_SEED_SCRIPT:-}" ] && [ -f "$EPHEMERAL_SEED_SCRIPT" ]; then
  echo "running seed script: $EPHEMERAL_SEED_SCRIPT"
  sh "$EPHEMERAL_SEED_SCRIPT"
elif [ -f "/app/seed/seed.sh" ]; then
  echo "running seed script: /app/seed/seed.sh"
  sh "/app/seed/seed.sh"
fi

exec "$@"
