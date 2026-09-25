# Architecture

Attesta is a Go HTTP application that renders server templates, serves a Vite
bundle, persists process data in MongoDB/GridFS, uses Appwrite for identity and
organization state, and asks Cerbos whether an actor may complete a substep.

This is navigation, not a replacement for code, configuration, or active-work
tracking. Read [`open-work.md`](open-work.md) for deferred changes.

## Boundaries and source locations

| Concern | Source of truth |
|---|---|
| HTTP routes and request handlers | `server/cmd/server/main.go`: `newMux`, `handleMyRoutes`, `handleStreamRoutes`, `handleProcessRoutes`, and `handleOrganizationRoutes` |
| Runtime configuration and workflow catalog | `server/cmd/server/main.go`: `runtimeConfig`; `server/config/workflow.yaml`; optional workflow catalog directory configured by environment |
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

## Operational shape

Local infrastructure is the shared Docker Compose stack under `deployment/`.
The Go server, its Vite bundle, and configuration are documented for humans in
the [README](../README.md), [Quickstart](../QUICKSTART.md), and
[Docker notes](../DOCKER.md). Those documents defer to scripts and
configuration when they differ.
