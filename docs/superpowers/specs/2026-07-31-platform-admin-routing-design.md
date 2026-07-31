# Platform admin routing rewire

Date: 2026-07-31  
Scope: `/admin` entry redirect, rename `/admin/orgs` → `/admin/organizations` (hard cut), platform-admin breadcrumbs, topbar account-menu label/href, `adminPath` helpers, hardcoded path sweep (templates/handlers/tests/docs)  
Out of scope: New admin landing page, last-visited section memory, unified `/admin` mux refactor (org-admin-style router), Categories URL changes, visual redesign beyond labels/crumbs/links

## Goal

Make platform admin navigation coherent now that the console has multiple sections (Organizations and Categories): `/admin` must work as an entry point, section URLs and labels must say “organizations”, breadcrumbs must match the org-admin pattern, and the topbar must point at the console root instead of a single section.

## Decisions

1. **Approach:** Minimal surface fix — keep flat mux registration; add root redirect + rename + breadcrumb/topbar fixes + path helpers.
2. **`/admin` and `/admin/`:** Auth-gated redirect to `/admin/organizations` (default section). Same platform-admin gate as today’s console.
3. **Section URLs:** Keep separate section paths. Rename orgs: `/admin/organizations` (and `/admin/organizations/logo/:id`). Categories stay at `/admin/categories` (+ nested subcategory POSTs).
4. **Hard cut:** `/admin/orgs` and `/admin/orgs/…` return **404**. No compatibility redirect.
5. **Path helpers:** Add `adminPath(rest string)` in `paths.go` (mirror `organizationPath`). Use it for console nav, redirects, HTMX URLs, logo paths, and list/pagination helpers instead of hardcoded `/admin/…` strings.
6. **Topbar:** Platform-admin account-menu item label **Admin**, href `/admin`.
7. **Breadcrumbs:** Always three crumbs, mirroring org admin:
   - `Dashboard` → `/my`
   - `Platform admin` → `/admin` (not current; not tied to a section URL)
   - Current section → `Organizations` or `Categories` with that section’s URL, `Current: true`
8. **ActivePanel keys:** `organizations` | `categories` (replace today’s `orgs` default).
9. **Page chrome:** Title remains “Platform admin dashboard”; subtitle continues to follow the active section.

## Routes

| Path | Behavior |
|------|----------|
| `GET /admin`, `GET /admin/` | Platform-admin gate → redirect to `/admin/organizations` |
| `/admin/organizations` (+ trailing slash) | Organizations console (today’s orgs handlers under the new path) |
| `/admin/organizations/logo/:id` | Organization logo asset |
| `/admin/categories` (+ nested paths) | Unchanged |
| `/admin/orgs`, `/admin/orgs/…` | **404** |

## Navigation

### Topbar / account menu

| Before | After |
|--------|-------|
| Label **Orgs**, href `/admin/orgs` | Label **Admin**, href `/admin` |

### Sidebar soft-nav

Unchanged structure:

| Title | Href |
|-------|------|
| Organizations | `/admin/organizations` |
| Categories | `/admin/categories` |

### Breadcrumbs

| Section | Trail |
|---------|--------|
| Organizations | `Dashboard` → `Platform admin` → `Organizations` (current) |
| Categories | `Dashboard` → `Platform admin` → `Categories` (current) |

Middle crumb always links to `/admin` (which redirects to the default section).

## Auth & errors

- `/admin` uses the same Cerbos / `requirePlatformAdmin` (or equivalent) gate as existing `/admin/organizations` and `/admin/categories`.
- Unauthenticated access follows today’s login redirect behavior for the console.
- Legacy `/admin/orgs*` → plain 404 (no soft redirect).

## Implementation touchpoints

- `server/cmd/server/main.go` — mux registration; rename orgs path prefixes; `/admin` redirect handler.
- `server/cmd/server/paths.go` — `adminPath`.
- `server/cmd/server/breadcrumbs.go` (+ tests) — always 3 crumbs; section labels/hrefs via helpers.
- `server/cmd/server/admin_console.go` — nav hrefs; ActivePanel `organizations`.
- Templates: `platform_admin.html`, `layout.html` (topbar), any hardcoded `/admin/orgs`.
- Helpers such as `platformAdminPath` (list/search pagination) → organizations path.
- Docs: `AGENTS.md` and any specs/README cites of `/admin/orgs`.

## Testing

- `/admin` and `/admin/` redirect to `/admin/organizations` for an authorized platform admin.
- `/admin/orgs` and `/admin/orgs/logo/…` return 404.
- Organizations console and logo still work under `/admin/organizations`.
- Categories routes unchanged.
- `buildPlatformAdminBreadcrumbs` covers both sections with the three-crumb shape.
- Template/HTMX/path tests updated for the new URLs and topbar label where covered.

## Out of scope (explicit)

- No dedicated `/admin` landing page body.
- No cookie/session “last section” memory.
- No `handleAdminRoutes` unified dispatcher (can revisit when a third section lands).
- No Categories URL rename.
- No visual redesign of the admin console beyond labels, crumbs, and links.
