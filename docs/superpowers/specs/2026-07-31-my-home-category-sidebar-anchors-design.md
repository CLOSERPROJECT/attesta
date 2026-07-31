# `/my` category sidebar (anchors) + shared `category_sidebar`

Date: 2026-07-31  
Branch / worktree: `feat/my-home-catalog`  
Related: `docs/superpowers/specs/2026-07-31-my-home-catalog-design.md`, `docs/superpowers/specs/2026-07-30-public-home-category-sidebar-design.md`

## Goal

Add a category sidebar to authenticated `/my` that matches the public-home category tree visually, but uses **in-page anchors** (not HTMX filters) to jump to catalog sections. Extract the sidebar into a shared **thin full** component used by both `/` and `/my`. On narrow viewports, wrap the `/my` sidebar in a CSS-only **left slide-in** `nav-drawer`.

## Decisions

1. **Leaf behavior on `/my`:** Subcategory leaves are anchors only (`href="#…"`). Category rows remain `<details>`/`<summary>` expand/collapse — not links. Public `/` keeps HTMX buttons (decision 7).
2. **Sidebar contents on `/my`:** Only categories/subcategories with ≥1 accessible stream for the current user. Include an Uncategorized leaf when that group exists.
3. **Layout on `/my`:** Two columns inside existing `u-max-w-7xl` / `.my-home`. Same page background — no public-home full-bleed streams split, no contrasting sidebar/results band colors.
4. **Responsive:** Desktop (`md`+): sidebar always docked in the left column. Mobile: hamburger opens a left `nav-drawer`; drawer chrome hidden on desktop.
5. **Active state:** Plain anchors only. No scroll-spy. Optional cheap `:target` styling is allowed; not required.
6. **Component split:**
   - **`category_sidebar`** — thin **full** component (template + CSS + view structs).
   - **`nav-drawer`** — **CSS-only** shell (markup contract in CSS header; inline in `pages/home.html`).
7. **Public `/`:** Keep HTMX filter semantics. Migrate inline sidebar markup to the shared template; leaf mode = PartialURL/PushURL buttons.

## Out of scope

- Scroll-spy / continuous active-leaf tracking
- Changing public filter URLs, partial endpoint, or selection resolution
- Extracting a second copy of the catalog results headers
- New app JS modules for drawer open/close (Popover API / declarative HTML only)
- Platform admin taxonomy CRUD changes

## Layout (`/my`)

```
[ page-header ]
  title + subtitle
  [Categories] hamburger     ← mobile only (nav-drawer trigger)
  [Create a stream]          ← existing, permission-gated

[ my-home body grid ]        ← inside u-max-w-7xl; same bg
  [ category_sidebar ]       ← desktop: docked column
  [ catalog stack ]          ← existing groups + cards
     section#cat-… per leaf
       subcategory header
       card grid

[ empty state ]              ← zero groups: no sidebar, no drawer trigger
```

Mobile: catalog full width. **One** `category_sidebar` instance in the DOM: on `md`+ it is the docked left column; below `md` that same node is the `nav-drawer` panel (responsive CSS / reparenting-not-required). Do not render two copies of the tree.

## Anchors & data

**Section ids:** Stable ids on each catalog leaf block:

- Taxonomy: `cat-{categorySlug}--{subCategorySlug}`
- Uncategorized: `cat-uncategorized`

Wrap subcategory header + its stream grid in a `<section id="…" class="…">` (or equivalent) with `scroll-margin-top` sufficient for the sticky topbar (`pages/home.css`).

**Group fields:** Extend `MyHomeStreamGroupView` with slugs needed for ids (`CategorySlug`, `SubCategorySlug`) and/or a precomputed `AnchorID`. Category display name/icon header rules from the prior catalog polish stay (category header only on first non-empty leaf under a category).

**Sidebar builder:** Derive the sidebar category list from the same accessible groups already built for `/my` (no second access pass). Categories with ≥1 leaf default `Expanded: true`. Uncategorized is a synthetic category named `Uncategorized` with a **single** leaf also labeled `Uncategorized` whose `Href` is `#cat-uncategorized` (keeps the same details→leaf shape as taxonomy paths).

**Empty catalog:** No sidebar and no hamburger (empty-state only).

## Components

### `category_sidebar` (full, thin)

| Piece | Path |
|-------|------|
| Template | `server/templates/components/category_sidebar.html` (`{{ define "category_sidebar" }}`) |
| CSS | `web/src/styles/components/category-sidebar.css` (`.category-sidebar-*`) |
| Views | Rename `PublicHomeCategoryView` / `PublicHomeSubCategoryView` → `CategorySidebarCategoryView` / `CategorySidebarLeafView`. Template root is a small wrapper (e.g. `CategorySidebarView{ Title, Categories []… }`) so the “Stream Categories” header stays inside the component. |

**Leaf rendering rule (thin mode dispatch):**

- If `Href` is non-empty → render `<a class="category-sidebar-subcategory" href="{{ .Href }}">`
- Else → render HTMX `<button>` with `PartialURL` / `PushURL` / `Active` as today’s public home does

No separate mode enum. Public builder fills PartialURL/PushURL and leaves `Href` empty. `/my` builder fills `Href` and leaves HTMX fields empty.

**CSS migration:** Move `.public-home-category-sidebar*` / `.public-home-category*` / `.public-home-subcategor*` tree rules from `pages/public-home.css` into `components/category-sidebar.css` under `.category-sidebar-*`. Update public home + any tests that assert old class strings. Public streams split layout rules that only position the aside stay in `public-home.css` (or thin wrappers).

**Public `/`:** Replace inline aside markup in `public_home.html` with `{{ template "category_sidebar" … }}`. Behavior unchanged.

### `nav-drawer` (CSS-only)

| Piece | Path |
|-------|------|
| CSS | `web/src/styles/components/nav-drawer.css` |
| Markup | Inline in `pages/home.html` (markup-tree header in the CSS file) |

**Mechanism:** Prefer HTML **Popover API** (`popover` + `popovertarget`) for open/close and backdrop dismiss without app JS. Hash navigation to catalog sections must remain independent of drawer open state (do not use `:target` to open the drawer).

**Ownership:** `nav-drawer` owns trigger, panel, slide-from-left, backdrop. It does not restyle the category tree. Desktop: hide trigger + treat panel as in-flow column (page CSS in `home.css` may cooperate).

## `/my` page CSS

`pages/home.css`:

- Grid for docked sidebar + catalog inside `.my-home`
- `scroll-margin-top` on catalog section anchors
- Any desktop/mobile visibility glue between `nav-drawer` and the grid that is page-specific

## Testing

- **Template:** `category_sidebar` with HTMX leaf attrs present when PartialURL set; anchor `<a href="#…">` when Href set; no HTMX attrs in anchor mode.
- **`/my` builder:** Sidebar categories/leaves match non-empty groups only; Uncategorized leaf when present; `Href` matches section `id`.
- **`/my` page markup:** When groups exist, hamburger/drawer trigger present (mobile contract); empty state omits sidebar and trigger.
- **Public home:** Existing filter/partial/push-url tests updated for renamed types/classes; behavior assertions unchanged.
- **CSS:** `task css:lint` after new component imports.

## Non-goals / unchanged

- Public home HTMX `/streams/public` contract
- Access matrix for which streams appear on `/my` (already specified)
- Card management shell / Create CTA placement from my-home catalog spec
