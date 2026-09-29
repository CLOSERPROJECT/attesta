#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SCRIPT="${ROOT}/scripts/worktree-env.sh"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf -- "$tmpdir"' EXIT
mkdir -p "${tmpdir}/scripts" "${tmpdir}/bin"
cp "${ROOT}/scripts/dev-ports.env" "${tmpdir}/scripts/dev-ports.env"

# Make allocation deterministic and force every label to the same seed. The
# collision walk must still produce ten distinct ports.
cat >"${tmpdir}/bin/wt" <<'EOF'
#!/usr/bin/env bash
[[ "$1" == "step" && "$2" == "eval" ]] || exit 1
[[ -z "${TEST_WT_SLEEP_SECONDS:-}" ]] || sleep "${TEST_WT_SLEEP_SECONDS}"
echo "${TEST_WT_PORT:-11000}"
EOF
cat >"${tmpdir}/bin/lsof" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
cat >"${tmpdir}/bin/nc" <<'EOF'
#!/usr/bin/env bash
[[ "${*: -1}" == "${TEST_BUSY_PORT:-}" ]]
EOF
chmod +x "${tmpdir}/bin/wt" "${tmpdir}/bin/lsof" "${tmpdir}/bin/nc"

PATH="${tmpdir}/bin:${PATH}" ATTESTA_ROOT_DIR="${tmpdir}" bash "${SCRIPT}" write
cat >"${tmpdir}/expected" <<'EOF'
# Generated ports for worktree tmpdir (branch detached).
# Seed includes checkout identity; Worktrunk hash_port preferred. Edit freely; not overwritten.
PORT=11000
VITE_PORT=11001
MONGODB_PORT=11002
CERBOS_PORT=11003
MONGO_EXPRESS_PORT=11004
MAILPIT_SMTP_PORT=11005
MAILPIT_UI_PORT=11006
APPWRITE_HTTP_PORT=11007
APPWRITE_HTTPS_PORT=11008
DOCKER_APP_PORT=11009
COMPOSE_PROJECT_NAME=PROJECT_NAME
EOF
project_name="$(ATTESTA_ROOT_DIR="${tmpdir}" bash "${SCRIPT}" project-name)"
sed -i.bak "s/worktree tmpdir/worktree ${project_name}/" "${tmpdir}/expected"
rm "${tmpdir}/expected.bak"
sed -i.bak "s/COMPOSE_PROJECT_NAME=PROJECT_NAME/COMPOSE_PROJECT_NAME=${project_name}/" "${tmpdir}/expected"
rm "${tmpdir}/expected.bak"
diff -u "${tmpdir}/expected" "${tmpdir}/.env.worktree" || fail "unexpected generated format"

# Re-running bootstrap preserves manual edits.
sed -i.bak 's/^PORT=.*/PORT=12000/' "${tmpdir}/.env.worktree"
rm "${tmpdir}/.env.worktree.bak"
PATH="${tmpdir}/bin:${PATH}" ATTESTA_ROOT_DIR="${tmpdir}" bash "${SCRIPT}" write
grep -q '^PORT=12000$' "${tmpdir}/.env.worktree" || fail "overwrote existing PORT"

# Resolved output drives both host processes and the Compose stack.
resolved="$(ATTESTA_ROOT_DIR="${tmpdir}" bash "${SCRIPT}" print)"
grep -q '^MONGODB_URI=mongodb://localhost:11002$' <<<"${resolved}" || fail "Mongo URL missing"
grep -q '^APPWRITE_ENDPOINT=http://localhost:11007/v1$' <<<"${resolved}" || fail "Appwrite URL missing"
grep -q '^MAILPIT_UI_URL=http://localhost:11006$' <<<"${resolved}" || fail "Mailpit URL missing"
grep -q '^COMPOSE_PROJECT_NAME=' <<<"${resolved}" || fail "Compose project missing"
grep -q '^APPWRITE_RESTORE_SEED_SQL=true$' <<<"${resolved}" || fail "seed flag missing"

# Duplicate manual edits are rejected before Docker sees them.
sed -i.bak 's/^VITE_PORT=.*/VITE_PORT=12000/' "${tmpdir}/.env.worktree"
rm "${tmpdir}/.env.worktree.bak"
if ATTESTA_ROOT_DIR="${tmpdir}" bash "${SCRIPT}" print \
  >"${tmpdir}/duplicate-out" 2>"${tmpdir}/duplicate-error"; then
  fail "duplicate ports unexpectedly succeeded"
fi
grep -q 'must use different ports' "${tmpdir}/duplicate-error" || fail "missing collision error"

# Classic mode remains available for the primary checkout.
rm "${tmpdir}/.env.worktree"
ATTESTA_ROOT_DIR="${tmpdir}" bash "${SCRIPT}" write --classic
grep -q '^PORT=3000$' "${tmpdir}/.env.worktree" || fail "classic PORT missing"
grep -q '^APPWRITE_HTTP_PORT=80$' "${tmpdir}/.env.worktree" || fail "classic Appwrite port missing"

# Positional manual ports are intentionally unsupported; edit the generated file.
rm "${tmpdir}/.env.worktree"
if PATH="${tmpdir}/bin:${PATH}" ATTESTA_ROOT_DIR="${tmpdir}" \
  bash "${SCRIPT}" write 3101 >"${tmpdir}/arg-out" 2>"${tmpdir}/arg-error"; then
  fail "positional port unexpectedly succeeded"
fi
grep -q 'accepts no port values' "${tmpdir}/arg-error" || fail "missing positional-port error"

# Codex creates managed worktrees at detached HEAD. Their checkout paths, not
# their shared branch label, must make the initial port selections distinct.
detached_one="$(mktemp -d)"
detached_two="$(mktemp -d)"
trap 'rm -rf -- "$tmpdir" "$detached_one" "$detached_two"' EXIT
for detached_root in "${detached_one}" "${detached_two}"; do
  mkdir -p "${detached_root}/scripts"
  cp "${ROOT}/scripts/dev-ports.env" "${detached_root}/scripts/dev-ports.env"
  PATH="/usr/bin:/bin" ATTESTA_ROOT_DIR="${detached_root}" bash "${SCRIPT}" write
done
if cmp -s "${detached_one}/.env.worktree" "${detached_two}/.env.worktree"; then
  fail "detached worktrees received identical generated ports"
fi

# Compose identity is stable and derives from the canonical checkout path, not
# only the directory basename. Codex and Worktrunk may place equally named
# worktrees under different parents.
primary_root="${tmpdir}/Legacy_Primary"
same_name_one="${tmpdir}/parent-one/shared-worktree"
same_name_two="${tmpdir}/parent-two/shared-worktree"
mkdir -p "${primary_root}" "$(dirname "${same_name_one}")" "$(dirname "${same_name_two}")"
git -C "${primary_root}" init -q
git -C "${primary_root}" -c user.name=Test -c user.email=test@example.com \
  commit -q --allow-empty -m initial
git -C "${primary_root}" worktree add -q -b test-one "${same_name_one}"
git -C "${primary_root}" worktree add -q -b test-two "${same_name_two}"
project_one="$(ATTESTA_ROOT_DIR="${same_name_one}" bash "${SCRIPT}" project-name)"
project_two="$(ATTESTA_ROOT_DIR="${same_name_two}" bash "${SCRIPT}" project-name)"
project_primary="$(ATTESTA_ROOT_DIR="${primary_root}" bash "${SCRIPT}" project-name)"
repository_primary="$(ATTESTA_ROOT_DIR="${primary_root}" bash "${SCRIPT}" repository-id)"
repository_one="$(ATTESTA_ROOT_DIR="${same_name_one}" bash "${SCRIPT}" repository-id)"
repository_two="$(ATTESTA_ROOT_DIR="${same_name_two}" bash "${SCRIPT}" repository-id)"
[[ -n "${project_primary}" && "${project_primary}" != deployment ]] \
  || fail "primary Compose project defaulted to ${project_primary}, want path-qualified name"
[[ "${project_one}" != "${project_two}" ]] \
  || fail "same-basename worktrees received the same Compose project name"
[[ "${project_one}" != "${project_primary}" && "${project_two}" != "${project_primary}" ]] \
  || fail "linked worktrees reused the primary Compose project name"
[[ "${project_one}" == "$(ATTESTA_ROOT_DIR="${same_name_one}" bash "${SCRIPT}" project-name)" ]] \
  || fail "Compose project name is not stable"
[[ "${project_one}" =~ ^[a-z0-9][a-z0-9_-]*$ ]] \
  || fail "Compose project name is not Compose-safe: ${project_one}"
[[ ${#project_one} -le 63 ]] \
  || fail "Compose project name is not reasonably bounded: ${project_one}"
[[ "${repository_primary}" == "${repository_one}" && "${repository_one}" == "${repository_two}" ]] \
  || fail "linked worktrees did not share one repository identity"
[[ "${repository_one}" =~ ^repo-[0-9a-f]{32}$ ]] \
  || fail "repository identity is not bounded and label-safe: ${repository_one}"

# Independent primary clones each get a path-qualified Compose project so they
# never share volumes without an explicit persisted override.
independent_one="${tmpdir}/independent-one/shared-primary"
independent_two="${tmpdir}/independent-two/shared-primary"
mkdir -p "${independent_one}" "${independent_two}"
git -C "${independent_one}" init -q
git -C "${independent_two}" init -q
independent_project_one="$(ATTESTA_ROOT_DIR="${independent_one}" bash "${SCRIPT}" project-name)"
independent_project_two="$(ATTESTA_ROOT_DIR="${independent_two}" bash "${SCRIPT}" project-name)"
independent_repository_one="$(ATTESTA_ROOT_DIR="${independent_one}" bash "${SCRIPT}" repository-id)"
independent_repository_two="$(ATTESTA_ROOT_DIR="${independent_two}" bash "${SCRIPT}" repository-id)"
[[ "${independent_project_one}" != "${independent_project_two}" ]] \
  || fail "independent primaries shared Compose project: ${independent_project_one}"
[[ "${independent_project_one}" != deployment && "${independent_project_two}" != deployment ]] \
  || fail "independent primaries reused the legacy deployment name"
[[ "${independent_repository_one}" != "${independent_repository_two}" ]] \
  || fail "independent clones received the same repository identity"

# An explicit legacy identity can be adopted once during bootstrap. The
# persisted value then drives every read command without another override.
override_root="${tmpdir}/override-root"
mkdir -p "${override_root}/scripts"
cp "${ROOT}/scripts/dev-ports.env" "${override_root}/scripts/dev-ports.env"
COMPOSE_PROJECT_NAME=legacy-name ATTESTA_ROOT_DIR="${override_root}" \
  bash "${SCRIPT}" write --classic
grep -qx 'COMPOSE_PROJECT_NAME=legacy-name' "${override_root}/.env.worktree" \
  || fail "explicit Compose project override was not persisted"
[[ "$(ATTESTA_ROOT_DIR="${override_root}" bash "${SCRIPT}" project-name)" == legacy-name ]] \
  || fail "project-name ignored the persisted Compose project"
grep -qx 'COMPOSE_PROJECT_NAME=legacy-name' \
  <<<"$(ATTESTA_ROOT_DIR="${override_root}" bash "${SCRIPT}" print)" \
  || fail "print ignored the persisted Compose project"
grep -qx 'export COMPOSE_PROJECT_NAME=legacy-name' \
  <<<"$(ATTESTA_ROOT_DIR="${override_root}" bash "${SCRIPT}" export)" \
  || fail "export ignored the persisted Compose project"
[[ "$(COMPOSE_PROJECT_NAME=intruder ATTESTA_ROOT_DIR="${override_root}" bash "${SCRIPT}" project-name)" == legacy-name ]] \
  || fail "environment overrode the persisted Compose project"
grep -qx 'COMPOSE_PROJECT_NAME=legacy-name' \
  <<<"$(COMPOSE_PROJECT_NAME=intruder ATTESTA_ROOT_DIR="${override_root}" bash "${SCRIPT}" print)" \
  || fail "print let the environment override the persisted Compose project"
[[ "$(COMPOSE_PROJECT_NAME=manual_project ATTESTA_ROOT_DIR="${same_name_one}" bash "${SCRIPT}" project-name)" == "manual_project" ]] \
  || fail "explicit Compose project override was not preserved"

# Older generated files did not persist a project name. They derive the stable
# canonical identity and still ignore inherited shell/.env values.
legacy_file_root="${tmpdir}/legacy-file-root"
mkdir -p "${legacy_file_root}/scripts"
cp "${ROOT}/scripts/dev-ports.env" "${legacy_file_root}/scripts/dev-ports.env"
cp "${ROOT}/scripts/dev-ports.env" "${legacy_file_root}/.env.worktree"
legacy_generated="$(ATTESTA_ROOT_DIR="${legacy_file_root}" bash "${SCRIPT}" project-name)"
[[ -n "${legacy_generated}" && "${legacy_generated}" != intruder ]] \
  || fail "legacy env file did not derive a safe project identity"
[[ "$(COMPOSE_PROJECT_NAME=intruder ATTESTA_ROOT_DIR="${legacy_file_root}" bash "${SCRIPT}" project-name)" == "${legacy_generated}" ]] \
  || fail "environment overrode a legacy env file's generated identity"

# The exporter is safe to evaluate from checkout paths containing shell syntax,
# and it does not leak its internal repository path into callers.
export_root="${tmpdir}/path with spaces/\$(touch should-not-run)"
mkdir -p "${export_root}/scripts"
cp "${ROOT}/scripts/dev-ports.env" "${export_root}/scripts/dev-ports.env"
ATTESTA_ROOT_DIR="${export_root}" bash "${SCRIPT}" write --classic >/dev/null
export_output="$(ATTESTA_ROOT_DIR="${export_root}" bash "${SCRIPT}" export)"
if grep -q '^export ROOT_DIR=' <<<"${export_output}"; then
  fail "export exposed internal ROOT_DIR"
fi
(
  unset ROOT_DIR
  eval "${export_output}"
  [[ "${PORT}" == 3000 ]]
  [[ -z "${ROOT_DIR+x}" ]]
) || fail "export output was not safe to evaluate"
[[ ! -e "${ROOT}/should-not-run" && ! -e "${tmpdir}/should-not-run" ]] \
  || fail "evaluating exports executed checkout-path shell syntax"

# A TCP probe can report a listener even when lsof cannot see it (for example
# a Docker-published port owned by another user).
listener_port=13000
tcp_root="${tmpdir}/tcp-root"
mkdir -p "${tcp_root}/scripts"
cp "${ROOT}/scripts/dev-ports.env" "${tcp_root}/scripts/dev-ports.env"
PATH="${tmpdir}/bin:${PATH}" TEST_WT_PORT="${listener_port}" TEST_BUSY_PORT="${listener_port}" \
  ATTESTA_ROOT_DIR="${tcp_root}" bash "${SCRIPT}" write >/dev/null
if grep -qx "PORT=${listener_port}" "${tcp_root}/.env.worktree"; then
  fail "allocated a port with an active TCP listener"
fi

# Stopped sibling worktrees still reserve every manually persisted port.
reservation_repo="${tmpdir}/reservation-repo"
reservation_sibling="${tmpdir}/reservation-sibling"
mkdir -p "${reservation_repo}/scripts"
cp "${ROOT}/scripts/dev-ports.env" "${reservation_repo}/scripts/dev-ports.env"
git -C "${reservation_repo}" init -q
git -C "${reservation_repo}" -c user.name=Test -c user.email=test@example.com \
  commit -q --allow-empty -m initial
git -C "${reservation_repo}" worktree add -q -b reservation-sibling "${reservation_sibling}"
cat >"${reservation_sibling}/.env.worktree" <<'EOF'
PORT=11000
VITE_PORT=11001
MONGODB_PORT=11002
CERBOS_PORT=11003
MONGO_EXPRESS_PORT=11004
MAILPIT_SMTP_PORT=11005
MAILPIT_UI_PORT=11006
APPWRITE_HTTP_PORT=11007
APPWRITE_HTTPS_PORT=11008
DOCKER_APP_PORT=11009
COMPOSE_PROJECT_NAME=reserved-sibling
EOF
PATH="${tmpdir}/bin:${PATH}" ATTESTA_ROOT_DIR="${reservation_repo}" \
  bash "${SCRIPT}" write >/dev/null
grep -qx 'PORT=11010' "${reservation_repo}/.env.worktree" \
  || fail "allocated a port reserved by a stopped sibling worktree"

# Concurrent first bootstraps serialize allocation and atomically publish two
# complete, disjoint files. Sleeping in the deterministic Worktrunk stub keeps
# the first allocator inside the critical section while the second starts.
concurrent_repo="${tmpdir}/concurrent-repo"
concurrent_sibling="${tmpdir}/concurrent-sibling"
mkdir -p "${concurrent_repo}/scripts"
cp "${ROOT}/scripts/dev-ports.env" "${concurrent_repo}/scripts/dev-ports.env"
git -C "${concurrent_repo}" init -q
git -C "${concurrent_repo}" add scripts/dev-ports.env
git -C "${concurrent_repo}" -c user.name=Test -c user.email=test@example.com \
  commit -qm initial
git -C "${concurrent_repo}" worktree add -q -b concurrent-sibling "${concurrent_sibling}"

PATH="${tmpdir}/bin:${PATH}" TEST_WT_PORT=14000 TEST_WT_SLEEP_SECONDS=0.02 \
  ATTESTA_ROOT_DIR="${concurrent_repo}" bash "${SCRIPT}" write \
  >"${tmpdir}/concurrent-primary.out" 2>"${tmpdir}/concurrent-primary.err" &
concurrent_primary_pid=$!
PATH="${tmpdir}/bin:${PATH}" TEST_WT_PORT=14000 TEST_WT_SLEEP_SECONDS=0.02 \
  ATTESTA_ROOT_DIR="${concurrent_sibling}" bash "${SCRIPT}" write \
  >"${tmpdir}/concurrent-sibling.out" 2>"${tmpdir}/concurrent-sibling.err" &
concurrent_sibling_pid=$!
concurrent_status=0
wait "${concurrent_primary_pid}" || concurrent_status=1
wait "${concurrent_sibling_pid}" || concurrent_status=1
if [[ "${concurrent_status}" != "0" ]]; then
  cat "${tmpdir}/concurrent-primary.err" "${tmpdir}/concurrent-sibling.err" >&2
  fail "concurrent port allocation did not complete"
fi

for env_file in "${concurrent_repo}/.env.worktree" "${concurrent_sibling}/.env.worktree"; do
  [[ -f "${env_file}" ]] || fail "concurrent allocation did not publish ${env_file}"
  [[ "$(grep -Ec '^(PORT|[A-Z][A-Z0-9_]*_PORT)=[0-9]+$' "${env_file}")" -eq 10 ]] \
    || fail "concurrent allocation published an incomplete port file: ${env_file}"
  [[ "$(grep -c '^COMPOSE_PROJECT_NAME=' "${env_file}")" -eq 1 ]] \
    || fail "concurrent allocation published an incomplete project identity: ${env_file}"
done
concurrent_ports="$(
  sed -n -E 's/^(PORT|[A-Z][A-Z0-9_]*_PORT)=([0-9]+)$/\2/p' \
    "${concurrent_repo}/.env.worktree" "${concurrent_sibling}/.env.worktree"
)"
[[ "$(wc -l <<<"${concurrent_ports}" | tr -d '[:space:]')" -eq 20 ]] \
  || fail "concurrent allocation did not publish twenty ports"
[[ "$(sort -u <<<"${concurrent_ports}" | wc -l | tr -d '[:space:]')" -eq 20 ]] \
  || fail "concurrent sibling worktrees received overlapping ports"
if find "${concurrent_repo}" "${concurrent_sibling}" -maxdepth 1 -name '.env.worktree.tmp.*' | grep -q .; then
  fail "concurrent allocation left a partial publication file"
fi

# A killed allocator must not leave the repository permanently blocked.
port_lock_dir="$(git -C "${concurrent_repo}" rev-parse --path-format=absolute --git-common-dir)/attesta-worktree-port-allocation.lock"
rm "${concurrent_repo}/.env.worktree"
mkdir "${port_lock_dir}"
printf '%s\n' 99999999 >"${port_lock_dir}/pid"
PATH="${tmpdir}/bin:${PATH}" TEST_WT_PORT=14000 \
  ATTESTA_ROOT_DIR="${concurrent_repo}" bash "${SCRIPT}" write >/dev/null
[[ -f "${concurrent_repo}/.env.worktree" ]] \
  || fail "port allocation did not recover from a stale repository lock"
[[ ! -e "${port_lock_dir}" ]] \
  || fail "stale repository port-allocation lock was not cleaned up"

echo "ok: worktree-env_test"
