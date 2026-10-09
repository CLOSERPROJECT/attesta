#!/bin/sh
# Shared Appwrite readiness + ATTESTA_SEED gate helpers. Source only; do not execute.
# POSIX sh so Coolify entrypoint and worktree-compose can both source this.

attesta_seed_enabled() {
  case "${ATTESTA_SEED:-false}" in
    1|true|TRUE|yes|YES|on|ON) return 0 ;;
    *) return 1 ;;
  esac
}

# wait_for_appwrite <health-url>
# Polls until HTTP 200 or 401, or until ATTESTA_APPWRITE_READY_TIMEOUT_SECONDS
# (default 120). Timeout 0 skips the probe.
wait_for_appwrite() {
  url="${1:-}"
  timeout="${ATTESTA_APPWRITE_READY_TIMEOUT_SECONDS:-120}"
  elapsed=0

  if [ -z "${url}" ]; then
    echo "error: wait_for_appwrite requires a health URL" >&2
    return 1
  fi

  case "${timeout}" in
    ''|*[!0-9]*)
      echo "error: ATTESTA_APPWRITE_READY_TIMEOUT_SECONDS must be a non-negative integer" >&2
      return 1
      ;;
  esac

  # Callers (tests, overrides) may set timeout 0 to skip the readiness probe.
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
