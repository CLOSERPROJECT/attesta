#!/usr/bin/env bash
# Shared Compose entrypoint for one fully isolated stack per worktree.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${ROOT_DIR}/deployment/docker-compose.local.yaml"
# shellcheck source=scripts/require-compose-version.sh
source "${ROOT_DIR}/scripts/require-compose-version.sh"

usage() {
  cat <<'USAGE'
Usage: scripts/worktree-compose.sh <command>

Commands:
  up [--build]  Start this worktree's infrastructure
  up-app [--build]
                Start infrastructure and the containerized Attesta app
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
  local worktree_exports
  cd "${ROOT_DIR}"
  if [[ -f "${ROOT_DIR}/.env" ]]; then
    set -a
    # shellcheck disable=SC1091
    source "${ROOT_DIR}/.env"
    set +a
  fi
  if ! worktree_exports="$(bash "${ROOT_DIR}/scripts/worktree-env.sh" export)"; then
    echo "error: could not resolve this worktree's environment" >&2
    return 1
  fi
  eval "${worktree_exports}"
  ATTESTA_WORKTREE_ROOT="$(cd "${ROOT_DIR}" && pwd -P)"
  ATTESTA_REPOSITORY_ID="$(bash "${ROOT_DIR}/scripts/worktree-env.sh" repository-id)"
  export COMPOSE_PROJECT_NAME APPWRITE_RESTORE_SEED_SQL ATTESTA_WORKTREE_ROOT ATTESTA_REPOSITORY_ID
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

verify_volume_ownership() {
  local format volumes name managed project owner repository unsafe=0
  format="{{.Name}}|{{.Label \"eu.forkbomb.attesta.managed\"}}|{{.Label \"eu.forkbomb.attesta.project\"}}|{{.Label \"eu.forkbomb.attesta.worktree\"}}|{{.Label \"eu.forkbomb.attesta.repository\"}}"
  if ! volumes="$(docker volume ls \
    --filter "label=com.docker.compose.project=${COMPOSE_PROJECT_NAME}" \
    --format "${format}")"; then
    echo "error: refusing to remove volumes because their ownership could not be verified" >&2
    return 1
  fi
  while IFS='|' read -r name managed project owner repository; do
    [[ -n "${name}" ]] || continue
    if [[ "${managed}" != "true" || "${project}" != "${COMPOSE_PROJECT_NAME}" || \
      "${owner}" != "${ATTESTA_WORKTREE_ROOT}" || "${repository}" != "${ATTESTA_REPOSITORY_ID}" ]]; then
      echo "error: refusing to remove volumes: ${name} has missing or mismatched Attesta ownership labels" >&2
      unsafe=1
    fi
  done <<<"${volumes}"
  [[ "${unsafe}" -eq 0 ]]
}

wait_for_appwrite() {
  local port="${APPWRITE_HTTP_PORT:-}"
  local timeout="${ATTESTA_APPWRITE_READY_TIMEOUT_SECONDS:-120}"
  local deadline code
  [[ "${port}" =~ ^[0-9]+$ ]] || {
    echo "error: APPWRITE_HTTP_PORT is required to wait for Appwrite readiness" >&2
    return 1
  }
  [[ "${timeout}" =~ ^[0-9]+$ ]] || {
    echo "error: ATTESTA_APPWRITE_READY_TIMEOUT_SECONDS must be a non-negative integer" >&2
    return 1
  }
  # Tests set timeout 0 to skip the readiness probe.
  if [[ "${timeout}" -eq 0 ]]; then
    return 0
  fi
  deadline=$((SECONDS + timeout))
  echo "waiting for Appwrite on localhost:${port}"
  while ((SECONDS < deadline)); do
    code="$(curl -s -o /dev/null -w '%{http_code}' \
      "http://127.0.0.1:${port}/v1/health" 2>/dev/null || true)"
    # Guests get 401 once Appwrite is serving; treat that as ready.
    if [[ "${code}" == "200" || "${code}" == "401" ]]; then
      return 0
    fi
    sleep 0.5
  done
  echo "error: Appwrite did not become ready on localhost:${port} within ${timeout}s" >&2
  return 1
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
  wait_for_appwrite
  compose ps
}

up_app() {
  local build=0
  if [[ "${1:-}" == "--build" ]]; then
    build=1
    shift
  fi
  [[ $# -eq 0 ]] || {
    echo "error: up-app accepts only --build" >&2
    exit 1
  }

  APPWRITE_INVITE_REDIRECT_URL="http://localhost:${DOCKER_APP_PORT}/invite/accept"
  APPWRITE_RESET_REDIRECT_URL="http://localhost:${DOCKER_APP_PORT}/reset/confirm"
  export APPWRITE_INVITE_REDIRECT_URL APPWRITE_RESET_REDIRECT_URL

  local services=()
  while IFS= read -r service; do
    [[ -n "${service}" ]] && services+=("${service}")
  done < <(compose config --services)
  [[ ${#services[@]} -gt 0 ]] || {
    echo "error: Compose configuration contains no services" >&2
    exit 1
  }
  bash "${ROOT_DIR}/scripts/worktree-data.sh" restore
  if [[ "${build}" -eq 1 ]]; then
    compose up -d --build "${services[@]}"
  else
    compose up -d "${services[@]}"
  fi
  wait_for_appwrite
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
  require_compose_version
  case "${command}" in
    up) up "$@" ;;
    up-app) up_app "$@" ;;
    stop) compose stop ;;
    down) compose down --remove-orphans ;;
    down-v)
      verify_volume_ownership
      compose down -v --remove-orphans
      ;;
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
