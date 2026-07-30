# Platform admin: categories CRUD editor

Date: 2026-07-30  
Scope: Replace browse-only `/admin/categories` tree with an HTMX-driven CRUD editor for taxonomy groups (`Category`) and leaves (`SubCategory`); new store mutation APIs; platform-admin templates/CSS; small Vite client hooks only as needed for icon grid + HTMX re-init after swaps  
Out of scope: Drag-and-drop reorder, editable slugs after create, `Description` on groups, drawer/modal chrome, dark Figma-only palette, public-home sidebar/catalog UX, expanding the taxonomy icon allowlist, replacing `seed-categories` full-replace

Figma references (structure/actions; implement as full panel inside existing admin console, not a drawer):

- [List / accordion editor](https://www.figma.com/design/UYbx6DfxoFp9CCjqaD3qEr/CLOSER---Prototypes---Forkbomb?node-id=617-39977)
- [Create/edit form](https://www.figma.com/design/UYbx6DfxoFp9CCjqaD3qEr/CLOSER---Prototypes---Forkbomb?node-id=668-34451)

## Goal

Give platform admins a same-page editor to create, edit, delete, and reorder discovery taxonomy nodes used by stream categorization and the public home filter. Keep English copy. Reuse existing platform-admin chrome; replace only the categories panel body.

## Decisions

1. **Approach:** Server-rendered editor + focused store mutations (not a JSON SPA API, not full `ReplaceTaxonomy` snapshots from the UI).
2. **Placement:** Replace the current browse tree at `/admin/categories`. No drawer header / close (X); normal admin nav is enough.
3. **CRUD scope:** Full create/edit/delete/reorder for both `Category` (group) and `SubCategory` (leaf).
4. **Accordion:** Always expanded. Omit expand/collapse controls (no fake chevron affordance).
5. **Forms:** Inline on the same page via HTMX panel swap. Edit replaces the summary row/card in place. Create form sits at the **insertion point** (end of groups list / end of that group’s leaves); on Add, scroll the form into view. Saved items **append last** (`sortOrder = max(siblings)+1`).
6. **Group description:** Groups have no `Description`. Hide that field on group forms. Leaves keep description.
7. **Slugs:** Derived with existing `canonifySlug(name)` on create only; never shown or edited afterward. Updates change name/icon/(description) only.
8. **Add subcategory:** Control on each group row (always visible because groups stay expanded). Top “+” creates a new group.
9. **Icon picker:** Click icon control → inline grid of allowlisted keys (`/static/taxonomy/*.svg`). Invalid keys rejected server-side (`ErrInvalidTaxonomyIcon`).
10. **Delete:** Confirm with existing `<dialog class="dialog">` when allowed. When not allowed, button is `disabled` with a tooltip explaining why. Eligibility computed on the server when building the view (cheap): group blocked if it has leaves; leaf blocked if any live catalog stream references the path (same rule as `DeleteSubCategory`). Progressive “fetch can-delete” is a fallback only if that ever becomes expensive — not required for v1.
11. **Reorder:** ↑/↓ arrows only (no SortableJS / drag-and-drop). Serialize in-flight reorders with HTMX (`hx-sync` and/or disable arrow controls while a reorder request is outstanding). No optimistic DOM reorder / debounced persist in v1.
12. **HTTP shape:** Two POST targets + small `intent` switches (Attesta house style, split by resource):
    - `POST /admin/categories` — group intents: `create` | `update` | `delete` | `reorder`
    - `POST /admin/categories/{slug}/subcategories` — leaf intents: `create` | `update` | `delete` | `reorder`
13. **GET modes:** Query flags open create/edit in the panel (e.g. `?new=group`, `?new=sub&parent=<slug>`, `?edit=group&slug=…`, `?edit=sub&parent=…&slug=…`). Cancel is an HTMX GET back to clean `/admin/categories`.
14. **Rendering:** HTMX swaps a stable categories panel wrapper (e.g. `#platform-admin-categories`, `hx-swap="outerHTML"`). Non-HTMX clients may still receive the full `platform_admin` page.
15. **Auth:** `requirePlatformAdmin` on all routes. No new Cerbos resource.
16. **Copy:** English throughout (Figma Italian labels are not used).
17. **Figma wording in meta pill:** Keep “`{N} groups · {M} categories`” where groups = `Category` and categories = `SubCategory` (matches Figma; domain types stay Category/SubCategory in code).
18. **Implementation workspace:** Feature work lands in a git worktree (per repo Taskfile / `.worktrees` workflow).

## Layout

```
[ Platform admin console — Categories tab ]
┌──────────────────────────────────────────────┐
│ Categories                                   │
│ Manage stream discovery taxonomy             │
│ [+]  ·  N groups · M categories              │
│                                              │
│ ┌ Group row (icon, name, ↑↓, add sub,        │
│ │            edit, delete)                   │
│ │  ┌ Leaf card … ↑↓ edit delete              │
│ │  └ …                                       │
│ │  [ create-sub form when ?new=sub … ]       │
│ └────────────────────────────────────────────│
│ [ create-group form when ?new=group ]        │
└──────────────────────────────────────────────┘
```

## Data model & store

Unchanged document shape:

- `Category`: name, icon, slug, sortOrder (no description)
- `SubCategory`: name, icon, slug, sortOrder, description, categorySlug

New store methods (MemoryStore + MongoStore); bump `TaxonomyRevision` on every successful write:

- `CreateCategory` / `UpdateCategory`
- `CreateSubCategory` / `UpdateSubCategory`
- `ReorderCategory` / `ReorderSubCategory` (swap with adjacent sibling by direction, or equivalent neighbor swap)
- Existing `DeleteCategory` / `DeleteSubCategory` (and server `DeleteSubCategory` catalog-reference guard)

Create validation: non-empty name; allowlisted icon; unique group slug globally; unique leaf slug within parent.

Seed path unchanged: `ReplaceTaxonomy`, `seed-categories`, empty-store bootstrap from `categories.yaml`.

## UI components / CSS

- Replace `platform_admin_categories_panel` markup with editor list + inline form partials.
- Follow `docs/css.md` and `.agents/skills/attesta-ui-components` for any extracted component CSS.
- Map Figma structure to existing admin tokens (not a one-off dark drawer theme).
- Reuse taxonomy `IconURL` helpers from `taxonomy_admin.go`.
- Icon grid and HTMX-after-swap init live in `web/src/main.js` if needed (minimal).

## Error handling

- Duplicate slug / invalid icon / empty name → re-render panel with form open + error message.
- Blocked delete on POST (race) → error message; item remains.
- Reorder at list ends: disable ↑ on first / ↓ on last sibling.

## Testing

- Store unit tests for create/update/reorder/delete, duplicate slug, append sortOrder, category-delete-with-children refusal.
- Handler tests: non-admin rejected; each intent on both POST targets; HTMX responses return the categories panel partial.
- View eligibility: leaf referenced by catalog → delete disabled; empty group → delete enabled.
- Template smoke: group form omits description; leaf form includes it; meta pill counts; reorder controls honor `hx-sync`/disabled-while-pending attributes as specified.

## Migration / rollout

No data migration. Existing Mongo taxonomy rows remain valid. Editor writes go through new APIs; ops may still full-replace via `seed-categories` when needed.
