# Worktree Host-Dev DX Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `task worktree:add` + `task dev` reliable across git worktrees with one shared Docker stack, macOS-safe process cleanup, and per-checkout `PORT`/`VITE_PORT` via `.env.local`.

**Architecture:** Thin Taskfile wrappers call focused bash scripts under `deployment/scripts/`. Shared helpers live in `deployment/scripts/dev-lib.sh` (sourced, not executed). Compose always runs from the primary worktree root. Host-dev loads `.env` then `.env.local`, then forces Appwrite redirect URLs and `VITE_DEV_SERVER` from the resolved ports.

**Tech Stack:** bash, Taskfile, Docker Compose, air, Vite, existing git worktree layout under `.worktrees/`.

**Spec:** `docs/superpowers/specs/2026-07-23-worktree-dev-dx-design.md`

## Global Constraints

- Share one Docker stack; no per-worktree Compose/Appwrite
- `.env.local` holds only `PORT` and `VITE_PORT` (already gitignored); never put secrets there
- Primary checkout prefers ports `3000` / `5173`; other worktrees use ranges `3100–3199` / `5200–5299`
- Host-dev always overrides `APPWRITE_INVITE_REDIRECT_URL`, `APPWRITE_RESET_REDIRECT_URL`, and `VITE_DEV_SERVER` from `PORT`/`VITE_PORT`
- Kill only processes belonging to **this** checkout’s paths; never another worktree’s
- `task start` no-ops when `closer-mongodb`, `attesta-cerbos`, and `appwrite` are running; Compose cwd = primary root
- Defer `npm install` to `dev:web`; init submodules in `worktree:add`
- Prefer minimal diffs; do not change Go server listen logic (already reads `PORT` / `VITE_DEV_SERVER`)

## File map

| File | Responsibility |
|------|----------------|
| `deployment/scripts/dev-lib.sh` | Shared helpers: primary root, process cwd, port listening, env file load |
| `deployment/scripts/worktree-ports.sh` | Ensure `.env.local` with stable free ports |
| `deployment/scripts/worktree-env.sh` | Symlink `.env`; call ports script |
| `deployment/scripts/infra-up.sh` | Healthy-check + compose up from primary |
| `deployment/scripts/dev-env.sh` | Sourceable: load env + force host-dev overrides; print URL |
| `deployment/scripts/dev-stop-local.sh` | Path-scoped kill of this checkout’s air/vite/server |
| `deployment/scripts/dev-lib_test.sh` | Bash unit tests for lib + port allocation helpers |
| `Taskfile.yml` | Wire tasks; replace inline `/proc` kill and hardcoded Vite port |
| `README.md` | Worktrees subsection + update port override docs |

---

### Task 1: Shared bash helpers + unit tests

**Files:**
- Create: `deployment/scripts/dev-lib.sh`
- Create: `deployment/scripts/dev-lib_test.sh`

**Interfaces:**
- Consumes: `git`, `lsof` (macOS), `/proc` when present (Linux)
- Produces (functions in `dev-lib.sh`, safe to `source`):
  - `primary_worktree_root` → prints absolute primary path
  - `process_cwd <pid>` → prints absolute cwd or empty
  - `tcp_port_listening <port>` → exit 0 if something listens on TCP port
  - `load_env_file <path>` → exports `KEY=VAL` lines (skip blank/`#`); does not override vars already set in the environment
  - `is_primary_worktree <abs_root>` → exit 0 if abs_root equals primary
  - `stable_port_index <abs_root>` → prints integer 0–99 (hash of path)
  - `pick_free_port <start> <count> <preferred>` → prints first free port in `[start, start+count)` trying preferred first, then bump; exit 1 if none

- [ ] **Step 1: Write failing tests**

Create `deployment/scripts/dev-lib_test.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=deployment/scripts/dev-lib.sh
source "$ROOT/deployment/scripts/dev-lib.sh"

fail() { echo "FAIL: $*" >&2; exit 1; }

# load_env_file must not clobber pre-set PORT
tmp="$(mktemp)"
printf 'PORT=9999\nVITE_PORT=8888\n' >"$tmp"
export PORT=3001
load_env_file "$tmp"
[[ "$PORT" == "3001" ]] || fail "expected PORT to stay 3001, got $PORT"
[[ "$VITE_PORT" == "8888" ]] || fail "expected VITE_PORT 8888, got ${VITE_PORT:-}"
unset VITE_PORT
rm -f "$tmp"

# stable_port_index is deterministic and in 0..99
a="$(stable_port_index "/tmp/attesta-wt-a")"
b="$(stable_port_index "/tmp/attesta-wt-a")"
c="$(stable_port_index "/tmp/attesta-wt-b")"
[[ "$a" == "$b" ]] || fail "index not stable"
[[ "$a" =~ ^[0-9]+$ && "$a" -ge 0 && "$a" -le 99 ]] || fail "index out of range: $a"
# different paths usually differ; if equal, still OK as long as in range
[[ "$c" =~ ^[0-9]+$ && "$c" -ge 0 && "$c" -le 99 ]] || fail "index out of range: $c"

# pick_free_port returns preferred when free
# Use a high ephemeral preferred unlikely busy; skip if busy
pref=58761
if ! tcp_port_listening "$pref"; then
  got="$(pick_free_port 58700 100 "$pref")"
  [[ "$got" == "$pref" ]] || fail "expected preferred $pref, got $got"
fi

echo "ok: dev-lib_test"
```

- [ ] **Step 2: Run tests — expect fail (missing lib)**

Run: `bash deployment/scripts/dev-lib_test.sh`

Expected: FAIL with `No such file` or `source: ...dev-lib.sh: No such file`

- [ ] **Step 3: Implement `dev-lib.sh`**

Create `deployment/scripts/dev-lib.sh`:

```bash
#!/usr/bin/env bash
# Shared helpers for worktree / host-dev scripts. Source only; do not execute.

primary_worktree_root() {
  git worktree list --porcelain | awk '/^worktree / { print substr($0, 10); exit }'
}

is_primary_worktree() {
  local root="$1"
  local primary
  primary="$(primary_worktree_root)"
  [[ -n "$primary" && "$root" == "$primary" ]]
}

process_cwd() {
  local pid="$1"
  if [[ -d "/proc/$pid" ]]; then
    readlink "/proc/$pid/cwd" 2>/dev/null || true
    return 0
  fi
  # macOS: lsof cwd
  lsof -a -p "$pid" -d cwd -Fn 2>/dev/null | awk '/^n/ { print substr($0, 2); exit }'
}

tcp_port_listening() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
    return $?
  fi
  # fallback
  nc -z 127.0.0.1 "$port" >/dev/null 2>&1
}

# Export KEY=VAL from file without overriding already-set environment variables.
load_env_file() {
  local path="$1"
  [[ -f "$path" ]] || return 0
  local line key val
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    key="${line%%=*}"
    val="${line#*=}"
    key="$(echo "$key" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
    [[ -z "$key" ]] && continue
    if [[ -n "${!key+x}" ]]; then
      continue
    fi
    export "$key=$val"
  done <"$path"
}

stable_port_index() {
  local root="$1"
  # cksum first field; map to 0..99
  local sum
  sum="$(printf '%s' "$root" | cksum | awk '{print $1}')"
  echo $((sum % 100))
}

pick_free_port() {
  local start="$1"
  local count="$2"
  local preferred="$3"
  local candidates=()
  local i p
  if [[ "$preferred" -ge "$start" && "$preferred" -lt $((start + count)) ]]; then
    candidates+=("$preferred")
  fi
  for ((i = 0; i < count; i++)); do
    p=$((start + i))
    if [[ "$p" -ne "$preferred" ]]; then
      candidates+=("$p")
    fi
  done
  for p in "${candidates[@]}"; do
    if ! tcp_port_listening "$p"; then
      echo "$p"
      return 0
    fi
  done
  echo "error: no free port in ${start}-$((start + count - 1))" >&2
  return 1
}
```

- [ ] **Step 4: Run tests — expect pass**

Run: `bash deployment/scripts/dev-lib_test.sh`

Expected: `ok: dev-lib_test`

- [ ] **Step 5: Commit**

```bash
git add deployment/scripts/dev-lib.sh deployment/scripts/dev-lib_test.sh
git commit -m "$(cat <<'EOF'
feat(dev): add shared host-dev bash helpers

Port/env helpers for worktree DX, with a small bash self-test.
EOF
)"
```

---

### Task 2: `worktree-ports.sh` + wire into env/add

**Files:**
- Create: `deployment/scripts/worktree-ports.sh`
- Modify: `deployment/scripts/worktree-env.sh`
- Modify: `Taskfile.yml` (`worktree:add`, `worktree:env` desc)
- Extend: `deployment/scripts/dev-lib_test.sh` (ports file behavior via invoking the script)

**Interfaces:**
- Consumes: `primary_worktree_root`, `is_primary_worktree`, `stable_port_index`, `pick_free_port`, `tcp_port_listening` from `dev-lib.sh`
- Produces: `worktree-ports.sh [target_root]` → writes `$target_root/.env.local` with `PORT` and `VITE_PORT` if missing; prints `PORT=… VITE_PORT=…`; exit 0 if file already has both keys

- [ ] **Step 1: Extend tests for ports script**

Append to `dev-lib_test.sh` (or create `worktree-ports_test.sh` that invokes the script):

```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
fail() { echo "FAIL: $*" >&2; exit 1; }

tmpdir="$(mktemp -d)"
# Pretend non-primary by using a path under tmp (script uses real git primary for is_primary;
# for unit coverage, call pick logic indirectly by running script twice on same dir).
bash "$ROOT/deployment/scripts/worktree-ports.sh" "$tmpdir"
[[ -f "$tmpdir/.env.local" ]] || fail "missing .env.local"
grep -q '^PORT=' "$tmpdir/.env.local" || fail "missing PORT"
grep -q '^VITE_PORT=' "$tmpdir/.env.local" || fail "missing VITE_PORT"
# second run must not overwrite
cp "$tmpdir/.env.local" "$tmpdir/.env.local.bak"
bash "$ROOT/deployment/scripts/worktree-ports.sh" "$tmpdir"
diff -u "$tmpdir/.env.local.bak" "$tmpdir/.env.local" || fail "overwrote .env.local"
rm -rf "$tmpdir"
echo "ok: worktree-ports_test"
```

Save as `deployment/scripts/worktree-ports_test.sh` and chmod +x.

- [ ] **Step 2: Run test — expect fail**

Run: `bash deployment/scripts/worktree-ports_test.sh`

Expected: FAIL (script missing)

- [ ] **Step 3: Implement `worktree-ports.sh`**

```bash
#!/usr/bin/env bash
# Ensure target root has .env.local with PORT and VITE_PORT.
# Usage: worktree-ports.sh [target_root]
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=deployment/scripts/dev-lib.sh
source "$SCRIPT_DIR/dev-lib.sh"

target_root="${1:-$(git rev-parse --show-toplevel)}"
target_root="$(cd "$target_root" && pwd)"
env_local="${target_root}/.env.local"

read_key() {
  local file="$1" key="$2"
  [[ -f "$file" ]] || return 0
  awk -F= -v k="$key" '$1 == k { print $2; exit }' "$file"
}

existing_port="$(read_key "$env_local" PORT)"
existing_vite="$(read_key "$env_local" VITE_PORT)"
if [[ -n "${existing_port}" && -n "${existing_vite}" ]]; then
  echo "ok: ${env_local} PORT=${existing_port} VITE_PORT=${existing_vite}"
  exit 0
fi

if is_primary_worktree "$target_root"; then
  preferred_port=3000
  preferred_vite=5173
  port_start=3000
  port_count=100
  vite_start=5173
  vite_count=100
else
  idx="$(stable_port_index "$target_root")"
  preferred_port=$((3100 + idx))
  preferred_vite=$((5200 + idx))
  port_start=3100
  port_count=100
  vite_start=5200
  vite_count=100
fi

# Keep existing single key if present
if [[ -n "${existing_port}" ]]; then
  port="$existing_port"
else
  port="$(pick_free_port "$port_start" "$port_count" "$preferred_port")"
fi
if [[ -n "${existing_vite}" ]]; then
  vite_port="$existing_vite"
else
  vite_port="$(pick_free_port "$vite_start" "$vite_count" "$preferred_vite")"
fi

printf 'PORT=%s\nVITE_PORT=%s\n' "$port" "$vite_port" >"$env_local"
echo "wrote ${env_local} PORT=${port} VITE_PORT=${vite_port}"
```

- [ ] **Step 4: Update `worktree-env.sh` to also ensure ports**

After successful link (or early “already linked/exists” paths), call ports before every exit 0. Refactor so all success paths fall through:

At end of `worktree-env.sh` (replace early `exit 0` success branches with setting a status message, then always):

```bash
bash "${SCRIPT_DIR:-$(cd "$(dirname "$0")" && pwd)}/worktree-ports.sh" "${target_root}"
```

Concrete approach: add at top `SCRIPT_DIR=...`, and change each successful exit to call `worktree-ports.sh` then exit. Example for the “already linked” branch:

```bash
  echo "ok: ${dst} already linked to ${src}"
  bash "$SCRIPT_DIR/worktree-ports.sh" "$target_root"
  exit 0
```

Same for “already exists (real file)”, and after `ln -s`.

- [ ] **Step 5: Update `task worktree:add`**

In `Taskfile.yml`, after `worktree-env.sh`, add submodule init and a summary print:

```yaml
          bash deployment/scripts/worktree-env.sh "${path}"
          git -C "${path}" submodule update --init --recursive
          # ports already ensured by worktree-env.sh; print URL from .env.local
          port="$(awk -F= '/^PORT=/ { print $2; exit }' "${path}/.env.local")"
          echo "worktree ready: ${path}"
          echo "  open http://localhost:${port}"
          echo "  infra is shared — run 'task start' once from primary if needed"
```

Update descs:

```yaml
  worktree:env:
    desc: Symlink primary .env and ensure .env.local ports. Usage - task worktree:env [-- path]
  worktree:add:
    desc: Create .worktrees/<branch>, link .env, init submodules, assign ports. Usage - task worktree:add -- <branch>
```

- [ ] **Step 6: Run port tests**

Run: `bash deployment/scripts/worktree-ports_test.sh`

Expected: `ok: worktree-ports_test`

- [ ] **Step 7: Commit**

```bash
git add deployment/scripts/worktree-ports.sh deployment/scripts/worktree-ports_test.sh deployment/scripts/worktree-env.sh Taskfile.yml
git commit -m "$(cat <<'EOF'
feat(dev): assign per-worktree PORT/VITE_PORT in .env.local

Bootstrap worktrees with stable free ports and submodule init.
EOF
)"
```

---

### Task 3: Harden `task start` via `infra-up.sh`

**Files:**
- Create: `deployment/scripts/infra-up.sh`
- Modify: `Taskfile.yml` (`start`, `start:build`, `dev` compose stop path)

**Interfaces:**
- Consumes: `primary_worktree_root` from `dev-lib.sh`
- Produces: `infra-up.sh [--build]` → exit 0 if healthy or after compose up from primary root

- [ ] **Step 1: Implement `infra-up.sh`**

```bash
#!/usr/bin/env bash
# Bring up shared local infra from the primary worktree root.
# Usage: infra-up.sh [--build]
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=deployment/scripts/dev-lib.sh
source "$SCRIPT_DIR/dev-lib.sh"

build=0
if [[ "${1:-}" == "--build" ]]; then
  build=1
fi

container_running() {
  local name="$1"
  [[ "$(docker inspect -f '{{.State.Running}}' "$name" 2>/dev/null || echo false)" == "true" ]]
}

if container_running closer-mongodb && container_running attesta-cerbos && container_running appwrite; then
  echo "ok: infra already up (closer-mongodb, attesta-cerbos, appwrite)"
  exit 0
fi

primary="$(primary_worktree_root)"
if [[ -z "$primary" || ! -d "$primary" ]]; then
  echo "error: could not resolve primary worktree root" >&2
  exit 1
fi

compose=(docker compose -f deployment/docker-compose.local.yaml)
cd "$primary"
if [[ "$build" -eq 1 ]]; then
  "${compose[@]}" up -d --build
else
  "${compose[@]}" up -d
fi
"${compose[@]}" ps
```

- [ ] **Step 2: Wire Taskfile**

```yaml
  start:
    desc: Bring up shared infra from primary (no-op if already healthy).
    cmds:
      - bash deployment/scripts/infra-up.sh

  start:build:
    desc: Bring up shared infra with image rebuild (from primary).
    cmds:
      - task: submodule:init
      - bash deployment/scripts/infra-up.sh --build

  stop:
    desc: Stop shared infra for all worktrees (keeps volumes).
    # cmds unchanged

  reset:
    desc: Stop shared infra and remove volumes (affects all worktrees).
    # cmds unchanged
```

Update `dev` so compose stop also runs from primary (avoid wrong project cwd):

```yaml
  dev:
    desc: Hot reload backend + frontend (shared infra; per-checkout ports via .env.local).
    cmds:
      - task: start
      - |
          sh -c '
          set -euo pipefail
          primary="$(git worktree list --porcelain | awk "/^worktree / { print substr(\$0, 10); exit }")"
          docker compose -f "$primary/deployment/docker-compose.local.yaml" stop attesta
          task dev:server &
          server_pid=$!
          task dev:web &
          web_pid=$!
          wait $server_pid $web_pid
          '
```

- [ ] **Step 3: Manual verify**

Run (with Docker available):

```bash
task start
task start
```

Expected: second invocation prints `ok: infra already up ...` and does not recreate containers.

- [ ] **Step 4: Commit**

```bash
git add deployment/scripts/infra-up.sh Taskfile.yml
git commit -m "$(cat <<'EOF'
feat(dev): no-op shared infra start when stack is healthy

Always compose from the primary worktree so bind mounts stay stable.
EOF
)"
```

---

### Task 4: Path-scoped process stop (`dev-stop-local.sh`)

**Files:**
- Create: `deployment/scripts/dev-stop-local.sh`
- Modify: `Taskfile.yml` (`dev:server` kill block removed in Task 5; this task only adds the script)

**Interfaces:**
- Consumes: `process_cwd`, `tcp_port_listening` from `dev-lib.sh`
- Produces: `dev-stop-local.sh [repo_root]` → kills this tree’s air / `tmp/attesta-server` / vite; if `PORT`/`VITE_PORT` set and held by foreign process → exit 1 with message

- [ ] **Step 1: Implement `dev-stop-local.sh`**

```bash
#!/usr/bin/env bash
# Stop host-dev processes that belong to this checkout only.
# Usage: dev-stop-local.sh [repo_root]
# Optional env: PORT, VITE_PORT — if set, fail when a foreign process holds them.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=deployment/scripts/dev-lib.sh
source "$SCRIPT_DIR/dev-lib.sh"

repo_root="${1:-$(git rev-parse --show-toplevel)}"
repo_root="$(cd "$repo_root" && pwd)"
server_dir="${repo_root}/server"
web_dir="${repo_root}/web"
bin_path="${server_dir}/tmp/attesta-server"

kill_matching() {
  local pattern="$1"
  local want_cwd="$2"
  local pid cwd
  for pid in $(pgrep -f "$pattern" 2>/dev/null || true); do
    cwd="$(process_cwd "$pid")"
    if [[ "$cwd" == "$want_cwd" ]]; then
      kill "$pid" >/dev/null 2>&1 || true
    fi
  done
}

# air in this server dir
kill_matching 'air -c \.air\.toml' "$server_dir"
# built binary for this tree
if [[ -e "$bin_path" ]]; then
  for pid in $(pgrep -f "$bin_path" 2>/dev/null || true); do
    kill "$pid" >/dev/null 2>&1 || true
  done
fi
# vite for this web dir (node running vite with cwd=web)
kill_matching 'vite' "$web_dir"

sleep 1

port_holder_ok() {
  local port="$1"
  local want_cwd="$2"
  tcp_port_listening "$port" || return 0
  local pid cwd
  pid="$(lsof -nP -iTCP:"$port" -sTCP:LISTEN -t 2>/dev/null | head -n1 || true)"
  [[ -n "$pid" ]] || return 0
  cwd="$(process_cwd "$pid")"
  if [[ "$cwd" == "$want_cwd" || "$cwd" == "$server_dir" || "$cwd" == "$web_dir" ]]; then
    return 0
  fi
  # also accept if command line references this repo_root
  local cmd
  cmd="$(ps -p "$pid" -o command= 2>/dev/null || true)"
  if [[ "$cmd" == *"$repo_root"* ]]; then
    return 0
  fi
  echo "error: port ${port} is in use by pid ${pid} (cwd=${cwd:-unknown}), not this worktree" >&2
  echo "  set PORT/VITE_PORT differently or stop that process" >&2
  return 1
}

if [[ -n "${PORT:-}" ]]; then
  port_holder_ok "$PORT" "$server_dir"
fi
if [[ -n "${VITE_PORT:-}" ]]; then
  port_holder_ok "$VITE_PORT" "$web_dir"
fi
```

- [ ] **Step 2: Smoke — dry logic**

Run: `bash -n deployment/scripts/dev-stop-local.sh`

Expected: no output (syntax OK)

- [ ] **Step 3: Commit**

```bash
git add deployment/scripts/dev-stop-local.sh
git commit -m "$(cat <<'EOF'
feat(dev): path-scoped stop for air/vite per worktree

Replace /proc-only cwd checks so macOS host-dev cleanup works.
EOF
)"
```

---

### Task 5: `dev-env.sh` + wire `dev:server` / `dev:web`

**Files:**
- Create: `deployment/scripts/dev-env.sh`
- Modify: `Taskfile.yml` (`dev:server`, `dev:web`)

**Interfaces:**
- Consumes: `load_env_file` from `dev-lib.sh`; ensures `.env.local` via `worktree-ports.sh`
- Produces: when **sourced**, exports env and prints `Attesta http://localhost:$PORT (vite :$VITE_PORT)`

- [ ] **Step 1: Implement `dev-env.sh`**

```bash
#!/usr/bin/env bash
# Source from Taskfile host-dev tasks (do not execute as main).
# Usage from repo root:
#   set -a
#   # shellcheck disable=SC1091
#   source deployment/scripts/dev-env.sh
#   set +a
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
# If sourced from a worktree Taskfile, prefer git toplevel as repo root
if git rev-parse --show-toplevel >/dev/null 2>&1; then
  REPO_ROOT="$(git rev-parse --show-toplevel)"
fi
# shellcheck source=deployment/scripts/dev-lib.sh
source "$SCRIPT_DIR/dev-lib.sh"

bash "$SCRIPT_DIR/worktree-ports.sh" "$REPO_ROOT" >/dev/null

# Preserve caller overrides: load_env_file skips already-set keys.
# Load order: .env then .env.local (local wins for unset keys only —
# so load .env first, then .env.local).
load_env_file "$REPO_ROOT/.env"
load_env_file "$REPO_ROOT/.env.local"

PORT="${PORT:-3000}"
VITE_PORT="${VITE_PORT:-5173}"
export PORT
export VITE_PORT
export APPWRITE_INVITE_REDIRECT_URL="http://localhost:${PORT}/invite/accept"
export APPWRITE_RESET_REDIRECT_URL="http://localhost:${PORT}/reset/confirm"
export VITE_DEV_SERVER="http://localhost:${VITE_PORT}"

echo "Attesta http://localhost:${PORT} (vite :${VITE_PORT})"
```

Note on load order vs “local overrides”: because `load_env_file` skips already-set keys, loading `.env` first then `.env.local` means `.env` wins for duplicate keys — **wrong** for `PORT` if someone put `PORT` in `.env`. Spec says `.env.local` overrides `.env`. Fix by either:

1. Unset PORT/VITE_PORT from file loads selectively, or
2. Load `.env.local` keys with force for PORT/VITE_PORT only.

Implement force after both loads:

```bash
load_env_file "$REPO_ROOT/.env"
# Force PORT/VITE_PORT from .env.local when present (unless already set in shell before sourcing)
if [[ -z "${ATTESTA_DEV_ENV_PRESERVE:-}" ]]; then
  :
fi
# Simpler explicit rule matching spec:
# - shell-pre-set PORT/VITE_PORT win (detect via marking)
```

Use this clearer implementation in the file:

```bash
_shell_port="${PORT-}"
_shell_vite="${VITE_PORT-}"
# Clear so files can populate, then restore shell overrides
unset PORT VITE_PORT 2>/dev/null || true
load_env_file "$REPO_ROOT/.env"
load_env_file "$REPO_ROOT/.env.local"
# shell wins
if [[ -n "${_shell_port}" ]]; then export PORT="$_shell_port"; fi
if [[ -n "${_shell_vite}" ]]; then export VITE_PORT="$_shell_vite"; fi
PORT="${PORT:-3000}"
VITE_PORT="${VITE_PORT:-5173}"
export PORT VITE_PORT
export APPWRITE_INVITE_REDIRECT_URL="http://localhost:${PORT}/invite/accept"
export APPWRITE_RESET_REDIRECT_URL="http://localhost:${PORT}/reset/confirm"
export VITE_DEV_SERVER="http://localhost:${VITE_PORT}"
echo "Attesta http://localhost:${PORT} (vite :${VITE_PORT})"
```

Important: when Taskfile runs `export $(grep .env | xargs)` today it always sets vars. New flow must **not** pre-export from Taskfile before sourcing `dev-env.sh`. Use only `source deployment/scripts/dev-env.sh`. For `PORT=3001 task dev`, Task/make passes env into the task process — those are “shell pre-set” and must win. Detecting “was set before source” requires saving at the very top before unset — as above. Caveat: empty `_shell_port` when unset vs set-empty; use `${PORT+x}`:

```bash
_shell_has_port=0
_shell_has_vite=0
[[ -n "${PORT+x}" ]] && _shell_has_port=1 && _shell_port="$PORT"
[[ -n "${VITE_PORT+x}" ]] && _shell_has_vite=1 && _shell_vite="$VITE_PORT"
unset PORT VITE_PORT 2>/dev/null || true
load_env_file "$REPO_ROOT/.env"
load_env_file "$REPO_ROOT/.env.local"
[[ "$_shell_has_port" -eq 1 ]] && export PORT="$_shell_port"
[[ "$_shell_has_vite" -eq 1 ]] && export VITE_PORT="$_shell_vite"
```

- [ ] **Step 2: Rewrite `dev:server` and `dev:web` in Taskfile**

```yaml
  dev:server:
    desc: Hot reload Go server (ports from .env.local; macOS-safe restart).
    cmds:
      - task: goa:generate
      - sh -c 'cd server && GOCACHE=${GOCACHE:-/tmp/gocache-attesta} go mod tidy'
      - bash deployment/scripts/dev-stop-local.sh
      - sh -c 'cd server && go install github.com/air-verse/air@latest'
      - |
          bash -c '
          set -euo pipefail
          source deployment/scripts/dev-env.sh
          bash deployment/scripts/dev-stop-local.sh
          cd server
          bin="$(go env GOBIN)"
          if [ -z "$bin" ]; then bin="$(go env GOPATH)/bin"; fi
          "$bin/air" -c .air.toml
          '

  dev:web:
    desc: Vite dev server (HMR) on VITE_PORT from .env.local.
    cmds:
      - sh -c 'cd web && npm install'
      - |
          bash -c '
          set -euo pipefail
          source deployment/scripts/dev-env.sh
          cd web
          npm run dev -- --port "$VITE_PORT" --strictPort
          '
```

Ensure `dev-stop-local.sh` sees `PORT`/`VITE_PORT` on the second call inside `dev:server` (after `dev-env.sh` source) so foreign holders fail loudly before air starts. First call without ports only clears this tree’s processes.

- [ ] **Step 3: Manual verify (primary)**

```bash
# ensure .env.local exists
bash deployment/scripts/worktree-ports.sh
task dev:web
```

In another terminal, confirm Vite listens on the `VITE_PORT` from `.env.local` (`lsof -nP -iTCP:$VITE_PORT -sTCP:LISTEN`). Stop with Ctrl+C.

```bash
task dev:server
```

Confirm log/listen on `PORT`, and HTML includes `http://localhost:$VITE_PORT/@vite/client` when both run via `task dev`.

- [ ] **Step 4: Commit**

```bash
git add deployment/scripts/dev-env.sh Taskfile.yml
git commit -m "$(cat <<'EOF'
feat(dev): drive host-dev from .env.local ports

Force Appwrite redirects and Vite HMR URL from PORT/VITE_PORT.
EOF
)"
```

---

### Task 6: README worktrees section + final checklist

**Files:**
- Modify: `README.md` (after the “To run the backend on another port” block in Getting Started, or new subsection)

- [ ] **Step 1: Add Worktrees subsection**

Insert after the port override snippet (~line 131):

```markdown
### Git worktrees

Linked worktrees under `.worktrees/` share one Docker stack (Mongo, Appwrite, Cerbos, Mailpit) and the primary checkout’s `.env` (symlinked). Each worktree gets its own `.env.local` with `PORT` and `VITE_PORT`.

```bash
task worktree:add -- my-feature
task start   # once, from any checkout; no-op if already up
cd .worktrees/my-feature
task dev     # prints http://localhost:<PORT>
```

Notes:

- `task stop` / `task reset` affect **all** worktrees (shared infra).
- Cerbos policies mounted into the running stack come from the **primary** checkout.
- Override ports with `PORT=3001 VITE_PORT=5174 task dev` when needed.
- Parallel `task dev` in two worktrees works when their `.env.local` ports differ.
```

Update the older “To run the backend on another port” example to mention Vite:

```bash
PORT=3001 VITE_PORT=5174 task dev
```

- [ ] **Step 2: Run automated script tests**

```bash
bash deployment/scripts/dev-lib_test.sh
bash deployment/scripts/worktree-ports_test.sh
bash -n deployment/scripts/infra-up.sh
bash -n deployment/scripts/dev-stop-local.sh
bash -n deployment/scripts/dev-env.sh
```

Expected: both `ok:` lines; `bash -n` silent.

- [ ] **Step 3: Manual checklist (spec verification)**

- [ ] `task start` twice → second no-op  
- [ ] `task worktree:add -- dx-probe` → symlink `.env`, non-empty submodule, `.env.local` ports  
- [ ] From worktree, `task dev` prints worktree URL; Vite `--strictPort` matches  
- [ ] Optional: second worktree `task dev` on different ports  
- [ ] macOS: restart `task dev` in same tree does not leave duplicate air on that tree’s port  
- [ ] Server env redirects use `http://localhost:$PORT/...` not `:3030`

Remove the probe worktree when done:

```bash
git worktree remove .worktrees/dx-probe
```

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "$(cat <<'EOF'
docs: document worktree host-dev DX

Shared infra, .env.local ports, and parallel task dev notes.
EOF
)"
```

---

## Spec coverage (self-review)

| Spec requirement | Task |
|------------------|------|
| Symlink `.env` | Task 2 (`worktree-env.sh`) |
| Submodule init on add | Task 2 |
| `.env.local` ports + ranges | Task 2 |
| `worktree:env` = link + ports | Task 2 |
| Infra healthy no-op | Task 3 |
| Compose from primary | Task 3 |
| Stop only `attesta` for host-dev | Task 3 |
| Redirect + `VITE_DEV_SERVER` from ports | Task 5 |
| Vite `--port` + `strictPort` | Task 5 |
| Path-scoped kill (macOS) | Task 4–5 |
| Foreign port error | Task 4 |
| README worktrees | Task 6 |
| Defer npm install | unchanged in `dev:web` (Task 5) |

No TBD/placeholder steps remain after clarifying `.env.local` override semantics in Task 5.
