#!/usr/bin/env bash
# Stop host-dev processes that belong to this checkout only.
# Usage: dev-stop-local.sh [--ports-only] [repo_root]
# Optional env: PORT, VITE_PORT — if set, fail when a foreign process holds them.
# --ports-only: skip path-scoped kills; only run port_holder_ok checks.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=deployment/scripts/dev-lib.sh
source "$SCRIPT_DIR/dev-lib.sh"

ports_only=false
if [[ "${1:-}" == "--ports-only" ]]; then
  ports_only=true
  shift
fi

repo_root="${1:-$(git rev-parse --show-toplevel)}"
repo_root="$(cd "$repo_root" && pwd)"
server_dir="${repo_root}/server"
web_dir="${repo_root}/web"
bin_path="${server_dir}/tmp/attesta-server"

kill_matching() {
  local pattern="$1"
  local want_cwd="$2"
  local pid cwd
  for pid in $(pgrep -f "$pattern" 2>/dev/null || true); do
    cwd="$(process_cwd "$pid")"
    if [[ "$cwd" == "$want_cwd" ]]; then
      kill "$pid" >/dev/null 2>&1 || true
    fi
  done
}

if [[ "$ports_only" != true ]]; then
  # air in this server dir
  kill_matching 'air -c \.air\.toml' "$server_dir"
  # built binary for this tree
  if [[ -e "$bin_path" ]]; then
    for pid in $(pgrep -f "$bin_path" 2>/dev/null || true); do
      kill "$pid" >/dev/null 2>&1 || true
    done
  fi
  # vite for this web dir (node running vite with cwd=web)
  kill_matching 'vite' "$web_dir"

  sleep 1
fi

port_holder_ok() {
  local port="$1"
  local want_cwd="$2"
  tcp_port_listening "$port" || return 0
  local pid cwd
  pid="$(lsof -nP -iTCP:"$port" -sTCP:LISTEN -t 2>/dev/null | head -n1 || true)"
  [[ -n "$pid" ]] || return 0
  cwd="$(process_cwd "$pid")"
  if [[ "$cwd" == "$want_cwd" || "$cwd" == "$server_dir" || "$cwd" == "$web_dir" ]]; then
    return 0
  fi
  # also accept if command line references this repo_root
  local cmd
  cmd="$(ps -p "$pid" -o command= 2>/dev/null || true)"
  if [[ "$cmd" == *"$repo_root"* ]]; then
    return 0
  fi
  echo "error: port ${port} is in use by pid ${pid} (cwd=${cwd:-unknown}), not this worktree" >&2
  echo "  set PORT/VITE_PORT differently or stop that process" >&2
  return 1
}

if [[ -n "${PORT:-}" ]]; then
  port_holder_ok "$PORT" "$server_dir"
fi
if [[ -n "${VITE_PORT:-}" ]]; then
  port_holder_ok "$VITE_PORT" "$web_dir"
fi
