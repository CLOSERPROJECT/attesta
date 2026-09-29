#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

test_root="$(mktemp -d)"
primary_root="$(mktemp -d)"
trap 'rm -rf -- "$test_root" "$primary_root"' EXIT
mkdir -p "${test_root}/scripts" "${test_root}/bin"
cp "${ROOT}/scripts/worktree-bootstrap.sh" "${test_root}/scripts/worktree-bootstrap.sh"
touch "${test_root}/mise.toml" "${test_root}/.worktreeinclude"
printf '.env\n' >"${test_root}/.worktreeinclude"
printf '.env\n' >"${primary_root}/.worktreeinclude"
printf 'SECRET=test\n' >"${test_root}/.env"
printf 'SECRET=primary\n' >"${primary_root}/.env"

cat >"${test_root}/scripts/worktree-env.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "env $*" >>"${TEST_LOG}"
if [[ "${1:-}" == "write" ]]; then
  touch "${ATTESTA_TEST_ROOT}/.env.worktree"
elif [[ "${1:-}" == "print" ]]; then
  echo "PORT=3000"
fi
EOF
cat >"${test_root}/scripts/worktree-data.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "data $*" >>"${TEST_LOG}"
EOF
cat >"${test_root}/bin/mise" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "mise $*" >>"${TEST_LOG}"
if [[ "$*" == "ls --missing --no-header" && ! -f "${ATTESTA_TEST_ROOT}/.mise-installed" ]]; then
  echo "task missing"
elif [[ "$*" == "install" ]]; then
  touch "${ATTESTA_TEST_ROOT}/.mise-installed"
fi
EOF
cat >"${test_root}/bin/git" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1 $2 $3" == "worktree list --porcelain" ]]; then
  printf 'worktree %s\nHEAD deadbeef\nbranch refs/heads/main\n\n' "${ATTESTA_TEST_PRIMARY}"
elif [[ "$1 $2 $3" == "submodule update --init" && "${4:-}" == "--recursive" ]]; then
  echo "git submodule update --init --recursive" >>"${TEST_LOG}"
else
  echo "unexpected git invocation: $*" >&2
  exit 1
fi
EOF
cat >"${test_root}/bin/wt" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "wt $*" >>"${TEST_LOG}"
if [[ "$*" == "step copy-ignored --require-include" ]]; then
  cp "${ATTESTA_TEST_PRIMARY}/.env" "${ATTESTA_TEST_ROOT}/.env"
else
  exit 1
fi
EOF
chmod +x "${test_root}/scripts/"*.sh "${test_root}/bin/"*

log_file="${test_root}/calls.log"
PATH="${test_root}/bin:/usr/bin:/bin" \
  TEST_LOG="${log_file}" \
  ATTESTA_TEST_ROOT="${test_root}" \
  ATTESTA_TEST_PRIMARY="${primary_root}" \
  bash "${test_root}/scripts/worktree-bootstrap.sh" >"${test_root}/output"

expected_prefix="$(cat <<EOF
mise trust --yes ${test_root}/mise.toml
mise ls --missing --no-header
mise install
env write
data snapshot-primary
git submodule update --init --recursive
env print
EOF
)"
[[ "$(cat "${log_file}")" == "${expected_prefix}" ]] \
  || fail "unexpected bootstrap calls: $(cat "${log_file}")"
grep -q 'bootstrap complete' "${test_root}/output" || fail "missing completion output"
if grep -q '^wt ' "${log_file}"; then
  fail "bootstrap invoked Worktrunk even though Codex had already copied ignored files"
fi

# Worktrunk calls the same script and retains its copy-ignored behavior when an
# allowlisted setup file is missing.
rm "${test_root}/.env"
: >"${log_file}"
PATH="${test_root}/bin:/usr/bin:/bin" \
  TEST_LOG="${log_file}" \
  ATTESTA_TEST_ROOT="${test_root}" \
  ATTESTA_TEST_PRIMARY="${primary_root}" \
  bash "${test_root}/scripts/worktree-bootstrap.sh" >"${test_root}/worktrunk-output"
grep -q '^wt step copy-ignored --require-include$' "${log_file}" \
  || fail "Worktrunk worktree did not use copy-ignored"
grep -q '^SECRET=primary$' "${test_root}/.env" \
  || fail "Worktrunk worktree did not receive primary .env"
if grep -q '^mise install$' "${log_file}"; then
  fail "idempotent bootstrap reinstalled an already complete mise toolset"
fi

# A plain Git worktree without Worktrunk can copy the same allowlisted ignored
# setup files directly from the primary checkout.
rm "${test_root}/.env" "${test_root}/bin/wt"
: >"${log_file}"
PATH="${test_root}/bin:/usr/bin:/bin" \
  TEST_LOG="${log_file}" \
  ATTESTA_TEST_ROOT="${test_root}" \
  ATTESTA_TEST_PRIMARY="${primary_root}" \
  bash "${test_root}/scripts/worktree-bootstrap.sh" >"${test_root}/plain-output"
grep -q '^SECRET=primary$' "${test_root}/.env" \
  || fail "plain worktree did not receive primary .env"

# Primary setup uses classic ports and does not snapshot itself.
rm "${test_root}/.env"
cp "${test_root}/scripts/worktree-bootstrap.sh" "${primary_root}/scripts-worktree-bootstrap.sh"
mkdir -p "${primary_root}/scripts" "${primary_root}/bin"
cp "${test_root}/scripts/worktree-bootstrap.sh" "${primary_root}/scripts/worktree-bootstrap.sh"
cp "${test_root}/scripts/worktree-env.sh" "${primary_root}/scripts/worktree-env.sh"
cp "${test_root}/scripts/worktree-data.sh" "${primary_root}/scripts/worktree-data.sh"
cp "${test_root}/bin/mise" "${primary_root}/bin/mise"
cp "${test_root}/bin/git" "${primary_root}/bin/git"
touch "${primary_root}/mise.toml"
: >"${log_file}"
PATH="${primary_root}/bin:/usr/bin:/bin" \
  TEST_LOG="${log_file}" \
  ATTESTA_TEST_ROOT="${primary_root}" \
  ATTESTA_TEST_PRIMARY="${primary_root}" \
  bash "${primary_root}/scripts/worktree-bootstrap.sh" >"${primary_root}/output"
grep -q '^env write --classic$' "${log_file}" || fail "primary did not use classic ports"
if grep -q '^data ' "${log_file}"; then
  fail "primary bootstrap unexpectedly captured a data snapshot"
fi

echo "ok: worktree-bootstrap_test"
