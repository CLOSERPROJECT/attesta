# Public stream page

Date: 2026-07-31  
Scope: new `GET /streams/:key` public page + templates/CSS/view models; wire landing `public_stream_card` as a link; recent completed-run cards (DPP digital link when present); read-only workflow blueprint reusing timeline/substep preview builders  
Out of scope: signup / “Open in workspace” CTAs; public instance detail under `/streams/:key/instance/…`; start/complete/terminate/delete; changing `/my/streams/:key`; category sidebar / HTMX partial behavior beyond keeping `/streams/public` working; Figma asset download

## Goal

Let guests open a shareable public page for a catalog stream from the homepage card. The page expands card metadata, lists recent **completed** runs as small cards (linking to the public DPP when available), and shows a full read-only workflow blueprint (steps → substeps with input/instructions, no actions).

## Decisions

1. **Approach:** Dedicated public page at `/streams/:key` (not anonymous `/my`, not a landing modal).
2. **Catalog gate:** Only workflow keys that exist in the same catalog used to build public home cards are reachable; unknown keys → **404**. (Direct URL does not require a category match; uncategorized catalog streams are reachable by key even if they never show in a subcategory filter.)
3. **Auth:** Same page for guests and signed-in users; no redirect to `/my`. Browse-only — no primary CTA.
4. **Entry:** Whole landing `public-stream-card` links to `/streams/:key` via new `Href` on `PublicStreamCardView`.
5. **Recent runs:** Only **completed** processes; newest first; cap after filter (**8**). Incomplete and terminated never listed.
6. **Run cards:** Minimal meta — status + completed date. With `process.DPP` → `<a>` to `/01/{gtin}/10/{lot}/21/{serial}` plus a DPP cue. Without DPP → static non-link card with the same meta.
7. **Blueprint:** Full locked-style substep detail via existing read-only timeline builders (`HideStatus`, preview/message modes). No complete forms, Formata posts, override links, or upload controls.
8. **Chrome:** Shared site topbar (same pattern as public home). Light shared tokens; no new landing palette.
9. **Routing:** Keep exact `/streams/public` as the HTMX results partial; `:key` must not capture `public`.

## Layout

```
[ shared topbar ]

[ header ]
  title
  description
  optional DPP chip (stream passport enabled)
  metrics: runs / steps / roles
  organizations (avatars + count)

[ recent completed runs ]
  heading: “Recent completed runs”
  grid/list of small cards (see above)
  empty: “No completed runs yet.”

[ blueprint ]
  heading: “Workflow”
  full read-only stream timeline

[ no primary CTA ]
```

Mobile: single column — header → runs → blueprint.

## Routing & data

| Piece | Behavior |
|-------|----------|
| `GET /streams/:key` | Public handler; accept trailing slash; prefer generating links without trailing slash |
| `GET /streams/public` | Unchanged HTMX partial |
| Header data | Reuse fields from `buildPublicStreamCardView` (name, description, passport, counts, orgs) |
| Recent runs | List completed processes for workflow key; map `DigitalLink` when DPP present (`digitalLinkPath` / existing helper) |
| Blueprint | Empty/synthetic process + `makeStreamInstanceDetailReadOnly` (or equivalent) → `stream_timeline` / `substep_shell` / `substep_body` |

### Handler sketch

1. Resolve `key` from path; if `key == "public"`, do not use this handler (dedicated route already registered).
2. Load catalog/runtime config for key; if missing → 404.
3. Build header view + recent completed run views (cap 8) + read-only detail/timeline.
4. Render page template with shared topbar layout flags consistent with public home.

## Components

| Piece | Role |
|-------|------|
| `public_stream_card` | Add `Href`; whole card is the click target |
| New page template (e.g. `public_stream`) | Page body; header + runs section + blueprint |
| Header block | Page-local markup (not nested `public_stream_card`) |
| New `public_stream_run_card` | Status + completed date; link shell iff `DigitalLink` set |
| Blueprint | Reuse `stream_timeline` + substep stack in read-only preview |
| CSS | New page stylesheet + thin run-card component CSS; follow `docs/css.md` / attesta-ui-components |

## Errors

| Case | Behavior |
|------|----------|
| Unknown key (not in workflow catalog) | 404 |
| Store / config failure | 500 (existing pattern) |
| Zero completed runs | 200; empty runs copy; blueprint still renders |
| Stream DPP disabled / run without DPP | Run card still shown if completed; not a link |

## Testing

- `GET /streams/:key` → 200 for a catalog stream; body includes title and timeline/blueprint chrome.
- Unknown key → 404.
- `/streams/public` still returns the results partial (no regression).
- Homepage card HTML includes `href="/streams/{key}"`.
- Recent section lists only completed; DPP run is an `<a>` to `/01/…`; non-DPP completed is not a link.
- Cap applied; active/terminated absent.
- Blueprint HTML contains no complete-action forms / start / terminate controls.

## Success criteria

- From a landing stream card, a guest reaches a bookmarkable `/streams/:key` page.
- Page shows catalog-level meta, recent completed runs (DPP-linked when possible), and a full read-only workflow blueprint.
- No auth-gated actions or marketing CTAs on the page.
- Public home filter partial and authenticated `/my/streams/:key` remain unchanged in behavior.
