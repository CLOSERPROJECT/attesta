# Architecture

How Attesta’s seams connect. Vocabulary layers: [`GLOSSARY-MAP.md`](../GLOSSARY-MAP.md). Decisions: [`adr/`](adr/). Ops: README / QUICKSTART / DOCKER. This is not a file inventory or runbook.

## Two product surfaces

| Surface | UI | Runtime law | HTTP |
| --- | --- | --- | --- |
| **`/my`** | Go templates + HTMX + `web/` | Formata Streams in Mongo (catalog), **and** Org-backed streams via `server/internal/org` (Active Config + Domain) | `server/cmd/server` |
| **`/app`** | SvelteKit `app/` | Active Org Config + Domain, scoped to one **Org stream instance** | `app/` → `/api/org/*` → `server/internal/org` |

`/my` is the affiliated landing: Stream discovery, Org-backed stream dashboards, and Formata stream chrome. `/app` is the instance-scoped operator Worklist for an Org stream instance ([ADR-0014](adr/0014-org-config-as-stream.md)). Shared membership (Organization, Affiliation, roles) gates both; see root [`GLOSSARY.md`](../GLOSSARY.md).

Platform admin (`/admin`) and public home (`/`) stay on the `/my` shell. They are not the Org Worklist.

## Shell → Org executor → SPA

```text
browser
  ├─ /my/…          → cmd/server (Formata Streams + Org-backed stream catalog/dashboard/timeline
  │                      via org package: ListStreams, instances, ProjectStreamTimeline)
  ├─ /app/{stream}/… → cmd/server handleAppSPA → app/ build (or APP_DEV_SERVER)
  │                      └─ fetch /api/org/… → org.API (Gate = verified session
  │                           → OperatorIdentity from Affiliation; stream + instance scope)
  │                           └─ Load / Commit / QR / artifacts
  ├─ /admin/…       → cmd/server (platform admin; Cerbos platform_admin)
  └─ /01/…/10/…/21/… → cmd/server Digital Link (Org DPP first, else Stream)
```

- **`cmd/server`** is the process shell: mux, Appwrite session, `/my` handlers (Formata and Org-backed), mounts `org.API`, serves `/app`.
- **`server/internal/org`** is the Org **dumb executor**: evaluates Active Config against Domain (Worklist projection, Commit, passport mint, HttpPort, stream instance/timeline projection). Plant open/close law lives in Config, not in Go or `app/` branches.
- **`app/`** is a dumb client: Draft is browser-local (ADR-0009); Domain writes only happen through Commit on `/api/org`. Routes are scoped to one Org stream instance.
- After Commit, the executor publishes the new **Domain revision** on an in-process, stream-scoped change feed; `/app` boards subscribe per stream and refresh through the same Load path as after their own submit ([ADR-0015](adr/0015-org-domain-change-feed.md)).

## Shared infrastructure seams

```text
Appwrite ──session / teams / membership──► both /my and /app+/api/org
Cerbos   ──one PDP, split policies──────► Stream (substep, formata, org_admin, …)
                                         Org   (resource task / action commit)
Mongo    ──one DB, split collections────► Formata: processes, formata_*, GridFS
                                         Org:    org_domains, org_configs,
                                                 org_active_config, org_dpps,
                                                 org_submission_links
                                                 (keys include stream; Active
                                                  Config and Domain are per
                                                  (org, stream))
```

Progress / entity keys with `.` are stored as `_` and normalized on read (same rule on Stream Process and Org Domain — see [`AGENTS.md`](../AGENTS.md)).

Organization role slugs (Appwrite) should align with Config role slugs where Cerbos checks membership.

## Where runtime law lives

```text
Seed / prepare
  ├─ Formata catalog (+ demo Stream instances)  → Formata /my runtime
  └─ Org Active Config + Domain per stream (+ identity) → Org runtime
         ▲
         │ authored plant payload (e.g. Feragame YAML with stream section)
         │ platform gates / onCommit / transforms → docs/agents/org-config-authoring.md
```

- Empty Formata catalog fails closed on the production path (no disk-YAML fallback for live Formata Streams).
- Org Active Config pins the Config version Domain is evaluated against, per **Org stream**.
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
- **Org-admin / Formata** under `/my/…` — shared membership + Cerbos `org_admin_console` / `formata_builder` on the `/my` shell; still not `/app`.

## Where to go next

| Need | Doc |
| --- | --- |
| Which glossary | [`GLOSSARY-MAP.md`](../GLOSSARY-MAP.md) |
| Platform decisions | [`adr/`](adr/) (format: [`adr/README.md`](adr/README.md)) |
| Org Config authoring | [`agents/org-config-authoring.md`](agents/org-config-authoring.md) |
| Feragame fixture | [`agents/feragame-fixture-authoring.md`](agents/feragame-fixture-authoring.md) |
| Digital Link / UNTP | [`untp.md`](untp.md) |
| Day-1 / Compose | [`QUICKSTART.md`](../QUICKSTART.md), [`DOCKER.md`](../DOCKER.md) |
