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
EOF
project_name="$(ATTESTA_ROOT_DIR="${tmpdir}" bash "${SCRIPT}" project-name)"
sed -i.bak "s/worktree tmpdir/worktree ${project_name}/" "${tmpdir}/expected"
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

echo "ok: worktree-env_test"
