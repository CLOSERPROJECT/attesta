#!/usr/bin/env bash
# Bring up shared local infra from the primary worktree root.
# Usage: infra-up.sh [--build]
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=deployment/scripts/dev-lib.sh
source "$SCRIPT_DIR/dev-lib.sh"

build=0
if [[ "${1:-}" == "--build" ]]; then
  build=1
fi

container_running() {
  local name="$1"
  [[ "$(docker inspect -f '{{.State.Running}}' "$name" 2>/dev/null || echo false)" == "true" ]]
}

if container_running closer-mongodb && container_running attesta-cerbos && container_running appwrite; then
  echo "ok: infra already up (closer-mongodb, attesta-cerbos, appwrite)"
  exit 0
fi

primary="$(primary_worktree_root)"
if [[ -z "$primary" || ! -d "$primary" ]]; then
  echo "error: could not resolve primary worktree root" >&2
  exit 1
fi

compose=(docker compose -f deployment/docker-compose.local.yaml)
cd "$primary"
if [[ "$build" -eq 1 ]]; then
  "${compose[@]}" up -d --build
else
  "${compose[@]}" up -d
fi
"${compose[@]}" ps
