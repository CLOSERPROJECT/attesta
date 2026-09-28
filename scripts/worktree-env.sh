#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${ATTESTA_ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
DEFAULTS_FILE="${ROOT_DIR}/scripts/dev-ports.env"
WORKTREE_ENV_FILE="${ROOT_DIR}/.env.worktree"
PORT_KEYS=(
  PORT VITE_PORT MONGODB_PORT CERBOS_PORT MONGO_EXPRESS_PORT
  MAILPIT_SMTP_PORT MAILPIT_UI_PORT APPWRITE_HTTP_PORT
  APPWRITE_HTTPS_PORT DOCKER_APP_PORT
)
RESOLVED_KEYS=(
  "${PORT_KEYS[@]}"
  MONGODB_URI CERBOS_URL APPWRITE_ENDPOINT APPWRITE_CONSOLE_URL
  MONGO_EXPRESS_URL MAILPIT_UI_URL APPWRITE_INVITE_REDIRECT_URL
  APPWRITE_RESET_REDIRECT_URL PUBLIC_BASE_URL VITE_DEV_SERVER SMTP_HOST
  SMTP_PORT COMPOSE_PROJECT_NAME APPWRITE_RESTORE_SEED_SQL
)

sanitize_compose_project_name() {
  basename "$1" | tr '[:upper:]' '[:lower:]' \
    | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//'
}

canonical_path() {
  (cd "$1" && pwd -P)
}

generated_compose_project_name() {
  local root slug checksum suffix max_slug_length
  root="$(canonical_path "${ROOT_DIR}")"
  slug="$(sanitize_compose_project_name "${root}")"
  [[ -n "${slug}" ]] || slug="attesta"

  checksum="$(printf '%s' "${root}" | cksum | awk '{print $1}')"
  printf -v suffix '%08x' "${checksum}"
  max_slug_length=$((63 - 1 - ${#suffix}))
  printf '%s-%s\n' "${slug:0:${max_slug_length}}" "${suffix}"
}

repository_identity() {
  local common_dir identity_file identity temporary random_hex
  common_dir="$(git -C "${ROOT_DIR}" rev-parse --path-format=absolute --git-common-dir)" \
    || fail "could not resolve the Git common directory"
  if [[ "${common_dir}" != /* ]]; then
    common_dir="${ROOT_DIR}/${common_dir}"
  fi
  common_dir="$(cd "${common_dir}" && pwd -P)" \
    || fail "Git common directory does not exist: ${common_dir}"
  identity_file="${common_dir}/attesta-repository-id"
  if [[ -r "${identity_file}" ]]; then
    IFS= read -r identity <"${identity_file}" || true
    [[ "${identity}" =~ ^repo-[0-9a-f]{32}$ ]] \
      || fail "invalid repository identity in ${identity_file}"
    printf '%s\n' "${identity}"
    return 0
  fi

  random_hex="$(od -An -N16 -tx1 /dev/urandom | tr -d '[:space:]')"
  [[ "${random_hex}" =~ ^[0-9a-f]{32}$ ]] \
    || fail "could not generate a repository identity"
  temporary="${identity_file}.tmp.$$-${RANDOM}"
  printf 'repo-%s\n' "${random_hex}" >"${temporary}"
  chmod 600 "${temporary}"
  # A hard link publishes one complete immutable value atomically. Concurrent
  # worktrees may generate candidates, but only the first link can win.
  ln "${temporary}" "${identity_file}" 2>/dev/null || true
  rm -f -- "${temporary}"

  [[ -r "${identity_file}" ]] \
    || fail "could not persist repository identity in ${common_dir}"
  IFS= read -r identity <"${identity_file}" || true
  [[ "${identity}" =~ ^repo-[0-9a-f]{32}$ ]] \
    || fail "invalid repository identity in ${identity_file}"
  printf '%s\n' "${identity}"
}

INITIAL_COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-}"
COMPOSE_PROJECT_NAME="${INITIAL_COMPOSE_PROJECT_NAME:-$(generated_compose_project_name)}"

usage() {
  cat <<'USAGE'
Usage: scripts/worktree-env.sh <command>

Commands:
  print         Print resolved ports, URLs, and Compose project name
  export        Same as print, prefixed with `export `
  project-name  Print this checkout's Compose-safe project name
  repository-id Print the stable identity shared by this clone's worktrees
  write         Write available Worktrunk-derived ports if missing; never overwrite
  write --classic
                Write classic defaults if .env.worktree is missing

Generated ports are deterministic for the checkout and service label. If a
candidate is busy or already selected, allocation walks forward until free.
Edit .env.worktree afterward for any manual override.
USAGE
}

fail() {
  echo "error: $*" >&2
  exit 1
}

validate_compose_project_name() {
  [[ "${COMPOSE_PROJECT_NAME}" =~ ^[a-z0-9][a-z0-9_-]*$ ]] \
    || fail "COMPOSE_PROJECT_NAME must start with a lowercase letter or digit and contain only lowercase letters, digits, hyphens, or underscores"
  [[ ${#COMPOSE_PROJECT_NAME} -le 63 ]] \
    || fail "COMPOSE_PROJECT_NAME must be at most 63 characters"
}

validate_port() {
  local name="$1"
  local value="$2"
  if [[ ! "${value}" =~ ^[0-9]+$ ]] || ((value < 1 || value > 65535)); then
    fail "${name} must be an integer from 1 to 65535 (got ${value:-<empty>})"
  fi
}

validate_ports() {
  local key other
  for key in "${PORT_KEYS[@]}"; do
    validate_port "${key}" "${!key:-}"
  done
  for key in "${PORT_KEYS[@]}"; do
    for other in "${PORT_KEYS[@]}"; do
      [[ "${key}" == "${other}" ]] && break
      [[ "${!key}" != "${!other}" ]] \
        || fail "${key} and ${other} must use different ports (${!key})"
    done
  done
}

port_free() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    if lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
      return 1
    fi
  fi
  # lsof can hide listeners owned by other users, including Docker-published
  # ports on macOS. A real TCP connection is the authoritative fallback.
  if command -v nc >/dev/null 2>&1; then
    ! nc -z 127.0.0.1 "${port}" >/dev/null 2>&1
  else
    ! (: </dev/tcp/127.0.0.1/"${port}") 2>/dev/null
  fi
}

ALLOCATED_PORTS=()
PORT_ALLOCATION_LOCK_DIR=""
PORT_ALLOCATION_LOCK_HELD=0
PORT_WRITE_TEMP=""

port_lock_parent() {
  local common_dir
  if common_dir="$(git -C "${ROOT_DIR}" rev-parse --git-common-dir 2>/dev/null)"; then
    if [[ "${common_dir}" != /* ]]; then
      common_dir="${ROOT_DIR}/${common_dir}"
    fi
    (cd "${common_dir}" && pwd -P)
    return
  fi
  canonical_path "${ROOT_DIR}"
}

acquire_port_allocation_lock() {
  local timeout="${ATTESTA_PORT_LOCK_TIMEOUT_SECONDS:-300}"
  [[ "${timeout}" =~ ^[0-9]+$ ]] \
    || fail "ATTESTA_PORT_LOCK_TIMEOUT_SECONDS must be a non-negative integer"

  PORT_ALLOCATION_LOCK_DIR="$(port_lock_parent)/attesta-worktree-port-allocation.lock"
  local deadline=$((SECONDS + timeout))
  local owner=""
  while ! mkdir "${PORT_ALLOCATION_LOCK_DIR}" 2>/dev/null; do
    owner=""
    if [[ -r "${PORT_ALLOCATION_LOCK_DIR}/pid" ]]; then
      IFS= read -r owner <"${PORT_ALLOCATION_LOCK_DIR}/pid" || true
    fi
    if [[ "${owner}" =~ ^[0-9]+$ ]] && ! kill -0 "${owner}" 2>/dev/null; then
      rm -f -- "${PORT_ALLOCATION_LOCK_DIR}/pid"
      rmdir "${PORT_ALLOCATION_LOCK_DIR}" 2>/dev/null || true
      continue
    fi
    if ((SECONDS >= deadline)); then
      fail "timed out after ${timeout}s waiting for repository port-allocation lock: ${PORT_ALLOCATION_LOCK_DIR}"
    fi
    sleep 0.1
  done
  PORT_ALLOCATION_LOCK_HELD=1
  if ! printf '%s\n' "$$" >"${PORT_ALLOCATION_LOCK_DIR}/pid"; then
    rm -f -- "${PORT_ALLOCATION_LOCK_DIR}/pid"
    rmdir "${PORT_ALLOCATION_LOCK_DIR}" 2>/dev/null || true
    PORT_ALLOCATION_LOCK_HELD=0
    fail "could not record repository port-allocation lock owner: ${PORT_ALLOCATION_LOCK_DIR}"
  fi
}

release_port_allocation_lock() {
  [[ "${PORT_ALLOCATION_LOCK_HELD}" == "1" ]] || return 0
  local owner=""
  if [[ -r "${PORT_ALLOCATION_LOCK_DIR}/pid" ]]; then
    IFS= read -r owner <"${PORT_ALLOCATION_LOCK_DIR}/pid" || true
  fi
  if [[ "${owner}" == "$$" ]]; then
    rm -f -- "${PORT_ALLOCATION_LOCK_DIR}/pid"
    rmdir "${PORT_ALLOCATION_LOCK_DIR}" 2>/dev/null || true
  fi
  PORT_ALLOCATION_LOCK_HELD=0
}

cleanup_port_write() {
  if [[ -n "${PORT_WRITE_TEMP}" ]]; then
    rm -f -- "${PORT_WRITE_TEMP}"
    PORT_WRITE_TEMP=""
  fi
  release_port_allocation_lock
}

finish_port_write() {
  release_port_allocation_lock
  trap - EXIT INT TERM HUP
}

port_unallocated() {
  local candidate="$1"
  local allocated
  for allocated in "${ALLOCATED_PORTS[@]:-}"; do
    [[ "${candidate}" != "${allocated}" ]] || return 1
  done
}

seed_port() {
  local label="$1"
  local identity checksum candidate template
  identity="$(cd "${ROOT_DIR}" && pwd -P)"
  checksum="$(printf '%s' "${identity}" | cksum | awk '{print $1}')"
  template="{{ \"attesta-${label}-${checksum}\" | hash_port }}"

  if command -v wt >/dev/null 2>&1; then
    candidate="$(wt step eval "${template}" 2>/dev/null || true)"
  elif command -v mise >/dev/null 2>&1; then
    candidate="$(mise exec -- wt step eval "${template}" 2>/dev/null || true)"
  else
    candidate=""
  fi
  if [[ "${candidate}" =~ ^[0-9]+$ ]] && ((candidate >= 1 && candidate <= 65535)); then
    echo "${candidate}"
    return 0
  fi

  # Worktrunk is optional for plain Git and Codex-managed worktrees. Keep the
  # same deterministic semantics when it is unavailable.
  checksum="$(printf '%s' "attesta-${label}-${checksum}" | cksum | awk '{print $1}')"
  echo $((10000 + checksum % 10000))
}

reserve_sibling_worktree_ports() {
  local current line checkout env_file env_line port worktree_output
  current="$(canonical_path "${ROOT_DIR}")"
  if ! git -C "${ROOT_DIR}" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    return 0
  fi
  if ! worktree_output="$(git -C "${ROOT_DIR}" worktree list --porcelain 2>/dev/null)"; then
    fail "could not enumerate sibling Git worktrees for port allocation"
  fi
  while IFS= read -r line; do
    [[ "${line}" == worktree\ * ]] || continue
    checkout="${line#worktree }"
    [[ -d "${checkout}" ]] || continue
    checkout="$(canonical_path "${checkout}")"
    [[ "${checkout}" != "${current}" ]] || continue
    env_file="${checkout}/.env.worktree"
    [[ -f "${env_file}" ]] || continue
    while IFS= read -r env_line || [[ -n "${env_line}" ]]; do
      if [[ "${env_line}" =~ ^(PORT|[A-Z][A-Z0-9_]*_PORT)=([0-9]+)$ ]]; then
        port="${BASH_REMATCH[2]}"
        if ((port >= 1 && port <= 65535)); then
          printf '%s\n' "${port}"
        fi
      fi
    done <"${env_file}"
  done <<<"${worktree_output}"
}

pick_port() {
  local label="$1"
  local target="$2"
  local candidate
  candidate="$(seed_port "${label}")"
  validate_port "Worktrunk hash_port for ${label}" "${candidate}"

  local attempts=0
  while [[ "${attempts}" -lt 200 ]]; do
    if port_unallocated "${candidate}" && port_free "${candidate}"; then
      ALLOCATED_PORTS+=("${candidate}")
      printf -v "${target}" '%s' "${candidate}"
      return 0
    fi
    candidate=$((1024 + (candidate - 1024 + 1) % 64512))
    attempts=$((attempts + 1))
  done
  fail "could not find a free port for ${label}"
}

branch_name() {
  git -C "${ROOT_DIR}" symbolic-ref --short -q HEAD 2>/dev/null || echo detached
}

allocate_ports() {
  local reserved reserved_output
  if ! reserved_output="$(reserve_sibling_worktree_ports)"; then
    fail "could not reserve sibling worktree ports"
  fi
  while IFS= read -r reserved; do
    [[ -n "${reserved}" ]] && ALLOCATED_PORTS+=("${reserved}")
  done <<<"${reserved_output}"
  pick_port app PORT
  pick_port vite VITE_PORT
  pick_port mongodb MONGODB_PORT
  pick_port cerbos CERBOS_PORT
  pick_port mongo-express MONGO_EXPRESS_PORT
  pick_port mailpit-smtp MAILPIT_SMTP_PORT
  pick_port mailpit-ui MAILPIT_UI_PORT
  pick_port appwrite-http APPWRITE_HTTP_PORT
  pick_port appwrite-https APPWRITE_HTTPS_PORT
  pick_port docker-app DOCKER_APP_PORT
}

load_resolved() {
  [[ -f "${DEFAULTS_FILE}" ]] || fail "missing ${DEFAULTS_FILE}"
  local key
  for key in "${PORT_KEYS[@]}"; do
    unset "${key}" 2>/dev/null || true
  done
  set -a
  # shellcheck disable=SC1090
  source "${DEFAULTS_FILE}"
  if [[ -f "${WORKTREE_ENV_FILE}" ]]; then
    # Once persisted, this file is authoritative. Inherited shell variables or
    # values copied through .env must never collapse separate Compose projects.
    unset COMPOSE_PROJECT_NAME
    # shellcheck disable=SC1090
    source "${WORKTREE_ENV_FILE}"
    COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-$(generated_compose_project_name)}"
  else
    COMPOSE_PROJECT_NAME="${INITIAL_COMPOSE_PROJECT_NAME:-$(generated_compose_project_name)}"
  fi
  set +a
  validate_compose_project_name
  validate_ports
  MONGODB_URI="mongodb://localhost:${MONGODB_PORT}"
  CERBOS_URL="http://localhost:${CERBOS_PORT}"
  APPWRITE_ENDPOINT="http://localhost:${APPWRITE_HTTP_PORT}/v1"
  APPWRITE_CONSOLE_URL="http://localhost:${APPWRITE_HTTP_PORT}"
  MONGO_EXPRESS_URL="http://localhost:${MONGO_EXPRESS_PORT}"
  MAILPIT_UI_URL="http://localhost:${MAILPIT_UI_PORT}"
  APPWRITE_INVITE_REDIRECT_URL="http://localhost:${PORT}/invite/accept"
  APPWRITE_RESET_REDIRECT_URL="http://localhost:${PORT}/reset/confirm"
  PUBLIC_BASE_URL="http://localhost:${PORT}"
  VITE_DEV_SERVER="http://localhost:${VITE_PORT}"
  SMTP_HOST="localhost"
  SMTP_PORT="${MAILPIT_SMTP_PORT}"
  APPWRITE_RESTORE_SEED_SQL=true
}

load_project_name() {
  if [[ -f "${WORKTREE_ENV_FILE}" ]]; then
    unset COMPOSE_PROJECT_NAME
    # shellcheck disable=SC1090
    source "${WORKTREE_ENV_FILE}"
    COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-$(generated_compose_project_name)}"
  else
    COMPOSE_PROJECT_NAME="${INITIAL_COMPOSE_PROJECT_NAME:-$(generated_compose_project_name)}"
  fi
  validate_compose_project_name
}

write_ports_file() {
  local header="$1"
  validate_compose_project_name
  validate_ports
  PORT_WRITE_TEMP="${WORKTREE_ENV_FILE}.tmp.$$-${RANDOM}"
  {
    echo "${header}"
    local key
    for key in "${PORT_KEYS[@]}"; do
      printf '%s=%s\n' "${key}" "${!key}"
    done
    printf 'COMPOSE_PROJECT_NAME=%s\n' "${COMPOSE_PROJECT_NAME}"
  } >"${PORT_WRITE_TEMP}"
  if [[ -e "${WORKTREE_ENV_FILE}" ]]; then
    rm -f -- "${PORT_WRITE_TEMP}"
    PORT_WRITE_TEMP=""
    echo "keeping existing ${WORKTREE_ENV_FILE}"
    return 0
  fi
  mv "${PORT_WRITE_TEMP}" "${WORKTREE_ENV_FILE}"
  PORT_WRITE_TEMP=""
  echo "wrote ${WORKTREE_ENV_FILE}"
}

cmd_write() {
  if [[ "${1:-}" == "--classic" ]]; then
    [[ $# -eq 1 ]] || fail "write --classic accepts no port values"
  else
    [[ $# -eq 0 ]] || fail "write accepts no port values; edit .env.worktree after generation"
  fi

  trap cleanup_port_write EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  trap 'exit 129' HUP
  acquire_port_allocation_lock

  if [[ -f "${WORKTREE_ENV_FILE}" ]]; then
    echo "keeping existing ${WORKTREE_ENV_FILE}"
    finish_port_write
    return 0
  fi

  if [[ "${1:-}" == "--classic" ]]; then
    load_resolved
    write_ports_file \
      "# Generated classic ports from scripts/dev-ports.env. Edit freely; not overwritten."
    finish_port_write
    return 0
  fi

  ALLOCATED_PORTS=()
  allocate_ports
  local branch
  branch="$(branch_name)"
  write_ports_file \
    "# Generated ports for worktree ${COMPOSE_PROJECT_NAME} (branch ${branch}).
# Seed includes checkout identity; Worktrunk hash_port preferred. Edit freely; not overwritten."
  finish_port_write
}

cmd_print() {
  load_resolved
  local key
  for key in "${RESOLVED_KEYS[@]}"; do
    printf '%s=%s\n' "${key}" "${!key}"
  done
}

cmd_export() {
  load_resolved
  local key
  for key in "${RESOLVED_KEYS[@]}"; do
    printf 'export %s=%q\n' "${key}" "${!key}"
  done
}

cmd_project_name() {
  load_project_name
  echo "${COMPOSE_PROJECT_NAME}"
}

cmd_repository_id() {
  repository_identity
}

main() {
  local command="${1:-}"
  shift || true
  case "${command}" in
    print) cmd_print "$@" ;;
    export) cmd_export "$@" ;;
    project-name) cmd_project_name "$@" ;;
    repository-id) cmd_repository_id "$@" ;;
    write) cmd_write "$@" ;;
    ""|-h|--help) usage ;;
    *)
      echo "unknown command: ${command}" >&2
      usage >&2
      exit 1
      ;;
  esac
}

main "$@"
