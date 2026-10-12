#!/usr/bin/env bash
# Docker cannot change labels on an existing local volume; pre-worktree volumes
# keep a stale compose config-hash without Attesta ownership labels, so Compose
# prompts to recreate them (and wipe data) on every up.
set -euo pipefail
umask 077

ROOT_DIR="${ATTESTA_ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/require-compose-version.sh
source "${SCRIPT_ROOT}/scripts/require-compose-version.sh"
HELPER_IMAGE="${ATTESTA_VOLUME_HELPER_IMAGE:-alpine:3.20}"
RELABEL_SAFETY_DIR=""
RELABEL_MIGRATED=0
RELABEL_TOTAL=0

fail() {
  echo "error: $*" >&2
  exit 1
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
    docker compose \
      --project-name "${COMPOSE_PROJECT_NAME}" \
      -f "${checkout_root}/deployment/docker-compose.local.yaml" \
      "$@"
  )
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

archive_volume() {
  local volume="$1"
  local destination="$2"
  docker run --rm -v "${volume}:/source:ro" "${HELPER_IMAGE}" \
    sh -c 'cd /source && exec tar -czf - .' archive >"${destination}"
  [[ -s "${destination}" ]] || fail "archive for ${volume} is empty"
}

extract_volume() {
  local archive="$1"
  local volume="$2"
  docker run --rm -i -v "${volume}:/target" "${HELPER_IMAGE}" \
    sh -c 'cd /target && exec tar -xzf -' extract <"${archive}"
}

volume_ownership() {
  local volume="$1"
  docker volume inspect \
    --format '{{ index .Labels "eu.forkbomb.attesta.managed" }}|{{ index .Labels "eu.forkbomb.attesta.project" }}|{{ index .Labels "eu.forkbomb.attesta.worktree" }}|{{ index .Labels "eu.forkbomb.attesta.repository" }}' \
    "${volume}"
}

expected_volume_ownership() {
  local project="$1"
  local physical_root="$2"
  local repository="$3"
  printf 'true|%s|%s|%s' "${project}" "${physical_root}" "${repository}"
}

relabel_volumes() {
  command -v docker >/dev/null 2>&1 \
    || fail "Docker is required to relabel Compose volumes"

  local project
  if ! project="$(compose_project "${ROOT_DIR}")"; then
    fail "could not resolve the target Compose project"
  fi
  local repository
  if ! repository="$(repository_identity "${ROOT_DIR}")"; then
    fail "could not resolve the repository identity for ${ROOT_DIR}"
  fi
  local physical_root
  physical_root="$(cd "${ROOT_DIR}" && pwd -P)"
  local expected
  expected="$(expected_volume_ownership "${project}" "${physical_root}" "${repository}")"

  local running_services=()
  local running_output
  if ! running_output="$(compose_for_root "${ROOT_DIR}" ps --status running --services)"; then
    fail "could not determine the target Compose running services"
  fi
  while IFS= read -r service; do
    [[ -n "${service}" ]] && running_services+=("${service}")
  done <<<"${running_output}"
  if [[ "${#running_services[@]}" -gt 0 ]]; then
    fail "relabel-volumes requires a stopped Compose stack (running: ${running_services[*]})"
  fi

  local logicals=()
  local logical_output
  if ! logical_output="$(compose_for_root "${ROOT_DIR}" config --volumes)"; then
    fail "could not list Compose volumes for ${project}"
  fi
  while IFS= read -r logical; do
    [[ -n "${logical}" ]] && logicals+=("${logical}")
  done <<<"${logical_output}"
  [[ ${#logicals[@]} -gt 0 ]] || return 0

  local logical resolved ownership
  local to_relabel=()
  local to_relabel_logicals=()
  for logical in "${logicals[@]}"; do
    if ! resolved="$(volume_name "${project}" "${logical}")"; then
      fail "could not inspect Compose volume ${logical}"
    fi
    [[ -n "${resolved}" ]] || continue
    if ! ownership="$(volume_ownership "${resolved}")"; then
      fail "could not inspect ownership of volume ${resolved}"
    fi
    if [[ "${ownership}" == "${expected}" ]]; then
      continue
    fi
    # Empty ownership is the legacy pre-worktree case. Any other mismatch means
    # another checkout owns the volume; never steal it.
    if [[ "${ownership}" != "|||" ]]; then
      fail "volume ${resolved} has ownership ${ownership}; expected ${expected} or unlabeled legacy volumes"
    fi
    to_relabel+=("${resolved}")
    to_relabel_logicals+=("${logical}")
  done
  [[ ${#to_relabel[@]} -gt 0 ]] || return 0

  RELABEL_SAFETY_DIR=""
  RELABEL_MIGRATED=0
  RELABEL_TOTAL=${#to_relabel[@]}
  local index=0
  cleanup_relabel() {
    # Keep rollback paths at script scope; Ubuntu Bash can unwind function locals
    # before EXIT traps after a fatal command.
    if [[ "${RELABEL_MIGRATED}" -lt "${RELABEL_TOTAL}" && -n "${RELABEL_SAFETY_DIR}" && -d "${RELABEL_SAFETY_DIR}" ]]; then
      echo "error: volume relabel incomplete; preserved archives in ${RELABEL_SAFETY_DIR}" >&2
      return 0
    fi
    if [[ -n "${RELABEL_SAFETY_DIR}" && -d "${RELABEL_SAFETY_DIR}" ]]; then
      rm -rf -- "${RELABEL_SAFETY_DIR}"
    fi
  }
  trap cleanup_relabel EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  trap 'exit 129' HUP

  RELABEL_SAFETY_DIR="$(mktemp -d "${TMPDIR:-/tmp}/attesta-relabel-volumes.XXXXXX")"

  echo "relabeling ${#to_relabel[@]} legacy Compose volume(s) for ${project}"
  for index in "${!to_relabel[@]}"; do
    resolved="${to_relabel[$index]}"
    logical="${to_relabel_logicals[$index]}"
    echo "  ${resolved}"
    archive_volume "${resolved}" "${RELABEL_SAFETY_DIR}/${logical}.tar.gz"
    docker volume rm "${resolved}" >/dev/null \
      || fail "could not remove unlabeled volume ${resolved} (is anything still using it?)"
    docker volume create \
      --label "com.docker.compose.project=${project}" \
      --label "com.docker.compose.volume=${logical}" \
      --label "eu.forkbomb.attesta.managed=true" \
      --label "eu.forkbomb.attesta.worktree=${physical_root}" \
      --label "eu.forkbomb.attesta.project=${project}" \
      --label "eu.forkbomb.attesta.repository=${repository}" \
      "${resolved}" >/dev/null \
      || fail "could not create labeled replacement for ${resolved}; archive is at ${RELABEL_SAFETY_DIR}/${logical}.tar.gz"
    extract_volume "${RELABEL_SAFETY_DIR}/${logical}.tar.gz" "${resolved}"
    RELABEL_MIGRATED=$((RELABEL_MIGRATED + 1))
  done

  rm -rf -- "${RELABEL_SAFETY_DIR}"
  RELABEL_SAFETY_DIR=""
  trap - EXIT INT TERM HUP
  echo "relabeled ${RELABEL_MIGRATED} Compose volume(s)"
}

main() {
  local command="${1:-}"
  shift || true
  [[ $# -eq 0 ]] || fail "${command:-relabel-volumes} accepts no arguments"
  case "${command}" in
    ""|relabel-volumes)
      require_compose_version
      relabel_volumes
      ;;
    -h|--help)
      cat <<'USAGE'
Usage: scripts/worktree-relabel-volumes.sh [relabel-volumes]

Copy legacy unlabeled Compose volumes onto same-named labeled replacements so
Compose stops prompting to recreate them. Requires a stopped stack.
USAGE
      ;;
    *)
      echo "unknown command: ${command}" >&2
      exit 1
      ;;
  esac
}

main "$@"
