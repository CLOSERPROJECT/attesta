#!/usr/bin/env bash
# Remove Docker resources left behind after an Attesta worktree directory is gone.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MANAGED_LABEL="eu.forkbomb.attesta.managed"
OWNER_LABEL="eu.forkbomb.attesta.worktree"
PROJECT_LABEL="eu.forkbomb.attesta.project"
REPOSITORY_LABEL="eu.forkbomb.attesta.repository"
REPOSITORY_ID=""
REGISTERED_WORKTREES=""
APPLY=0

usage() {
  cat <<'USAGE'
Usage: scripts/worktree-gc.sh [--apply]

Find Attesta Compose projects whose owning worktree directory no longer exists.
The default is a dry-run. Pass --apply to remove those projects' containers,
networks, and volumes. Active or still-present worktrees are never removed.
USAGE
}

fail() {
  echo "error: $*" >&2
  exit 1
}

registered_worktree() {
  local owner="$1"
  local registered
  while IFS= read -r registered; do
    [[ "${registered}" == "${owner}" ]] && return 0
  done <<<"${REGISTERED_WORKTREES}"
  return 1
}

load_registered_worktrees() {
  local worktree_output line
  if ! worktree_output="$(git -C "${ROOT_DIR}" worktree list --porcelain 2>/dev/null)"; then
    echo "error: could not enumerate registered Git worktrees" >&2
    return 1
  fi

  REGISTERED_WORKTREES=""
  while IFS= read -r line; do
    [[ "${line}" == worktree\ * ]] || continue
    REGISTERED_WORKTREES+="${line#worktree }"$'\n'
  done <<<"${worktree_output}"
}

repository_identity() {
  ATTESTA_ROOT_DIR="${ROOT_DIR}" \
    bash "${ROOT_DIR}/scripts/worktree-env.sh" repository-id
}

list_candidates() {
  local format volume_candidates network_candidates
  format="{{.Label \"${OWNER_LABEL}\"}}|{{.Label \"${PROJECT_LABEL}\"}}"
  if ! volume_candidates="$(docker volume ls \
      --filter "label=${MANAGED_LABEL}=true" \
      --filter "label=${REPOSITORY_LABEL}=${REPOSITORY_ID}" \
      --format "${format}")"; then
    echo "error: could not enumerate managed Attesta volumes" >&2
    return 1
  fi
  if ! network_candidates="$(docker network ls \
      --filter "label=${MANAGED_LABEL}=true" \
      --filter "label=${REPOSITORY_LABEL}=${REPOSITORY_ID}" \
      --format "${format}")"; then
    echo "error: could not enumerate managed Attesta networks" >&2
    return 1
  fi
  printf '%s\n%s\n' "${volume_candidates}" "${network_candidates}"
}

remove_project() {
  local project="$1"
  local owner="$2"
  local container_output network_output volume_output resource
  local containers=() networks=() volumes=()

  if ! container_output="$(docker ps -aq \
      --filter "label=${MANAGED_LABEL}=true" \
      --filter "label=${REPOSITORY_LABEL}=${REPOSITORY_ID}" \
      --filter "label=${PROJECT_LABEL}=${project}" \
      --filter "label=${OWNER_LABEL}=${owner}")"; then
    echo "error: could not enumerate containers for Attesta Compose project ${project}" >&2
    return 1
  fi
  if ! network_output="$(docker network ls -q \
      --filter "label=${MANAGED_LABEL}=true" \
      --filter "label=${REPOSITORY_LABEL}=${REPOSITORY_ID}" \
      --filter "label=${PROJECT_LABEL}=${project}" \
      --filter "label=${OWNER_LABEL}=${owner}")"; then
    echo "error: could not enumerate networks for Attesta Compose project ${project}" >&2
    return 1
  fi
  if ! volume_output="$(docker volume ls -q \
      --filter "label=${MANAGED_LABEL}=true" \
      --filter "label=${REPOSITORY_LABEL}=${REPOSITORY_ID}" \
      --filter "label=${PROJECT_LABEL}=${project}" \
      --filter "label=${OWNER_LABEL}=${owner}")"; then
    echo "error: could not enumerate volumes for Attesta Compose project ${project}" >&2
    return 1
  fi

  while IFS= read -r resource; do
    [[ -n "${resource}" ]] && containers+=("${resource}")
  done <<<"${container_output}"
  while IFS= read -r resource; do
    [[ -n "${resource}" ]] && networks+=("${resource}")
  done <<<"${network_output}"
  while IFS= read -r resource; do
    [[ -n "${resource}" ]] && volumes+=("${resource}")
  done <<<"${volume_output}"

  if [[ ${#containers[@]} -gt 0 ]]; then
    docker container rm -f "${containers[@]}"
  fi
  if [[ ${#networks[@]} -gt 0 ]]; then
    for resource in "${networks[@]}"; do
      if ! docker network rm "${resource}"; then
        echo "warning: could not remove network ${resource}; it may still have active endpoints" >&2
      fi
    done
  fi
  if [[ ${#volumes[@]} -gt 0 ]]; then
    docker volume rm "${volumes[@]}"
  fi
  return 0
}

main() {
  case "${1:-}" in
    "") ;;
    --apply) APPLY=1 ;;
    -h|--help)
      usage
      return 0
      ;;
    *)
      usage >&2
      fail "unknown argument: $1"
      ;;
  esac
  [[ $# -le 1 ]] || fail "expected at most one argument"
  command -v docker >/dev/null 2>&1 || fail "Docker is required"
  REPOSITORY_ID="$(repository_identity)"
  load_registered_worktrees || fail "garbage collection aborted"

  local candidates
  if ! candidates="$(list_candidates)"; then
    fail "garbage collection aborted"
  fi
  if ! candidates="$(sort -u <<<"${candidates}")"; then
    fail "could not sort managed Attesta resources"
  fi
  if [[ -z "${candidates}" ]]; then
    echo "no managed Attesta Compose resources found"
    return 0
  fi

  local owner project owner_count
  while IFS='|' read -r owner project; do
    [[ -n "${owner}" && -n "${project}" ]] || continue
    [[ "${owner}" == /* ]] || {
      echo "warning: skipping ${project}: invalid worktree owner label" >&2
      continue
    }
    [[ "${project}" =~ ^[a-z0-9][a-z0-9_-]*$ ]] || {
      echo "warning: skipping invalid Compose project label: ${project}" >&2
      continue
    }

    owner_count="$(awk -F'|' -v project="${project}" \
      '$2 == project { print $1 }' <<<"${candidates}" | sort -u | awk 'NF { count++ } END { print count+0 }')"
    if [[ "${owner_count}" -ne 1 ]]; then
      echo "warning: skipping ${project}: resources disagree on owning worktree" >&2
      continue
    fi
    [[ ! -d "${owner}" ]] || continue
    if registered_worktree "${owner}"; then
      continue
    fi

    if [[ "${APPLY}" -eq 0 ]]; then
      echo "would remove Attesta Compose project ${project} (missing worktree ${owner})"
    else
      echo "removing Attesta Compose project ${project} (missing worktree ${owner})"
      if ! remove_project "${project}" "${owner}"; then
        fail "cleanup aborted for Attesta Compose project ${project}"
      fi
    fi
  done <<<"${candidates}"
}

main "$@"
