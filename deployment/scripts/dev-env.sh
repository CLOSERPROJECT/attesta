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

_is_primary=0
is_primary_worktree "$REPO_ROOT" && _is_primary=1
if [[ "$_is_primary" -eq 0 && ! -f "$REPO_ROOT/.env.worktree" ]]; then
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

# Primary checkouts may intentionally point host processes at external dev
# services. Linked worktrees always replace these with their isolated stack.
_primary_endpoint_keys=(MONGODB_URI CERBOS_URL APPWRITE_ENDPOINT SMTP_HOST SMTP_PORT)
_primary_endpoint_set=()
_primary_endpoint_values=()
if [[ "$_is_primary" -eq 1 ]]; then
  for _index in "${!_primary_endpoint_keys[@]}"; do
    _key="${_primary_endpoint_keys[$_index]}"
    if [[ -n "${!_key+x}" ]]; then
      _primary_endpoint_set[$_index]=1
      _primary_endpoint_values[$_index]="${!_key}"
    fi
  done
fi

if ! _worktree_exports="$(bash "$REPO_ROOT/scripts/worktree-env.sh" export)"; then
  return 1 2>/dev/null || exit 1
fi
eval "$_worktree_exports"
if [[ "$_is_primary" -eq 1 ]]; then
  [[ "$_shell_has_port" -eq 1 ]] && export PORT="$_shell_port"
  [[ "$_shell_has_vite" -eq 1 ]] && export VITE_PORT="$_shell_vite"
  for _index in "${!_primary_endpoint_keys[@]}"; do
    if [[ "${_primary_endpoint_set[$_index]:-0}" -eq 1 ]]; then
      _key="${_primary_endpoint_keys[$_index]}"
      export "${_key}=${_primary_endpoint_values[$_index]}"
    fi
  done
fi
export PORT VITE_PORT
export APPWRITE_INVITE_REDIRECT_URL="http://localhost:${PORT}/invite/accept"
export APPWRITE_RESET_REDIRECT_URL="http://localhost:${PORT}/reset/confirm"
export PUBLIC_BASE_URL="http://localhost:${PORT}"
export VITE_DEV_SERVER="http://localhost:${VITE_PORT}"

echo "Attesta http://localhost:${PORT} (vite :${VITE_PORT})"
