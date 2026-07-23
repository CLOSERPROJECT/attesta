# Authenticated app under `/my` (public `/`)

Date: 2026-07-23  
Scope: Full rewrite of authenticated app URL prefixes; public blank homepage at `/`; hard cut of legacy `/w/` and `/org-admin/` mounts  
Out of scope: Public homepage content/design; improving `/my` stream-picker UX; compatibility redirects; moving platform `/admin` under `/my`; renaming Cerbos/Appwrite/Mongo domain models

## Goal

Make `/` a public homepage (blank placeholder for now) and move the authenticated product surface under `/my/*`, with clearer stream/instance path segments. Platform admin stays at root `/admin`. Org settings move to `/my/organization`.

## Decisions

1. **Approach:** Remount handlers and rewrite URL helpers end-to-end (no path-rewrite shim, no dual-mount period).
2. **Auth prefix:** `/my` (not `/dashboard`).
3. **Stream picker:** `GET /my` = today’s authenticated `/` content (to be improved later).
4. **Stream dashboard:** `GET /my/streams/:streamId` (today’s `/w/:key/`).
5. **Instance:** `GET /my/streams/:streamId/instance/:instanceId` (today’s `/w/:key/process/:id`).
6. **Org admin:** `/my/organization/...` (today’s `/org-admin/...`).
7. **Platform admin:** stays at `/admin/orgs` (and related logo routes).
8. **Hard cut:** `/w/*`, `/org-admin/*`, and `/dashboard/*` return 404. No redirects.
9. **Logged-in visit to `/`:** stay on the public blank homepage (no auto-redirect to `/my`).
10. **Post-login / app-home default:** `/my`.

## Route map

| Role | Path | Auth | Notes |
|------|------|------|--------|
| Public homepage | `GET /` | no | Blank placeholder; layout chrome OK |
| Stream picker | `GET /my` | yes | Today’s `/` stream list |
| Stream dashboard | `GET /my/streams/:streamId` | yes | Instance list for one stream |
| Start instance | `POST /my/streams/:streamId/instance/start` | yes | Was `/process/start` |
| Delete stream | `POST /my/streams/:streamId/delete` | yes | Same verb |
| Instance detail | `GET /my/streams/:streamId/instance/:instanceId` | yes | Was `/process/:id` |
| Instance subpaths | `/my/streams/:streamId/instance/:instanceId/...` | yes | `content`, `downloads`, `terminate`, `substep/...`, exports, attachment file |
| Stream SSE | `GET /my/streams/:streamId/events` | yes | Was `/w/:key/events` |
| Org profile | `GET /my/organization/profile` | yes | Was `/org-admin/profile` |
| Org roles | `GET/POST /my/organization/roles` | yes | Was `/org-admin/roles` |
| Org members | `GET /my/organization/members` | yes | Was `/org-admin/members` |
| Org users (forms) | `GET/POST /my/organization/users` | yes | GET redirects to profile; POST keeps invite/set_roles/delete intents |
| Formata builder | `/my/organization/formata-builder` | yes | Was `/org-admin/formata-builder` |
| Org logo | `/my/organization/logo/...` | yes | Was `/org-admin/logo/...` |
| Platform admin | `/admin/orgs` (+ logo) | yes | Unchanged at root |
| Auth / public | `/login`, `/signup`, `/logout`, `/invite/`, `/reset`, `/01/…`, `/docs`, `/about`, `/api/catalog` | as today | Stay at root |
| Legacy | `/w/*`, `/org-admin/*`, `/dashboard/*` | — | 404 |

**IDs:** `:streamId` = today’s workflow key; `:instanceId` = process ObjectID hex. Persistence and domain types unchanged — path segments only.

**Trailing slashes:** Preserve today’s behavior for stream picker and stream dashboard (accept `/my` and `/my/`; accept `/my/streams/:id` and `/my/streams/:id/`). Prefer generating links without a trailing slash except where existing helpers already normalize.

**Verb rename:** public URL uses `instance` instead of `process` (e.g. start was `/w/:key/process/start`). Internal Go types (`Process`, handlers named `handle*Process*`) may keep existing names unless a local rename clarifies call sites; no required domain rename in this change.

## Architecture

### Routing (`newMux` + handlers)

- `GET /` → new public blank homepage handler (no `requireAuthenticatedPage`).
- `/my/` → auth-gated surface:
  - `/my` and `/my/` → stream picker (today’s `handleHome`)
  - `/my/streams/:id/...` → stream/instance router (today’s `handleWorkflowRoutes` / `handleProcessRoutes`)
  - `/my/organization/...` → org settings (today’s `handleOrgAdmin*`)
- Unregister `/w/` and `/org-admin/*`. Keep `/admin/*` at root.
- Every `/my/*` handler continues to use `requireAuthenticatedPage` / `requireAuthenticatedPost` as today.

### Central URL builders

Rewrite helpers so templates/handlers do not hardcode old prefixes:

- Stream base path: `/my/streams/{key}` (replace or adapt `workflowPath`)
- Instance detail: `/my/streams/{key}/instance/{id}`
- Org links: `/my/organization/profile` (and siblings)
- Login default / `safeNextPath` fallback: `/my` (today often `/`)
- Pagination, breadcrumbs, post-start/delete/complete redirects → new paths

Prefer one helper per resource so cards, breadcrumbs, HTMX URLs, and redirects stay consistent.

### Frontend

Update hardcoded `/w/...` fetch and EventSource URLs in `web/src/main.js` to `/my/streams/...`.

### Templates / chrome

- Topbar “My Org” → `/my/organization/profile`
- Breadcrumbs root “Streams” → `/my` (not public `/`)
- Stream/instance crumbs use new stream and instance paths
- Org admin page forms, nav, and inline JS path maps → `/my/organization/...`

### Docs

Update `AGENTS.md` (and README/QUICKSTART route mentions if present) so agents do not regenerate `/w/` or `/org-admin/` URLs.

## Auth and errors

- Unauthenticated access to `/my/*` → `/login?next=<request URI>` (existing pattern).
- Login/signup default `next` fallback → `/my`.
- Signup that today lands on `/org-admin/profile` → `/my/organization/profile`.
- Platform-admin login `next` may still target `/admin/orgs`.
- Cerbos resource names and role slugs (e.g. `org-admin`, `org_admin_console`) are unchanged — URL rename only.
- Missing stream/instance → 404 as today.
- Legacy prefixes → 404 (no redirect body).

## Testing

- Update handler/route/template tests that assert `/w/…`, `/org-admin/…`, or login `next=/` to the new paths.
- Keep/extend coverage that legacy `/dashboard` is 404; assert `/w/` and `/org-admin/` 404 after remount.
- Smoke cases:
  - Unauthenticated `/my` → login with `next`
  - Authenticated `GET /` → 200 blank homepage, no redirect to `/my`
  - Instance URLs build `/my/streams/{id}/instance/{id}`
  - Org topbar → `/my/organization/profile`
- Frontend EventSource/content fetch paths match new stream/instance URLs.
- No required changes to Cerbos policy files, Mongo progress-key encoding, or Appwrite identity model for this work.

## Non-goals (explicit)

- Designing or filling the public homepage
- Redesigning the `/my` stream picker
- Compatibility redirects from old URLs
- Nesting platform admin under `/my`
- Renaming internal process/workflow domain types solely for vocabulary alignment
