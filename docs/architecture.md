# Architecture

Attesta is a Go HTTP application that renders server templates, serves a Vite
bundle, persists process data in MongoDB/GridFS, uses Appwrite for identity and
organization state, and asks Cerbos whether an actor may complete a substep.

This is navigation, not a replacement for code, configuration, or active-work
tracking. Unresolved work lives as GitHub issues (see
[`agents/issue-tracker.md`](agents/issue-tracker.md)).

## Boundaries and source locations

| Concern | Source of truth |
|---|---|
| HTTP routes and request handlers | `server/cmd/server/main.go`: `newMux`, `handleMyRoutes`, `handleStreamRoutes`, `handleProcessRoutes`, and `handleOrganizationRoutes` |
| Runtime Stream catalog | Formata builder Streams in Mongo (`Server.workflowCatalog` / `loadWorkflowCatalog`); empty Formata fails closed — no disk YAML fallback on the production path |
| Catalog template pack and taxonomy seed | On-disk `server/config/`: Stream YAML templates via `openCatalogTemplatePack` for empty-install foundation and Seed; `categories.yaml` taxonomy seed via `resolveCategoriesSeedPath` |
| Process completion and lifecycle artifacts | `server/cmd/server/process_completion.go`; persistence interface and implementations in `server/cmd/server/store.go` |
| Persistence and attachments | `server/cmd/server/store.go`: Mongo collections and GridFS `attachments` bucket |
| Identity, sessions, and organization membership | `server/cmd/server/main.go`: `readSession` and `currentUser`; Appwrite client code in `server/cmd/server/identity*.go` |
| Completion authorization | `server/cmd/server/authorizer.go`; `cerbos/policies/substep_policy.yaml` |
| Live updates | `server/cmd/server/main.go`: `SSEHub` and stream-scoped events; `web/src/main.js` |
| Digital Product Passport and UNTP surface | `server/cmd/server/dpp.go`, `server/cmd/server/untp.go`, and the public handler in `server/cmd/server/main.go`; see [`untp.md`](untp.md) |
| Templates and static assets | `server/cmd/server/templates.go`; `server/templates/`; `web/src/`; generated `web/dist/` is served under `/static/` |

## Route shape

The public homepage is `/`; authenticated application entry is `/my`; stream
instances live under `/my/streams/:key/`. Platform administration begins at
`/admin`. The handler registrations are authoritative for the complete route
surface.

## Decisions

| Theme | ADR |
|---|---|
| Presentation-only stream edits keep instances | [`adr/0001-stream-presentation-edit-without-purge.md`](adr/0001-stream-presentation-edit-without-purge.md) |
| Affiliation required before stream authoring | [`adr/0002-affiliation-gated-stream-authoring.md`](adr/0002-affiliation-gated-stream-authoring.md) |
| Email verification gate after auth | [`adr/0003-email-verification-gate.md`](adr/0003-email-verification-gate.md) |
| Host-specific demo seed invoke | [`adr/0004-host-specific-demo-seed-invoke.md`](adr/0004-host-specific-demo-seed-invoke.md) |
| Living Worklist is server-projected | [`adr/0005-living-worklist-server-projection.md`](adr/0005-living-worklist-server-projection.md) |
| Living `/app` home is act-now Worklist | [`adr/0006-living-operator-worklist-home.md`](adr/0006-living-operator-worklist-home.md) |
| Living Passport production in Org Config | [`adr/0007-living-passport-production.md`](adr/0007-living-passport-production.md) |
| Living Task submit once/many | [`adr/0008-living-task-submit-cardinality.md`](adr/0008-living-task-submit-cardinality.md) |
| Living client Draft + Bozza chrome | [`adr/0009-living-client-draft-persistence.md`](adr/0009-living-client-draft-persistence.md) |
| Living HttpPort path-shaped side effects | [`adr/0010-living-http-side-effects.md`](adr/0010-living-http-side-effects.md) |
| Living QR Config-declared resolve | [`adr/0011-living-qr-resolve.md`](adr/0011-living-qr-resolve.md) |
| Living Commit explicit mint intent | [`adr/0012-living-commit-mint-intent.md`](adr/0012-living-commit-mint-intent.md) |
| Living `/app` three copy tracks | [`adr/0013-living-app-locale-tracks.md`](adr/0013-living-app-locale-tracks.md) |

Living Mongo collections (`living_domains`, `living_org_configs`,
`living_active_config`) and entity/fact key `.`→`_` encoding at the Mongo
boundary follow the same progress-key rule as Stream Process storage (see
[`AGENTS.md`](../AGENTS.md)).

## Operational shape

Local infrastructure is the shared Docker Compose stack under `deployment/`.
The Go server, its Vite bundle, and configuration are documented for humans in
the [README](../README.md), [Quickstart](../QUICKSTART.md), and
[Docker notes](../DOCKER.md). Those documents defer to scripts and
configuration when they differ.
