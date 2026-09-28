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
if [[ " $* " == *" worktree list --porcelain "* ]]; then
  printf 'worktree %s\nHEAD deadbeef\nbranch refs/heads/main\n' "${FAKE_PRIMARY}"
  exit 0
fi
exit 1
EOF

cat >"${tmpdir}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${DOCKER_CALLS_FILE}"

if [[ "${1:-}" == volume && "${2:-}" == ls ]]; then
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
      mkdir -p "${FAKE_RESTORE_DIR}"
      cat >"${FAKE_RESTORE_DIR}/${logical}.tar.gz"
      : >"${FAKE_VOLUMES_DIR}/${volume}/nonempty"
      ;;
  esac
  exit 0
fi

case " $* " in
  *" ps --status running --services "*) printf '%s' "${FAKE_RUNNING_SERVICES-appwrite
mongodb
mariadb
}" ;;
  *" ps --all --services "*) printf '%s' "${FAKE_ALL_SERVICES:-}" ;;
  *" mariadb-admin ping "*) echo 'mysqld is alive' ;;
  *" exec -T mariadb "*" mariadb --user=root "*)
    mkdir -p "${FAKE_RESTORE_DIR}"
    printf '%s\n' "$*" >"${FAKE_RESTORE_DIR}/sanitized"
    ;;
esac
EOF
chmod +x "${tmpdir}/bin/git" "${tmpdir}/bin/docker"

common_env=(
  PATH="${tmpdir}/bin:${PATH}"
  FAKE_PRIMARY="${primary}"
  FAKE_VOLUMES_DIR="${tmpdir}/volumes"
  FAKE_RESTORE_DIR="${tmpdir}/restored"
  DOCKER_CALLS_FILE="${tmpdir}/docker-calls"
)

primary_project="$(env "${common_env[@]}" ATTESTA_ROOT_DIR="${primary}" \
  bash "${primary}/scripts/worktree-env.sh" project-name)"
target_project="$(env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" \
  bash "${target}/scripts/worktree-env.sh" project-name)"
[[ "${primary_project}" =~ ^primary-[0-9a-f]{8}$ ]] \
  || fail "primary Compose identity is not path-qualified: ${primary_project}"
[[ "${target_project}" != "${primary_project}" ]] \
  || fail "target reused the primary Compose identity"

durable_volumes=(
  mongodb_data appwrite-mariadb appwrite-uploads appwrite-imports
  appwrite-functions appwrite-sites appwrite-builds
)
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

# Existing worktree state wins; restore never overwrites or partially merges it.
: >"${tmpdir}/docker-calls"
rm -rf -- "${tmpdir}/restored"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${target}" FAKE_RUNNING_SERVICES='' bash "${SCRIPT}" restore >/dev/null
if [[ -e "${tmpdir}/restored" ]]; then
  cat "${tmpdir}/docker-calls" >&2
  fail "restore overwrote existing worktree data"
fi
! grep -q ' extract ' "${tmpdir}/docker-calls" || fail "restore touched a developed worktree"

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
[[ ! -e "${target}/.worktree-data" ]] || fail "snapshot bundle was not forgotten"

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

# Any partial primary volume set is incoherent and must fail closed.
mkdir -p "${tmpdir}/volumes/${primary_project}_mongodb_data"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${fresh_target}" \
  bash "${SCRIPT}" snapshot-primary >"${tmpdir}/partial-output" 2>"${tmpdir}/partial-error"; then
  fail "partial primary volume set unexpectedly succeeded"
fi
grep -qi 'partial.*durable volume' "${tmpdir}/partial-error" \
  || fail "partial primary volume error was unclear"
[[ ! -e "${fresh_target}/.worktree-data/snapshot-v1" ]] \
  || fail "partial primary volume set published a snapshot"

echo "ok: worktree-data_test"
