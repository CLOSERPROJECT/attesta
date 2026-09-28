#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf -- "$tmpdir"' EXIT
mkdir -p "${tmpdir}/bin"

COMPOSE_PROJECT_NAME=label-test \
ATTESTA_WORKTREE_ROOT="${tmpdir}/owned-worktree" \
  docker compose \
    --env-file "${ROOT}/deployment/appwrite/.env.appwrite.example" \
    -f "${ROOT}/deployment/docker-compose.local.yaml" \
    config --format json >"${tmpdir}/compose.json"

node -e '
  const fs = require("node:fs");
  const config = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
  const owner = process.argv[2];
  const valid = Object.values(config.services).every(({ labels }) =>
    labels?.["eu.forkbomb.attesta.managed"] === "true" &&
    labels?.["eu.forkbomb.attesta.project"] === "label-test" &&
    labels?.["eu.forkbomb.attesta.worktree"] === owner
  );
  process.exit(valid ? 0 : 1);
' "${tmpdir}/compose.json" "${tmpdir}/owned-worktree" \
  || fail "not every Compose service has the exact Attesta ownership labels"

cat >"${tmpdir}/bin/bash" <<'EOF'
#!/bin/bash
set -euo pipefail
if [[ "${1:-}" == */scripts/worktree-data.sh ]]; then
  printf 'data %s\n' "${2:-}" >>"${DOCKER_CALLS_FILE}"
  exit 0
fi
exec /bin/bash "$@"
EOF

cat >"${tmpdir}/bin/docker" <<'EOF'
#!/bin/bash
set -euo pipefail
if [[ " $* " == *" config --services "* ]]; then
  printf '%s\n' mongodb cerbos mailpit mariadb redis appwrite traefik attesta
  exit 0
fi
printf '%s\n' "$*" >>"${DOCKER_CALLS_FILE}"
EOF
chmod +x "${tmpdir}/bin/bash" "${tmpdir}/bin/docker"

expected_project="$(/bin/bash "${ROOT}/scripts/worktree-env.sh" project-name)"

PATH="${tmpdir}/bin:${PATH}" DOCKER_CALLS_FILE="${tmpdir}/calls" \
  /bin/bash "${ROOT}/scripts/worktree-compose.sh" up

grep -Fq -- "--project-name ${expected_project}" "${tmpdir}/calls" \
  || fail "Compose project name missing"
grep -q ' up -d .*mongodb.*appwrite.*traefik' "${tmpdir}/calls" || fail "infra services missing"
if grep ' up -d ' "${tmpdir}/calls" | sed 's/.* up -d //' | grep -q '\(^\| \)attesta\($\| \)'; then
  fail "host-dev up unexpectedly starts the attesta container"
fi
grep -q ' ps$' "${tmpdir}/calls" || fail "status call missing"

restore_line="$(grep -n '^data restore$' "${tmpdir}/calls" | cut -d: -f1)"
up_line="$(grep -n ' up -d ' "${tmpdir}/calls" | cut -d: -f1)"
[[ -n "${restore_line}" && -n "${up_line}" && "${restore_line}" -lt "${up_line}" ]] \
  || fail "data bundle was not restored before writer services started"

: >"${tmpdir}/calls"
PATH="${tmpdir}/bin:${PATH}" DOCKER_CALLS_FILE="${tmpdir}/calls" \
  /bin/bash "${ROOT}/scripts/worktree-compose.sh" stop
grep -q -- "--project-name ${expected_project} .* stop$" "${tmpdir}/calls" \
  || fail "stop did not stay scoped to this Compose project"
if grep -q -- ' -v\|--volumes' "${tmpdir}/calls"; then
  fail "stop unexpectedly removes data volumes"
fi

: >"${tmpdir}/calls"
PATH="${tmpdir}/bin:${PATH}" DOCKER_CALLS_FILE="${tmpdir}/calls" \
  /bin/bash "${ROOT}/scripts/worktree-compose.sh" down-v
grep -q -- "--project-name ${expected_project} .* down -v --remove-orphans$" "${tmpdir}/calls" \
  || fail "down-v did not remove only this Compose project's volumes"

echo "ok: worktree-compose_test"
