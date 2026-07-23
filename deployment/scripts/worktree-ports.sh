#!/usr/bin/env bash
# Ensure target root has .env.local with PORT and VITE_PORT.
# Usage: worktree-ports.sh [target_root]
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=deployment/scripts/dev-lib.sh
source "$SCRIPT_DIR/dev-lib.sh"

target_root="${1:-$(git rev-parse --show-toplevel)}"
target_root="$(cd "$target_root" && pwd)"
env_local="${target_root}/.env.local"

read_key() {
  local file="$1" key="$2"
  [[ -f "$file" ]] || return 0
  awk -F= -v k="$key" '$1 == k { print $2; exit }' "$file"
}

existing_port="$(read_key "$env_local" PORT)"
existing_vite="$(read_key "$env_local" VITE_PORT)"
if [[ -n "${existing_port}" && -n "${existing_vite}" ]]; then
  echo "ok: ${env_local} PORT=${existing_port} VITE_PORT=${existing_vite}"
  exit 0
fi

if is_primary_worktree "$target_root"; then
  preferred_port=3000
  preferred_vite=5173
  port_start=3000
  port_count=100
  vite_start=5173
  vite_count=100
else
  idx="$(stable_port_index "$target_root")"
  preferred_port=$((3100 + idx))
  preferred_vite=$((5200 + idx))
  port_start=3100
  port_count=100
  vite_start=5200
  vite_count=100
fi

# Keep existing single key if present
if [[ -n "${existing_port}" ]]; then
  port="$existing_port"
else
  port="$(pick_free_port "$port_start" "$port_count" "$preferred_port")"
fi
if [[ -n "${existing_vite}" ]]; then
  vite_port="$existing_vite"
else
  vite_port="$(pick_free_port "$vite_start" "$vite_count" "$preferred_vite")"
fi

printf 'PORT=%s\nVITE_PORT=%s\n' "$port" "$vite_port" >"$env_local"
echo "wrote ${env_local} PORT=${port} VITE_PORT=${vite_port}"
