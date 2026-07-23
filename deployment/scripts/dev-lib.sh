#!/usr/bin/env bash
# Shared helpers for worktree / host-dev scripts. Source only; do not execute.

primary_worktree_root() {
  git worktree list --porcelain | awk '/^worktree / { print substr($0, 10); exit }'
}

is_primary_worktree() {
  local root="$1"
  local primary
  primary="$(primary_worktree_root)"
  [[ -n "$primary" && "$root" == "$primary" ]]
}

process_cwd() {
  local pid="$1"
  if [[ -d "/proc/$pid" ]]; then
    readlink "/proc/$pid/cwd" 2>/dev/null || true
    return 0
  fi
  # macOS: lsof cwd
  lsof -a -p "$pid" -d cwd -Fn 2>/dev/null | awk '/^n/ { print substr($0, 2); exit }'
}

tcp_port_listening() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
    return $?
  fi
  # fallback
  nc -z 127.0.0.1 "$port" >/dev/null 2>&1
}

# Read a single KEY=value from an env file (empty if missing).
read_env_key() {
  local path="$1"
  local key="$2"
  [[ -f "$path" ]] || return 0
  awk -F= -v k="$key" '$1 == k { print $2; exit }' "$path"
}

# Export KEY=VAL from file without overriding already-set environment variables.
load_env_file() {
  local path="$1"
  [[ -f "$path" ]] || return 0
  local line key val
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    key="${line%%=*}"
    val="${line#*=}"
    key="$(echo "$key" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
    [[ -z "$key" ]] && continue
    if [[ -n "${!key+x}" ]]; then
      continue
    fi
    export "$key=$val"
  done <"$path"
}

stable_port_index() {
  local root="$1"
  # cksum first field; map to 0..99
  local sum
  sum="$(printf '%s' "$root" | cksum | awk '{print $1}')"
  echo $((sum % 100))
}

pick_free_port() {
  local start="$1"
  local count="$2"
  local preferred="$3"
  local candidates=()
  local i p
  if [[ "$preferred" -ge "$start" && "$preferred" -lt $((start + count)) ]]; then
    candidates+=("$preferred")
  fi
  for ((i = 0; i < count; i++)); do
    p=$((start + i))
    if [[ "$p" -ne "$preferred" ]]; then
      candidates+=("$p")
    fi
  done
  for p in "${candidates[@]}"; do
    if ! tcp_port_listening "$p"; then
      echo "$p"
      return 0
    fi
  done
  echo "error: no free port in ${start}-$((start + count - 1))" >&2
  return 1
}
