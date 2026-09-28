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
printf '%s\n' "$*" >>"${DOCKER_CALLS_FILE}"

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
_APP_DB_ROOT_PASS="test-root-password" \
_APP_DB_SCHEMA="test-appwrite" \
  /bin/sh "${SCRIPT}" >"${tmpdir}/stdout"

[[ -s "${tmpdir}/calls" ]] || fail "restore issued no Docker calls"
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

echo "ok: restore-appwrite-seed_test"
