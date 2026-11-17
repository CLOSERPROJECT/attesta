# Architecture

How Attesta’s seams connect. Vocabulary layers: [`GLOSSARY-MAP.md`](../GLOSSARY-MAP.md). Decisions: [`adr/`](adr/). Ops: README / QUICKSTART / DOCKER. This is not a file inventory or runbook.

## Two product surfaces

| Surface | UI | Runtime law | HTTP |
| --- | --- | --- | --- |
| **Stream** (`/my`) | Go templates + HTMX + `web/` | Formata Streams in Mongo (catalog); on-disk `server/config/` is seed/template only | `server/cmd/server` |
| **Org** (`/app`) | SvelteKit `app/` | Active Org Config + Domain | `app/` → `/api/org/*` → `server/internal/org` |

**Org ↛ Stream:** `/app` does not load Streams or Stream instances. Org Config ≠ Stream blueprint. Shared membership (Organization, Affiliation, roles) gates both; see root [`GLOSSARY.md`](../GLOSSARY.md).

Platform admin (`/admin`) and public home (`/`) stay on the Stream shell. They are not the Org Worklist.

## Shell → Org executor → SPA

```text
browser
  ├─ /my/…          → cmd/server (Stream HTMX)
  ├─ /app/…         → cmd/server handleAppSPA → app/ build (or APP_DEV_SERVER)
  │                      └─ fetch /api/org/… → org.API (Gate = verified session
  │                           → OperatorIdentity from Affiliation)
  │                           └─ Load / Commit / QR / artifacts
  ├─ /admin/…       → cmd/server (platform admin; Cerbos platform_admin)
  └─ /01/…/10/…/21/… → cmd/server Digital Link (Org DPP first, else Stream)
```

- **`cmd/server`** is the process shell: mux, Appwrite session, Stream handlers, mounts `org.API`, serves `/app`.
- **`server/internal/org`** is the Org **dumb executor**: evaluates Active Config against Domain (Worklist projection, Commit, passport mint, HttpPort). Plant open/close law lives in Config, not in Go or `app/` branches.
- **`app/`** is a dumb client: Draft is browser-local (ADR-0009); Domain writes only happen through Commit on `/api/org`.

## Shared infrastructure seams

```text
Appwrite ──session / teams / membership──► both /my and /app+/api/org
Cerbos   ──one PDP, split policies──────► Stream (substep, formata, org_admin, …)
                                         Org   (resource task / action commit)
Mongo    ──one DB, split collections────► Stream: processes, formata_*, GridFS
                                         Org:    org_domains, org_configs,
                                                 org_active_config, org_dpps,
                                                 org_submission_links
```

Progress / entity keys with `.` are stored as `_` and normalized on read (same rule on Stream Process and Org Domain — see [`AGENTS.md`](../AGENTS.md)).

Organization role slugs (Appwrite) should align with Config role slugs where Cerbos checks membership.

## Where runtime law lives

```text
Seed / prepare
  ├─ Formata catalog (+ demo Stream instances)  → Stream runtime
  └─ Org Active Config + Domain (+ identity)    → Org runtime
         ▲
         │ authored plant payload (e.g. Feragame YAML)
         │ platform gates / onCommit / transforms → docs/agents/org-config-authoring.md
```

- Empty Formata catalog fails closed on the production path (no disk-YAML fallback for live Streams).
- Org Active Config pins the Config version Domain is evaluated against.
- Feragame is one Organization’s Config/Domain **payload**, not platform ADR law.

## Dual passport, one Digital Link

Issuance is isolated; the public URL is shared.

```text
Commit (Org, Config passport)  → org_dpps  ─┐
Process done (Stream dpp.*)    → process   ─┼─► GET /01/{gtin}/10/{lot}/21/{serial}
                                            │      shell: Org identity hit first
                                            │      else Stream process Digital Link
                                            └─ shared untp helpers; stores stay apart
```

Org hit: Org HTML/credential/linkset; Stream-only extras (`/events`, process attachments) are not served on that branch. Detail: [`untp.md`](untp.md), ADR-0007.

## Adjacent seams (not the executor)

- **HttpPort** — Config-declared side effects after Commit; env sidecar (`ORG_HTTP_*`); artifact bytes via `/api/org/artifacts/` (ADR-0010). Not the passport issue path.
- **Org-admin / Formata** under `/my/…` — shared membership + Cerbos `org_admin_console` / `formata_builder` on the Stream shell; still not `/app`.

## Where to go next

| Need | Doc |
| --- | --- |
| Which glossary | [`GLOSSARY-MAP.md`](../GLOSSARY-MAP.md) |
| Platform decisions | [`adr/`](adr/) (format: [`adr/README.md`](adr/README.md)) |
| Org Config authoring | [`agents/org-config-authoring.md`](agents/org-config-authoring.md) |
| Feragame fixture | [`agents/feragame-fixture-authoring.md`](agents/feragame-fixture-authoring.md) |
| Digital Link / UNTP | [`untp.md`](untp.md) |
| Day-1 / Compose | [`QUICKSTART.md`](../QUICKSTART.md), [`DOCKER.md`](../DOCKER.md) |
