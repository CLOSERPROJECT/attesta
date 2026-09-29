#!/usr/bin/env bash
# Capture and restore one coherent, offline snapshot of Attesta's durable data.
set -euo pipefail
umask 077

ROOT_DIR="${ATTESTA_ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/require-compose-version.sh
source "${SCRIPT_ROOT}/scripts/require-compose-version.sh"
DATA_DIR="${ROOT_DIR}/.worktree-data"
BUNDLE_DIR="${DATA_DIR}/snapshot-v1"
MANIFEST="${BUNDLE_DIR}/manifest"
FRESH_SEED_MARKER="${DATA_DIR}/fresh-seed"
HELPER_IMAGE="${ATTESTA_VOLUME_HELPER_IMAGE:-alpine:3.20}"
FORMAT="attesta-worktree-data"
VERSION="1"
DURABLE_VOLUMES=(
  mongodb_data
  appwrite-mariadb
  appwrite-uploads
  appwrite-imports
  appwrite-functions
  appwrite-sites
  appwrite-builds
)
SNAPSHOT_LOCK_DIR=""
SNAPSHOT_LOCK_HELD=0
SNAPSHOT_PRIMARY_ROOT=""
SNAPSHOT_RUNNING_SERVICES=()
SNAPSHOT_QUIESCED=0
SNAPSHOT_TEMP_BUNDLE=""
RESTORE_CREATED_VOLUMES=()
RESTORE_TOUCHED_REUSED_VOLUMES=()
RESTORE_COMPLETE=0
RESTORE_MARIADB_MAY_EXIST=0

usage() {
  cat <<'USAGE'
Usage: scripts/worktree-data.sh <command>

Commands:
  snapshot-primary  Capture one immutable, coherent snapshot from the primary checkout
  restore           Restore the bundle only into a new, stopped worktree stack
  sanitize          Remove copied Appwrite sessions, certificates, and routing rules
  forget            Delete this checkout's saved snapshot bundle
USAGE
}

fail() {
  echo "error: $*" >&2
  exit 1
}

git_common_dir() {
  local common_dir
  common_dir="$(git -C "${ROOT_DIR}" rev-parse --path-format=absolute --git-common-dir)" \
    || fail "could not resolve the Git common directory"
  if [[ "${common_dir}" != /* ]]; then
    common_dir="${ROOT_DIR}/${common_dir}"
  fi
  [[ -d "${common_dir}" ]] \
    || fail "Git common directory does not exist: ${common_dir}"
  (cd "${common_dir}" && pwd -P)
}

acquire_snapshot_lock() {
  local timeout="${ATTESTA_SNAPSHOT_LOCK_TIMEOUT_SECONDS:-300}"
  [[ "${timeout}" =~ ^[0-9]+$ ]] \
    || fail "ATTESTA_SNAPSHOT_LOCK_TIMEOUT_SECONDS must be a non-negative integer"

  SNAPSHOT_LOCK_DIR="$(git_common_dir)/attesta-worktree-snapshot.lock"
  local deadline=$((SECONDS + timeout))
  local owner=""
  while ! mkdir "${SNAPSHOT_LOCK_DIR}" 2>/dev/null; do
    owner=""
    if [[ -r "${SNAPSHOT_LOCK_DIR}/pid" ]]; then
      IFS= read -r owner <"${SNAPSHOT_LOCK_DIR}/pid" || true
    fi
    if [[ "${owner}" =~ ^[0-9]+$ ]] && ! kill -0 "${owner}" 2>/dev/null; then
      rm -f -- "${SNAPSHOT_LOCK_DIR}/pid"
      rmdir "${SNAPSHOT_LOCK_DIR}" 2>/dev/null || true
      continue
    fi
    if (( SECONDS >= deadline )); then
      fail "timed out after ${timeout}s waiting for repository snapshot lock: ${SNAPSHOT_LOCK_DIR}"
    fi
    sleep 0.1
  done
  printf '%s\n' "$$" >"${SNAPSHOT_LOCK_DIR}/pid"
  SNAPSHOT_LOCK_HELD=1
}

release_snapshot_lock() {
  [[ "${SNAPSHOT_LOCK_HELD}" == "1" ]] || return 0
  local owner=""
  if [[ -r "${SNAPSHOT_LOCK_DIR}/pid" ]]; then
    IFS= read -r owner <"${SNAPSHOT_LOCK_DIR}/pid" || true
  fi
  if [[ "${owner}" == "$$" ]]; then
    rm -f -- "${SNAPSHOT_LOCK_DIR}/pid"
    rmdir "${SNAPSHOT_LOCK_DIR}" 2>/dev/null || true
  fi
  SNAPSHOT_LOCK_HELD=0
}

primary_worktree() {
  git -C "${ROOT_DIR}" worktree list --porcelain \
    | awk '/^worktree / { print substr($0, 10); exit }'
}

compose_exports() {
  local checkout_root="$1"
  ATTESTA_ROOT_DIR="${checkout_root}" \
    bash "${ROOT_DIR}/scripts/worktree-env.sh" export
}

compose_project() {
  local checkout_root="$1"
  local exports
  if ! exports="$(compose_exports "${checkout_root}")"; then
    fail "could not resolve the Compose environment for ${checkout_root}"
  fi
  (
    eval "${exports}"
    printf '%s\n' "${COMPOSE_PROJECT_NAME}"
  )
}

repository_identity() {
  local checkout_root="$1"
  ATTESTA_ROOT_DIR="${checkout_root}" \
    bash "${ROOT_DIR}/scripts/worktree-env.sh" repository-id
}

compose_for_root() {
  local checkout_root="$1"
  shift
  local exports repository
  if ! exports="$(compose_exports "${checkout_root}")"; then
    fail "could not resolve the Compose environment for ${checkout_root}"
  fi
  if ! repository="$(repository_identity "${checkout_root}")"; then
    fail "could not resolve the repository identity for ${checkout_root}"
  fi
  (
    eval "${exports}"
    export ATTESTA_WORKTREE_ROOT ATTESTA_REPOSITORY_ID
    ATTESTA_WORKTREE_ROOT="$(cd "${checkout_root}" && pwd -P)"
    ATTESTA_REPOSITORY_ID="${repository}"
    if [[ "${ATTESTA_DATA_DISABLE_SEED:-0}" == "1" ]]; then
      export APPWRITE_RESTORE_SEED_SQL=false
    fi
    docker compose \
      --project-name "${COMPOSE_PROJECT_NAME}" \
      -f "${checkout_root}/deployment/docker-compose.local.yaml" \
      "$@"
  )
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    fail "sha256sum or shasum is required"
  fi
}

volume_name() {
  local project="$1"
  local logical="$2"
  local names
  if ! names="$(docker volume ls \
      --filter "label=com.docker.compose.project=${project}" \
      --filter "label=com.docker.compose.volume=${logical}" \
      --format '{{.Name}}')"; then
    echo "error: could not enumerate Docker volume ${project}/${logical}" >&2
    return 1
  fi
  local count
  count="$(printf '%s\n' "${names}" | awk 'NF { count++ } END { print count + 0 }')"
  [[ "${count}" -le 1 ]] || fail "multiple Docker volumes found for ${project}/${logical}"
  printf '%s' "${names}"
}

# Pre-worktree local Compose used the directory name "deployment" as its
# project. Those volumes are still labeled and must not be ignored as "no data".
legacy_deployment_volume_count() {
  local logical names count
  local found=0
  for logical in "${DURABLE_VOLUMES[@]}"; do
    if ! names="$(docker volume ls \
        --filter "label=com.docker.compose.project=deployment" \
        --filter "label=com.docker.compose.volume=${logical}" \
        --format '{{.Name}}')"; then
      fail "could not enumerate legacy deployment volume ${logical}"
    fi
    count="$(printf '%s\n' "${names}" | awk 'NF { count++ } END { print count + 0 }')"
    [[ "${count}" -le 1 ]] || fail "multiple legacy deployment volumes for ${logical}"
    [[ "${count}" -eq 0 ]] || found=$((found + 1))
  done
  printf '%s\n' "${found}"
}

archive_volume() {
  local volume="$1"
  local destination="$2"
  docker run --rm -v "${volume}:/source:ro" "${HELPER_IMAGE}" \
    sh -c 'cd /source && exec tar -czf - .' archive >"${destination}"
  [[ -s "${destination}" ]] || fail "archive for ${volume} is empty"
}

volume_is_empty() {
  local volume="$1"
  docker run --rm -v "${volume}:/source:ro" "${HELPER_IMAGE}" \
    sh -c 'test -z "$(find /source -mindepth 1 -print -quit)"' test-empty
}

empty_volume() {
  local volume="$1"
  docker run --rm -v "${volume}:/target" "${HELPER_IMAGE}" \
    sh -c 'find /target -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +' clear
  volume_is_empty "${volume}"
}

assert_runtime_network_quiescent() {
  local project="$1"
  local network="${project}_runtimes"
  local networks attached
  if ! networks="$(docker network ls \
    --filter "name=^${network}$" \
    --format '{{.Name}}')"; then
    fail "could not inspect the Appwrite runtimes network"
  fi
  if ! grep -Fqx -- "${network}" <<<"${networks}"; then
    return 0
  fi
  if ! attached="$(docker network inspect \
    --format '{{range $id, $_ := .Containers}}{{println $id}}{{end}}' \
    "${network}")"; then
    fail "could not inspect attached Appwrite runtime containers on ${network}"
  fi
  if [[ -n "${attached//[[:space:]]/}" ]]; then
    fail "Appwrite runtime containers remain attached to ${network}; stop function executions before snapshotting"
  fi
}

extract_volume() {
  local archive="$1"
  local volume="$2"
  docker run --rm -i -v "${volume}:/target" "${HELPER_IMAGE}" \
    sh -c 'cd /target && exec tar -xzf -' extract <"${archive}"
}

verify_bundle() {
  [[ -f "${MANIFEST}" ]] || fail "snapshot bundle has no manifest"
  grep -Fqx "format ${FORMAT}" "${MANIFEST}" \
    || fail "snapshot bundle format is invalid"
  grep -Fqx "version ${VERSION}" "${MANIFEST}" \
    || fail "snapshot bundle version is unsupported"

  local logical artifact expected actual entries
  for logical in "${DURABLE_VOLUMES[@]}"; do
    artifact="${logical}.tar.gz"
    [[ -s "${BUNDLE_DIR}/${artifact}" ]] \
      || fail "snapshot bundle is partial: missing ${artifact}"
    entries="$(awk -v file="${artifact}" '$1 == "sha256" && $2 == file { count++ } END { print count + 0 }' "${MANIFEST}")"
    [[ "${entries}" == "1" ]] \
      || fail "snapshot manifest must contain one checksum for ${artifact}"
    expected="$(awk -v file="${artifact}" '$1 == "sha256" && $2 == file { print $3 }' "${MANIFEST}")"
    actual="$(sha256_file "${BUNDLE_DIR}/${artifact}")"
    [[ "${expected}" == "${actual}" ]] \
      || fail "snapshot checksum mismatch for ${artifact}"
  done
}

write_manifest() {
  local directory="$1"
  local source_root="$2"
  local source_project="$3"
  local manifest="${directory}/manifest"
  {
    echo "format ${FORMAT}"
    echo "version ${VERSION}"
    printf 'created_at %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
    echo "source_root ${source_root}"
    echo "source_project ${source_project}"
    local logical artifact
    for logical in "${DURABLE_VOLUMES[@]}"; do
      artifact="${logical}.tar.gz"
      printf 'sha256 %s %s\n' "${artifact}" "$(sha256_file "${directory}/${artifact}")"
    done
  } >"${manifest}"
}

write_fresh_seed_choice() {
  local reason="$1"
  mkdir -p "${DATA_DIR}"
  local temporary="${DATA_DIR}/.fresh-seed.tmp.$$"
  {
    echo "format ${FORMAT}"
    echo "version ${VERSION}"
    printf 'created_at %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
    printf 'reason %s\n' "${reason}"
  } >"${temporary}"
  mv "${temporary}" "${FRESH_SEED_MARKER}"
}

snapshot_primary() {
  local primary
  primary="$(primary_worktree)"
  [[ -n "${primary}" && -d "${primary}" ]] \
    || fail "could not resolve primary worktree root"
  if [[ "$(cd "${primary}" && pwd -P)" == "$(cd "${ROOT_DIR}" && pwd -P)" ]]; then
    echo "primary worktree: no data snapshot needed"
    return 0
  fi

  SNAPSHOT_PRIMARY_ROOT="${primary}"
  SNAPSHOT_RUNNING_SERVICES=()
  SNAPSHOT_QUIESCED=0
  SNAPSHOT_TEMP_BUNDLE=""

  resume_primary() {
    if [[ "${SNAPSHOT_QUIESCED}" == "1" ]]; then
      if [[ "${#SNAPSHOT_RUNNING_SERVICES[@]}" -gt 0 ]]; then
        compose_for_root "${SNAPSHOT_PRIMARY_ROOT}" start "${SNAPSHOT_RUNNING_SERVICES[@]}" >/dev/null
      fi
      SNAPSHOT_QUIESCED=0
    fi
  }

  cleanup_snapshot() {
    # Keep rollback state at script scope because Ubuntu Bash unwinds function
    # locals before running the EXIT trap after a fatal command.
    if [[ -n "${SNAPSHOT_TEMP_BUNDLE}" ]]; then
      rm -rf -- "${SNAPSHOT_TEMP_BUNDLE}"
    fi
    if [[ "${SNAPSHOT_QUIESCED}" == "1" ]]; then
      resume_primary >/dev/null 2>&1 || \
        echo "error: could not restore the primary Compose state" >&2
    fi
    release_snapshot_lock
  }
  finish_snapshot() {
    release_snapshot_lock
    trap - EXIT INT TERM HUP
  }
  trap cleanup_snapshot EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  trap 'exit 129' HUP

  acquire_snapshot_lock

  if [[ -f "${FRESH_SEED_MARKER}" ]]; then
    echo "keeping saved fresh-seed choice for this worktree"
    finish_snapshot
    return 0
  fi

  if [[ -e "${BUNDLE_DIR}" ]]; then
    verify_bundle
    echo "keeping existing ${BUNDLE_DIR}"
    finish_snapshot
    return 0
  fi

  command -v docker >/dev/null 2>&1 \
    || fail "Docker is required to snapshot primary worktree data"

  local target_project
  if ! target_project="$(compose_project "${ROOT_DIR}")"; then
    fail "could not resolve the target Compose project"
  fi
  # Volumes without a local bundle or fresh-seed marker are leftovers from a
  # previous checkout at this path (same Compose project name). Keeping them
  # skips the primary copy and leaves an empty Appwrite that fails identity
  # bootstrap. Remove them and continue.
  if target_has_state "${target_project}"; then
    echo "removing orphan Compose volumes from a previous checkout at this path"
    remove_orphan_target_state "${target_project}"
  fi

  local project
  if ! project="$(compose_project "${primary}")"; then
    fail "could not resolve the primary Compose project"
  fi
  local source_volumes=()
  local logical resolved
  local found_volumes=0
  for logical in "${DURABLE_VOLUMES[@]}"; do
    if ! resolved="$(volume_name "${project}" "${logical}")"; then
      fail "could not inspect primary durable volume ${logical}"
    fi
    source_volumes+=("${resolved}")
    [[ -z "${resolved}" ]] || found_volumes=$((found_volumes + 1))
  done
  if [[ "${found_volumes}" -eq 0 ]]; then
    local legacy_found=0
    if [[ "${project}" != "deployment" ]]; then
      legacy_found="$(legacy_deployment_volume_count)"
    fi
    if [[ "${legacy_found}" -gt 0 ]]; then
      fail "primary Compose project ${project} has no durable volumes, but legacy project \"deployment\" still has ${legacy_found} Attesta volume(s); add COMPOSE_PROJECT_NAME=deployment to the primary .env.worktree once so bootstrap can copy that data"
    fi
    write_fresh_seed_choice "primary-had-no-durable-volumes"
    echo "no primary durable data found; worktree will use the fresh seed"
    finish_snapshot
    return 0
  fi
  [[ "${found_volumes}" -eq "${#DURABLE_VOLUMES[@]}" ]] \
    || fail "partial primary durable volume set: found ${found_volumes} of ${#DURABLE_VOLUMES[@]}"

  local running_output
  if ! running_output="$(compose_for_root "${primary}" ps --status running --services)"; then
    fail "could not determine the primary Compose running services"
  fi
  while IFS= read -r service; do
    [[ -n "${service}" ]] && SNAPSHOT_RUNNING_SERVICES+=("${service}")
  done <<<"${running_output}"

  mkdir -p "${DATA_DIR}"
  SNAPSHOT_TEMP_BUNDLE="${DATA_DIR}/.snapshot-v1.tmp.$$"
  mkdir "${SNAPSHOT_TEMP_BUNDLE}"

  if [[ "${#SNAPSHOT_RUNNING_SERVICES[@]}" -gt 0 ]]; then
    SNAPSHOT_QUIESCED=1
    echo "quiescing primary Compose services"
    compose_for_root "${primary}" stop "${SNAPSHOT_RUNNING_SERVICES[@]}" >/dev/null
  fi

  assert_runtime_network_quiescent "${project}"

  echo "snapshotting primary durable volumes"
  local index=0
  for logical in "${DURABLE_VOLUMES[@]}"; do
    archive_volume "${source_volumes[$index]}" "${SNAPSHOT_TEMP_BUNDLE}/${logical}.tar.gz"
    index=$((index + 1))
  done
  write_manifest "${SNAPSHOT_TEMP_BUNDLE}" "${primary}" "${project}"

  # Restore the exact pre-snapshot running set before publishing the bundle.
  resume_primary
  mv "${SNAPSHOT_TEMP_BUNDLE}" "${BUNDLE_DIR}"
  SNAPSHOT_TEMP_BUNDLE=""
  echo "wrote ${BUNDLE_DIR}"
  finish_snapshot
}

target_has_state() {
  local project="$1"
  local existing_services=()
  local services_output
  if ! services_output="$(compose_for_root "${ROOT_DIR}" ps --all --services)"; then
    fail "could not determine whether the target Compose stack already exists"
  fi
  while IFS= read -r service; do
    [[ -n "${service}" ]] && existing_services+=("${service}")
  done <<<"${services_output}"
  [[ "${#existing_services[@]}" -eq 0 ]] || return 0

  local logical resolved
  for logical in "${DURABLE_VOLUMES[@]}"; do
    if ! resolved="$(volume_name "${project}" "${logical}")"; then
      fail "could not inspect target durable volume ${logical}"
    fi
    if [[ -n "${resolved}" ]] && ! volume_is_empty "${resolved}"; then
      return 0
    fi
  done
  return 1
}

remove_orphan_target_state() {
  local project="$1"
  local logical resolved
  compose_for_root "${ROOT_DIR}" down -v --remove-orphans >/dev/null \
    || fail "could not tear down orphan Compose project ${project}"
  for logical in "${DURABLE_VOLUMES[@]}"; do
    if ! resolved="$(volume_name "${project}" "${logical}")"; then
      fail "could not inspect orphan durable volume ${logical}"
    fi
    if [[ -n "${resolved}" ]]; then
      docker volume rm -f "${resolved}" >/dev/null \
        || fail "could not delete orphan volume ${resolved}"
    fi
  done
}

validate_existing_volume_ownership() {
  local volume="$1"
  local project="$2"
  local physical_root="$3"
  local repository="$4"
  local ownership
  if ! ownership="$(docker volume inspect \
    --format '{{ index .Labels "eu.forkbomb.attesta.managed" }}|{{ index .Labels "eu.forkbomb.attesta.project" }}|{{ index .Labels "eu.forkbomb.attesta.worktree" }}|{{ index .Labels "eu.forkbomb.attesta.repository" }}' \
    "${volume}")"; then
    fail "could not verify ownership of pre-existing volume ${volume}"
  fi
  if [[ "${ownership}" != "true|${project}|${physical_root}|${repository}" ]]; then
    fail "pre-existing volume ${volume} is not owned by this worktree"
  fi
}

wait_for_mariadb() {
  local attempt=0
  until compose_for_root "${ROOT_DIR}" exec -T mariadb sh -c \
    'mariadb-admin ping --silent --user=root --password="$MYSQL_ROOT_PASSWORD"' \
    >/dev/null 2>&1; do
    attempt=$((attempt + 1))
    [[ "${attempt}" -lt 30 ]] || fail "Appwrite MariaDB did not become ready"
    sleep 1
  done
}

sanitize_appwrite() {
  wait_for_mariadb
  compose_for_root "${ROOT_DIR}" exec -T mariadb sh -c \
    'exec mariadb --user=root --password="$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" --execute "DELETE FROM _console_certificates; DELETE FROM _console_rules; DELETE FROM _console_sessions; DELETE FROM _1_sessions;"'
}

restore_snapshot() {
  local target_volumes=()
  local target_volume_reused=()
  RESTORE_CREATED_VOLUMES=()
  RESTORE_TOUCHED_REUSED_VOLUMES=()
  RESTORE_COMPLETE=0
  RESTORE_MARIADB_MAY_EXIST=0
  cleanup_restore() {
    # EXIT traps run after Bash has started unwinding function-local scope on
    # some platforms, so rollback state intentionally lives at script scope.
    if [[ "${RESTORE_COMPLETE}" != "1" ]]; then
      if [[ "${RESTORE_MARIADB_MAY_EXIST}" == "1" ]]; then
        ATTESTA_DATA_DISABLE_SEED=1 compose_for_root "${ROOT_DIR}" stop mariadb >/dev/null 2>&1 || true
        ATTESTA_DATA_DISABLE_SEED=1 compose_for_root "${ROOT_DIR}" rm -f mariadb >/dev/null 2>&1 || true
      fi
      if [[ "${#RESTORE_CREATED_VOLUMES[@]}" -gt 0 ]]; then
        docker volume rm -f "${RESTORE_CREATED_VOLUMES[@]}" >/dev/null 2>&1 || true
      fi
      local reused
      for reused in "${RESTORE_TOUCHED_REUSED_VOLUMES[@]:-}"; do
        [[ -n "${reused}" ]] || continue
        if ! empty_volume "${reused}" >/dev/null 2>&1; then
          echo "error: could not clear partially restored pre-existing volume ${reused}" >&2
        fi
      done
    fi
    release_snapshot_lock
  }
  finish_restore() {
    release_snapshot_lock
    trap - EXIT INT TERM HUP
  }
  trap cleanup_restore EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  trap 'exit 129' HUP
  acquire_snapshot_lock

  if [[ -f "${FRESH_SEED_MARKER}" ]]; then
    echo "fresh-seed choice present; no worktree data snapshot will be restored"
    finish_restore
    return 0
  fi
  [[ -e "${BUNDLE_DIR}" ]] || {
    echo "no worktree data snapshot to restore"
    finish_restore
    return 0
  }
  # Integrity is checked before any Docker operation or target mutation.
  verify_bundle
  command -v docker >/dev/null 2>&1 \
    || fail "Docker is required to restore worktree data"

  local project
  if ! project="$(compose_project "${ROOT_DIR}")"; then
    fail "could not resolve the target Compose project"
  fi

  local running_services=()
  local running_output
  if ! running_output="$(compose_for_root "${ROOT_DIR}" ps --status running --services)"; then
    fail "could not determine the target Compose running services"
  fi
  while IFS= read -r service; do
    [[ -n "${service}" ]] && running_services+=("${service}")
  done <<<"${running_output}"
  if [[ "${#running_services[@]}" -gt 0 ]]; then
    # task start / task:dev re-enter restore on every run. A live stack already
    # owns its volumes; only refuse when restore would still need to mutate them.
    if target_has_state "${project}"; then
      echo "Compose stack already running; skipping data restore"
      finish_restore
      return 0
    fi
    fail "restore must run before this worktree's Compose stack starts (stack is up but durable volumes are empty; run task stop, then task start)"
  fi

  local repository
  if ! repository="$(repository_identity "${ROOT_DIR}")"; then
    fail "could not resolve the repository identity for ${ROOT_DIR}"
  fi
  local physical_root
  physical_root="$(cd "${ROOT_DIR}" && pwd -P)"
  if target_has_state "${project}"; then
    echo "worktree already contains data; keeping it"
    finish_restore
    return 0
  fi

  local logical resolved target_name
  for logical in "${DURABLE_VOLUMES[@]}"; do
    if ! resolved="$(volume_name "${project}" "${logical}")"; then
      fail "could not inspect target durable volume ${logical}"
    fi
    if [[ -z "${resolved}" ]]; then
      target_name="${project}_${logical}"
      resolved="$(docker volume create \
        --label "com.docker.compose.project=${project}" \
        --label "com.docker.compose.volume=${logical}" \
        --label "eu.forkbomb.attesta.managed=true" \
        --label "eu.forkbomb.attesta.worktree=${physical_root}" \
        --label "eu.forkbomb.attesta.project=${project}" \
        --label "eu.forkbomb.attesta.repository=${repository}" \
        "${target_name}")"
      RESTORE_CREATED_VOLUMES+=("${resolved}")
      target_volume_reused+=(0)
    else
      validate_existing_volume_ownership \
        "${resolved}" "${project}" "${physical_root}" "${repository}"
      target_volume_reused+=(1)
    fi
    target_volumes+=("${resolved}")
  done

  local index=0
  for logical in "${DURABLE_VOLUMES[@]}"; do
    if [[ "${target_volume_reused[$index]}" == "1" ]]; then
      RESTORE_TOUCHED_REUSED_VOLUMES+=("${target_volumes[$index]}")
    fi
    extract_volume "${BUNDLE_DIR}/${logical}.tar.gz" "${target_volumes[$index]}"
    index=$((index + 1))
  done

  echo "sanitizing copied Appwrite runtime state"
  RESTORE_MARIADB_MAY_EXIST=1
  ATTESTA_DATA_DISABLE_SEED=1 compose_for_root "${ROOT_DIR}" up -d mariadb >/dev/null
  sanitize_appwrite
  ATTESTA_DATA_DISABLE_SEED=1 compose_for_root "${ROOT_DIR}" stop mariadb >/dev/null
  ATTESTA_DATA_DISABLE_SEED=1 compose_for_root "${ROOT_DIR}" rm -f mariadb >/dev/null
  RESTORE_MARIADB_MAY_EXIST=0

  RESTORE_COMPLETE=1
  finish_restore
  echo "restored coherent worktree data snapshot"
}

forget_snapshot() {
  trap release_snapshot_lock EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  trap 'exit 129' HUP
  acquire_snapshot_lock
  write_fresh_seed_choice "snapshot-forgotten"
  rm -rf -- "${BUNDLE_DIR}"
  release_snapshot_lock
  trap - EXIT INT TERM HUP
  echo "removed saved worktree data snapshot and selected the fresh seed"
}

main() {
  local command="${1:-}"
  shift || true
  [[ $# -eq 0 ]] || fail "${command} accepts no arguments"
  case "${command}" in
    snapshot-primary)
      require_compose_version
      snapshot_primary
      ;;
    restore)
      require_compose_version
      restore_snapshot
      ;;
    sanitize)
      require_compose_version
      sanitize_appwrite
      ;;
    forget) forget_snapshot ;;
    ""|-h|--help) usage ;;
    *)
      echo "unknown command: ${command}" >&2
      usage >&2
      exit 1
      ;;
  esac
}

main "$@"
