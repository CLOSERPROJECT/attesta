#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SCRIPT="${ROOT}/deployment/scripts/restore-appwrite-seed.sh"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf -- "$tmpdir"' EXIT
mkdir -p "${tmpdir}/bin"

cat >"${tmpdir}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$*" == "compose version --short" ]]; then
  echo "${DOCKER_COMPOSE_VERSION:-2.40.3}"
  exit 0
fi
printf '%s\n' "$*" >>"${DOCKER_CALLS_FILE}"
printf '%s\n' "${ATTESTA_REPOSITORY_ID:-<unset>}" >>"${DOCKER_REPOSITORY_IDS_FILE}"

if [[ " $* " == *" exec -T mariadb mariadb "* ]]; then
  cat >>"${DOCKER_STDIN_FILE}"
fi

# Exercise the script's compatibility fallback when optional workers are absent.
if [[ " $* " == *" restart appwrite appwrite-worker-mails appwrite-worker-messaging appwrite-realtime "* ]]; then
  exit 1
fi
EOF
chmod +x "${tmpdir}/bin/docker"

expected_project="$(bash "${ROOT}/scripts/worktree-env.sh" project-name)"
expected_compose="${ROOT}/deployment/docker-compose.local.yaml"

PATH="${tmpdir}/bin:${PATH}" \
DOCKER_CALLS_FILE="${tmpdir}/calls" \
DOCKER_STDIN_FILE="${tmpdir}/stdin" \
DOCKER_REPOSITORY_IDS_FILE="${tmpdir}/repository-ids" \
_APP_DB_ROOT_PASS="test-root-password" \
_APP_DB_SCHEMA="test-appwrite" \
  /bin/bash "${SCRIPT}" >"${tmpdir}/stdout"

[[ -s "${tmpdir}/calls" ]] || fail "restore issued no Docker calls"
if grep -Evq '^repo-[0-9a-f]{32}$' "${tmpdir}/repository-ids"; then
  fail "restore did not scope Compose resources to this repository clone"
fi
while IFS= read -r call; do
  [[ "${call}" == "compose --project-name ${expected_project} -f ${expected_compose} "* ]] \
    || fail "unscoped Compose call: ${call}"
done <"${tmpdir}/calls"

if grep -Eq '(^| )exec (appwrite-)?mariadb( |$)' "${tmpdir}/calls"; then
  fail "restore uses a fixed container name or direct docker exec"
fi
if grep -Fq 'appwrite-mariadb' "${tmpdir}/calls"; then
  fail "restore retains the obsolete fixed MariaDB container name"
fi
grep -Fq ' exec -T mariadb healthcheck.sh --connect --innodb_initialized' "${tmpdir}/calls" \
  || fail "readiness check does not address the MariaDB service"
grep -Fq ' exec -T mariadb mariadb --user=root --password=test-root-password test-appwrite' "${tmpdir}/calls" \
  || fail "database import does not use the configured credentials"
grep -Fq ' restart appwrite appwrite-worker-mails appwrite-worker-messaging appwrite-realtime' "${tmpdir}/calls" \
  || fail "full Appwrite restart was not attempted"
grep -Fxq "compose --project-name ${expected_project} -f ${expected_compose} restart appwrite" "${tmpdir}/calls" \
  || fail "Appwrite-only restart fallback was not attempted"

grep -Fq 'DROP TABLE IF EXISTS' "${tmpdir}/stdin" \
  || fail "checked-in seed was not sent to MariaDB"
grep -Fq 'DELETE FROM _console_certificates;' "${tmpdir}/stdin" \
  || fail "certificate cleanup SQL missing"
grep -Fq 'DELETE FROM _console_rules;' "${tmpdir}/stdin" \
  || fail "routing cleanup SQL missing"
grep -Fq 'DELETE FROM _console_sessions;' "${tmpdir}/stdin" \
  || fail "console-session cleanup SQL missing"
grep -Fq 'DELETE FROM _1_sessions;' "${tmpdir}/stdin" \
  || fail "project-session cleanup SQL missing"

# Export validation errors stop the restore before any Docker mutation.
failing_root="${tmpdir}/failing-root"
mkdir -p "${failing_root}/deployment/scripts" "${failing_root}/deployment/appwrite" \
  "${failing_root}/scripts"
cp "${SCRIPT}" "${failing_root}/deployment/scripts/restore-appwrite-seed.sh"
cp "${ROOT}/scripts/require-compose-version.sh" "${failing_root}/scripts/require-compose-version.sh"
printf '%s\n' 'SELECT 1;' >"${failing_root}/deployment/appwrite/appwrite-seed.sql"
cat >"${failing_root}/scripts/worktree-env.sh" <<'EOF'
#!/usr/bin/env bash
echo "invalid worktree environment" >&2
exit 23
EOF
chmod +x "${failing_root}/scripts/worktree-env.sh"
: >"${tmpdir}/failing-calls"
if PATH="${tmpdir}/bin:${PATH}" \
  DOCKER_CALLS_FILE="${tmpdir}/failing-calls" \
  DOCKER_STDIN_FILE="${tmpdir}/failing-stdin" \
  DOCKER_REPOSITORY_IDS_FILE="${tmpdir}/failing-repository-ids" \
  bash "${failing_root}/deployment/scripts/restore-appwrite-seed.sh" \
  >"${tmpdir}/failing-out" 2>"${tmpdir}/failing-error"; then
  fail "restore ignored exporter failure"
fi
[[ ! -s "${tmpdir}/failing-calls" ]] \
  || fail "restore touched Docker after exporter failure"
grep -q 'invalid worktree environment' "${tmpdir}/failing-error" \
  || fail "restore hid exporter failure"

# Old Compose releases fail before the local configuration or database is touched.
: >"${tmpdir}/old-compose-calls"
if PATH="${tmpdir}/bin:${PATH}" \
  DOCKER_COMPOSE_VERSION=2.24.3 \
  DOCKER_CALLS_FILE="${tmpdir}/old-compose-calls" \
  DOCKER_STDIN_FILE="${tmpdir}/old-compose-stdin" \
  DOCKER_REPOSITORY_IDS_FILE="${tmpdir}/old-compose-repository-ids" \
  bash "${SCRIPT}" >"${tmpdir}/old-compose-out" 2>"${tmpdir}/old-compose-error"; then
  fail "restore accepted Docker Compose older than 2.24.4"
fi
grep -q 'Docker Compose 2.24.4 or newer is required' "${tmpdir}/old-compose-error" \
  || fail "old Compose rejection did not explain the required version"
[[ ! -s "${tmpdir}/old-compose-calls" ]] \
  || fail "restore mutated Docker after the Compose version preflight failed"

echo "ok: restore-appwrite-seed_test"
