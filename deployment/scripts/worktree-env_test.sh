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
echo 11000
EOF
cat >"${tmpdir}/bin/lsof" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
chmod +x "${tmpdir}/bin/wt" "${tmpdir}/bin/lsof"

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
[[ "${project_one}" != "${project_two}" ]] \
  || fail "same-basename worktrees received the same Compose project name"
[[ "${project_one}" == "$(ATTESTA_ROOT_DIR="${same_name_one}" bash "${SCRIPT}" project-name)" ]] \
  || fail "Compose project name is not stable"
[[ "${project_one}" =~ ^[a-z0-9][a-z0-9_-]*$ ]] \
  || fail "Compose project name is not Compose-safe: ${project_one}"
[[ ${#project_one} -le 63 ]] \
  || fail "Compose project name is not reasonably bounded: ${project_one}"

# Independent primary clones with the same basename also remain isolated.
independent_one="${tmpdir}/independent-one/shared-primary"
independent_two="${tmpdir}/independent-two/shared-primary"
mkdir -p "${independent_one}" "${independent_two}"
git -C "${independent_one}" init -q
git -C "${independent_two}" init -q
independent_project_one="$(ATTESTA_ROOT_DIR="${independent_one}" bash "${SCRIPT}" project-name)"
independent_project_two="$(ATTESTA_ROOT_DIR="${independent_two}" bash "${SCRIPT}" project-name)"
[[ "${independent_project_one}" != "${independent_project_two}" ]] \
  || fail "same-basename primary clones received the same Compose project name"
[[ "${independent_project_one}" =~ ^[a-z0-9][a-z0-9_-]*$ ]] \
  || fail "primary Compose project name is not safe: ${independent_project_one}"
[[ ${#independent_project_one} -le 63 ]] \
  || fail "primary Compose project name is not bounded: ${independent_project_one}"

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
[[ "$(COMPOSE_PROJECT_NAME=manual_project ATTESTA_ROOT_DIR="${same_name_one}" bash "${SCRIPT}" project-name)" == "manual_project" ]] \
  || fail "explicit Compose project override was not preserved"

echo "ok: worktree-env_test"
