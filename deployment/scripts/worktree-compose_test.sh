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

# Host .env / shell may already export PUBLIC_BASE_URL; clear it so
# DOCKER_APP_PORT drives the containerized Attesta default under test.
unset PUBLIC_BASE_URL COMPOSE_PROJECT_NAME

COMPOSE_PROJECT_NAME=label-test-a \
ATTESTA_WORKTREE_ROOT="${tmpdir}/owned-worktree-a" \
ATTESTA_REPOSITORY_ID=repo-test-a \
APPWRITE_HTTP_PORT=18080 \
APPWRITE_HTTPS_PORT=18443 \
DOCKER_APP_PORT=19030 \
  docker compose \
    --env-file "${ROOT}/deployment/appwrite/.env.appwrite.example" \
    -f "${ROOT}/deployment/docker-compose.local.yaml" \
    config --format json >"${tmpdir}/local-a.json"

COMPOSE_PROJECT_NAME=label-test-b \
ATTESTA_WORKTREE_ROOT="${tmpdir}/owned-worktree-b" \
ATTESTA_REPOSITORY_ID=repo-test-b \
APPWRITE_HTTP_PORT=28080 \
APPWRITE_HTTPS_PORT=28443 \
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
  const isOwnedBy = (resource, project, owner, repository) =>
    resource.labels?.["eu.forkbomb.attesta.managed"] === "true" &&
    resource.labels?.["eu.forkbomb.attesta.project"] === project &&
    resource.labels?.["eu.forkbomb.attesta.worktree"] === owner &&
    resource.labels?.["eu.forkbomb.attesta.repository"] === repository;

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
  assert(!mounts(standalone).some(target => target.startsWith("/seed/")), "standalone acquired a local seed mount");

  assert(Object.values(localA.services).every(service => service.container_name == null), "local services retain fixed container names");
  assert(Object.values(localA.services).every(resource => isOwnedBy(resource, "label-test-a", ownerA, "repo-test-a")), "local services lack exact ownership labels");
  assert(Object.values(localA.networks).every(resource => isOwnedBy(resource, "label-test-a", ownerA, "repo-test-a")), "local networks lack exact ownership labels");
  assert(Object.values(localA.volumes).every(resource => isOwnedBy(resource, "label-test-a", ownerA, "repo-test-a")), "local volumes lack exact ownership labels");
  assert(portSet(localA).has("18080") && portSet(localA).has("18443"), "local Appwrite ports were not overridden");
  assert(localA.networks.appwrite.name === "label-test-a_appwrite", "local Appwrite network is not project-scoped");
  assert(localA.networks.runtimes.name === "label-test-a_runtimes", "local runtimes network is not project-scoped");
  assert(localA.services["openruntimes-executor"].environment.OPR_EXECUTOR_NETWORK === "label-test-a_runtimes", "executor uses the wrong runtimes network");
  assert(String(localA.services.attesta.ports[0].published) === "19030", "containerized Attesta port did not use DOCKER_APP_PORT");
  assert(localA.services.attesta.environment.PUBLIC_BASE_URL === "http://localhost:19030", "containerized PUBLIC_BASE_URL did not use DOCKER_APP_PORT");
  assert(localA.services.attesta.environment.ATTESTA_ROOT_DIR === "/app", "containerized ATTESTA_ROOT_DIR is not /app");
  assert(!mounts(localA).some(target => target.startsWith("/seed/")), "local MariaDB still mounts retired SQL seed artifacts");
  assert(localA.services.mariadb.entrypoint == null, "local MariaDB still uses a seed wrapper entrypoint");

  const resourcesA = resourceNames(localA);
  const overlap = [...resourceNames(localB)].filter(name => resourcesA.has(name));
  assert(overlap.length === 0, `local projects share resources: ${overlap.join(", ")}`);
' "${tmpdir}/standalone.json" "${tmpdir}/local-a.json" "${tmpdir}/local-b.json" \
  "${tmpdir}/owned-worktree-a" || fail "standalone/local Compose separation failed"

cat >"${tmpdir}/bin/bash" <<'EOF'
#!/bin/bash
set -euo pipefail
if [[ "${1:-}" == */scripts/worktree-env.sh && "${FAIL_WORKTREE_ENV:-0}" == "1" ]]; then
  echo "simulated invalid worktree environment" >&2
  exit 23
fi
exec /bin/bash "$@"
EOF

cat >"${tmpdir}/bin/docker" <<'EOF'
#!/bin/bash
set -euo pipefail
if [[ "$*" == "compose version --short" ]]; then
  echo "${DOCKER_COMPOSE_VERSION:-2.40.3}"
  exit 0
fi
if [[ -n "${DOCKER_ENV_FILE:-}" ]]; then
  printf '%s\n' "${ATTESTA_REPOSITORY_ID:-<unset>}" >>"${DOCKER_ENV_FILE}"
fi
if [[ -n "${DOCKER_PUBLIC_BASE_URL_FILE:-}" ]]; then
  printf '%s\n' "${PUBLIC_BASE_URL:-<unset>}" >>"${DOCKER_PUBLIC_BASE_URL_FILE}"
fi
if [[ "$*" == volume\ ls\ --filter\ label=com.docker.compose.project=* ]]; then
  if [[ "${VOLUME_ENUMERATION_FAILURE:-0}" == "1" ]]; then
    echo "simulated volume enumeration failure" >&2
    exit 71
  fi
  owner="${ATTESTA_WORKTREE_ROOT:-}"
  repository="${ATTESTA_REPOSITORY_ID:-}"
  if [[ "${VOLUME_OWNER_MODE:-}" == "mismatch" ]]; then
    owner="/another/worktree"
  fi
  if [[ "${VOLUME_OWNER_MODE:-}" == "unlabeled" ]]; then
    printf 'legacy-volume||||\n'
  else
    printf 'managed-volume|true|%s|%s|%s\n' \
      "${COMPOSE_PROJECT_NAME}" "${owner}" "${repository}"
  fi
  exit 0
fi
if [[ " $* " == *" config --services "* ]]; then
  printf '%s\n' mongodb cerbos mailpit mariadb redis appwrite traefik attesta
  exit 0
fi
printf '%s\n' "$*" >>"${DOCKER_CALLS_FILE}"
EOF
cat >"${tmpdir}/bin/go" <<'EOF'
#!/bin/bash
set -euo pipefail
# Stub prepare so compose up tests do not need a live Appwrite stack.
if [[ " $* " == *" prepare "* ]]; then
  printf 'attesta-prepare\n' >>"${DOCKER_CALLS_FILE}"
  exit 0
fi
echo "unexpected go invocation: $*" >&2
exit 1
EOF
chmod +x "${tmpdir}/bin/bash" "${tmpdir}/bin/docker" "${tmpdir}/bin/go"

expected_project="$(/bin/bash "${ROOT}/scripts/worktree-env.sh" project-name)"

grep -Fq 'go run ./cmd/server prepare --policy=worktree' \
  "${ROOT}/scripts/worktree-compose.sh" \
  || fail "worktree-compose does not invoke attesta prepare with worktree policy"
grep -Fq 'ATTESTA_ROOT_DIR="${ROOT_DIR}"' \
  "${ROOT}/scripts/worktree-compose.sh" \
  || fail "worktree-compose does not export ATTESTA_ROOT_DIR for prepare"
grep -Fq 'ATTESTA_SEED_COMMAND' "${ROOT}/scripts/worktree-compose.sh" \
  && fail "worktree-compose still exposes retired ATTESTA_SEED_COMMAND hatch"
[[ ! -e "${ROOT}/deployment/attesta-appwrite-ready.sh" ]] \
  || fail "retired attesta-appwrite-ready.sh still present"
grep -Fq './attesta prepare' "${ROOT}/deployment/coolify-entrypoint.sh" \
  || fail "Coolify entrypoint does not invoke attesta prepare"
grep -Fq 'bootstrap-appwrite' "${ROOT}/deployment/coolify-entrypoint.sh" \
  && fail "Coolify entrypoint still runs bootstrap-appwrite in shell"
grep -Eq '(^|[[:space:]])(\./)?attesta seed([[:space:]]|$)' "${ROOT}/deployment/coolify-entrypoint.sh" \
  && fail "Coolify entrypoint still runs attesta seed in shell"
grep -Eq 'load_appwrite_runtime_env|^[[:space:]]*\.[[:space:]]' "${ROOT}/deployment/coolify-entrypoint.sh" \
  && fail "Coolify entrypoint still sources runtime Appwrite env in shell"

if PATH="${tmpdir}/bin:${PATH}" DOCKER_CALLS_FILE="${tmpdir}/calls" \
  FAIL_WORKTREE_ENV=1 COMPOSE_PROJECT_NAME=stale-project \
  /bin/bash "${ROOT}/scripts/worktree-compose.sh" ps \
  >"${tmpdir}/env-failure.out" 2>"${tmpdir}/env-failure.err"; then
  fail "Compose continued after worktree environment export failed"
fi
grep -q 'simulated invalid worktree environment' "${tmpdir}/env-failure.err" \
  || fail "worktree environment failure was hidden"
[[ ! -e "${tmpdir}/calls" ]] \
  || fail "Docker was called after worktree environment export failed"

if PATH="${tmpdir}/bin:${PATH}" DOCKER_CALLS_FILE="${tmpdir}/calls" \
  DOCKER_COMPOSE_VERSION=2.24.3 \
  /bin/bash "${ROOT}/scripts/worktree-compose.sh" config \
  >"${tmpdir}/old-compose.out" 2>"${tmpdir}/old-compose.err"; then
  fail "Compose older than 2.24.4 was accepted"
fi
grep -q 'Docker Compose 2.24.4 or newer is required' "${tmpdir}/old-compose.err" \
  || fail "old Compose rejection did not explain the required version"
[[ ! -e "${tmpdir}/calls" ]] \
  || fail "local Compose configuration was parsed before the version preflight"

PATH="${tmpdir}/bin:${PATH}" DOCKER_CALLS_FILE="${tmpdir}/calls" \
  DOCKER_ENV_FILE="${tmpdir}/repository-ids" \
  /bin/bash "${ROOT}/scripts/worktree-compose.sh" up

expected_repository_id="$(/bin/bash "${ROOT}/scripts/worktree-env.sh" repository-id)"
if grep -Fvxq "${expected_repository_id}" "${tmpdir}/repository-ids"; then
  fail "Compose resources did not receive the stable repository identity"
fi

grep -Fq -- "--project-name ${expected_project}" "${tmpdir}/calls" \
  || fail "Compose project name missing"
grep -q ' up -d .*mongodb.*appwrite.*traefik' "${tmpdir}/calls" || fail "infra services missing"
if grep ' up -d ' "${tmpdir}/calls" | sed 's/.* up -d //' | grep -q '\(^\| \)attesta\($\| \)'; then
  fail "host-dev up unexpectedly starts the attesta container"
fi
grep -q '^attesta-prepare$' "${tmpdir}/calls" || fail "attesta prepare did not run after infra up"
grep -q ' ps$' "${tmpdir}/calls" || fail "status call missing"
! grep -q 'relabel' "${tmpdir}/calls" \
  || fail "compose up still invoked retired volume relabel"
! grep -q '^data ' "${tmpdir}/calls" \
  || fail "compose up still invoked retired worktree-data snapshot/restore"

: >"${tmpdir}/calls"
PATH="${tmpdir}/bin:${PATH}" DOCKER_CALLS_FILE="${tmpdir}/calls" \
  DOCKER_PUBLIC_BASE_URL_FILE="${tmpdir}/container-public-base-url" \
  /bin/bash "${ROOT}/scripts/worktree-compose.sh" up-app --build
grep -q ' up -d --build .*attesta' "${tmpdir}/calls" \
  || fail "containerized start did not build and start the Attesta service"
grep -q '^attesta-prepare$' "${tmpdir}/calls" \
  || fail "containerized start did not run attesta prepare after infra up"
docker_app_port="$(/bin/bash "${ROOT}/scripts/worktree-env.sh" print | awk -F= '$1 == "DOCKER_APP_PORT" { print $2 }')"
expected_public_base_url="http://localhost:${docker_app_port}"
if grep -Fvxq "${expected_public_base_url}" "${tmpdir}/container-public-base-url"; then
  fail "containerized start did not pass DOCKER_APP_PORT PUBLIC_BASE_URL to Compose"
fi
task_list="$(task --list-all --json)"
grep -Eq '"name"[[:space:]]*:[[:space:]]*"start:docker"' <<<"${task_list}" \
  || fail "Taskfile does not expose the containerized Attesta start path"
grep -Fq '"github:max-sixty/worktrunk" = { version = "0.77.0", bin = "wt" }' "${ROOT}/mise.toml" \
  || fail "Worktrunk is not pinned to the tested 0.77.0 release"

: >"${tmpdir}/calls"
PATH="${tmpdir}/bin:${PATH}" DOCKER_CALLS_FILE="${tmpdir}/calls" \
  /bin/bash "${ROOT}/scripts/worktree-compose.sh" stop
grep -q -- "--project-name ${expected_project} .* stop$" "${tmpdir}/calls" \
  || fail "stop did not stay scoped to this Compose project"
if grep -q -- ' -v\|--volumes' "${tmpdir}/calls"; then
  fail "stop unexpectedly removes data volumes"
fi

: >"${tmpdir}/calls"
if PATH="${tmpdir}/bin:${PATH}" DOCKER_CALLS_FILE="${tmpdir}/calls" \
  VOLUME_OWNER_MODE=mismatch \
  /bin/bash "${ROOT}/scripts/worktree-compose.sh" down-v \
  >"${tmpdir}/unsafe-down.out" 2>"${tmpdir}/unsafe-down.err"; then
  fail "down-v accepted a managed volume owned by another worktree"
fi
grep -q 'refusing to remove volumes' "${tmpdir}/unsafe-down.err" \
  || fail "down-v ownership refusal was not explained"
if grep -q ' down -v ' "${tmpdir}/calls"; then
  fail "down-v reached Compose after the ownership mismatch"
fi

: >"${tmpdir}/calls"
if PATH="${tmpdir}/bin:${PATH}" DOCKER_CALLS_FILE="${tmpdir}/calls" \
  VOLUME_OWNER_MODE=unlabeled \
  /bin/bash "${ROOT}/scripts/worktree-compose.sh" down-v \
  >"${tmpdir}/unlabeled-down.out" 2>"${tmpdir}/unlabeled-down.err"; then
  fail "down-v accepted an unlabeled legacy project volume"
fi
grep -q 'missing or mismatched Attesta ownership labels' "${tmpdir}/unlabeled-down.err" \
  || fail "unlabeled legacy volume refusal was not explained"
if grep -q ' down -v ' "${tmpdir}/calls"; then
  fail "down-v reached Compose after finding an unlabeled legacy volume"
fi

: >"${tmpdir}/calls"
if PATH="${tmpdir}/bin:${PATH}" DOCKER_CALLS_FILE="${tmpdir}/calls" \
  VOLUME_ENUMERATION_FAILURE=1 \
  /bin/bash "${ROOT}/scripts/worktree-compose.sh" down-v \
  >"${tmpdir}/enumeration-down.out" 2>"${tmpdir}/enumeration-down.err"; then
  fail "down-v continued after volume enumeration failed"
fi
grep -q 'ownership could not be verified' "${tmpdir}/enumeration-down.err" \
  || fail "volume enumeration failure was not explained"
if grep -q ' down -v ' "${tmpdir}/calls"; then
  fail "down-v reached Compose after volume enumeration failed"
fi

: >"${tmpdir}/calls"
PATH="${tmpdir}/bin:${PATH}" DOCKER_CALLS_FILE="${tmpdir}/calls" \
  /bin/bash "${ROOT}/scripts/worktree-compose.sh" down-v
grep -q -- "--project-name ${expected_project} .* down -v --remove-orphans$" "${tmpdir}/calls" \
  || fail "down-v did not remove only this Compose project's volumes"

echo "ok: worktree-compose_test"
