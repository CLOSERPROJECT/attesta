<div align="center">

<img
    src="https://raw.githubusercontent.com/CLOSERPROJECT/attesta/refs/heads/master/web/public/favicon.png"
    alt="Attesta logo"
    height="90"/>

# Attesta <!-- omit in toc -->

## Authenticated, role-gated stream traceability with evidence capture, policy checks, and Digital Product Passport publishing. <!-- omit in toc -->

</div>

<br>

Attesta turns regulated, multi-party streams into verifiable digital records.
Each organization sees the actions it is allowed to complete, submits structured
data or file evidence, and follows a live timeline from intake to final
notarization.

It is built for traceability demos where accountability matters: policy checks
enforce roles and step order, MongoDB stores process state and attachments,
Appwrite manages organizations and users, and completed streams can publish a
GS1 Digital Link / Digital Product Passport for external verification.

- 🔐 Appwrite-backed sessions, organizations, roles, invites, and user labels.
- 🧾 YAML or Formata Builder streams.
- 🛡️ Cerbos authorization for role and sequence-gated actions.
- 📎 MongoDB and GridFS storage for evidence and attachments.
- ⚡ HTMX pages with SSE updates for live process views.
- 🪪 Optional DPP landing pages and JSON exports under `/01/...`.
- 📡 OpenAPI documentation served at `/docs`.

<br>

---

<div id="toc">

## 🚩 Table of Contents <!-- omit in toc -->

- [🧱 Stack](#-stack)
- [✅ Requirements](#-requirements)
- [⚡ Quick Start](#-quick-start)
- [⚙️ Configuration](#️-configuration)
- [🪪 Digital Product Passport](#-digital-product-passport)
- [🗂️ Project Layout](#️-project-layout)
- [🚢 Deployment Notes](#-deployment-notes)
- [💼 License](#-license)
- [💶 Funding](#-funding)
- [📖 More Documentation](#-more-documentation)

</div>

## 🧱 Stack

- Go 1.25+, `net/http`, `html/template`
- MongoDB 7 and GridFS
- Appwrite for accounts, sessions, teams, memberships, labels, and org assets
- Cerbos PDP for authorization
- Vite, JavaScript, CSS, HTMX, SSE
- Docker Compose for local infrastructure

**[🔝 back to top](#toc)**

---

## ✅ Requirements

Use either the manual toolchain:

- Go 1.25+
- Node.js 18+ with npm
- [Task](https://taskfile.dev)
- Docker and Docker Compose 2.24.4+

Or use [mise](https://mise.jdx.dev/installing-mise.html) plus Docker:

```bash
mise trust
mise install
```

**[🔝 back to top](#toc)**

---

## ⚡ Quick Start

Clone the repo and create local environment settings:

```bash
git clone https://github.com/CLOSERPROJECT/attesta
cd attesta
cp .env.example .env
```

Start local infrastructure:

```bash
task start
```

Bootstrap Appwrite:

1. Open `http://localhost/console/register`.
2. Create the first Appwrite console account.
3. Create an Appwrite project.
4. Copy the project ID into `.env` as `APPWRITE_PROJECT_ID`.
5. Create an API key with Auth permissions and Storage file read/write permissions.
6. Copy the API key into `.env` as `APPWRITE_API_KEY`.
7. Create a storage bucket with bucket ID `org-assets`.

Run the app in development mode:

```bash
task dev
```

To run the complete stack in Docker instead, use `task start:docker` (or
`task start:docker:build` after changing the image). The containerized app is
published on `DOCKER_APP_PORT`; normal `task start` remains infrastructure-only.

Open:

- Attesta: `http://localhost:3000`
- API docs: `http://localhost:3000/docs`
- Appwrite Console: `http://localhost`
- Mailpit: `http://localhost:8025`

To run the backend on another port:

```bash
PORT=3001 VITE_PORT=5174 task dev
```

### Git worktrees

Attesta follows the same Worktrunk bootstrap shape as Credimi.
The primary checkout keeps the classic ports from `scripts/dev-ports.env`.
Every parallel checkout receives deterministic, available application and infrastructure ports in a gitignored `.env.worktree`; bootstrap never overwrites that file.

```bash
mise install
wt config shell install
wt config approvals add  # review and approve the shared bootstrap/cleanup hooks once
wt switch -c my-feature
task dev
```

Codex-managed worktrees use the checked-in local environment to run the same bootstrap automatically.
Codex also copies the ignored paths allowlisted in `.worktreeinclude`.
For a plain Git worktree, run bootstrap explicitly:

```bash
bash scripts/worktree-bootstrap.sh
task dev
```

Bootstrap copies the allowlisted ignored development files when needed, trusts the checkout's `mise.toml`, initializes submodules, writes `.env.worktree` only when it is missing, and captures one coordinated primary-data bundle.
After bootstrap, `task dev` reruns the same setup idempotently.

Each checkout owns a separate Compose project derived from its canonical path, so
containers, networks, MongoDB, and Appwrite volumes stay isolated across linked
worktrees and independent clones—even when two checkouts share a directory name.
`task start`, `task stop`, `task reset`, `task status`, and `task logs` are all scoped to the current worktree.

Before this worktree setup, `docker compose -f deployment/docker-compose.local.yaml` used `deployment` as its implicit project name.
To keep those existing volumes attached, add `COMPOSE_PROJECT_NAME=deployment` once to the primary `.env.worktree` (or pass it for the first `task worktree:bootstrap` write). New checkouts do not need that.

Bootstrap of a linked worktree fails closed if the primary's managed project has no durable volumes while legacy `deployment_*` volumes still exist under a mismatched primary project name.

As in Credimi, Worktrunk's `hash_port` derives the initial value for each service.
Bootstrap checks live TCP listeners and the ports reserved in sibling worktrees, then walks forward on collisions before writing `PORT`, `VITE_PORT`, `MONGODB_PORT`, `CERBOS_PORT`, `MONGO_EXPRESS_PORT`, `MAILPIT_SMTP_PORT`, `MAILPIT_UI_PORT`, `APPWRITE_HTTP_PORT`, `APPWRITE_HTTPS_PORT`, and `DOCKER_APP_PORT`.
Edit the generated `.env.worktree` for any manual override.
Later bootstraps preserve it, and duplicate manual values are rejected before Docker starts.

The snapshot bundle is stored with private permissions in ignored `.worktree-data/`.
It contains MongoDB together with Appwrite MariaDB and its durable uploads, imports, functions, sites, and builds.
Bootstrap temporarily quiesces the primary Compose project while capturing the bundle, then restores its prior running state.
Snapshot creation is serialized across all worktrees through the repository's shared Git directory.
Restore is all-or-nothing and happens only before an uninitialized worktree stack starts; partial or corrupt bundles fail closed.
If the primary has no durable Docker volumes yet, bootstrap creates no bundle and records a `.worktree-data/fresh-seed` choice so later starts do not recopy new primary data into that worktree.
Existing worktree data also suppresses a new primary snapshot.
Redis, caches, certificates, sessions, and host routing state are regenerated, so sign in again in the new worktree.
The bundle is point-in-time: later changes in either checkout do not synchronize.

`task stop` preserves the current worktree's volumes.
`task reset` deletes the volumes, so the next start restores the saved coordinated bundle.
`task purge` deletes the volumes and bundle and records the fresh-seed choice; its next start initializes from the checked-in seed.
To deliberately copy the primary again after a purge, remove `.worktree-data/fresh-seed` before that next start.
Worktrunk removal runs a non-interactive `pre-remove` hook that deletes the worktree's Compose volumes.
Codex can remove managed worktrees without that hook; use the checked-in purge action before archiving, or run `task worktree:gc` from an active checkout to preview orphaned Attesta Docker resources, then `task worktree:gc:apply` to remove them.
Garbage collection is scoped to this repository clone and ignores foreign or unlabeled legacy resources.
Reset and purge enumerate every volume in the current Compose project and refuse deletion unless all ownership labels match the current checkout.
Volumes preserved from the legacy `deployment` project are intentionally unlabeled, so reset and purge refuse them too.
To discard those after migration, first inspect the exact list with `docker volume ls --filter label=com.docker.compose.project=deployment`, then remove only the confirmed legacy volume names explicitly; the next start creates fully labeled replacements.

**[🔝 back to top](#toc)**

---

## ⚙️ Configuration

Common environment variables:

- `PORT` or `ADDR` - backend listen address, default `:3000`
- `VITE_PORT` - Vite dev-server port, default `5173`
- `MONGODB_PORT`, `CERBOS_PORT` - host ports for this checkout's infrastructure
- `APPWRITE_HTTP_PORT`, `APPWRITE_HTTPS_PORT` - this checkout's Appwrite entrypoints
- `MAILPIT_SMTP_PORT`, `MAILPIT_UI_PORT` - this checkout's local mail service
- `MONGODB_URI`, `CERBOS_URL`, `APPWRITE_ENDPOINT`, `SMTP_HOST`, `SMTP_PORT` - explicit `.env` or shell values are preserved for host development in the primary checkout; linked worktrees always resolve these to their isolated local ports
- `APPWRITE_PROJECT_ID`
- `APPWRITE_API_KEY`
- `APPWRITE_INVITE_REDIRECT_URL`
- `APPWRITE_RESET_REDIRECT_URL`
- `APPWRITE_ORG_ASSETS_BUCKET` - default `org-assets`
- `WORKFLOW_CONFIG` - default `config/workflow.yaml`
- `ATTACHMENT_MAX_BYTES` - default 25 MiB
- `ANYONE_CAN_CREATE_ACCOUNT` - default `true` (open registration); set `false` as an emergency kill switch for signup
- `SESSION_TTL_DAYS`
- `COOKIE_SECURE`

See `.env.example` for local defaults.

**[🔝 back to top](#toc)**

---

## 🪪 Digital Product Passport

DPP generation is configured per stream:

```yaml
dpp:
  enabled: true
  gtin: "09506000134352"
  lotInputKey: "batchId"
  lotDefault: ""
  serialInputKey: ""
  serialStrategy: "process_id_hex"
```

When a process first reaches `done`, Attesta stores stable DPP identifiers on
the process and exposes a public Digital Link page:

```text
/01/{GTIN}/10/{LOT}/21/{SERIAL}
```

Use `Accept: application/json` or `?format=json` to retrieve the JSON export.

**[🔝 back to top](#toc)**

---

## 🗂️ Project Layout

```text
server/       Go server, templates, generated API docs, stream config
web/          Vite frontend bundle source and build output
cerbos/       Cerbos config and policies
deployment/   Dockerfiles and Docker Compose files
Taskfile.yml  Common local development commands
```

**[🔝 back to top](#toc)**

---

## 🚢 Deployment Notes

Before deploying:

1. Set Appwrite project, API key, invite URL, reset URL, and org assets bucket.
2. Set `COOKIE_SECURE=true` behind HTTPS.
3. Open registration is the product default (`ANYONE_CAN_CREATE_ACCOUNT` defaults to `true`); set it to `false` only as an emergency kill switch.
4. Verify MongoDB and Cerbos connectivity.
5. Bootstrap initial organizations and org-admin users in Appwrite.
6. Confirm stream YAML organization and role slugs match Appwrite state.
7. Decide whether public DPP Digital Link routes should be exposed.

Docker and Coolify details live in [DOCKER.md](DOCKER.md).

**[🔝 back to top](#toc)**

---

## 💼 License

```
Attesta
Copyright 2025-2026 Forkbomb bv (forkbomb.eu), The Forkbomb Company.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
```

**[🔝 back to top](#toc)**

---

## 💶 Funding

CLOSER (Circular raw materiaLs for european Open Strategic autonomy on chips and
microElectronics pRoduction, Project No. 101161109) is funded by the European
Union under the Interregional Innovation Investments (I3) Instrument of the
European Regional Development Fund, managed by the European Innovation Council
and SMEs Executive Agency (EISMEA).

Even Closer (Project No. 101228240) is funded by the European Union under the
Interregional Innovation Investments (I3) Instrument of the European Regional
Development Fund, managed by the European Innovation Council and SMEs Executive
Agency (EISMEA).

This repository is part of CLOSER and Even Closer projects and has received funding from the
European Union. Views and opinions expressed are those of the author(s) only and
do not necessarily reflect those of the European Union or EISMEA. Neither the
European Union nor the granting authority can be held responsible for them.

**[🔝 back to top](#toc)**

---

## 📖 More Documentation

- [QUICKSTART.md](QUICKSTART.md) - step-by-step local setup.
- [DOCKER.md](DOCKER.md) - Docker Compose, Coolify, and preview deployment notes.
- [docs/architecture.md](docs/architecture.md) - system boundaries and source navigation.
- [docs/untp.md](docs/untp.md) - Digital Product Passport and UNTP implementation.
- [Taskfile.yml](Taskfile.yml) - available local commands.
