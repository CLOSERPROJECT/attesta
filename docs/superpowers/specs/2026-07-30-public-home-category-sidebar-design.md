# Public home: category filter sidebar

Date: 2026-07-30  
Scope: `server/templates/pages/public_home.html`, new results partial + sidebar markup, `web/src/styles/pages/public-home.css` (and optional component CSS), `web/src/main.js` (replace `initLandingTabs`), `server/cmd/server` public-home handlers / view models, Lucide static SVGs for sidebar chrome  
Out of scope: Public stream card redesign, platform admin taxonomy CRUD, `/my` stream picker, a full “browse all streams” catalog page, mounting Svelte/`@lucide/svelte` on the marketing page

Figma reference (structure only; use light tokens): [Filter Sidebar](https://www.figma.com/design/UYbx6DfxoFp9CCjqaD3qEr/CLOSER---Prototypes---Forkbomb?node-id=580-28986)

## Goal

Replace the horizontal category tab strip on `/#streams` with a Figma-style accordion sidebar. Filter public stream cards by taxonomy subcategory via HTMX (swap results only). Show every category and subcategory from the live taxonomy store, even when a leaf has zero streams. Theme the sidebar with shared light tokens (no dark Figma palette, no new `--landing-*`).

## Decisions

1. **Approach:** Sidebar + HTMX results partial (not full-section swap, not client-only show/hide).
2. **Parent category click:** Expand/collapse only. Filtering runs only when a subcategory is selected.
3. **Accordion:** Multi-open (more than one category may be expanded), matching Figma.
4. **Default selection:** Always expand the **first category** and select its **first subcategory** (taxonomy `sortOrder`). Do not prefer leaves that have streams.
5. **Taxonomy source:** Live store via `loadTaxonomyTree` (seeded from `categories.yaml`). Render all nodes even if no catalog streams match.
6. **Mobile:** Sidebar stacks **above** the results grid (full width); two-column from tablet/desktop up.
7. **Empty leaf:** Short empty copy + primary CTA **Create a stream** → guests `/login?next=/my/organization/formata-builder`; signed-in users `/my/organization/formata-builder`.
8. **Remove:** Horizontal `.public-home-tabs*` UI, scroll arrows, `initLandingTabs`, and the section footer **See all streams** signup link.
9. **Icons:** Sidebar chrome = Lucide **`LayoutGrid`** (header) + **`ChevronRight` / `ChevronDown`** as **static SVGs** under `web/public/` (same glyphs as Formata’s `@lucide/svelte`; not Figma MCP assets; not a Svelte runtime on `/`). Category row icons = existing taxonomy `IconURL` (`/static/taxonomy/…`).
10. **URL sync:** Subcategory selection updates the browser URL to `/?category=<slug>&subCategory=<slug>` via `hx-push-url` so refresh/share restores selection. Invalid/missing params on full page fall back to first/first. Hash `#streams` is optional (nice-to-have; not required for correctness).

## Layout

```
[ Section head — “Stream Categories” / “Explore Public Streams” ]
┌─────────────────────┬────────────────────────────────┐
│ Filter sidebar      │ Results (#public-home-stream-  │
│ • header + Lucide   │   results)                     │
│ • category accordion│ • public_stream_card grid OR   │
│ • subcategory list  │   empty + Create a stream CTA  │
└─────────────────────┴────────────────────────────────┘
```

- Desktop: sidebar ~280–320px left; results fill remaining width.
- Mobile: sidebar full-width above results.
- Section head stays centered above the two-region body.

### Sidebar structure (light tokens)

| Piece | Behavior |
|-------|----------|
| Header | Lucide `LayoutGrid` icon + “Stream Categories” |
| Category row | Taxonomy icon + name + chevron; click toggles expand |
| Expanded category | Stronger surface/border (Figma “open” treatment → `--card` / `--muted` / `--border`) |
| Subcategory | Indent; click triggers HTMX filter |
| Active subcategory | Left accent bar + `--secondary` (or equivalent) surface; `--primary` accent — not dark-theme greens |

No forced `color-scheme: dark` on the sidebar. Follow existing public-home / `data-theme` token usage.

## Data flow

### Full page (`GET /`)

1. Load taxonomy tree from store. If empty, render sidebar empty state (no hard-coded fallback categories) and an empty results panel with Create CTA.
2. Resolve selection: query `category` + `subCategory` if both resolve to a known path; else first category / first subcategory.
3. Build filtered public stream cards for that path (see Filtering).
4. SSR sidebar (expanded/active flags) + results partial content inside the swap target.

### Partial (`GET /streams/public`)

Public HTMX results endpoint (register in `newMux`; must not collide with authenticated `/my/streams/…`).

- Query: `category` and `subCategory` both required.
- Unknown or missing slugs → **404** with a short error/empty fragment suitable for the swap target (full page owns defaulting; partial does not).
- Response: **only** the results fragment (card grid or empty CTA), not layout/sidebar.
- Wire with `hx-get` + `hx-target` on `#public-home-stream-results` + `hx-swap="innerHTML"` + `hx-push-url="/?category=…&subCategory=…"`.

### Filtering

- Catalog workflows expose `categorySlug` / `subCategorySlug` after effective categorization.
- Extend `PublicStreamCardView` (or filter before mapping) so cards can be selected by that pair.
- Include a stream only when both slugs match the selected path.
- Keep a per-response card cap (today: `publicHomeStreamCardLimit = 6`) **after** filtering.
- Uncategorized streams never appear in a subcategory filter result.

### Client behavior

- Subcategory anchors/buttons: HTMX attrs as above; update `is-active` / `aria-selected` on the sidebar without re-fetching the sidebar (small JS or `hx-on`).
- Category rows: local expand/collapse only (no HTMX). Prefer `details`/`summary` or a tiny JS toggle; remove `initLandingTabs`.
- Loading: optional `hx-indicator` on the results panel; keep minimal.

## Empty & auth CTA

When the filtered set is empty:

- Message along the lines of “No public streams in this category yet.”
- Button **Create a stream**:
  - If request has no session: `href="/login?next=/my/organization/formata-builder"`
  - If signed in: `href="/my/organization/formata-builder"`
- Reuse existing button classes (`btn btn-primary`).

## Errors

| Case | Behavior |
|------|----------|
| Taxonomy load failure on `/` | Existing 500 path |
| Catalog load failure | Existing 500 path |
| Unknown slugs on full page | Fall back to first/first |
| Unknown/missing slugs on partial | 404 + short fragment in swap target |
| Partial server error | 500 + short error fragment in results target; do not break the rest of the landing page |

## Files (expected touch set)

| Area | Change |
|------|--------|
| `public_home.html` | Replace tabs with sidebar + results shell; wire HTMX |
| New partial template | Results-only fragment (grid / empty) |
| `public-home.css` (+ optional component CSS) | Two-column / stack layout; sidebar states; drop tabs styles |
| `main.js` | Remove/replace `initLandingTabs`; accordion + active sync if needed |
| `handlePublicHome` + new handler | Taxonomy + selection + filter; register partial route in `newMux` |
| `PublicStreamCardView` / `publicStreamCards` | Category fields and/or filtered builder |
| `web/public/…` | Lucide static SVGs for chrome icons |
| Tests | `home_handler_test.go` (+ focused new tests): taxonomy render, default first/first, query selection, filter, empty CTA hrefs, partial without layout chrome |

## Testing

- Full page includes all taxonomy categories/subs from a seeded store (including zero-stream leaves).
- Default HTML has first category expanded and first subcategory active; results match that pair.
- `?category=&subCategory=` selects a different leaf when valid.
- HTMX partial returns cards or empty CTA only (no full landing chrome).
- Empty CTA uses `login?next=…formata-builder` when anonymous.
- Horizontal tab classes / “See all streams” absent.
- `task css:lint` (or project equivalent) still passes for touched CSS.

## Success criteria

- `#streams` matches Figma sidebar interaction (accordion + active subcategory) on light tokens.
- Selecting a subcategory updates the card grid via HTMX without full page reload.
- Every taxonomy category/subcategory is listed regardless of stream count.
- Empty leaves offer a clear path into Formata via login-next when needed.
