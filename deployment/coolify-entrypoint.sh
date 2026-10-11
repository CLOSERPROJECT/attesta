#!/bin/sh
set -e

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

# Hard-fail prepare: wait → bootstrap → apply env → seed (see ADR-0004).
./attesta prepare

# Ensure the long-running server inherits bootstrap credentials.
load_appwrite_runtime_env
exec "$@"
