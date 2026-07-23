# Worktree host-dev DX (shared infra + auto ports)

Date: 2026-07-23  
Scope: Taskfile + shell scripts for worktree bootstrap, shared Docker infra start, and host `task dev` port/process isolation  
Out of scope: Per-worktree Docker/Appwrite stacks, changing Appwrite bootstrap UX, sharing `node_modules` across worktrees, Coolify/ephemeral deploy paths

## Goal

Make git worktrees a reliable place to run `task dev`: one shared local infra stack, no container name fights, macOS-safe process cleanup, and automatic per-checkout backend/Vite ports so one-at-a-time is smooth and parallel checkouts are possible without manual port juggling.

## Decisions

1. **Share one Docker stack** (Mongo, Cerbos, Appwrite, Mailpit, mongo-express). Worktrees do not get isolated data.
2. **Harden `task start`**: if the stack is already healthy, no-op; never recreate fighting containers from a worktree.
3. **Per-checkout app ports** via gitignored `.env.local` (`PORT`, `VITE_PORT`), not via editing the shared/symlinked `.env`.
4. **Host-dev redirect URLs always follow `PORT`**, overriding hardcoded values from the shared `.env` (today often `:3030` for the Compose `attesta` service).
5. **Kill only this checkout’s** air/vite/server processes (path-scoped). Do not kill another worktree’s processes.
6. **Defer `npm install`** to `task dev:web` (keep `worktree:add` fast). Submodule init belongs in bootstrap.

## Current friction (why)

| Issue | Effect |
|-------|--------|
| Fixed `:3000` / `:5173` | Second `task dev` fails with bind errors |
| `PORT=…` does not move Vite or `VITE_DEV_SERVER` | Partial port override breaks HMR |
| Shared `.env` redirect URLs | Invite/reset links miss the host-dev port |
| Fixed Compose `container_name`s | `task start` from a worktree fights the same stack |
| `worktree:add` only links `.env` | Empty submodules (e.g. `formata-arch`), no ports file |
| `dev:server` kill uses `/proc/$pid/cwd` | Broken on macOS; stale air can linger |

## Architecture

```text
Primary checkout                    Worktree checkout
─────────────────                   ─────────────────
.env  (real file, secrets)  ──ln──► .env
.env.local (PORT/VITE defaults)     .env.local (own PORT/VITE)
Docker stack (once)  ◄───────────── task start (no-op if healthy)
task dev ── air :PORT               task dev ── air :PORT'
       └─ vite :VITE_PORT                  └─ vite :VITE_PORT'
```

**Units**

| Unit | Responsibility | Interface |
|------|----------------|-----------|
| `deployment/scripts/worktree-env.sh` | Symlink primary `.env` | `worktree-env.sh [target_root]` |
| `deployment/scripts/worktree-ports.sh` (new) | Ensure `.env.local` with free/stable ports | `worktree-ports.sh [target_root]` |
| `deployment/scripts/infra-up.sh` (new) | Healthy-check then compose up | used by `task start` |
| `deployment/scripts/dev-env.sh` (new) | Load `.env` + `.env.local`, export host-dev overrides | sourced by `dev:server` / `dev:web` |
| `deployment/scripts/dev-stop-local.sh` (new) | Path-scoped kill of this checkout’s air/vite/bin | used by `dev:server` / `dev` |
| `Taskfile.yml` | Wire tasks; thin wrappers | `worktree:add`, `start`, `dev*` |

Script names may be collapsed (e.g. one `worktree-setup.sh`) if it stays readable; behavior above is normative.

## Bootstrap

### `task worktree:add -- <branch>`

1. Create `.worktrees/<branch>` (existing branch or `-b`).
2. Symlink primary `.env` (existing `worktree-env.sh`).
3. `git -C <path> submodule update --init --recursive`.
4. Ensure `<path>/.env.local` via port allocator (do not overwrite an existing `.env.local`).
5. Print path, branch, `http://localhost:$PORT`, and that infra is shared.

### `task worktree:env`

**env link + ensure ports** for the target root. Submodule init stays on `worktree:add` and on `goa:generate` / `submodule:init` as today.

## Shared infra (`task start`)

**Healthy** means these containers exist and are running (names from current Compose):

- `closer-mongodb`
- `attesta-cerbos`
- `appwrite` (representative of the Appwrite graph)

Optional: also check `attesta-mailpit`. Do not require the `attesta` app container (host-dev stops it).

**Behavior**

- If healthy → print that infra is already up; exit 0.
- Else → run Compose from the **primary worktree root** (first `git worktree list` entry), not from the caller’s cwd when that cwd is a linked worktree. This keeps Cerbos `../cerbos` bind-mounts and Appwrite paths stable.
- Command: `docker compose -f deployment/docker-compose.local.yaml up -d` (and existing `start:build` variant with `--build`) with working directory = primary root.
- Compose continues to use fixed `container_name`s; project name stays as today. No per-worktree project.

**`task stop` / `reset` / `purge`**

Unchanged semantics; document that they affect **all** worktrees sharing the stack.

**`task dev` and the `attesta` container**

Keep stopping the Compose `attesta` service so host air can own the app process. Do not stop Mongo/Appwrite/Cerbos/Mailpit.

## Ports & `.env.local`

`.env.local` is already gitignored.

**Keys**

```bash
PORT=3000
VITE_PORT=5173
```

**Allocation (`worktree-ports.sh`)**

1. If `.env.local` already defines both ports → leave as-is (unless a flag forces regenerate — not required for v1).
2. Else choose defaults:
   - Primary worktree (first entry in `git worktree list`): prefer `3000` / `5173`.
   - Other worktrees: derive a stable pair from a hash of the absolute worktree path into ranges **backend `3100–3199`**, **vite `5200–5299`** (same hash index for both ranges).
3. If a chosen port is already listening, bump within the range (wrap once); if none free, fail with a clear error.
4. Write `.env.local` with `PORT` and `VITE_PORT` only (no secrets).

**Runtime load order** (`dev-env.sh`)

1. Export from repo-root `.env` (skip comments/blank).
2. Export from `.env.local` (overrides).
3. Shell env (`PORT=3001 task dev`) wins over both when already set before the script runs — document that Taskfile must not clobber pre-set `PORT`/`VITE_PORT`.
4. Force for host-dev (always, after load):

   ```bash
   APPWRITE_INVITE_REDIRECT_URL=http://localhost:${PORT}/invite/accept
   APPWRITE_RESET_REDIRECT_URL=http://localhost:${PORT}/reset/confirm
   VITE_DEV_SERVER=http://localhost:${VITE_PORT}
   ```

5. Print: `Attesta http://localhost:$PORT (vite :$VITE_PORT)`.

**Vite**

```bash
cd web && npm run dev -- --port "$VITE_PORT" --strictPort
```

(or equivalent in `package.json` / Taskfile). `strictPort` fails loudly instead of silently picking another port (which would desync `VITE_DEV_SERVER`).

**Air / Go server**

Unchanged listen via existing `PORT` / `ADDR` (`listenAddrFromEnv`). `VITE_DEV_SERVER` already read in `main.go`.

## Process lifecycle

Replace `/proc/$pid/cwd` matching with macOS- and Linux-safe path scoping:

1. Before starting air in this checkout: kill processes that are clearly **this** tree’s:
   - binary path equals `$ROOT/server/tmp/attesta-server`, or
   - command line contains `air -c .air.toml` and cwd is `$ROOT/server` (cwd via `lsof -a -p $pid -d cwd` on macOS, `/proc/$pid/cwd` on Linux when present).
2. Same for vite: cwd `$ROOT/web` or cmdline tied to that path.
3. Do **not** kill processes whose cwd is another worktree.
4. If `$PORT` or `$VITE_PORT` is listening but the holder is **not** this checkout’s process → exit with message naming the port and suggesting another `PORT`/`VITE_PORT` or stopping the other process.

One-at-a-time: stop `task dev` in A, start in B — B uses its `.env.local` ports; no collision. Parallel: both run if ports differ.

## Docs

- `README.md` — short “Worktrees” subsection: `task worktree:add` → `task start` once → `cd` worktree → `task dev` → use printed URL; note shared DB/auth; note `stop`/`reset` are global.
- Task `desc:` strings for `worktree:add`, `start`, `dev` updated to match.
- No AGENTS.md change required unless it documents the broken `/proc` kill or worktree flow.

## Testing / verification

Automated tests are optional for shell; prefer a short manual checklist in the plan:

1. From primary: `task start` twice → second is no-op.
2. `task worktree:add -- dx-probe` → `.env` symlink, submodules non-empty, `.env.local` with ports ≠ primary (or documented defaults).
3. `PORT`/`VITE_PORT` from worktree `.env.local` appear in printed URL; Vite strictPort matches.
4. Two worktrees: both `task dev` bind successfully on different ports; HMR script tags point at each worktree’s `VITE_DEV_SERVER`.
5. On macOS: restart `task dev` in same worktree kills prior air/vite for that tree only.
6. Invite/reset redirect env in the running server process matches `http://localhost:$PORT/...` (not `:3030`).

## Non-goals (explicit)

- Per-worktree Compose project / Appwrite / volumes
- Editing Cerbos policies from a linked worktree while infra was started from primary (bind-mount is primary’s `cerbos/`; document that policy edits for the running stack belong in primary unless you recreate the stack)
- Symlinking `web/node_modules` between trees
