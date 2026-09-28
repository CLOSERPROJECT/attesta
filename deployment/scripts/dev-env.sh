#!/usr/bin/env bash
# Source from Taskfile host-dev tasks (do not execute as main).
# Usage from repo root:
#   set -a
#   # shellcheck disable=SC1091
#   source deployment/scripts/dev-env.sh
#   set +a
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
# If sourced from a worktree Taskfile, prefer git toplevel as repo root
if git rev-parse --show-toplevel >/dev/null 2>&1; then
  REPO_ROOT="$(git rev-parse --show-toplevel)"
fi
# shellcheck source=deployment/scripts/dev-lib.sh
source "$SCRIPT_DIR/dev-lib.sh"

if ! is_primary_worktree "$REPO_ROOT" && [[ ! -f "$REPO_ROOT/.env.worktree" ]]; then
  echo "error: ${REPO_ROOT}/.env.worktree is missing" >&2
  echo "run: bash scripts/worktree-bootstrap.sh" >&2
  return 1 2>/dev/null || exit 1
fi

_shell_has_port=0
_shell_has_vite=0
[[ -n "${PORT+x}" ]] && _shell_has_port=1 && _shell_port="$PORT"
[[ -n "${VITE_PORT+x}" ]] && _shell_has_vite=1 && _shell_vite="$VITE_PORT"
unset PORT VITE_PORT 2>/dev/null || true
load_env_file "$REPO_ROOT/.env"
eval "$(bash "$REPO_ROOT/scripts/worktree-env.sh" export)"
[[ "$_shell_has_port" -eq 1 ]] && export PORT="$_shell_port"
[[ "$_shell_has_vite" -eq 1 ]] && export VITE_PORT="$_shell_vite"
export PORT VITE_PORT
export APPWRITE_INVITE_REDIRECT_URL="http://localhost:${PORT}/invite/accept"
export APPWRITE_RESET_REDIRECT_URL="http://localhost:${PORT}/reset/confirm"
export PUBLIC_BASE_URL="http://localhost:${PORT}"
export VITE_DEV_SERVER="http://localhost:${VITE_PORT}"

echo "Attesta http://localhost:${PORT} (vite :${VITE_PORT})"
