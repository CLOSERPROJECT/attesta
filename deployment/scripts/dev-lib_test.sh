#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=deployment/scripts/dev-lib.sh
source "$ROOT/deployment/scripts/dev-lib.sh"

fail() { echo "FAIL: $*" >&2; exit 1; }

# load_env_file must not clobber pre-set PORT
tmp="$(mktemp)"
printf 'PORT=9999\nVITE_PORT=8888\n' >"$tmp"
export PORT=3001
load_env_file "$tmp"
[[ "$PORT" == "3001" ]] || fail "expected PORT to stay 3001, got $PORT"
[[ "$VITE_PORT" == "8888" ]] || fail "expected VITE_PORT 8888, got ${VITE_PORT:-}"
unset VITE_PORT
rm -f "$tmp"

# read_env_key extracts a single key from an env file
tmp="$(mktemp)"
printf 'PORT=4000\nVITE_PORT=5000\n' >"$tmp"
[[ "$(read_env_key "$tmp" "PORT")" == "4000" ]] || fail "read_env_key PORT expected 4000"
[[ "$(read_env_key "$tmp" "VITE_PORT")" == "5000" ]] || fail "read_env_key VITE_PORT expected 5000"
[[ -z "$(read_env_key "$tmp" "MISSING")" ]] || fail "read_env_key MISSING should be empty"
rm -f "$tmp"

# stable_port_index is deterministic and in 0..99
a="$(stable_port_index "/tmp/attesta-wt-a")"
b="$(stable_port_index "/tmp/attesta-wt-a")"
c="$(stable_port_index "/tmp/attesta-wt-b")"
[[ "$a" == "$b" ]] || fail "index not stable"
[[ "$a" =~ ^[0-9]+$ && "$a" -ge 0 && "$a" -le 99 ]] || fail "index out of range: $a"
# different paths usually differ; if equal, still OK as long as in range
[[ "$c" =~ ^[0-9]+$ && "$c" -ge 0 && "$c" -le 99 ]] || fail "index out of range: $c"

# pick_free_port returns preferred when free
# Use a high ephemeral preferred unlikely busy; skip if busy
pref=58761
if ! tcp_port_listening "$pref"; then
  got="$(pick_free_port 58700 100 "$pref")"
  [[ "$got" == "$pref" ]] || fail "expected preferred $pref, got $got"
fi

echo "ok: dev-lib_test"
