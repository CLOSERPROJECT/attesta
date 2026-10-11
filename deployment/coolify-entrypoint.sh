#!/bin/sh
set -e

# shellcheck source=deployment/attesta-appwrite-ready.sh
. "$(dirname "$0")/attesta-appwrite-ready.sh"

load_appwrite_runtime_env() {
  env_file="${ATTESTA_APPWRITE_ENV_FILE:-/tmp/attesta-appwrite.env}"
  if [ -f "${env_file}" ]; then
    # Runtime file wins over Coolify-injected APPWRITE_API_KEY (often stale on empty Appwrite).
    set -a
    # shellcheck disable=SC1090
    . "${env_file}"
    set +a
  fi
}

if attesta_seed_enabled; then
  endpoint="${APPWRITE_ENDPOINT:-http://appwrite/v1}"
  wait_for_appwrite "${endpoint%/}/health"
  echo "running attesta bootstrap-appwrite (ATTESTA_SEED=${ATTESTA_SEED})"
  ./attesta bootstrap-appwrite
  load_appwrite_runtime_env
  echo "running attesta seed (empty-only; ATTESTA_SEED=${ATTESTA_SEED})"
  ./attesta seed
fi

load_appwrite_runtime_env
exec "$@"
