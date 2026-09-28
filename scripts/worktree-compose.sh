#!/usr/bin/env bash
# Shared Compose entrypoint for one fully isolated stack per worktree.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${ROOT_DIR}/deployment/docker-compose.local.yaml"

usage() {
  cat <<'USAGE'
Usage: scripts/worktree-compose.sh <command>

Commands:
  up [--build]  Start this worktree's infrastructure
  stop          Stop this worktree's infrastructure containers
  down          Remove this worktree's containers and networks; keep volumes
  down-v        Remove this worktree's containers, networks, and volumes
  ps            Show this worktree's Compose status
  logs          Follow this worktree's Compose logs
  mongo-ping    Ping this worktree's MongoDB service
  config        Render the resolved Compose configuration
USAGE
}

load_env() {
  cd "${ROOT_DIR}"
  if [[ -f "${ROOT_DIR}/.env" ]]; then
    set -a
    # shellcheck disable=SC1091
    source "${ROOT_DIR}/.env"
    set +a
  fi
  eval "$(bash "${ROOT_DIR}/scripts/worktree-env.sh" export)"
  ATTESTA_WORKTREE_ROOT="$(cd "${ROOT_DIR}" && pwd -P)"
  export COMPOSE_PROJECT_NAME APPWRITE_RESTORE_SEED_SQL ATTESTA_WORKTREE_ROOT
}

compose() {
  docker compose \
    --project-name "${COMPOSE_PROJECT_NAME}" \
    -f "${COMPOSE_FILE}" \
    "$@"
}

infra_services() {
  compose config --services | while IFS= read -r service; do
    [[ "${service}" == "attesta" ]] || echo "${service}"
  done
}

up() {
  local build=0
  if [[ "${1:-}" == "--build" ]]; then
    build=1
    shift
  fi
  [[ $# -eq 0 ]] || {
    echo "error: up accepts only --build" >&2
    exit 1
  }

  local services=()
  while IFS= read -r service; do
    [[ -n "${service}" ]] && services+=("${service}")
  done < <(infra_services)
  [[ ${#services[@]} -gt 0 ]] || {
    echo "error: Compose configuration contains no infrastructure services" >&2
    exit 1
  }
  bash "${ROOT_DIR}/scripts/worktree-data.sh" restore
  if [[ "${build}" -eq 1 ]]; then
    compose up -d --build "${services[@]}"
  else
    compose up -d "${services[@]}"
  fi
  compose ps
}

main() {
  local command="${1:-}"
  shift || true
  if [[ "${command}" == "" || "${command}" == "-h" || "${command}" == "--help" ]]; then
    usage
    return 0
  fi
  load_env
  case "${command}" in
    up) up "$@" ;;
    stop) compose stop ;;
    down) compose down --remove-orphans ;;
    down-v) compose down -v --remove-orphans ;;
    ps) compose ps ;;
    logs) compose logs -f ;;
    mongo-ping) compose exec -T mongodb mongosh --quiet --eval "db.adminCommand('ping')" ;;
    config) compose config ;;
    *)
      echo "unknown command: ${command}" >&2
      usage >&2
      exit 1
      ;;
  esac
}

main "$@"
