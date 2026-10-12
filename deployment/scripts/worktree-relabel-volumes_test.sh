#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SCRIPT="${ROOT}/scripts/worktree-relabel-volumes.sh"

fail() { echo "FAIL: $*" >&2; exit 1; }

tmpdir="$(mktemp -d)"
trap 'rm -rf -- "$tmpdir"' EXIT
target="${tmpdir}/target"
mkdir -p "${target}/scripts" "${tmpdir}/bin" "${tmpdir}/volumes" "${tmpdir}/git-common"
cp "${ROOT}/scripts/worktree-env.sh" "${target}/scripts/worktree-env.sh"
cp "${ROOT}/scripts/dev-ports.env" "${target}/scripts/dev-ports.env"
mkdir -p "${target}/deployment"
: >"${target}/deployment/docker-compose.local.yaml"

cat >"${tmpdir}/bin/git" <<'EOF'
#!/usr/bin/env bash
if [[ " $* " == *" --git-common-dir "* ]]; then
  printf '%s\n' "${FAKE_GIT_COMMON_DIR}"
  exit 0
fi
if [[ " $* " == *" --is-inside-work-tree "* ]]; then
  exit 0
fi
if [[ " $* " == *" worktree list --porcelain "* ]]; then
  printf 'worktree %s\nHEAD deadbeef\nbranch refs/heads/main\n' "${FAKE_PRIMARY:-${FAKE_GIT_COMMON_DIR}}"
  exit 0
fi
if [[ " $* " == *" symbolic-ref "* ]]; then
  echo main
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

if [[ "${1:-}" == run ]]; then
  mount="" previous=""
  for arg in "$@"; do
    [[ "$previous" == -v ]] && mount="$arg"
    previous="$arg"
  done
  volume="${mount%%:*}"
  logical="${volume#*_}"
  case " $* " in
    *" archive "*)
      printf 'archive:%s' "$logical"
      ;;
    *" extract "*)
      cat >/dev/null
      : >"${FAKE_VOLUMES_DIR}/${volume}/nonempty"
      ;;
  esac
  exit 0
fi

case " $* " in
  *" config --volumes "*)
    if [[ -n "${FAKE_COMPOSE_VOLUMES:-}" ]]; then
      # shellcheck disable=SC2086
      printf '%s\n' ${FAKE_COMPOSE_VOLUMES}
    else
      printf '%s\n' mongodb_data appwrite-cache
    fi
    ;;
  *" ps --status running --services "*) printf '%s' "${FAKE_RUNNING_SERVICES-}" ;;
esac
EOF
chmod +x "${tmpdir}/bin/bash" "${tmpdir}/bin/git" "${tmpdir}/bin/docker"

common_env=(
  -u COMPOSE_PROJECT_NAME
  PATH="${tmpdir}/bin:${PATH}"
  FAKE_GIT_COMMON_DIR="${tmpdir}/git-common"
  FAKE_VOLUMES_DIR="${tmpdir}/volumes"
  DOCKER_CALLS_FILE="${tmpdir}/docker-calls"
)

prepare_checkout() {
  local checkout="$1"
  mkdir -p "${checkout}/scripts" "${checkout}/deployment"
  cp "${ROOT}/scripts/worktree-env.sh" "${checkout}/scripts/worktree-env.sh"
  cp "${ROOT}/scripts/dev-ports.env" "${checkout}/scripts/dev-ports.env"
  : >"${checkout}/deployment/docker-compose.local.yaml"
}

relabel_target="${tmpdir}/relabel-target"
prepare_checkout "${relabel_target}"
relabel_project="$(env "${common_env[@]}" ATTESTA_ROOT_DIR="${relabel_target}" \
  bash "${relabel_target}/scripts/worktree-env.sh" project-name)"
relabel_repository="$(env "${common_env[@]}" ATTESTA_ROOT_DIR="${relabel_target}" \
  bash "${relabel_target}/scripts/worktree-env.sh" repository-id)"
relabel_root="$(cd "${relabel_target}" && pwd -P)"
mkdir -p "${tmpdir}/volumes/${relabel_project}_mongodb_data" \
  "${tmpdir}/volumes/${relabel_project}_appwrite-cache"
printf '|||\n' >"${tmpdir}/volumes/${relabel_project}_mongodb_data/.ownership"
printf '|||\n' >"${tmpdir}/volumes/${relabel_project}_appwrite-cache/.ownership"
: >"${tmpdir}/volumes/${relabel_project}_mongodb_data/nonempty"
: >"${tmpdir}/volumes/${relabel_project}_appwrite-cache/nonempty"
: >"${tmpdir}/docker-calls"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${relabel_target}" FAKE_RUNNING_SERVICES='' \
  FAKE_COMPOSE_VOLUMES='mongodb_data appwrite-cache' \
  bash "${SCRIPT}" >"${tmpdir}/relabel-output"
grep -qi 'relabeled 2 Compose volume' "${tmpdir}/relabel-output" \
  || fail "legacy volumes were not relabeled: $(cat "${tmpdir}/relabel-output")"
grep -Fxq "true|${relabel_project}|${relabel_root}|${relabel_repository}" \
  "${tmpdir}/volumes/${relabel_project}_mongodb_data/.ownership" \
  || fail "mongodb volume did not receive Attesta ownership labels"
grep -Fxq "true|${relabel_project}|${relabel_root}|${relabel_repository}" \
  "${tmpdir}/volumes/${relabel_project}_appwrite-cache/.ownership" \
  || fail "cache volume did not receive Attesta ownership labels"
[[ -f "${tmpdir}/volumes/${relabel_project}_mongodb_data/nonempty" ]] \
  || fail "relabel lost mongodb data"

: >"${tmpdir}/docker-calls"
env "${common_env[@]}" ATTESTA_ROOT_DIR="${relabel_target}" FAKE_RUNNING_SERVICES='' \
  FAKE_COMPOSE_VOLUMES='mongodb_data appwrite-cache' \
  bash "${SCRIPT}" >"${tmpdir}/relabel-skip-output"
[[ ! -s "${tmpdir}/relabel-skip-output" ]] \
  || fail "second relabel should be a no-op: $(cat "${tmpdir}/relabel-skip-output")"
! grep -q ' volume create ' "${tmpdir}/docker-calls" \
  || fail "already-labeled volumes were recreated"

# Foreign ownership must fail closed instead of being stolen.
foreign_target="${tmpdir}/foreign-target"
prepare_checkout "${foreign_target}"
foreign_project="$(env "${common_env[@]}" ATTESTA_ROOT_DIR="${foreign_target}" \
  bash "${foreign_target}/scripts/worktree-env.sh" project-name)"
mkdir -p "${tmpdir}/volumes/${foreign_project}_mongodb_data"
printf 'true|%s|/other/worktree|repo-other\n' "${foreign_project}" \
  >"${tmpdir}/volumes/${foreign_project}_mongodb_data/.ownership"
if env "${common_env[@]}" ATTESTA_ROOT_DIR="${foreign_target}" FAKE_RUNNING_SERVICES='' \
  FAKE_COMPOSE_VOLUMES='mongodb_data' \
  bash "${SCRIPT}" >"${tmpdir}/foreign-output" 2>"${tmpdir}/foreign-error"; then
  fail "foreign volume ownership was overwritten"
fi
grep -qi 'ownership' "${tmpdir}/foreign-error" \
  || fail "foreign ownership rejection was unclear"

if env "${common_env[@]}" ATTESTA_ROOT_DIR="${relabel_target}" \
  FAKE_RUNNING_SERVICES=mongodb FAKE_COMPOSE_VOLUMES='mongodb_data' \
  bash "${SCRIPT}" >"${tmpdir}/running-output" 2>"${tmpdir}/running-error"; then
  fail "relabel ran while Compose services were up"
fi
grep -qi 'stopped Compose stack' "${tmpdir}/running-error" \
  || fail "running-stack rejection was unclear"

if env "${common_env[@]}" ATTESTA_ROOT_DIR="${relabel_target}" \
  DOCKER_COMPOSE_VERSION=2.24.3 FAKE_RUNNING_SERVICES='' \
  bash "${SCRIPT}" >"${tmpdir}/old-compose.out" 2>"${tmpdir}/old-compose.err"; then
  fail "Compose older than 2.24.4 was accepted"
fi
grep -q 'Docker Compose 2.24.4 or newer is required' "${tmpdir}/old-compose.err" \
  || fail "old Compose rejection did not explain the required version"

echo "ok: worktree-relabel-volumes_test"
