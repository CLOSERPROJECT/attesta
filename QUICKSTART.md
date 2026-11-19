# Quick Start

Ordered local-dev recipe for this checkout. Product overview → [README.md](README.md). Compose services, Coolify, lifecycle detail → [DOCKER.md](DOCKER.md).

## Prerequisites

- Docker + Docker Compose 2.24.4+
- Go 1.25+
- Node.js 18+ (npm) and Bun (for `/app`)
- [Task](https://taskfile.dev)

Or install the pinned toolchain with [mise](https://mise.jdx.dev/installing-mise.html):

```bash
mise trust
mise install
```

For parallel worktrees, `mise install` also pins Worktrunk (`wt`).

## Clone and env

```bash
git clone https://github.com/CLOSERPROJECT/attesta
cd attesta
cp .env.example .env
```

Ports and Compose identity live in gitignored `.env.worktree` (written once by bootstrap; later edits are preserved). Appwrite runtime credentials after prepare live in `.env.appwrite`.

## Worktrees

```bash
mise install
wt config shell install
wt config approvals add   # once: review shared bootstrap/cleanup hooks
wt switch -c my-feature
```

- Worktrunk / Codex / Cursor worktrees run `scripts/worktree-bootstrap.sh` automatically; plain Git worktrees: `bash scripts/worktree-bootstrap.sh`.
- Bootstrap writes `.env.worktree` only when missing (path-qualified Compose project + host ports).
- Each checkout starts with empty Compose volumes. Demo data comes only from the Go seed after Appwrite credentials exist (`ATTESTA_SEED` / `task seed`).

## Start infrastructure

```bash
task start
```

Brings up this checkout’s Compose project (MongoDB, Cerbos, Appwrite, Mailpit, …) and runs `attesta prepare` when `ATTESTA_SEED` is set (worktree-env default: `true`).

Alternatives:

- `task start:docker` — full stack including the containerized app on `DOCKER_APP_PORT`
- `task start:docker:build` — rebuild that image first

`task start` is infrastructure-only; host app processes come from `task dev` below.

Stop / reset / purge / orphan GC for this checkout: see [DOCKER.md](DOCKER.md). After `task reset` or `task purge`, the next start can re-apply the Go demo seed when credentials exist.

## Bootstrap Appwrite

Preferred: with `ATTESTA_SEED=true`, `task start` / `task start:docker` run `attesta prepare --policy=worktree` (wait → bootstrap → empty-only seed by value). Prepare writes checkout-scoped `.env.appwrite` (`APPWRITE_PROJECT_ID` / `APPWRITE_API_KEY`) for the server. Soft-skips when bootstrap cannot finish yet; re-run `task start` after credentials exist, or run `task seed`.

Console defaults when unset: `admin@appwrite.local` / `admin@appwrite.local`.

Manual console fallback (stack already up):

1. Open the Appwrite console (`APPWRITE_CONSOLE_URL` from `bash scripts/worktree-env.sh print`).
2. Create the first console account.
3. Create the Attesta project.
4. Create an API key for Attesta.
5. Create the `org-assets` storage bucket.
6. Put `APPWRITE_PROJECT_ID` / `APPWRITE_API_KEY` in `.env`, or let prepare write `.env.appwrite`.
7. Re-run `task start` (or `task seed`) so the empty-only Go seed can apply.

Invite / reset / verify links use `PUBLIC_BASE_URL` (from worktree-env / compose) plus fixed paths; if unset, they use the request host. Inspect mail on `MAILPIT_UI_URL` from the same print command.

## Run the app

```bash
task dev
```

Starts infrastructure (via `task start`), then hot-reloads the Go server, Vite (`web`), and SvelteKit `/app` on this checkout’s ports.

One-shot backend without HMR: `task run:server`. Prefer Task over bare `npm` / `go run`.

## Open the demo

Print this checkout’s URLs and ports:

```bash
bash scripts/worktree-env.sh print
```

Use the printed values (do not assume `:3000` / `:80` / `:8025` on every worktree):

| Surface              | Env key                                       |
| -------------------- | --------------------------------------------- |
| Attesta (host-dev)   | `PORT` → `http://localhost:${PORT}/`          |
| Vite HMR             | `VITE_PORT`                                   |
| SvelteKit `/app` HMR | `APP_PORT`                                    |
| Appwrite console     | `APPWRITE_CONSOLE_URL`                        |
| Mailpit              | `MAILPIT_UI_URL`                              |
| Containerized app    | `DOCKER_APP_PORT` (after `task start:docker`) |

After login, the stream picker is `/my`. Stream routes live under `/my/streams/{workflowKey}/...` (legacy `/w/` and `/org-admin/` return 404). Platform admin (when enabled): `/admin`. API docs: `/docs`.

## Seed

Demo data comes only from the Go seed (`attesta seed` / `ATTESTA_SEED`). Product boot does not auto-load Feragame or wipe Domain.

- Empty-only (default): `task seed` / prepare seed is a no-op while the marker org exists.
- Rewrite: `task seed -- --replace` (wipes and rebuilds org Domain + passports). After Org Config-as-Stream, re-seed — there is no migration of old `org_*` rows.
- Identity: org `feragame`, all-roles `admin@feragame.com`, plus `administration@feragame.com`, `logistic@feragame.com`, `line@feragame.com` (password = email). Existing stacks need `--replace` once to pick up new users. Everyone lands on `/my` (stream dashboard), then opens a truck into `/app`.
- Domain + Active Config: same seed sets feragame-config as Active on stream key `recycling` (from the fixture) and, when Domain is missing/empty or `--replace` is used, rebuilds a staged Domain for that stream. No harness `/api/runs` snapshot PUT.
- Staged Domain (parallel-truck demo): two active trucks — truck A with one finished lot (DPP issued) and one open lot mid-weigh; truck B still in intake. Truck B's "Apri lotto" shows line busy while A's lot is open.
- Feragame scopes (skip Formata; always run; scoped to the fixture stream):
  - `task seed:feragame-config` — Active Config + wipe Domain empty (no Appwrite)
  - `task seed:feragame-domain` — Config + staged Domain
  - `task seed:feragame-identity` — Feragame Appwrite + Config + empty Domain
  - `task seed:feragame-full` — Appwrite + Config + staged Domain
  - `--keep-domain` on config/identity only: refresh Active Config for the stream without touching that stream's Domain
- If the marker org exists but Feragame Domain is still empty (pre-staging seed), run `task seed -- --replace` or `task seed:feragame-full` once.
- Memory-backed Domain clears on server restart; re-run `task seed` (or `task seed:feragame-full` / `--replace`) after a Memory-backed restart if you need the fixture again. Normal host-dev with Mongo uses durable Domain stores.

See `attesta seed --help` (via `task seed -- --help`).

## Migration notes

- Existing Mongo-backed sessions are not migrated. After Appwrite credentials are configured, users must log in again through Appwrite.
- Existing Mongo invite and password-reset tokens are not migrated. Re-issue any still-needed invite or reset from the Appwrite-backed Attesta flows.
- Cutting over an existing demo dataset: migrate organizations, roles, and active users into Appwrite before relying on `/my/organization/*`.

## Tests

```bash
task test    # unit suite (Go + worktree scripts + CSS lint + /app unit)
task cover   # same suite with >= 90% Go coverage gate (override with COVER_MIN)
```

Integration (needs Compose Mongo/Cerbos):

```bash
task test:integration
```

## Optional DPP

Digital Link / UNTP behavior: [README.md](README.md) (Digital Product Passport) and [docs/untp.md](docs/untp.md).
