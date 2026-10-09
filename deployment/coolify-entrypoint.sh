#!/bin/sh
set -e

attesta_seed_enabled() {
  case "${ATTESTA_SEED:-false}" in
    1|true|TRUE|yes|YES|on|ON) return 0 ;;
    *) return 1 ;;
  esac
}

# Wait until Appwrite serves /v1/health (guest 401 or 200). Same readiness
# signal as scripts/worktree-compose.sh wait_for_appwrite.
wait_for_appwrite() {
  timeout="${ATTESTA_APPWRITE_READY_TIMEOUT_SECONDS:-120}"
  endpoint="${APPWRITE_ENDPOINT:-http://appwrite/v1}"
  url="${endpoint%/}/health"
  elapsed=0

  case "${timeout}" in
    ''|*[!0-9]*)
      echo "error: ATTESTA_APPWRITE_READY_TIMEOUT_SECONDS must be a non-negative integer" >&2
      return 1
      ;;
  esac

  # Overrides may set timeout 0 to skip the readiness probe.
  if [ "${timeout}" -eq 0 ]; then
    return 0
  fi

  command -v curl >/dev/null 2>&1 || {
    echo "error: curl is required to wait for Appwrite readiness" >&2
    return 1
  }

  echo "waiting for Appwrite at ${url}"
  while [ "${elapsed}" -lt "${timeout}" ]; do
    code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 "${url}" 2>/dev/null || true)"
    # Guests get 401 once Appwrite is serving; treat that as ready.
    if [ "${code}" = "200" ] || [ "${code}" = "401" ]; then
      return 0
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done

  echo "error: Appwrite did not become ready at ${url} within ${timeout}s" >&2
  return 1
}

if attesta_seed_enabled; then
  wait_for_appwrite
  echo "running attesta seed (empty-only; ATTESTA_SEED=${ATTESTA_SEED})"
  ./attesta seed
fi

exec "$@"
