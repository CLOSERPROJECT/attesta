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
real_git="$(command -v git)"

cat >"${tmpdir}/bin/git" <<'EOF'
#!/bin/bash
set -euo pipefail
if [[ "${FAIL_GIT_WORKTREE_LIST:-0}" == "1" && "$*" == *'worktree list --porcelain'* ]]; then
  echo "injected git worktree failure" >&2
  exit 73
fi
exec "${REAL_GIT}" "$@"
EOF
chmod +x "${tmpdir}/bin/git"

cat >"${tmpdir}/bin/docker" <<'EOF'
#!/bin/bash
set -euo pipefail
printf '%s\n' "$*" >>"${DOCKER_CALLS_FILE}"

if [[ -n "${FAIL_DOCKER_PREFIX:-}" && "$*" == "${FAIL_DOCKER_PREFIX}"* ]]; then
  echo "injected Docker enumeration failure" >&2
  exit 73
fi

if [[ "$*" == "network rm orphan-network" && "${NETWORK_BUSY:-0}" == "1" ]]; then
  echo "network has active endpoints" >&2
  exit 1
fi

case "$*" in
  'volume ls --filter label=eu.forkbomb.attesta.managed=true --filter label=eu.forkbomb.attesta.repository='*' --format '*|'network ls --filter label=eu.forkbomb.attesta.managed=true --filter label=eu.forkbomb.attesta.repository='*' --format '*)
    printf '%s|%s\n' "${ORPHAN_ROOT}" orphan-stack
    printf '%s|%s\n' "${ACTIVE_ROOT}" active-stack
    printf '%s|%s\n' "${EXISTING_ROOT}" existing-stack
    ;;
  'volume ls --filter label=eu.forkbomb.attesta.managed=true --format '*|'network ls --filter label=eu.forkbomb.attesta.managed=true --format '*)
    printf '%s|%s\n' "${ORPHAN_ROOT}" orphan-stack
    printf '%s|%s\n' "${FOREIGN_ROOT}" foreign-stack
    ;;
  'ps -aq --filter label=eu.forkbomb.attesta.managed=true --filter label=eu.forkbomb.attesta.repository='*' --filter label=eu.forkbomb.attesta.project=orphan-stack --filter label=eu.forkbomb.attesta.worktree='*)
    echo orphan-container
    ;;
  'ps -aq --filter label=com.docker.compose.project=orphan-stack')
    echo orphan-container
    echo unrelated-same-project-container
    ;;
  'network ls -q --filter label=eu.forkbomb.attesta.managed=true --filter label=eu.forkbomb.attesta.repository='*' --filter label=eu.forkbomb.attesta.project=orphan-stack --filter label=eu.forkbomb.attesta.worktree='*)
    echo orphan-network
    ;;
  'volume ls -q --filter label=eu.forkbomb.attesta.managed=true --filter label=eu.forkbomb.attesta.repository='*' --filter label=eu.forkbomb.attesta.project=orphan-stack --filter label=eu.forkbomb.attesta.worktree='*)
    echo orphan-volume
    ;;
esac
EOF
chmod +x "${tmpdir}/bin/docker"

common_env=(
  "PATH=${tmpdir}/bin:${PATH}"
  "REAL_GIT=${real_git}"
  "DOCKER_CALLS_FILE=${tmpdir}/calls"
  "ORPHAN_ROOT=${tmpdir}/deleted-codex-worktree"
  "ACTIVE_ROOT=${ROOT}"
  "EXISTING_ROOT=${tmpdir}/still-on-disk"
  "FOREIGN_ROOT=${tmpdir}/missing-in-another-clone"
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
if grep -q 'foreign-stack' "${tmpdir}/dry-run"; then
  fail "dry-run selected resources belonging to another repository clone"
fi
grep -q 'volume ls --filter label=eu.forkbomb.attesta.managed=true --filter label=eu.forkbomb.attesta.repository=' "${tmpdir}/calls" \
  || fail "candidate volumes were not scoped to this repository clone"
grep -q 'network ls --filter label=eu.forkbomb.attesta.managed=true --filter label=eu.forkbomb.attesta.repository=' "${tmpdir}/calls" \
  || fail "candidate networks were not scoped to this repository clone"

: >"${tmpdir}/calls"
env "${common_env[@]}" NETWORK_BUSY=1 bash "${ROOT}/scripts/worktree-gc.sh" --apply \
  >"${tmpdir}/apply" 2>"${tmpdir}/apply.err"
grep -q '^container rm -f orphan-container$' "${tmpdir}/calls" \
  || fail "apply did not remove the orphaned project's containers"
grep -Eq "^ps -aq --filter label=eu.forkbomb.attesta.managed=true --filter label=eu.forkbomb.attesta.repository=repo-[0-9a-f]{32} --filter label=eu.forkbomb.attesta.project=orphan-stack --filter label=eu.forkbomb.attesta.worktree=${tmpdir}/deleted-codex-worktree$" "${tmpdir}/calls" \
  || fail "container selection did not require the exact Attesta ownership labels"
if grep -q 'label=com.docker.compose.project=orphan-stack' "${tmpdir}/calls"; then
  fail "container selection still relies on the generic Compose project label"
fi
if grep -q 'unrelated-same-project-container' "${tmpdir}/calls"; then
  fail "apply selected an unrelated container with the same Compose project name"
fi
grep -q '^network rm orphan-network$' "${tmpdir}/calls" \
  || fail "apply did not remove the orphaned project's networks"
grep -q 'warning: could not remove network orphan-network' "${tmpdir}/apply.err" \
  || fail "a busy network did not produce a safe warning"
grep -q '^volume rm orphan-volume$' "${tmpdir}/calls" \
  || fail "a busy network prevented orphan volume processing"
grep -Eq "^network ls -q --filter label=eu.forkbomb.attesta.managed=true --filter label=eu.forkbomb.attesta.repository=repo-[0-9a-f]{32} --filter label=eu.forkbomb.attesta.project=orphan-stack --filter label=eu.forkbomb.attesta.worktree=${tmpdir}/deleted-codex-worktree$" "${tmpdir}/calls" \
  || fail "network selection did not require exact repository, project, and owner labels"
grep -Eq "^volume ls -q --filter label=eu.forkbomb.attesta.managed=true --filter label=eu.forkbomb.attesta.repository=repo-[0-9a-f]{32} --filter label=eu.forkbomb.attesta.project=orphan-stack --filter label=eu.forkbomb.attesta.worktree=${tmpdir}/deleted-codex-worktree$" "${tmpdir}/calls" \
  || fail "volume selection did not require exact repository, project, and owner labels"
if grep -q 'active-stack\|existing-stack' "${tmpdir}/calls"; then
  fail "apply touched a live or existing worktree"
fi

assert_enumeration_failure() {
  local name="$1"
  shift
  : >"${tmpdir}/calls"
  if env "${common_env[@]}" "$@" bash "${ROOT}/scripts/worktree-gc.sh" --apply \
      >"${tmpdir}/${name}.out" 2>"${tmpdir}/${name}.err"; then
    fail "${name} enumeration failure did not fail closed"
  fi
  if grep -Eq '^(container|network|volume) rm ' "${tmpdir}/calls"; then
    fail "${name} enumeration failure removed Docker resources"
  fi
  grep -q 'garbage collection aborted\|cleanup aborted' "${tmpdir}/${name}.err" \
    || fail "${name} enumeration failure did not explain the abort"
}

assert_enumeration_failure git-worktrees FAIL_GIT_WORKTREE_LIST=1
assert_enumeration_failure candidate-volumes \
  'FAIL_DOCKER_PREFIX=volume ls --filter label=eu.forkbomb.attesta.managed=true'
assert_enumeration_failure candidate-networks \
  'FAIL_DOCKER_PREFIX=network ls --filter label=eu.forkbomb.attesta.managed=true'
assert_enumeration_failure project-containers 'FAIL_DOCKER_PREFIX=ps -aq '
assert_enumeration_failure project-networks 'FAIL_DOCKER_PREFIX=network ls -q '
assert_enumeration_failure project-volumes 'FAIL_DOCKER_PREFIX=volume ls -q '

echo "ok: worktree-gc_test"
