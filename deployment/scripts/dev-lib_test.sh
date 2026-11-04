#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=deployment/scripts/dev-lib.sh
source "$ROOT/deployment/scripts/dev-lib.sh"

fail() { echo "FAIL: $*" >&2; exit 1; }

# load_env_file must not clobber pre-set PORT
tmp="$(mktemp)"
printf 'PORT=9999\nVITE_PORT=8888\n' >"$tmp"
unset VITE_PORT
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

# process_cwd must not abort under pipefail when the PID is gone (dev-stop-local).
cwd="$(process_cwd 99999999)"
[[ -z "$cwd" ]] || fail "expected empty cwd for missing pid, got $cwd"

echo "ok: dev-lib_test"
