#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf -- "$tmpdir"' EXIT
primary="${tmpdir}/primary"
linked="${tmpdir}/linked"
mkdir -p "${primary}/deployment/scripts" "${primary}/scripts"
cp "${ROOT}/deployment/scripts/dev-env.sh" "${primary}/deployment/scripts/dev-env.sh"
cp "${ROOT}/deployment/scripts/dev-lib.sh" "${primary}/deployment/scripts/dev-lib.sh"
cp "${ROOT}/scripts/worktree-env.sh" "${primary}/scripts/worktree-env.sh"
cp "${ROOT}/scripts/dev-ports.env" "${primary}/scripts/dev-ports.env"

cat >"${primary}/.env" <<'EOF'
MONGODB_URI=mongodb://primary.example:27099/attesta
CERBOS_URL=https://cerbos.primary.example
APPWRITE_ENDPOINT=https://appwrite.primary.example/v1
SMTP_HOST=smtp.primary.example
SMTP_PORT=2525
COMPOSE_PROJECT_NAME=must-not-win
EOF

git -C "${primary}" init -q
git -C "${primary}" add .
git -C "${primary}" -c user.name=Test -c user.email=test@example.com commit -qm initial
git -C "${primary}" worktree add -q -b linked "${linked}"

ATTESTA_ROOT_DIR="${primary}" bash "${primary}/scripts/worktree-env.sh" write --classic >/dev/null
ATTESTA_ROOT_DIR="${linked}" bash "${linked}/scripts/worktree-env.sh" write --classic >/dev/null
sed -i.bak \
  -e 's/^PORT=.*/PORT=13000/' \
  -e 's/^VITE_PORT=.*/VITE_PORT=13001/' \
  -e 's/^MONGODB_PORT=.*/MONGODB_PORT=13002/' \
  -e 's/^CERBOS_PORT=.*/CERBOS_PORT=13003/' \
  -e 's/^MAILPIT_SMTP_PORT=.*/MAILPIT_SMTP_PORT=13004/' \
  -e 's/^APPWRITE_HTTP_PORT=.*/APPWRITE_HTTP_PORT=13005/' \
  "${linked}/.env.worktree"
rm "${linked}/.env.worktree.bak"

primary_output="$({
  cd "${primary}"
  APPWRITE_ENDPOINT=https://shell.primary.example/v1 bash -c '
    set -euo pipefail
    source deployment/scripts/dev-env.sh
    printf "%s\n" \
      "MONGODB_URI=${MONGODB_URI}" \
      "CERBOS_URL=${CERBOS_URL}" \
      "APPWRITE_ENDPOINT=${APPWRITE_ENDPOINT}" \
      "SMTP_HOST=${SMTP_HOST}" \
      "SMTP_PORT=${SMTP_PORT}" \
      "COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME}"
  '
})"
grep -qx 'MONGODB_URI=mongodb://primary.example:27099/attesta' <<<"${primary_output}" \
  || fail "primary MongoDB endpoint was clobbered"
grep -qx 'CERBOS_URL=https://cerbos.primary.example' <<<"${primary_output}" \
  || fail "primary Cerbos endpoint was clobbered"
grep -qx 'APPWRITE_ENDPOINT=https://shell.primary.example/v1' <<<"${primary_output}" \
  || fail "primary shell Appwrite endpoint was clobbered"
grep -qx 'SMTP_HOST=smtp.primary.example' <<<"${primary_output}" \
  || fail "primary SMTP host was clobbered"
grep -qx 'SMTP_PORT=2525' <<<"${primary_output}" \
  || fail "primary SMTP port was clobbered"
if grep -qx 'COMPOSE_PROJECT_NAME=must-not-win' <<<"${primary_output}"; then
  fail "primary .env overrode persisted Compose identity"
fi

linked_output="$({
  cd "${linked}"
  APPWRITE_ENDPOINT=https://unsafe.example/v1 COMPOSE_PROJECT_NAME=unsafe bash -c '
    set -euo pipefail
    source deployment/scripts/dev-env.sh
    printf "%s\n" \
      "PORT=${PORT}" \
      "MONGODB_URI=${MONGODB_URI}" \
      "CERBOS_URL=${CERBOS_URL}" \
      "APPWRITE_ENDPOINT=${APPWRITE_ENDPOINT}" \
      "SMTP_HOST=${SMTP_HOST}" \
      "SMTP_PORT=${SMTP_PORT}" \
      "COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME}"
  '
})"
grep -qx 'PORT=13000' <<<"${linked_output}" || fail "linked worktree ignored its persisted port"
grep -qx 'MONGODB_URI=mongodb://localhost:13002' <<<"${linked_output}" \
  || fail "linked worktree retained the primary MongoDB endpoint"
grep -qx 'CERBOS_URL=http://localhost:13003' <<<"${linked_output}" \
  || fail "linked worktree retained the primary Cerbos endpoint"
grep -qx 'APPWRITE_ENDPOINT=http://localhost:13005/v1' <<<"${linked_output}" \
  || fail "linked worktree retained an inherited Appwrite endpoint"
grep -qx 'SMTP_HOST=localhost' <<<"${linked_output}" || fail "linked SMTP host is not isolated"
grep -qx 'SMTP_PORT=13004' <<<"${linked_output}" || fail "linked SMTP port is not isolated"
if grep -Eq 'COMPOSE_PROJECT_NAME=(unsafe|must-not-win)' <<<"${linked_output}"; then
  fail "linked worktree retained an inherited Compose identity"
fi

# Invalid persisted ports fail before a host process can start with fallback
# endpoints from .env or the shell.
sed -i.bak 's/^VITE_PORT=.*/VITE_PORT=13000/' "${linked}/.env.worktree"
rm "${linked}/.env.worktree.bak"
if (
  cd "${linked}"
  bash -c 'set -euo pipefail; source deployment/scripts/dev-env.sh; echo survived'
) >"${tmpdir}/invalid-out" 2>"${tmpdir}/invalid-error"; then
  fail "invalid worktree environment unexpectedly succeeded"
fi
if grep -q '^survived$' "${tmpdir}/invalid-out"; then
  fail "dev environment continued after exporter failure"
fi
grep -q 'must use different ports' "${tmpdir}/invalid-error" \
  || fail "dev environment hid the exporter failure"

echo "ok: dev-env_test"
