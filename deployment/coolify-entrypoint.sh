#!/bin/sh
set -e

# shellcheck source=deployment/attesta-appwrite-ready.sh
. "$(dirname "$0")/attesta-appwrite-ready.sh"

if attesta_seed_enabled; then
  endpoint="${APPWRITE_ENDPOINT:-http://appwrite/v1}"
  wait_for_appwrite "${endpoint%/}/health"
  echo "running attesta seed (empty-only; ATTESTA_SEED=${ATTESTA_SEED})"
  ./attesta seed
fi

exec "$@"
