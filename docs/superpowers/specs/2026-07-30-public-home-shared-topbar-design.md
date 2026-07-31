# Public home: restore shared topbar + footer legal mix

Date: 2026-07-30  
Scope: `server/templates/layout.html`, `server/templates/pages/public_home.html`, new `server/templates/components/public_home_footer_nav.html`, `web/src/styles/pages/public-home.css`, `docs/css.md`, public-home handler tests  
Out of scope: Restoring marketing nav / Get Started in chrome, re-enabling layout `.site-footer` on `/`, redesigning hero or stream catalog, wiring parked footer nav links again

## Goal

Make `/` use the same app topbar as other pages. Keep the public-home footer shell and brand/bottom meta, but replace the Platform / Developer / Enterprise link columns with the long site legal copy. Park the link-column markup for later reuse without shipping it in the live page.

## Decisions

1. **Approach:** Layout flip for topbar only (option A). Do not mount a second topbar inside the landing body.
2. **Marketing header:** Remove `.public-home-header` entirely (brand, Product/Solutions/Pricing/Docs, Sign In / Get Started / custom account menu) and delete matching CSS.
3. **Signed-out chrome:** Shared topbar Login only — no Get Started in the topbar. Signup remains available via in-page CTAs (hero, etc.).
4. **Signed-in chrome:** Shared account menu (Dashboard, Orgs / My organization when flagged, Sign out) plus theme toggle and Platform Admin label when applicable — same as other pages.
5. **Page shell:** Keep `page-landing` on `<main>` so landing content stays flush under the topbar (`padding: 0`).
6. **Site footer:** Layout continues to skip `.site-footer` when `Body == public_home_body` (no double footer).
7. **Home footer shell:** Keep `.public-home-footer` structure: brand column, top row, bottom rule / short © line / socials.
8. **Link columns:** Move Platform / Developer / Enterprise markup into `components/public_home_footer_nav.html` (`define "public_home_footer_nav"`). Do **not** `{{ template }}` it from the live page yet.
9. **Legal copy:** In the former columns slot, render the same long paragraphs currently used in layout `.site-footer` (AGPL / Forkbomb, CLOSER, Even Closer, EU disclaimer). Style with home-footer tokens (muted, small type); do not restyle that slot as a second `.site-footer` chrome block.
10. **Docs:** Update `docs/css.md` exception for `public-home.css` — landing no longer hides the app topbar; it still uses its own footer (not the shared site footer).

## Layout (after)

```
[ shared .topbar — brand | theme | Login or account ]
<main class="page page-landing">
  <div class="public-home">
    [ hero / streams / features / developers — unchanged ]
    <footer class="public-home-footer">
      [ brand blurb | legal prose (ex-columns slot) ]
      [ rule | short © | socials ]
    </footer>
  </div>
</main>
```

Parked (not rendered): `public_home_footer_nav` link columns.

## Files

| Area | Change |
|------|--------|
| `layout.html` | Render `.topbar` for `public_home_body`; keep skipping `.site-footer` for that body |
| `public_home.html` | Delete header; replace footer cols with legal prose |
| `public_home_footer_nav.html` | New component holding parked link columns |
| `public-home.css` | Remove header rules; add legal-prose styles in footer cols slot; drop dead header responsive rules |
| `docs/css.md` | Correct public-home chrome exception |
| `home_handler_test.go` (etc.) | Expect shared topbar Login / account chrome; drop assertions for `.public-home-signin` / custom header |

## Test plan

- Anonymous `GET /`: body includes `class="topbar"` and Login; no `.public-home-header`; no Get Started in topbar.
- Signed-in `GET /`: topbar account menu (and admin links when flagged); no duplicate marketing account chrome.
- Footer: brand + legal paragraphs present; Platform / Streams / Developer link columns absent from HTML response.
- Parked component file exists and parses with templates (define registered) but is not invoked from the page.
- Theme toggle on `/` works with shared `data-theme` (no landing-only header conflict).

## Non-goals

- Bringing back marketing anchor nav in the topbar.
- Duplicating legal copy into a shared partial used by both footers (optional later DRY).
- Changing public-home footer bottom meta / social icons beyond fitting next to the new legal block.
