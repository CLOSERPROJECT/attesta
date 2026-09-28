#!/usr/bin/env bash
# Remove Docker resources left behind after an Attesta worktree directory is gone.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MANAGED_LABEL="eu.forkbomb.attesta.managed"
OWNER_LABEL="eu.forkbomb.attesta.worktree"
PROJECT_LABEL="eu.forkbomb.attesta.project"
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
  git -C "${ROOT_DIR}" worktree list --porcelain 2>/dev/null \
    | sed -n 's/^worktree //p' \
    | grep -Fqx -- "${owner}"
}

list_candidates() {
  local format
  format="{{.Label \"${OWNER_LABEL}\"}}|{{.Label \"${PROJECT_LABEL}\"}}"
  docker volume ls \
    --filter "label=${MANAGED_LABEL}=true" \
    --format "${format}"
  docker network ls \
    --filter "label=${MANAGED_LABEL}=true" \
    --format "${format}"
}

remove_project() {
  local project="$1"
  local owner="$2"
  local resources=()

  while IFS= read -r resource; do
    [[ -n "${resource}" ]] && resources+=("${resource}")
  done < <(docker ps -aq \
    --filter "label=${MANAGED_LABEL}=true" \
    --filter "label=${PROJECT_LABEL}=${project}" \
    --filter "label=${OWNER_LABEL}=${owner}")
  if [[ ${#resources[@]} -gt 0 ]]; then
    docker container rm -f "${resources[@]}"
  fi

  resources=()
  while IFS= read -r resource; do
    [[ -n "${resource}" ]] && resources+=("${resource}")
  done < <(docker network ls -q \
    --filter "label=${MANAGED_LABEL}=true" \
    --filter "label=${PROJECT_LABEL}=${project}")
  if [[ ${#resources[@]} -gt 0 ]]; then
    docker network rm "${resources[@]}"
  fi

  resources=()
  while IFS= read -r resource; do
    [[ -n "${resource}" ]] && resources+=("${resource}")
  done < <(docker volume ls -q \
    --filter "label=${MANAGED_LABEL}=true" \
    --filter "label=${PROJECT_LABEL}=${project}")
  if [[ ${#resources[@]} -gt 0 ]]; then
    docker volume rm "${resources[@]}"
  fi
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

  local candidates
  candidates="$(list_candidates | sort -u)"
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
      remove_project "${project}" "${owner}"
    fi
  done <<<"${candidates}"
}

main "$@"
