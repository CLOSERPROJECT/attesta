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

COMPOSE_PROJECT_NAME_OVERRIDE="${COMPOSE_PROJECT_NAME:-}"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME_OVERRIDE:-$(generated_compose_project_name)}"

usage() {
  cat <<'USAGE'
Usage: scripts/worktree-env.sh <command>

Commands:
  print         Print resolved ports, URLs, and Compose project name
  export        Same as print, prefixed with `export `
  project-name  Print this checkout's Compose-safe project name
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
    ! lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
  else
    ! (echo >/dev/tcp/127.0.0.1/"${port}") 2>/dev/null
  fi
}

ALLOCATED_PORTS=()

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
    # shellcheck disable=SC1090
    source "${WORKTREE_ENV_FILE}"
  fi
  set +a
  if [[ -n "${COMPOSE_PROJECT_NAME_OVERRIDE}" ]]; then
    COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME_OVERRIDE}"
  fi
  validate_compose_project_name
  validate_ports
}

load_project_name() {
  if [[ -z "${COMPOSE_PROJECT_NAME_OVERRIDE}" && -f "${WORKTREE_ENV_FILE}" ]]; then
    # shellcheck disable=SC1090
    source "${WORKTREE_ENV_FILE}"
  fi
  if [[ -n "${COMPOSE_PROJECT_NAME_OVERRIDE}" ]]; then
    COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME_OVERRIDE}"
  fi
  validate_compose_project_name
}

write_ports_file() {
  local header="$1"
  validate_compose_project_name
  validate_ports
  {
    echo "${header}"
    local key
    for key in "${PORT_KEYS[@]}"; do
      printf '%s=%s\n' "${key}" "${!key}"
    done
    printf 'COMPOSE_PROJECT_NAME=%s\n' "${COMPOSE_PROJECT_NAME}"
  } >"${WORKTREE_ENV_FILE}"
  echo "wrote ${WORKTREE_ENV_FILE}"
}

cmd_write() {
  if [[ -f "${WORKTREE_ENV_FILE}" ]]; then
    echo "keeping existing ${WORKTREE_ENV_FILE}"
    return 0
  fi

  if [[ "${1:-}" == "--classic" ]]; then
    [[ $# -eq 1 ]] || fail "write --classic accepts no port values"
    load_resolved
    write_ports_file \
      "# Generated classic ports from scripts/dev-ports.env. Edit freely; not overwritten."
    return 0
  fi

  [[ $# -eq 0 ]] || fail "write accepts no port values; edit .env.worktree after generation"
  ALLOCATED_PORTS=()
  allocate_ports
  local branch
  branch="$(branch_name)"
  write_ports_file \
    "# Generated ports for worktree ${COMPOSE_PROJECT_NAME} (branch ${branch}).
# Seed includes checkout identity; Worktrunk hash_port preferred. Edit freely; not overwritten."
}

cmd_print() {
  load_resolved
  cat <<EOF
ROOT_DIR=${ROOT_DIR}
PORT=${PORT}
VITE_PORT=${VITE_PORT}
MONGODB_PORT=${MONGODB_PORT}
CERBOS_PORT=${CERBOS_PORT}
MONGO_EXPRESS_PORT=${MONGO_EXPRESS_PORT}
MAILPIT_SMTP_PORT=${MAILPIT_SMTP_PORT}
MAILPIT_UI_PORT=${MAILPIT_UI_PORT}
APPWRITE_HTTP_PORT=${APPWRITE_HTTP_PORT}
APPWRITE_HTTPS_PORT=${APPWRITE_HTTPS_PORT}
DOCKER_APP_PORT=${DOCKER_APP_PORT}
MONGODB_URI=mongodb://localhost:${MONGODB_PORT}
CERBOS_URL=http://localhost:${CERBOS_PORT}
APPWRITE_ENDPOINT=http://localhost:${APPWRITE_HTTP_PORT}/v1
APPWRITE_CONSOLE_URL=http://localhost:${APPWRITE_HTTP_PORT}
MONGO_EXPRESS_URL=http://localhost:${MONGO_EXPRESS_PORT}
MAILPIT_UI_URL=http://localhost:${MAILPIT_UI_PORT}
APPWRITE_INVITE_REDIRECT_URL=http://localhost:${PORT}/invite/accept
APPWRITE_RESET_REDIRECT_URL=http://localhost:${PORT}/reset/confirm
PUBLIC_BASE_URL=http://localhost:${PORT}
VITE_DEV_SERVER=http://localhost:${VITE_PORT}
SMTP_HOST=localhost
SMTP_PORT=${MAILPIT_SMTP_PORT}
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME}
APPWRITE_RESTORE_SEED_SQL=true
EOF
}

cmd_export() {
  cmd_print | sed 's/^/export /'
}

cmd_project_name() {
  load_project_name
  echo "${COMPOSE_PROJECT_NAME}"
}

main() {
  local command="${1:-}"
  shift || true
  case "${command}" in
    print) cmd_print "$@" ;;
    export) cmd_export "$@" ;;
    project-name) cmd_project_name "$@" ;;
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
