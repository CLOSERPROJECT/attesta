#!/usr/bin/env bash
# Capture and restore one coherent, offline snapshot of Attesta's durable data.
set -euo pipefail
umask 077

ROOT_DIR="${ATTESTA_ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
DATA_DIR="${ROOT_DIR}/.worktree-data"
BUNDLE_DIR="${DATA_DIR}/snapshot-v1"
MANIFEST="${BUNDLE_DIR}/manifest"
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
  exports="$(compose_exports "${checkout_root}")"
  (
    eval "${exports}"
    printf '%s\n' "${COMPOSE_PROJECT_NAME}"
  )
}

compose_for_root() {
  local checkout_root="$1"
  shift
  local exports
  exports="$(compose_exports "${checkout_root}")"
  (
    eval "${exports}"
    export ATTESTA_WORKTREE_ROOT
    ATTESTA_WORKTREE_ROOT="$(cd "${checkout_root}" && pwd -P)"
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
  names="$(docker volume ls \
    --filter "label=com.docker.compose.project=${project}" \
    --filter "label=com.docker.compose.volume=${logical}" \
    --format '{{.Name}}')"
  local count
  count="$(printf '%s\n' "${names}" | awk 'NF { count++ } END { print count + 0 }')"
  [[ "${count}" -le 1 ]] || fail "multiple Docker volumes found for ${project}/${logical}"
  printf '%s' "${names}"
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

snapshot_primary() {
  local primary
  primary="$(primary_worktree)"
  [[ -n "${primary}" && -d "${primary}" ]] \
    || fail "could not resolve primary worktree root"
  if [[ "$(cd "${primary}" && pwd -P)" == "$(cd "${ROOT_DIR}" && pwd -P)" ]]; then
    echo "primary worktree: no data snapshot needed"
    return 0
  fi
  command -v docker >/dev/null 2>&1 \
    || fail "Docker is required to snapshot primary worktree data"

  if [[ -e "${BUNDLE_DIR}" ]]; then
    verify_bundle
    echo "keeping existing ${BUNDLE_DIR}"
    return 0
  fi

  local project
  project="$(compose_project "${primary}")"
  local source_volumes=()
  local logical resolved
  local found_volumes=0
  for logical in "${DURABLE_VOLUMES[@]}"; do
    resolved="$(volume_name "${project}" "${logical}")"
    source_volumes+=("${resolved}")
    [[ -z "${resolved}" ]] || found_volumes=$((found_volumes + 1))
  done
  if [[ "${found_volumes}" -eq 0 ]]; then
    echo "no primary durable data found; worktree will use the fresh seed"
    return 0
  fi
  [[ "${found_volumes}" -eq "${#DURABLE_VOLUMES[@]}" ]] \
    || fail "partial primary durable volume set: found ${found_volumes} of ${#DURABLE_VOLUMES[@]}"

  local running_services=()
  while IFS= read -r service; do
    [[ -n "${service}" ]] && running_services+=("${service}")
  done < <(compose_for_root "${primary}" ps --status running --services)

  mkdir -p "${DATA_DIR}"
  local temp_bundle="${DATA_DIR}/.snapshot-v1.tmp.$$"
  mkdir "${temp_bundle}"
  local quiesced=0

  resume_primary() {
    if [[ "${quiesced}" == "1" ]]; then
      if [[ "${#running_services[@]}" -gt 0 ]]; then
        compose_for_root "${primary}" start "${running_services[@]}" >/dev/null
      fi
      quiesced=0
    fi
  }

  cleanup_snapshot() {
    rm -rf -- "${temp_bundle}"
    if [[ "${quiesced}" == "1" ]]; then
      resume_primary >/dev/null 2>&1 || \
        echo "error: could not restore the primary Compose state" >&2
    fi
  }
  trap cleanup_snapshot EXIT

  if [[ "${#running_services[@]}" -gt 0 ]]; then
    quiesced=1
    echo "quiescing primary Compose services"
    compose_for_root "${primary}" stop "${running_services[@]}" >/dev/null
  fi

  echo "snapshotting primary durable volumes"
  local index=0
  for logical in "${DURABLE_VOLUMES[@]}"; do
    archive_volume "${source_volumes[$index]}" "${temp_bundle}/${logical}.tar.gz"
    index=$((index + 1))
  done
  write_manifest "${temp_bundle}" "${primary}" "${project}"

  # Restore the exact pre-snapshot running set before publishing the bundle.
  resume_primary
  mv "${temp_bundle}" "${BUNDLE_DIR}"
  trap - EXIT
  echo "wrote ${BUNDLE_DIR}"
}

target_has_state() {
  local project="$1"
  local existing_services=()
  while IFS= read -r service; do
    [[ -n "${service}" ]] && existing_services+=("${service}")
  done < <(compose_for_root "${ROOT_DIR}" ps --all --services)
  [[ "${#existing_services[@]}" -eq 0 ]] || return 0

  local logical resolved
  for logical in "${DURABLE_VOLUMES[@]}"; do
    resolved="$(volume_name "${project}" "${logical}")"
    if [[ -n "${resolved}" ]] && ! volume_is_empty "${resolved}"; then
      return 0
    fi
  done
  return 1
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
  [[ -e "${BUNDLE_DIR}" ]] || {
    echo "no worktree data snapshot to restore"
    return 0
  }
  # Integrity is checked before any Docker operation or target mutation.
  verify_bundle
  command -v docker >/dev/null 2>&1 \
    || fail "Docker is required to restore worktree data"

  local running_services=()
  while IFS= read -r service; do
    [[ -n "${service}" ]] && running_services+=("${service}")
  done < <(compose_for_root "${ROOT_DIR}" ps --status running --services)
  [[ "${#running_services[@]}" -eq 0 ]] \
    || fail "restore must run before this worktree's Compose stack starts"

  local project
  project="$(compose_project "${ROOT_DIR}")"
  local physical_root
  physical_root="$(cd "${ROOT_DIR}" && pwd -P)"
  if target_has_state "${project}"; then
    echo "worktree already contains data; keeping it"
    return 0
  fi

  local target_volumes=()
  local restore_complete=0
  cleanup_restore() {
    if [[ "${restore_complete}" != "1" ]]; then
      ATTESTA_DATA_DISABLE_SEED=1 compose_for_root "${ROOT_DIR}" stop mariadb >/dev/null 2>&1 || true
      ATTESTA_DATA_DISABLE_SEED=1 compose_for_root "${ROOT_DIR}" rm -f mariadb >/dev/null 2>&1 || true
      if [[ "${#target_volumes[@]}" -gt 0 ]]; then
        docker volume rm -f "${target_volumes[@]}" >/dev/null 2>&1 || true
      fi
    fi
  }
  trap cleanup_restore EXIT

  local logical resolved target_name
  for logical in "${DURABLE_VOLUMES[@]}"; do
    resolved="$(volume_name "${project}" "${logical}")"
    if [[ -z "${resolved}" ]]; then
      target_name="${project}_${logical}"
      resolved="$(docker volume create \
        --label "com.docker.compose.project=${project}" \
        --label "com.docker.compose.volume=${logical}" \
        --label "eu.forkbomb.attesta.managed=true" \
        --label "eu.forkbomb.attesta.worktree=${physical_root}" \
        --label "eu.forkbomb.attesta.project=${project}" \
        "${target_name}")"
    fi
    target_volumes+=("${resolved}")
  done

  local index=0
  for logical in "${DURABLE_VOLUMES[@]}"; do
    extract_volume "${BUNDLE_DIR}/${logical}.tar.gz" "${target_volumes[$index]}"
    index=$((index + 1))
  done

  echo "sanitizing copied Appwrite runtime state"
  ATTESTA_DATA_DISABLE_SEED=1 compose_for_root "${ROOT_DIR}" up -d mariadb >/dev/null
  sanitize_appwrite
  ATTESTA_DATA_DISABLE_SEED=1 compose_for_root "${ROOT_DIR}" stop mariadb >/dev/null
  ATTESTA_DATA_DISABLE_SEED=1 compose_for_root "${ROOT_DIR}" rm -f mariadb >/dev/null

  restore_complete=1
  trap - EXIT
  echo "restored coherent worktree data snapshot"
}

forget_snapshot() {
  rm -rf -- "${BUNDLE_DIR}"
  rm -f -- "${DATA_DIR}/mongodb.archive.gz"
  rmdir "${DATA_DIR}" 2>/dev/null || true
  echo "removed saved worktree data snapshot"
}

main() {
  local command="${1:-}"
  shift || true
  [[ $# -eq 0 ]] || fail "${command} accepts no arguments"
  case "${command}" in
    snapshot-primary) snapshot_primary ;;
    restore) restore_snapshot ;;
    sanitize) sanitize_appwrite ;;
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
