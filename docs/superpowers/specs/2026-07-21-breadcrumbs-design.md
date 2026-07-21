# Breadcrumbs component (replace page-header Back)

Date: 2026-07-21  
Scope: Full `breadcrumbs` UI component (Go view, HTML partial, CSS), wired into stream, process, org-admin, and platform-admin page headers; remove `page_header_back`  
Out of scope: DPP, home, auth/invite/reset pages, Formata builder, JS truncation menus, JSON-LD structured data, changing page `h1` copy

## Goal

Replace the single “Back” link in page headers with a proper breadcrumb trail that shows hierarchy (including the current page as a non-link last crumb) and is reusable for future authenticated pages.

## Decisions

1. **Full component** (not a page-header micro-partial): `BreadcrumbsView` / `BreadcrumbItem` in `components.go`, `templates/components/breadcrumbs.html`, `web/src/styles/components/breadcrumbs.css`.
2. **Trail includes current page** as the last crumb (`Href` empty → non-link with `aria-current="page"`). Page `h1` titles stay as they are today (accepted duplication).
3. **Root label** is `Streams`, linking to `/`.
4. **v1 call sites:** stream dashboard, process instance, org admin (with section crumb), platform admin. Home / DPP / auth stay without breadcrumbs.
5. **Remove** `page_header_back` (and `page_header.html` if it only defines that partial), `.page-header-back` styles, related tests/docs, and the now-unused `icon-back` icon define.

## Component shape

```go
type BreadcrumbItem struct {
	Label string
	Href  string // empty => current page (non-link)
}

type BreadcrumbsView struct {
	Items []BreadcrumbItem
}
```

**Template:** `{{ define "breadcrumbs" }}` — call site `{{ template "breadcrumbs" .Breadcrumbs }}`.

**Markup:**
- `<nav class="breadcrumbs" aria-label="Breadcrumb">`
- Ordered list of items
- Ancestors: `<a href="…">Label</a>`
- Current (empty `Href`): `<span aria-current="page">Label</span>`
- Convention: only the **last** item has an empty `Href`; the template treats any empty `Href` as non-link + `aria-current`
- Separators via CSS only (e.g. `/` between items), not separate list entries
- If `Items` is empty or nil: render nothing

**Page-header slot:** first child of `section.page-header` (same position as today’s Back), above `page-header-body` / `page-header-head`.

**Assembly:** handlers / view builders set `Breadcrumbs` on the page view. Optional small helpers only to avoid duplicated trail construction — no fluent `With*` API on the view structs.

## Per-page trails

| Page | Trail |
|------|--------|
| Stream `/w/:key/` | Streams → *stream name* (current) |
| Process `/w/:key/process/:id` | Streams → *stream name* → `Instance: {name\|id}` (current) |
| Org admin | Streams → Organization admin → Profile \| Roles \| Members (current) |
| Platform admin `/admin/orgs` | Streams → Platform admin (current) |

### Label and href rules

- **Streams** → `/`
- **Stream name:** workflow display name; if empty, fall back to workflow key. On process pages, this crumb links to `/w/{key}/`.
- **Process current crumb:** literal prefix `Instance: ` plus trimmed instance name when present, otherwise process ID (e.g. `Instance: abc123`).
- **Organization admin** (middle) → `/org-admin/profile` (default landing). Section labels map from `ActivePanel`: `profile` → Profile, `roles` → Roles, `members` → Members (current, non-link).
- **Platform admin:** single console page; two crumbs only (no extra section).

## Visual / CSS

- Compact trail above the title: `--text-sm`, muted ancestors (`--muted-foreground`), current crumb at normal body emphasis
- No back-arrow icon
- Single horizontal row; wrap only if needed (`overflow-wrap`); no JS ellipsis menu in v1
- Reuse existing tokens (`--space-*`, `--font-semibold`, link/focus from reset)
- New module `components/breadcrumbs.css`, imported from `components.css`
- Remove `.page-header-back` from `page-header.css`; update markup-tree comments to `nav.breadcrumbs?`

## Docs to update

- `docs/css.md` — component index, page-header notes (drop `page_header_back`)
- `.agents/skills/attesta-ui-components/SKILL.md` — full-component row for breadcrumbs; remove back micro-partial exception
- `AGENTS.md` — only if it still cites `page_header_back` in the page-header bullet

## Testing

- Replace `TestPageHeaderBackRendersHrefAndLabel` with breadcrumbs template tests covering: linked ancestors, current `aria-current="page"`, empty `Items` renders nothing
- Update `templates_test.go` expected defines: remove `page_header_back`, add `breadcrumbs`
- Extend or add page template smoke coverage so stream / process / org-admin / platform-admin HTML includes the expected trail
- Run `task css:lint` after CSS changes

## Non-goals (explicit)

- Changing `h1` text or removing subtitle duplication
- Breadcrumbs on DPP, home, or auth flows in this change
- Mobile collapse / overflow “…” menus
- Schema.org / JSON-LD breadcrumb markup
