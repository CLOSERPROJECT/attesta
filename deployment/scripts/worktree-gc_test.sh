#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf -- "$tmpdir"' EXIT
mkdir -p "${tmpdir}/bin" "${tmpdir}/still-on-disk"

cat >"${tmpdir}/bin/docker" <<'EOF'
#!/bin/bash
set -euo pipefail
printf '%s\n' "$*" >>"${DOCKER_CALLS_FILE}"

case "$*" in
  'volume ls --filter label=eu.forkbomb.attesta.managed=true --format '*|'network ls --filter label=eu.forkbomb.attesta.managed=true --format '*)
    printf '%s|%s\n' "${ORPHAN_ROOT}" orphan-stack
    printf '%s|%s\n' "${ACTIVE_ROOT}" active-stack
    printf '%s|%s\n' "${EXISTING_ROOT}" existing-stack
    ;;
  'ps -aq --filter label=com.docker.compose.project=orphan-stack')
    echo orphan-container
    ;;
  'network ls -q --filter label=eu.forkbomb.attesta.managed=true --filter label=eu.forkbomb.attesta.project=orphan-stack')
    echo orphan-network
    ;;
  'volume ls -q --filter label=eu.forkbomb.attesta.managed=true --filter label=eu.forkbomb.attesta.project=orphan-stack')
    echo orphan-volume
    ;;
esac
EOF
chmod +x "${tmpdir}/bin/docker"

common_env=(
  "PATH=${tmpdir}/bin:${PATH}"
  "DOCKER_CALLS_FILE=${tmpdir}/calls"
  "ORPHAN_ROOT=${tmpdir}/deleted-codex-worktree"
  "ACTIVE_ROOT=${ROOT}"
  "EXISTING_ROOT=${tmpdir}/still-on-disk"
)

env "${common_env[@]}" bash "${ROOT}/scripts/worktree-gc.sh" >"${tmpdir}/dry-run"
grep -q 'would remove Attesta Compose project orphan-stack' "${tmpdir}/dry-run" \
  || fail "dry-run did not report the orphaned project"
if grep -q 'container rm\|network rm\|volume rm' "${tmpdir}/calls"; then
  fail "dry-run removed Docker resources"
fi
if grep -q 'active-stack\|existing-stack' "${tmpdir}/dry-run"; then
  fail "dry-run selected a live or existing worktree"
fi

: >"${tmpdir}/calls"
env "${common_env[@]}" bash "${ROOT}/scripts/worktree-gc.sh" --apply >"${tmpdir}/apply"
grep -q '^container rm -f orphan-container$' "${tmpdir}/calls" \
  || fail "apply did not remove the orphaned project's containers"
grep -q '^network rm orphan-network$' "${tmpdir}/calls" \
  || fail "apply did not remove the orphaned project's networks"
grep -q '^volume rm orphan-volume$' "${tmpdir}/calls" \
  || fail "apply did not remove the orphaned project's volumes"
if grep -q 'active-stack\|existing-stack' "${tmpdir}/calls"; then
  fail "apply touched a live or existing worktree"
fi

echo "ok: worktree-gc_test"
