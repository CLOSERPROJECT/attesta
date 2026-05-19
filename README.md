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
- Docker and Docker Compose

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

Open:

- Attesta: `http://localhost:3000`
- API docs: `http://localhost:3000/docs`
- Appwrite Console: `http://localhost`
- Mailpit: `http://localhost:8025`

To run the backend on another port:

```bash
PORT=3001 task dev
```

**[🔝 back to top](#toc)**

---

## ⚙️ Configuration

Common environment variables:

- `PORT` or `ADDR` - backend listen address, default `:3000`
- `MONGODB_URI` - default `mongodb://localhost:27017`
- `CERBOS_URL` - default `http://localhost:3592`
- `APPWRITE_ENDPOINT` - default `http://appwrite/v1`
- `APPWRITE_PROJECT_ID`
- `APPWRITE_API_KEY`
- `APPWRITE_INVITE_REDIRECT_URL`
- `APPWRITE_RESET_REDIRECT_URL`
- `APPWRITE_ORG_ASSETS_BUCKET` - default `org-assets`
- `WORKFLOW_CONFIG` - default `config/workflow.yaml`
- `ATTACHMENT_MAX_BYTES` - default 25 MiB
- `ANYONE_CAN_CREATE_ACCOUNT`
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
3. Keep `ANYONE_CAN_CREATE_ACCOUNT=false` unless public signup is intended.
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

This repository is part of the CLOSER project and has received funding from the
European Union. Views and opinions expressed are those of the author(s) only and
do not necessarily reflect those of the European Union or EISMEA. Neither the
European Union nor the granting authority can be held responsible for them.

**[🔝 back to top](#toc)**

---

## 📖 More Documentation

- [QUICKSTART.md](QUICKSTART.md) - step-by-step local setup.
- [DOCKER.md](DOCKER.md) - Docker Compose, Coolify, and preview deployment notes.
- [Taskfile.yml](Taskfile.yml) - available local commands.