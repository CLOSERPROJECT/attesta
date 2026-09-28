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

docker compose \
  --env-file "${ROOT}/deployment/appwrite/.env.appwrite.example" \
  -f "${ROOT}/deployment/appwrite/docker-compose.appwrite.yaml" \
  config --format json >"${tmpdir}/standalone.json"

COMPOSE_PROJECT_NAME=label-test-a \
ATTESTA_WORKTREE_ROOT="${tmpdir}/owned-worktree-a" \
APPWRITE_HTTP_PORT=18080 \
APPWRITE_HTTPS_PORT=18443 \
APPWRITE_RESTORE_SEED_SQL=true \
  docker compose \
    --env-file "${ROOT}/deployment/appwrite/.env.appwrite.example" \
    -f "${ROOT}/deployment/docker-compose.local.yaml" \
    config --format json >"${tmpdir}/local-a.json"

COMPOSE_PROJECT_NAME=label-test-b \
ATTESTA_WORKTREE_ROOT="${tmpdir}/owned-worktree-b" \
APPWRITE_HTTP_PORT=28080 \
APPWRITE_HTTPS_PORT=28443 \
APPWRITE_RESTORE_SEED_SQL=true \
  docker compose \
    --env-file "${ROOT}/deployment/appwrite/.env.appwrite.example" \
    -f "${ROOT}/deployment/docker-compose.local.yaml" \
    config --format json >"${tmpdir}/local-b.json"

node -e '
  const fs = require("node:fs");
  const [standalonePath, localAPath, localBPath, ownerA] = process.argv.slice(1);
  const standalone = JSON.parse(fs.readFileSync(standalonePath, "utf8"));
  const localA = JSON.parse(fs.readFileSync(localAPath, "utf8"));
  const localB = JSON.parse(fs.readFileSync(localBPath, "utf8"));
  const assert = (condition, message) => {
    if (!condition) throw new Error(message);
  };
  const portSet = config => new Set(config.services.traefik.ports.map(port => String(port.published)));
  const mounts = config => config.services.mariadb.volumes.map(volume => volume.target);
  const resourceNames = config => new Set([
    ...Object.values(config.networks).map(resource => resource.name),
    ...Object.values(config.volumes).map(resource => resource.name),
  ]);
  const isOwnedBy = (resource, project, owner) =>
    resource.labels?.["eu.forkbomb.attesta.managed"] === "true" &&
    resource.labels?.["eu.forkbomb.attesta.project"] === project &&
    resource.labels?.["eu.forkbomb.attesta.worktree"] === owner;

  assert(standalone.services.traefik.container_name === "appwrite-traefik", "standalone Traefik name changed");
  assert(standalone.services.appwrite.container_name === "appwrite", "standalone Appwrite name changed");
  assert(standalone.services.mariadb.container_name === "appwrite-mariadb", "standalone MariaDB name changed");
  assert(portSet(standalone).has("80") && portSet(standalone).has("443"), "standalone ports changed");
  assert(standalone.networks.gateway.name === "gateway", "standalone gateway network changed");
  assert(standalone.networks.appwrite.name === "appwrite", "standalone Appwrite network changed");
  assert(standalone.networks.runtimes.name === "runtimes", "standalone runtimes network changed");
  assert(Object.values(standalone.services).every(service => !service.labels?.["eu.forkbomb.attesta.managed"]), "standalone acquired worktree service labels");
  assert(Object.values(standalone.networks).every(network => !network.labels?.["eu.forkbomb.attesta.managed"]), "standalone acquired worktree network labels");
  assert(Object.values(standalone.volumes).every(volume => !volume.labels?.["eu.forkbomb.attesta.managed"]), "standalone acquired worktree volume labels");
  assert(!mounts(standalone).includes("/seed/appwrite-seed.sql"), "standalone acquired the local seed mount");
  assert(!mounts(standalone).includes("/docker-entrypoint-initdb.d/10-preview-seed.sh"), "standalone acquired the local seed hook");
  assert(!("APPWRITE_RESTORE_SEED_SQL" in standalone.services.mariadb.environment), "standalone acquired the local seed switch");

  assert(Object.values(localA.services).every(service => service.container_name == null), "local services retain fixed container names");
  assert(Object.values(localA.services).every(resource => isOwnedBy(resource, "label-test-a", ownerA)), "local services lack exact ownership labels");
  assert(Object.values(localA.networks).every(resource => isOwnedBy(resource, "label-test-a", ownerA)), "local networks lack exact ownership labels");
  assert(Object.values(localA.volumes).every(resource => isOwnedBy(resource, "label-test-a", ownerA)), "local volumes lack exact ownership labels");
  assert(portSet(localA).has("18080") && portSet(localA).has("18443"), "local Appwrite ports were not overridden");
  assert(localA.networks.appwrite.name === "label-test-a_appwrite", "local Appwrite network is not project-scoped");
  assert(localA.networks.runtimes.name === "label-test-a_runtimes", "local runtimes network is not project-scoped");
  assert(localA.services["openruntimes-executor"].environment.OPR_EXECUTOR_NETWORK === "label-test-a_runtimes", "executor uses the wrong runtimes network");
  assert(mounts(localA).includes("/seed/appwrite-seed.sql"), "local seed mount missing");
  assert(mounts(localA).includes("/docker-entrypoint-initdb.d/10-preview-seed.sh"), "local seed hook missing");
  assert(localA.services.mariadb.environment.APPWRITE_RESTORE_SEED_SQL === "true", "local seed switch missing");

  const resourcesA = resourceNames(localA);
  const overlap = [...resourceNames(localB)].filter(name => resourcesA.has(name));
  assert(overlap.length === 0, `local projects share resources: ${overlap.join(", ")}`);
' "${tmpdir}/standalone.json" "${tmpdir}/local-a.json" "${tmpdir}/local-b.json" \
  "${tmpdir}/owned-worktree-a" || fail "standalone/local Compose separation failed"

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
