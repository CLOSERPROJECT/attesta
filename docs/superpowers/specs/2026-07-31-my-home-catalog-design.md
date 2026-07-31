# `/my` home: accessible streams catalog

Date: 2026-07-31  
Scope: Authenticated `/my` stream home (`handleHome`, `home_picker_body` / `pages/home.html`), public stream card + optional management shell, CSS-only empty-state extraction, access filtering + taxonomy grouping builders/tests.  
Out of scope: Public `/` sidebar/filter behavior, public stream page content redesign, platform admin taxonomy CRUD, changing Cerbos policy semantics for edit/delete (reuse existing checks), SSE/HTMX for the `/my` listing.

Related: `docs/superpowers/specs/2026-07-30-public-home-category-sidebar-design.md` (public home filters; `/my` does **not** reuse the sidebar).

## Goal

Replace the authenticated `/my` “Choose a stream” grid (`stream_card` + create card) with a catalog that:

- Shows **only streams the user may access** (with admin exceptions below)
- Groups them under the same **category / subcategory result headers** used on the public home results panel
- Reuses **`public_stream_card`** visuals, with optional clone/edit/delete actions
- Puts **Create a stream** in the page header actions (not a grid card)
- Uses a shared **empty-state** component when the user has zero accessible streams

## Decisions

1. **Approach:** Stack reused results groups (no sidebar, no HTMX filter on `/my`).
2. **Card visual:** `public_stream_card`; management menu when permitted (shell/wrapper so the menu does not navigate).
3. **Card click:** `/my/streams/:key/`.
4. **Uncategorized:** Final group with an “Uncategorized” header for accessible streams lacking taxonomy.
5. **Create:** `page-header-actions` → Formata builder when `canViewFormataBuilder`; not inside the empty block.
6. **Empty:** Extract public stream runs empty pattern into a CSS-only component; `/my` and public stream runs share it.
7. **Card cap:** Public home’s `publicHomeStreamCardLimit` does **not** apply on `/my`.

## Access rules

Include a catalog stream on `/my` when **any** of:

| Actor | Streams shown |
|-------|----------------|
| Platform admin (`user.IsPlatformAdmin`) | **All** runtime catalog streams |
| Org admin (`userIsOrgAdmin`) for their org | All streams whose `organizations` include the user’s `OrgSlug` (workflow role not required) |
| Everyone else | User’s `OrgSlug` is a participating org **and** at least one of their `RoleSlugs` is listed on that stream’s `roles` for that org |

Catalog source: same runtime workflow catalog as today’s picker.  
Store/taxonomy/catalog load failures → HTTP 500. Zero accessible streams → HTTP 200 + empty state.

## Layout

```
[ page-header ]
  title + short subtitle
  [Create a stream]     ← page-header-actions (permission-gated)
  confirmation / error  ← existing query flash

[ catalog stack ]       ← no sidebar, no landing hero
  for each taxonomy path with ≥1 accessible stream (taxonomy sortOrder):
    category header     (.public-home-results-category-header)
    subcategory header  (.public-home-results-subcategory-header)
    public_stream_card grid
  if any uncategorized accessible streams:
    “Uncategorized” header
    same card grid

[ empty state ]         ← only when zero accessible streams
  shared empty-state component (icon + title + hint)
```

Mobile/desktop: single column stack; reuse public-home results header/grid classes. Light wrapper styles in `pages/home.css` only if needed.

## Cards & actions

- Metrics, DPP chip, org avatars: same as homepage `public_stream_card`.
- `Href`: `/my/streams/:key/`.
- Optional ellipsis menu (clone / edit / delete) using the same permission sources as today’s `workflowOptions` / `stream_card` (Formata builder for clone; Cerbos for edit/delete; edit-requires-purge + delete dialogs preserved).
- Homepage cards stay menu-free when action flags are unset/false.
- Prefer a shell around the public card (menu + dialogs outside the primary link) rather than nesting interactive controls inside the card `<a>`.

## Templates & CSS

| Piece | Plan |
|-------|------|
| `/my` body | Rewrite `home_picker_body` in `pages/home.html` |
| Group chrome | Reuse/share category + subcategory headers + grid from `public_home_stream_results` (small shared group partial if needed; avoid a large `/` rewrite) |
| Card | Extend `public_stream_card` or add shell for actions/dialogs |
| Create card | Stop using `stream_card_create` on `/my` |
| Empty | CSS-only `empty-state` component; migrate `public-stream-runs-empty` markup/classes to it; `/my` empty uses home-specific title/hint |
| Old `stream_card` | Remove from `/my`; delete only if nothing else references it |

Follow `.agents/skills/attesta-ui-components` and `docs/css.md` for tiers and layers.

## Data flow

1. `handleHome` authenticates; loads catalog + taxonomy + user.
2. Filter accessible keys (rules above).
3. Build public-style card views (instance/step/role/org metrics) with `/my` hrefs and action flags.
4. Group by effective `categorySlug` / `subCategorySlug`; emit non-empty taxonomy groups in `sortOrder`; append Uncategorized.
5. Render full page (no `/my` listing HTMX/SSE endpoint).

## Testing

- Access matrix: platform admin (all); org admin (org streams, no role required); role member (intersection); outsider (empty).
- Grouping: taxonomy order; skip empty leaves; Uncategorized last; no 6-card cap.
- Markup: result headers; card hrefs under `/my/streams/`; Create in header when allowed; menu absent without perms; empty-state when zero.
- Empty-state extraction: public stream runs empty still renders via shared component contract.

## Non-goals / unchanged

- Public home sidebar, HTMX `/streams/public`, and public card links to `/streams/:key`.
- Breadcrumbs on `/my` (none today).
- Auth/session cookie behavior.
