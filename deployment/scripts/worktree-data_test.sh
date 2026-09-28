#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SCRIPT="${ROOT}/scripts/worktree-data.sh"

fail() { echo "FAIL: $*" >&2; exit 1; }
assert_file() { [[ -s "$1" ]] || fail "missing or empty file: $1"; }

tmpdir="$(mktemp -d)"
trap 'rm -rf -- "$tmpdir"' EXIT
primary="${tmpdir}/primary"
target="${tmpdir}/target"
mkdir -p "${primary}/scripts" "${target}/scripts" "${tmpdir}/bin" "${tmpdir}/volumes"
target_physical="$(cd "${target}" && pwd -P)"
for checkout in "${primary}" "${target}"; do
  cp "${ROOT}/scripts/worktree-env.sh" "${checkout}/scripts/worktree-env.sh"
  cp "${ROOT}/scripts/dev-ports.env" "${checkout}/scripts/dev-ports.env"
  mkdir -p "${checkout}/deployment"
  : >"${checkout}/deployment/docker-compose.local.yaml"
done

cat >"${tmpdir}/bin/git" <<'EOF'
#!/usr/bin/env bash
if [[ " $* " == *" --git-common-dir "* ]]; then
  printf '%s\n' "${FAKE_GIT_COMMON_DIR}"
  exit 0
fi
if [[ " $* " == *" worktree list --porcelain "* ]]; then
  printf 'worktree %s\nHEAD deadbeef\nbranch refs/heads/main\n' "${FAKE_PRIMARY}"
  exit 0
fi
exit 1
EOF

cat >"${tmpdir}/bin/bash" <<'EOF'
#!/bin/bash
set -euo pipefail
if [[ "${1:-}" == */scripts/worktree-env.sh && "${2:-}" == export && "${FAKE_WORKTREE_EXPORT_FAILURE:-0}" == 1 ]]; then
  echo "injected worktree export failure" >&2
  exit 75
fi
if [[ "${1:-}" == */scripts/worktree-env.sh && "${2:-}" == repository-id && "${FAKE_REPOSITORY_ID_FAILURE:-0}" == 1 ]]; then
  echo "injected repository identity failure" >&2
  exit 76
fi
exec /bin/bash "$@"
EOF

cat >"${tmpdir}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$*" == "compose version --short" ]]; then
  echo "${DOCKER_COMPOSE_VERSION:-2.40.3}"
  exit 0
fi
printf '%s\n' "$*" >>"${DOCKER_CALLS_FILE}"

if [[ " $* " == *" ps --status running --services "* && "${FAKE_PS_FAILURE:-}" == "running" ]]; then
  exit 72
fi
if [[ " $* " == *" ps --all --services "* && "${FAKE_PS_FAILURE:-}" == "all" ]]; then
  exit 73
fi

if [[ "${1:-}" == volume && "${2:-}" == ls ]]; then
  if [[ "${FAKE_VOLUME_LS_FAILURE:-0}" == 1 ]]; then
    echo "injected volume enumeration failure" >&2
    exit 77
  fi
  project="" logical=""
  for arg in "$@"; do
    case "$arg" in
      label=com.docker.compose.project=*) project="${arg##*=}" ;;
      label=com.docker.compose.volume=*) logical="${arg##*=}" ;;
    esac
  done
  [[ -d "${FAKE_VOLUMES_DIR}/${project}_${logical}" ]] && printf '%s\n' "${project}_${logical}"
  exit 0
fi

if [[ "${1:-}" == volume && "${2:-}" == create ]]; then
  name="${*: -1}"
  mkdir -p "${FAKE_VOLUMES_DIR}/${name}"
  managed="" project="" owner="" repository="" previous=""
  for arg in "$@"; do
    if [[ "${previous}" == --label ]]; then
      case "${arg}" in
        eu.forkbomb.attesta.managed=*) managed="${arg#*=}" ;;
        eu.forkbomb.attesta.project=*) project="${arg#*=}" ;;
        eu.forkbomb.attesta.worktree=*) owner="${arg#*=}" ;;
        eu.forkbomb.attesta.repository=*) repository="${arg#*=}" ;;
      esac
    fi
    previous="${arg}"
  done
  printf '%s|%s|%s|%s\n' "${managed}" "${project}" "${owner}" "${repository}" \
    >"${FAKE_VOLUMES_DIR}/${name}/.ownership"
  printf '%s\n' "$name"
  exit 0
fi

if [[ "${1:-}" == volume && "${2:-}" == rm ]]; then
  shift 2
  for name in "$@"; do
    [[ "$name" == -f ]] || rm -rf -- "${FAKE_VOLUMES_DIR}/${name}"
  done
  exit 0
fi

if [[ "${1:-}" == volume && "${2:-}" == inspect ]]; then
  name="${*: -1}"
  [[ -f "${FAKE_VOLUMES_DIR}/${name}/.ownership" ]] || exit 1
  cat "${FAKE_VOLUMES_DIR}/${name}/.ownership"
  exit 0
fi

if [[ "${1:-}" == network && "${2:-}" == ls ]]; then
  [[ -n "${FAKE_RUNTIME_NETWORK:-}" ]] && printf '%s\n' "${FAKE_RUNTIME_NETWORK}"
  exit 0
fi

if [[ "${1:-}" == network && "${2:-}" == inspect ]]; then
  [[ -n "${FAKE_RUNTIME_CONTAINER_IDS:-}" ]] && printf '%s\n' "${FAKE_RUNTIME_CONTAINER_IDS}"
  exit 0
fi

if [[ "${1:-}" == run ]]; then
  mount="" previous=""
  for arg in "$@"; do
    [[ "$previous" == -v ]] && mount="$arg"
    previous="$arg"
  done
  volume="${mount%%:*}"
  logical="${volume#*_}"
  case " $* " in
    *" test-empty "*)
      [[ ! -e "${FAKE_VOLUMES_DIR}/${volume}/nonempty" ]] || exit 1
      ;;
    *" archive "*)
      [[ "${FAKE_ARCHIVE_FAILURE:-}" != "$logical" ]] || exit 23
      printf 'archive:%s' "$logical"
      ;;
    *" extract "*)
      [[ "${FAKE_EXTRACT_FAILURE:-}" != "$logical" ]] || exit 24
      if [[ -n "${FAKE_RESTORE_GUARD_DIR:-}" && "$logical" == mongodb_data ]]; then
        if ! mkdir "${FAKE_RESTORE_GUARD_DIR}" 2>/dev/null; then
          : >"${FAKE_RESTORE_OVERLAP_FILE}"
          exit 74
        fi
        sleep "${FAKE_RESTORE_PAUSE_SECONDS:-0}"
        rmdir "${FAKE_RESTORE_GUARD_DIR}" 2>/dev/null || true
      fi
      mkdir -p "${FAKE_RESTORE_DIR}"
      cat >"${FAKE_RESTORE_DIR}/${logical}.tar.gz"
      : >"${FAKE_VOLUMES_DIR}/${volume}/nonempty"
      ;;
    *" clear "*)
      rm -f -- "${FAKE_VOLUMES_DIR}/${volume}/nonempty"
      ;;
  esac
  exit 0
fi

case " $* " in
  *" stop "*)
    if [[ -n "${FAKE_SNAPSHOT_GUARD_DIR:-}" ]]; then
      if ! mkdir "${FAKE_SNAPSHOT_GUARD_DIR}" 2>/dev/null; then
        : >"${FAKE_SNAPSHOT_OVERLAP_FILE}"
        exit 71
      fi
      sleep "${FAKE_SNAPSHOT_PAUSE_SECONDS:-0}"
    fi
    ;;
  *" start "*)
    if [[ -n "${FAKE_SNAPSHOT_GUARD_DIR:-}" ]]; then
      rmdir "${FAKE_SNAPSHOT_GUARD_DIR}" 2>/dev/null || true
    fi
    ;;
  *" ps --status running --services "*) printf '%s' "${FAKE_RUNNING_SERVICES-appwrite
mongodb
mariadb
}" ;;
  *" ps --all --services "*) [[ -z "${FAKE_ALL_SERVICES:-}" ]] || printf '%s\n' "${FAKE_ALL_SERVICES}" ;;
  *" mariadb-admin ping "*) echo 'mysqld is alive' ;;
  *" exec -T mariadb "*" mariadb --user=root "*)
    mkdir -p "${FAKE_RESTORE_DIR}"
    printf '%s\n' "$*" >"${FAKE_RESTORE_DIR}/sanitized"
    ;;
esac
EOF
chmod +x "${tmpdir}/bin/bash" "${tmpdir}/bin/git" "${tmpdir}/bin/docker"

common_env=(
  PATH="${tmpdir}/bin:${PATH}"
  FAKE_PRIMARY="${primary}"
  FAKE_GIT_COMMON_DIR="${tmpdir}/git-common"
  FAKE_VOLUMES_DIR="${tmpdir}/volumes"
  FAKE_RESTORE_DIR="${tmpdir}/restored"
  DOCKER_CALLS_FILE="${tmpdir}/docker-calls"
)
mkdir -p "${tmpdir}/git-common"

: >"${tmpdir}/docker-calls"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" \
  DOCKER_COMPOSE_VERSION=2.24.3 bash "${SCRIPT}" snapshot-primary \
  >"${tmpdir}/old-compose.out" 2>"${tmpdir}/old-compose.err"; then
  fail "data lifecycle accepted Docker Compose older than 2.24.4"
fi
grep -q 'Docker Compose 2.24.4 or newer is required' "${tmpdir}/old-compose.err" \
  || fail "old Compose rejection did not explain the required version"
[[ ! -s "${tmpdir}/docker-calls" ]] \
  || fail "data lifecycle touched Docker resources before its version preflight"

primary_project="$(env "${common_env[@]}" ATTESTA_ROOT_DIR="${primary}" \
  bash "${primary}/scripts/worktree-env.sh" project-name)"
target_project="$(env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" \
  bash "${target}/scripts/worktree-env.sh" project-name)"
target_repository="$(env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" \
  bash "${target}/scripts/worktree-env.sh" repository-id)"
[[ "${primary_project}" =~ ^primary-[0-9a-f]{8}$ ]] \
  || fail "primary Compose identity is not path-qualified: ${primary_project}"
[[ "${target_project}" != "${primary_project}" ]] \
  || fail "target reused the primary Compose identity"

durable_volumes=(
  mongodb_data appwrite-mariadb appwrite-uploads appwrite-imports
  appwrite-functions appwrite-sites appwrite-builds
)

prepare_checkout() {
  local checkout="$1"
  mkdir -p "${checkout}/scripts" "${checkout}/deployment"
  cp "${ROOT}/scripts/worktree-env.sh" "${checkout}/scripts/worktree-env.sh"
  cp "${ROOT}/scripts/dev-ports.env" "${checkout}/scripts/dev-ports.env"
  : >"${checkout}/deployment/docker-compose.local.yaml"
}

for logical in "${durable_volumes[@]}"; do
  mkdir -p "${tmpdir}/volumes/${primary_project}_${logical}"
  : >"${tmpdir}/volumes/${primary_project}_${logical}/nonempty"
done

# A snapshot is an immutable, checksummed bundle of every durable store.
env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" bash "${SCRIPT}" snapshot-primary
bundle="${target}/.worktree-data/snapshot-v1"
assert_file "${bundle}/manifest"
for logical in "${durable_volumes[@]}"; do
  artifact="${logical}.tar.gz"
  assert_file "${bundle}/${artifact}"
  grep -Eq "^sha256 ${artifact} [[:xdigit:]]{64}$" "${bundle}/manifest" \
    || fail "manifest has no checksum for ${artifact}"
done
grep -qx 'format attesta-worktree-data' "${bundle}/manifest" || fail "manifest format missing"
grep -qx 'version 1' "${bundle}/manifest" || fail "manifest version missing"
grep -Eq '^created_at .+Z$' "${bundle}/manifest" || fail "manifest timestamp missing"
grep -Fqx "source_root ${primary}" "${bundle}/manifest" || fail "manifest source root missing"
grep -Fqx "source_project ${primary_project}" "${bundle}/manifest" \
  || fail "manifest source project missing"

# Writers are quiesced for the entire raw-volume snapshot, then prior state resumes.
grep -q ' stop appwrite mongodb mariadb$' "${tmpdir}/docker-calls" \
  || fail "primary running services were not quiesced"
grep -q ' start appwrite mongodb mariadb$' "${tmpdir}/docker-calls" \
  || fail "primary running services were not resumed"

# Repeated setup preserves the original point-in-time bundle.
calls_before="$(wc -l <"${tmpdir}/docker-calls")"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" bash "${SCRIPT}" snapshot-primary >/dev/null
calls_after="$(wc -l <"${tmpdir}/docker-calls")"
[[ "$calls_before" == "$calls_after" ]] || fail "snapshot was regenerated"

# Compose status is part of the safety decision. A failed `ps` must never be
# mistaken for an empty or stopped stack.
ps_all_target="${tmpdir}/ps-all-target"
prepare_checkout "${ps_all_target}"
: >"${tmpdir}/docker-calls"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${ps_all_target}" FAKE_PS_FAILURE=all \
  bash "${SCRIPT}" snapshot-primary >"${tmpdir}/ps-all.out" 2>"${tmpdir}/ps-all.err"; then
  fail "snapshot treated a failed all-services query as an empty target"
fi
! grep -q ' archive ' "${tmpdir}/docker-calls" \
  || fail "snapshot archived data after the target-state query failed"

ps_running_target="${tmpdir}/ps-running-target"
prepare_checkout "${ps_running_target}"
: >"${tmpdir}/docker-calls"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${ps_running_target}" FAKE_PS_FAILURE=running \
  bash "${SCRIPT}" snapshot-primary >"${tmpdir}/ps-running.out" 2>"${tmpdir}/ps-running.err"; then
  fail "snapshot treated a failed running-services query as no running writers"
fi
! grep -q ' archive ' "${tmpdir}/docker-calls" \
  || fail "snapshot archived data after the running-services query failed"

# Dynamic Appwrite runtimes are outside Compose's service list. Raw volume
# capture refuses to proceed while one remains attached after Compose writers
# are quiesced.
runtime_target="${tmpdir}/runtime-target"
prepare_checkout "${runtime_target}"
: >"${tmpdir}/docker-calls"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${runtime_target}" \
  FAKE_RUNTIME_NETWORK="${primary_project}_runtimes" \
  FAKE_RUNTIME_CONTAINER_IDS=dynamic-runtime-123 \
  bash "${SCRIPT}" snapshot-primary >"${tmpdir}/runtime.out" 2>"${tmpdir}/runtime.err"; then
  fail "snapshot accepted a dynamic container on the Appwrite runtimes network"
fi
grep -qi 'runtime.*container' "${tmpdir}/runtime.err" \
  || fail "dynamic runtime refusal was not actionable"
! grep -q ' archive ' "${tmpdir}/docker-calls" \
  || fail "snapshot archived data while a dynamic runtime container was attached"
grep -q ' start appwrite mongodb mariadb$' "${tmpdir}/docker-calls" \
  || fail "dynamic runtime refusal did not restore the primary running set"

# A live repository lock fails with an actionable bounded error instead of
# hanging bootstrap indefinitely.
timeout_target="${tmpdir}/timeout-target"
mkdir -p "${timeout_target}/scripts" "${timeout_target}/deployment"
cp "${ROOT}/scripts/worktree-env.sh" "${timeout_target}/scripts/worktree-env.sh"
cp "${ROOT}/scripts/dev-ports.env" "${timeout_target}/scripts/dev-ports.env"
: >"${timeout_target}/deployment/docker-compose.local.yaml"
lock_dir="${tmpdir}/git-common/attesta-worktree-snapshot.lock"
mkdir "${lock_dir}"
printf '%s\n' "$$" >"${lock_dir}/pid"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${timeout_target}" \
  ATTESTA_SNAPSHOT_LOCK_TIMEOUT_SECONDS=0 \
  bash "${SCRIPT}" snapshot-primary >"${tmpdir}/timeout-output" 2>"${tmpdir}/timeout-error"; then
  fail "snapshot unexpectedly ignored a live repository lock"
fi
grep -Fq 'timed out after 0s waiting for repository snapshot lock' "${tmpdir}/timeout-error" \
  || fail "snapshot lock timeout error was not actionable"
rm -f -- "${lock_dir}/pid"
rmdir "${lock_dir}"

# Two first-time worktree snapshots share one repository lock. The fake
# primary rejects overlapping stop/archive windows, so both commands can only
# succeed when the public CLI serializes the entire operation.
concurrent_a="${tmpdir}/concurrent-a"
concurrent_b="${tmpdir}/concurrent-b"
for checkout in "${concurrent_a}" "${concurrent_b}"; do
  mkdir -p "${checkout}/scripts" "${checkout}/deployment"
  cp "${ROOT}/scripts/worktree-env.sh" "${checkout}/scripts/worktree-env.sh"
  cp "${ROOT}/scripts/dev-ports.env" "${checkout}/scripts/dev-ports.env"
  : >"${checkout}/deployment/docker-compose.local.yaml"
done
guard_dir="${tmpdir}/primary-snapshot-active"
overlap_file="${tmpdir}/primary-snapshot-overlap"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${concurrent_a}" \
  FAKE_SNAPSHOT_GUARD_DIR="${guard_dir}" \
  FAKE_SNAPSHOT_OVERLAP_FILE="${overlap_file}" \
  FAKE_SNAPSHOT_PAUSE_SECONDS=1 \
  bash "${SCRIPT}" snapshot-primary >"${tmpdir}/concurrent-a-output" 2>"${tmpdir}/concurrent-a-error" &
pid_a=$!
env "${common_env[@]}" ATTESTA_ROOT_DIR="${concurrent_b}" \
  FAKE_SNAPSHOT_GUARD_DIR="${guard_dir}" \
  FAKE_SNAPSHOT_OVERLAP_FILE="${overlap_file}" \
  FAKE_SNAPSHOT_PAUSE_SECONDS=1 \
  bash "${SCRIPT}" snapshot-primary >"${tmpdir}/concurrent-b-output" 2>"${tmpdir}/concurrent-b-error" &
pid_b=$!
concurrent_status=0
wait "${pid_a}" || concurrent_status=1
wait "${pid_b}" || concurrent_status=1
if [[ "${concurrent_status}" != "0" ]]; then
  cat "${tmpdir}/concurrent-a-error" "${tmpdir}/concurrent-b-error" >&2
  fail "concurrent first-time snapshots did not both succeed"
fi
[[ ! -e "${overlap_file}" ]] \
  || fail "concurrent snapshots overlapped the primary quiesce window"
assert_file "${concurrent_a}/.worktree-data/snapshot-v1/manifest"
assert_file "${concurrent_b}/.worktree-data/snapshot-v1/manifest"

# The fresh-seed marker is authoritative if forget is interrupted after
# recording the choice but before deleting the old bundle.
printf 'format attesta-worktree-data\nversion 1\n' >"${target}/.worktree-data/fresh-seed"
: >"${tmpdir}/docker-calls"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" FAKE_RUNNING_SERVICES='' \
  bash "${SCRIPT}" restore >"${tmpdir}/marker-wins-output"
grep -qi 'fresh.seed choice' "${tmpdir}/marker-wins-output" \
  || fail "restore did not explain the authoritative fresh-seed choice"
[[ ! -s "${tmpdir}/docker-calls" ]] \
  || fail "restore touched Docker while a fresh-seed choice was present"
rm -f -- "${target}/.worktree-data/fresh-seed"

# Restore cannot interpret a failed running-services query as a stopped stack.
: >"${tmpdir}/docker-calls"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" \
  FAKE_PS_FAILURE=running bash "${SCRIPT}" restore \
  >"${tmpdir}/restore-ps.out" 2>"${tmpdir}/restore-ps.err"; then
  fail "restore treated a failed running-services query as a stopped target"
fi
! grep -q ' extract ' "${tmpdir}/docker-calls" \
  || fail "restore extracted data after its running-services query failed"

# Worktree environment failures cannot fall back to an inherited project or
# continue into Docker mutations.
export_failure_target="${tmpdir}/export-failure-target"
prepare_checkout "${export_failure_target}"
mkdir -p "${export_failure_target}/.worktree-data"
cp -R "${bundle}" "${export_failure_target}/.worktree-data/snapshot-v1"
: >"${tmpdir}/docker-calls"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${export_failure_target}" \
  COMPOSE_PROJECT_NAME=stale-project FAKE_WORKTREE_EXPORT_FAILURE=1 \
  FAKE_RUNNING_SERVICES='' bash "${SCRIPT}" restore \
  >"${tmpdir}/export-failure.out" 2>"${tmpdir}/export-failure.err"; then
  fail "restore inherited a stale Compose project after export failure"
fi
grep -q 'injected worktree export failure' "${tmpdir}/export-failure.err" \
  || fail "restore hid its worktree export failure"
! grep -Eq 'volume create| extract ' "${tmpdir}/docker-calls" \
  || fail "restore mutated volumes after worktree export failure"

repository_failure_target="${tmpdir}/repository-failure-target"
prepare_checkout "${repository_failure_target}"
mkdir -p "${repository_failure_target}/.worktree-data"
cp -R "${bundle}" "${repository_failure_target}/.worktree-data/snapshot-v1"
: >"${tmpdir}/docker-calls"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${repository_failure_target}" \
  FAKE_REPOSITORY_ID_FAILURE=1 FAKE_RUNNING_SERVICES='' \
  bash "${SCRIPT}" restore \
  >"${tmpdir}/repository-failure.out" 2>"${tmpdir}/repository-failure.err"; then
  fail "restore continued after repository identity failure"
fi
grep -q 'injected repository identity failure' "${tmpdir}/repository-failure.err" \
  || fail "restore hid its repository identity failure"
! grep -Eq 'volume create| extract ' "${tmpdir}/docker-calls" \
  || fail "restore mutated volumes after repository identity failure"

# A failed Docker volume enumeration is unknown state, never an empty target.
volume_ls_failure_target="${tmpdir}/volume-ls-failure-target"
prepare_checkout "${volume_ls_failure_target}"
mkdir -p "${volume_ls_failure_target}/.worktree-data"
cp -R "${bundle}" "${volume_ls_failure_target}/.worktree-data/snapshot-v1"
: >"${tmpdir}/docker-calls"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${volume_ls_failure_target}" \
  FAKE_VOLUME_LS_FAILURE=1 FAKE_RUNNING_SERVICES='' \
  bash "${SCRIPT}" restore \
  >"${tmpdir}/volume-ls-failure.out" 2>"${tmpdir}/volume-ls-failure.err"; then
  fail "restore treated failed volume enumeration as an empty target"
fi
grep -q 'injected volume enumeration failure' "${tmpdir}/volume-ls-failure.err" \
  || fail "restore hid its volume enumeration failure"
! grep -Eq 'volume create| extract ' "${tmpdir}/docker-calls" \
  || fail "restore mutated volumes after volume enumeration failure"

# Restore happens before the stack, restores all volumes, then sanitizes Appwrite runtime state.
: >"${tmpdir}/docker-calls"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" FAKE_RUNNING_SERVICES='' bash "${SCRIPT}" restore
for logical in "${durable_volumes[@]}"; do
  cmp "${bundle}/${logical}.tar.gz" "${tmpdir}/restored/${logical}.tar.gz" \
    || fail "${logical} was not restored"
done
assert_file "${tmpdir}/restored/sanitized"
grep -q '_console_sessions' "${tmpdir}/restored/sanitized" || fail "Appwrite sessions not sanitized"
grep -q '_console_certificates' "${tmpdir}/restored/sanitized" || fail "Appwrite certificates not sanitized"
grep -q '_console_rules' "${tmpdir}/restored/sanitized" || fail "Appwrite rules not sanitized"
grep -Fq -- "--label eu.forkbomb.attesta.managed=true" "${tmpdir}/docker-calls" \
  || fail "restored volumes are not marked as managed"
grep -Fq -- "--label eu.forkbomb.attesta.worktree=${target_physical}" "${tmpdir}/docker-calls" \
  || fail "restored volumes have no physical worktree owner"
grep -Fq -- "--label eu.forkbomb.attesta.repository=${target_repository}" "${tmpdir}/docker-calls" \
  || fail "restored volumes have no repository identity"

# Existing worktree state wins; restore never overwrites or partially merges it.
: >"${tmpdir}/docker-calls"
rm -rf -- "${tmpdir}/restored"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" FAKE_RUNNING_SERVICES='' bash "${SCRIPT}" restore >/dev/null
if [[ -e "${tmpdir}/restored" ]]; then
  cat "${tmpdir}/docker-calls" >&2
  fail "restore overwrote existing worktree data"
fi
! grep -q ' extract ' "${tmpdir}/docker-calls" || fail "restore touched a developed worktree"

# An empty pre-existing Compose volume is reusable only when all Attesta
# ownership labels identify this exact checkout and repository.
ownership_target="${tmpdir}/ownership-target"
prepare_checkout "${ownership_target}"
mkdir -p "${ownership_target}/.worktree-data"
cp -R "${bundle}" "${ownership_target}/.worktree-data/snapshot-v1"
ownership_project="$(env "${common_env[@]}" ATTESTA_ROOT_DIR="${ownership_target}" \
  bash "${ownership_target}/scripts/worktree-env.sh" project-name)"
mkdir -p "${tmpdir}/volumes/${ownership_project}_mongodb_data"
printf 'true|%s|%s|%s\n' "${ownership_project}" "/another/worktree" "${target_repository}" \
  >"${tmpdir}/volumes/${ownership_project}_mongodb_data/.ownership"
: >"${tmpdir}/docker-calls"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${ownership_target}" FAKE_RUNNING_SERVICES='' \
  bash "${SCRIPT}" restore >"${tmpdir}/ownership.out" 2>"${tmpdir}/ownership.err"; then
  fail "restore accepted an empty volume owned by another worktree"
fi
grep -qi 'owned by this worktree' "${tmpdir}/ownership.err" \
  || fail "pre-existing volume ownership refusal was not actionable"
! grep -q ' extract ' "${tmpdir}/docker-calls" \
  || fail "restore extracted data into a volume with mismatched ownership"

# Failed restore cleans up only volumes it created. An exact-owned empty volume
# that existed before restore remains available for diagnosis or retry.
cleanup_target="${tmpdir}/cleanup-target"
prepare_checkout "${cleanup_target}"
mkdir -p "${cleanup_target}/.worktree-data"
cp -R "${bundle}" "${cleanup_target}/.worktree-data/snapshot-v1"
cleanup_project="$(env "${common_env[@]}" ATTESTA_ROOT_DIR="${cleanup_target}" \
  bash "${cleanup_target}/scripts/worktree-env.sh" project-name)"
cleanup_repository="$(env "${common_env[@]}" ATTESTA_ROOT_DIR="${cleanup_target}" \
  bash "${cleanup_target}/scripts/worktree-env.sh" repository-id)"
cleanup_physical="$(cd "${cleanup_target}" && pwd -P)"
preexisting_volume="${cleanup_project}_mongodb_data"
mkdir -p "${tmpdir}/volumes/${preexisting_volume}"
printf 'true|%s|%s|%s\n' "${cleanup_project}" "${cleanup_physical}" "${cleanup_repository}" \
  >"${tmpdir}/volumes/${preexisting_volume}/.ownership"
: >"${tmpdir}/docker-calls"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${cleanup_target}" FAKE_RUNNING_SERVICES='' \
  FAKE_EXTRACT_FAILURE=appwrite-functions bash "${SCRIPT}" restore >/dev/null 2>&1; then
  fail "partial restore with a pre-existing volume unexpectedly succeeded"
fi
[[ -d "${tmpdir}/volumes/${preexisting_volume}" ]] \
  || fail "failed restore deleted a pre-existing owned volume"
[[ ! -e "${tmpdir}/volumes/${preexisting_volume}/nonempty" ]] \
  || fail "failed restore left partial data in a pre-existing owned volume"
if find "${tmpdir}/volumes" -maxdepth 1 -type d -name "${cleanup_project}_*" \
  ! -name "${preexisting_volume}" | grep -q .; then
  fail "failed restore retained volumes that it created"
fi

# Two restores of one checkout serialize the complete decision and mutation
# window. The second observes the first restore's finished state and leaves it.
concurrent_restore_target="${tmpdir}/concurrent-restore-target"
prepare_checkout "${concurrent_restore_target}"
mkdir -p "${concurrent_restore_target}/.worktree-data"
cp -R "${bundle}" "${concurrent_restore_target}/.worktree-data/snapshot-v1"
restore_guard="${tmpdir}/restore-active"
restore_overlap="${tmpdir}/restore-overlap"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${concurrent_restore_target}" \
  FAKE_RUNNING_SERVICES='' FAKE_RESTORE_GUARD_DIR="${restore_guard}" \
  FAKE_RESTORE_OVERLAP_FILE="${restore_overlap}" FAKE_RESTORE_PAUSE_SECONDS=1 \
  bash "${SCRIPT}" restore >"${tmpdir}/restore-a.out" 2>"${tmpdir}/restore-a.err" &
restore_a_pid=$!
for _ in $(seq 1 100); do
  [[ -d "${restore_guard}" ]] && break
  sleep 0.02
done
[[ -d "${restore_guard}" ]] || fail "first restore never entered its mutation window"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${concurrent_restore_target}" \
  FAKE_RUNNING_SERVICES='' FAKE_RESTORE_GUARD_DIR="${restore_guard}" \
  FAKE_RESTORE_OVERLAP_FILE="${restore_overlap}" FAKE_RESTORE_PAUSE_SECONDS=1 \
  bash "${SCRIPT}" restore >"${tmpdir}/restore-b.out" 2>"${tmpdir}/restore-b.err" &
restore_b_pid=$!
restore_status=0
wait "${restore_a_pid}" || restore_status=1
wait "${restore_b_pid}" || restore_status=1
if [[ "${restore_status}" != 0 ]]; then
  cat "${tmpdir}/restore-a.err" "${tmpdir}/restore-b.err" >&2
  fail "concurrent restores did not both finish safely"
fi
[[ ! -e "${restore_overlap}" ]] || fail "concurrent restores overlapped mutation windows"

# Forget cannot publish the fresh-seed choice or delete the bundle while a
# restore from that bundle is still in progress.
forget_race_target="${tmpdir}/forget-race-target"
prepare_checkout "${forget_race_target}"
mkdir -p "${forget_race_target}/.worktree-data"
cp -R "${bundle}" "${forget_race_target}/.worktree-data/snapshot-v1"
forget_guard="${tmpdir}/forget-race-restore-active"
forget_overlap="${tmpdir}/forget-race-overlap"
forget_finished="${tmpdir}/forget-finished"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${forget_race_target}" \
  FAKE_RUNNING_SERVICES='' FAKE_RESTORE_GUARD_DIR="${forget_guard}" \
  FAKE_RESTORE_OVERLAP_FILE="${forget_overlap}" FAKE_RESTORE_PAUSE_SECONDS=1 \
  bash "${SCRIPT}" restore >"${tmpdir}/forget-race-restore.out" \
  2>"${tmpdir}/forget-race-restore.err" &
forget_restore_pid=$!
for _ in $(seq 1 100); do
  [[ -d "${forget_guard}" ]] && break
  sleep 0.02
done
[[ -d "${forget_guard}" ]] || fail "restore never entered the restore-vs-forget window"
(
  env "${common_env[@]}" ATTESTA_ROOT_DIR="${forget_race_target}" \
    bash "${SCRIPT}" forget >"${tmpdir}/forget-race-forget.out" \
    2>"${tmpdir}/forget-race-forget.err"
  : >"${forget_finished}"
) &
forget_pid=$!
sleep 0.2
[[ ! -e "${forget_finished}" ]] \
  || fail "forget completed while restore still consumed the bundle"
forget_status=0
wait "${forget_restore_pid}" || forget_status=1
wait "${forget_pid}" || forget_status=1
if [[ "${forget_status}" != 0 ]]; then
  cat "${tmpdir}/forget-race-restore.err" "${tmpdir}/forget-race-forget.err" >&2
  fail "serialized restore and forget did not both succeed"
fi
assert_file "${forget_race_target}/.worktree-data/fresh-seed"
[[ ! -e "${forget_race_target}/.worktree-data/snapshot-v1" ]] \
  || fail "forget did not remove the bundle after restore released it"

# A corrupt bundle fails closed before Docker or target volumes are touched.
printf tampered >>"${bundle}/mongodb_data.tar.gz"
: >"${tmpdir}/docker-calls"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" FAKE_RUNNING_SERVICES='' \
  bash "${SCRIPT}" restore 2>"${tmpdir}/restore-error"; then
  fail "corrupt bundle restored successfully"
fi
grep -q checksum "${tmpdir}/restore-error" || fail "corrupt bundle error was unclear"
[[ ! -s "${tmpdir}/docker-calls" ]] || fail "corrupt bundle touched Docker"

env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" bash "${SCRIPT}" forget >/dev/null
[[ ! -e "${target}/.worktree-data/snapshot-v1" ]] || fail "snapshot bundle was not forgotten"
assert_file "${target}/.worktree-data/fresh-seed"

# Purge/forget is an explicit fresh-seed choice, not an invitation to copy the
# primary again on the next bootstrap.
: >"${tmpdir}/docker-calls"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" \
  bash "${SCRIPT}" snapshot-primary >"${tmpdir}/forgotten-output"
grep -qi 'fresh.seed choice' "${tmpdir}/forgotten-output" \
  || fail "saved fresh-seed choice was not explained"
[[ ! -e "${target}/.worktree-data/snapshot-v1" ]] \
  || fail "forgotten snapshot was recreated from the primary"
! grep -q ' archive ' "${tmpdir}/docker-calls" \
  || fail "fresh-seed choice still archived primary data"

# The primary checkout never snapshots itself.
env "${common_env[@]}" ATTESTA_ROOT_DIR="${primary}" bash "${SCRIPT}" snapshot-primary >/dev/null

# A mid-snapshot failure remains atomic and always restores the primary running set.
failure_target="${tmpdir}/failure-target"
mkdir -p "${failure_target}/scripts" "${failure_target}/deployment"
cp "${ROOT}/scripts/worktree-env.sh" "${failure_target}/scripts/worktree-env.sh"
cp "${ROOT}/scripts/dev-ports.env" "${failure_target}/scripts/dev-ports.env"
: >"${failure_target}/deployment/docker-compose.local.yaml"
failure_project="$(env "${common_env[@]}" ATTESTA_ROOT_DIR="${failure_target}" \
  bash "${failure_target}/scripts/worktree-env.sh" project-name)"
: >"${tmpdir}/docker-calls"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${failure_target}" \
  FAKE_ARCHIVE_FAILURE=appwrite-functions bash "${SCRIPT}" snapshot-primary >/dev/null 2>&1; then
  fail "partial snapshot unexpectedly succeeded"
fi
[[ ! -e "${failure_target}/.worktree-data/snapshot-v1" ]] \
  || fail "partial snapshot was published"
grep -q ' stop appwrite mongodb mariadb$' "${tmpdir}/docker-calls" \
  || fail "failed snapshot did not quiesce the primary"
grep -q ' start appwrite mongodb mariadb$' "${tmpdir}/docker-calls" \
  || fail "failed snapshot did not restore primary state"

# A mid-restore failure removes all partially restored target volumes.
: >"${tmpdir}/docker-calls"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${failure_target}" bash "${SCRIPT}" snapshot-primary >/dev/null
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${failure_target}" FAKE_RUNNING_SERVICES='' \
  FAKE_EXTRACT_FAILURE=appwrite-functions bash "${SCRIPT}" restore >/dev/null 2>&1; then
  fail "partial restore unexpectedly succeeded"
fi
if find "${tmpdir}/volumes" -maxdepth 1 -type d -name "${failure_project}_*" | grep -q .; then
  fail "partial restore volumes were not removed"
fi
grep -Fq "volume rm -f ${failure_project}_" "${tmpdir}/docker-calls" \
  || fail "failed restore did not clean up target volumes"

# A completely fresh primary has nothing coherent to copy and uses normal seed startup.
fresh_target="${tmpdir}/fresh-target"
mkdir -p "${fresh_target}/scripts" "${fresh_target}/deployment"
cp "${ROOT}/scripts/worktree-env.sh" "${fresh_target}/scripts/worktree-env.sh"
cp "${ROOT}/scripts/dev-ports.env" "${fresh_target}/scripts/dev-ports.env"
: >"${fresh_target}/deployment/docker-compose.local.yaml"
rm -rf -- "${tmpdir}/volumes"/"${primary_project}"_*
env "${common_env[@]}" ATTESTA_ROOT_DIR="${fresh_target}" \
  bash "${SCRIPT}" snapshot-primary >"${tmpdir}/fresh-output"
grep -qi 'no primary durable data.*fresh seed' "${tmpdir}/fresh-output" \
  || fail "fresh-primary fallback was not explained"
[[ ! -e "${fresh_target}/.worktree-data/snapshot-v1" ]] \
  || fail "fresh primary produced an empty snapshot"
assert_file "${fresh_target}/.worktree-data/fresh-seed"

# Once a first bootstrap chooses a fresh seed, later primary data must not be
# copied into that worktree.
for logical in "${durable_volumes[@]}"; do
  mkdir -p "${tmpdir}/volumes/${primary_project}_${logical}"
  : >"${tmpdir}/volumes/${primary_project}_${logical}/nonempty"
done
: >"${tmpdir}/docker-calls"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${fresh_target}" \
  bash "${SCRIPT}" snapshot-primary >/dev/null
[[ ! -e "${fresh_target}/.worktree-data/snapshot-v1" ]] \
  || fail "later bootstrap replaced the explicit fresh-seed choice"
! grep -q ' archive ' "${tmpdir}/docker-calls" \
  || fail "later bootstrap archived primary data after fresh-seed choice"

# Existing target state wins even when no bundle or choice marker exists.
state_target="${tmpdir}/state-target"
mkdir -p "${state_target}/scripts" "${state_target}/deployment"
cp "${ROOT}/scripts/worktree-env.sh" "${state_target}/scripts/worktree-env.sh"
cp "${ROOT}/scripts/dev-ports.env" "${state_target}/scripts/dev-ports.env"
: >"${state_target}/deployment/docker-compose.local.yaml"
: >"${tmpdir}/docker-calls"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${state_target}" \
  FAKE_ALL_SERVICES=mongodb bash "${SCRIPT}" snapshot-primary >"${tmpdir}/state-output"
grep -qi 'already contains data' "${tmpdir}/state-output" \
  || fail "existing target state was not preserved"
[[ ! -e "${state_target}/.worktree-data/snapshot-v1" ]] \
  || fail "existing target state triggered a primary snapshot"
! grep -q ' archive ' "${tmpdir}/docker-calls" \
  || fail "existing target state still archived primary data"

# Any partial primary volume set is incoherent and must fail closed.
partial_target="${tmpdir}/partial-target"
mkdir -p "${partial_target}/scripts" "${partial_target}/deployment"
cp "${ROOT}/scripts/worktree-env.sh" "${partial_target}/scripts/worktree-env.sh"
cp "${ROOT}/scripts/dev-ports.env" "${partial_target}/scripts/dev-ports.env"
: >"${partial_target}/deployment/docker-compose.local.yaml"
rm -rf -- "${tmpdir}/volumes"/"${primary_project}"_*
mkdir -p "${tmpdir}/volumes/${primary_project}_mongodb_data"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${partial_target}" \
  bash "${SCRIPT}" snapshot-primary >"${tmpdir}/partial-output" 2>"${tmpdir}/partial-error"; then
  fail "partial primary volume set unexpectedly succeeded"
fi
grep -qi 'partial.*durable volume' "${tmpdir}/partial-error" \
  || fail "partial primary volume error was unclear"
[[ ! -e "${partial_target}/.worktree-data/snapshot-v1" ]] \
  || fail "partial primary volume set published a snapshot"

echo "ok: worktree-data_test"
