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

bash "$SCRIPT_DIR/worktree-ports.sh" "$REPO_ROOT" >/dev/null

_shell_has_port=0
_shell_has_vite=0
[[ -n "${PORT+x}" ]] && _shell_has_port=1 && _shell_port="$PORT"
[[ -n "${VITE_PORT+x}" ]] && _shell_has_vite=1 && _shell_vite="$VITE_PORT"
unset PORT VITE_PORT 2>/dev/null || true
load_env_file "$REPO_ROOT/.env"
load_env_file "$REPO_ROOT/.env.local"
# .env.local wins over .env for ports (unless shell pre-set)
if [[ -f "$REPO_ROOT/.env.local" ]]; then
  _local_port="$(read_env_key "$REPO_ROOT/.env.local" "PORT")"
  _local_vite="$(read_env_key "$REPO_ROOT/.env.local" "VITE_PORT")"
  [[ -n "$_local_port" ]] && export PORT="$_local_port"
  [[ -n "$_local_vite" ]] && export VITE_PORT="$_local_vite"
fi
[[ "$_shell_has_port" -eq 1 ]] && export PORT="$_shell_port"
[[ "$_shell_has_vite" -eq 1 ]] && export VITE_PORT="$_shell_vite"
PORT="${PORT:-3000}"
VITE_PORT="${VITE_PORT:-5173}"
export PORT VITE_PORT
export APPWRITE_INVITE_REDIRECT_URL="http://localhost:${PORT}/invite/accept"
export APPWRITE_RESET_REDIRECT_URL="http://localhost:${PORT}/reset/confirm"
export VITE_DEV_SERVER="http://localhost:${VITE_PORT}"

echo "Attesta http://localhost:${PORT} (vite :${VITE_PORT})"
