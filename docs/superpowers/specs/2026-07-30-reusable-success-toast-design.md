# Reusable success toasts

Date: 2026-07-30  
Scope: Add a reusable, zero-dependency success toast primitive; opt in categories-editor confirmations (create / edit / delete / reorder) with no layout shift  
Out of scope: Migrating orgs / login / home / reset banners; error toasts; toast libraries; changing confirmation query-param / redirect helpers; Go toast DTO / OOB HTMX fragments

Related: platform admin categories CRUD (`2026-07-30-platform-admin-categories-crud-design.md`); worktree `feat/admin-categories-crud`

## Goal

Success messages can appear as non-blocking toasts instead of in-flow `.confirmation` banners. Categories is the first consumer for all four success paths. Other pages opt in later with the same markup marker. No new JS package.

## Decisions

1. **Approach:** Opt-in promote — server keeps rendering a normal confirmation node; client lifts marked nodes into a fixed toast host.
2. **Opt-in marker:** `data-toast` on the confirmation element. Without it, behavior stays the current in-flow `.confirmation` banner.
3. **No layout shift:** `.confirmation[data-toast]` is out of normal document flow from first paint (CSS fixed / toast placement). The categories panel never jumps before or without JS.
4. **Errors:** Stay inline (`.error`). Never toasted in v1.
5. **No library:** Small CSS module + a few lines in `web/src/main.js`. `web/` stays free of runtime dependencies for this.
6. **No Go API change:** Handlers keep passing `Confirmation` strings. No `ToastView`, no OOB swap requirement.
7. **First consumer:** Categories panel only — the existing `{{ if .CategoriesEditor.Confirmation }}` block adds `data-toast`. Create / edit / delete / reorder all reuse that one node.
8. **Host:** Empty `#toast-host` in `layout.html` with `aria-live="polite"`.
9. **Dismiss:** Auto-dismiss ~3s after enter; optional click-to-dismiss; stack newest on top with a small cap (~3) so rapid reorder does not pile up forever.
10. **HTMX:** Re-scan on `htmx:afterSwap` (and initial load) so panel swaps that include a confirmation still toast.

## Layout

```
Viewport
┌────────────────────────────────────────┐
│  page content (unchanged flow)         │
│                                        │
│                          ┌───────────┐ │
│                          │ toast #1  │ │  ← #toast-host (fixed corner)
│                          │ toast #2  │ │
│                          └───────────┘ │
└────────────────────────────────────────┘
```

In the categories panel source HTML (before/without JS promote), the confirmation node may still be present in the DOM tree but must not occupy layout space:

```html
<p class="confirmation" data-toast>Subcategory reordered</p>
```

## Components & files

| Piece | Placement | Notes |
|-------|-----------|--------|
| Toast host | `server/templates/layout.html` | `#toast-host.toast-host`, `aria-live="polite"` |
| Toast CSS | `web/src/styles/components/toast.css` | CSS-only tier; markup contract in file header; import from `components.css` |
| Promote / dismiss | `web/src/main.js` | Find `[data-toast]`, move into host, animate, auto-dismiss |
| Categories opt-in | `server/templates/pages/platform_admin.html` | Add `data-toast` on categories confirmation `<p>` |
| Existing `.confirmation` | `web/src/styles/components/shared.css` | Unchanged for non-toast banners |

Reuse success color tokens already used by `.confirmation` (`--success-muted`, `--success-muted-foreground`). Follow `docs/css.md` layer rules.

## Client behavior

On `DOMContentLoaded` and `htmx:afterSwap`:

1. Query confirmation nodes with `[data-toast]` inside the loaded/swapped subtree (avoid double-promoting nodes already inside `#toast-host`).
2. Move each into `#toast-host` (newest first / on top).
3. Play enter transition → wait ~3s → exit transition → remove from DOM.
4. Click (or explicit dismiss control, if present) closes early.
5. If more than ~3 toasts are visible, drop the oldest.

Progressive enhancement: with CSS-only out-of-flow placement, a confirmation still appears as a fixed toast even if JS fails to run; without dismiss it may linger until navigation — acceptable for v1.

## First consumer (categories)

Unchanged server strings, for example:

- Category group created / updated / deleted / reordered
- Subcategory created / updated / deleted / reordered

Exact copy stays whatever handlers already emit. Only the template marker changes.

## Out of scope (v1)

- Platform-admin orgs confirmations
- Login / home / reset / stream-delete confirmation banners
- Error or warning toasts
- Sonner / Toastify / any toast npm package
- Changing `redirectPlatformAdminWithMessage` or query `confirmation=` helpers
- Dedicated Go view struct for toasts

## Testing & QA

- Template / markup test: categories confirmation includes `data-toast` when `CategoriesEditor.Confirmation` is set.
- Existing categories handler tests that assert confirmation text in the body continue to pass (text still present; may live under host after client promote — server response still contains the string).
- Manual QA: create → edit → delete → rapid reorder; confirm no panel jump; toast appears and auto-dismisses; errors still inline.

## Implementation workspace

Land on `feat/admin-categories-crud` worktree (same PR track as categories CRUD polish) unless split later.
