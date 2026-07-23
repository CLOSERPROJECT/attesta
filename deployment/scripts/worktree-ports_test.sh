#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
fail() { echo "FAIL: $*" >&2; exit 1; }

tmpdir="$(mktemp -d)"
# Pretend non-primary by using a path under tmp (script uses real git primary for is_primary;
# for unit coverage, call pick logic indirectly by running script twice on same dir).
bash "$ROOT/deployment/scripts/worktree-ports.sh" "$tmpdir"
[[ -f "$tmpdir/.env.local" ]] || fail "missing .env.local"
grep -q '^PORT=' "$tmpdir/.env.local" || fail "missing PORT"
grep -q '^VITE_PORT=' "$tmpdir/.env.local" || fail "missing VITE_PORT"
# second run must not overwrite
cp "$tmpdir/.env.local" "$tmpdir/.env.local.bak"
bash "$ROOT/deployment/scripts/worktree-ports.sh" "$tmpdir"
diff -u "$tmpdir/.env.local.bak" "$tmpdir/.env.local" || fail "overwrote .env.local"
rm -rf "$tmpdir"
echo "ok: worktree-ports_test"
