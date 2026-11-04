# Paraglide locale wiring — Attesta SPA (`/app`)

Research date: 2026-11-04. Sources: `app/` SvelteKit tree, generated `app/src/lib/paraglide/` (local compile; gitignored), `server/cmd/server/app_spa.go`.

---

## 1. Current locale resolution

### Compiler / project settings

| Setting | Value |
|--------|--------|
| `project.inlang/settings.json` | `baseLocale`: **`en`**, `locales`: **`en`**, **`it`** |
| Message catalogs | `app/messages/{en,it}.json` — stub **`hello_world`** only |
| Vite plugin | `paraglideVitePlugin({ project, outdir: './src/lib/paraglide', emitTsDeclarations: true })` — **no explicit `strategy` or custom `urlPatterns` in `vite.config.ts`** |
| Generated runtime strategy | **`["cookie", "globalVariable", "baseLocale"]`** (see `runtime.js`) |
| Cookie | Name **`PARAGLIDE_LOCALE`**, `path=/`, long `max-age` |

**Not active today:** `"url"`, `"preferredLanguage"` (Accept-Language), `"localStorage"`. Accept-Language is **not** consulted on the client or in middleware vary headers with the current strategy.

### Resolution order (browser, after JS loads)

1. **`cookie`** — read `PARAGLIDE_LOCALE` if set.
2. **`globalVariable`** — in-memory `_locale` after first `getLocale()` / `setLocale()`.
3. **`baseLocale`** — **`en`**.

On first `getLocale()` in the browser, Paraglide resolves (typically cookie miss → `baseLocale`), then calls **`setLocale(resolved, { reload: false })`** to seed cookie + global state without reload.

**Server-side (SvelteKit `hooks.server.ts` only):** `paraglideMiddleware()` uses the same strategy list via `shouldRedirect` / `extractLocaleFromRequest*` — cookie first, then **`en`**. It does **not** delocalize URLs because **`"url"` is not in `strategy`**. It **does** run `transformPageChunk` to replace `%paraglide.lang%` and `%paraglide.dir%` in HTML when SvelteKit serves a document through this handle.

### URL shape with `paths.base: '/app'`

| Layer | Behavior |
|-------|----------|
| **Public URLs** | `https://<host>/app`, `https://<host>/app/<jobId>/…` (living routes) |
| **SvelteKit** | `paths.base: '/app'` in `vite.config.ts` |
| **sv-router** | `<Router base="app" />`; internal paths are **base-stripped** (e.g. `/intake/NIR-1/register`) — see `nav-place.test.ts` |
| **Paraglide default URL pattern** | Treats the **first pathname segment** as a optional locale prefix (`/:locale/...`). Segment **`app` is not a locale**, so `/app/intake/...` is **not** interpreted as localized. |
| **`hooks.ts` `reroute`** | `deLocalizeUrl(request.url).pathname` — only strips a leading segment if it is a known locale. With `/app/...`, the first segment is **`app`**, so paths are **unchanged**. |
| **Hidden switcher (`+layout.svelte`)** | `localizeHref(page.url.pathname, { locale })` then SvelteKit `resolve(...)`. With default patterns, switching to **`it`** would tend toward **`/it/app/...`** (locale **before** mount path), **not** `/app/it/...` — misaligned with `paths.base` and Go static routing unless URL patterns are customized. |

Generated `urlPatterns` in `runtime.js` describe root-level `/it/:path` vs unprefixed `en`; they are **not** wired into locale detection because **`"url"` ∉ `strategy`**. They still affect **`localizeHref` / `deLocalizeUrl`** helpers when used.

### Static hosting implications

Production serves the SPA from Go, **not** through SvelteKit middleware:

- **`server/cmd/server/app_spa.go`**: GET/HEAD under `/app` → files from `app/build` (or `ATTESTA_APP_DIR`); extensionless paths **fallback to `index.html`** (SPA).
- **No `paraglideMiddleware`** on this path — no cookie negotiation, no `%paraglide.*%` substitution, no redirects at the edge.
- **`+layout.ts`**: `export const ssr = false` — CSR-only; locale for `m.*` is entirely **client-side** after bundle load.
- **`hooks.server.ts`** applies only when the app is served by **SvelteKit** (`vite dev`, `vite preview`, Playwright e2e webServer). Production operators hitting Go get the **built `index.html` as stored on disk** (placeholders replaced only if the build/preview pipeline ran the handle transform for that artifact).

**Practical effect:** `<html lang="...">` may stay as placeholders or a single build-time locale in production unless the client updates `document.documentElement.lang` (or SSR/prerender is re-enabled for the shell).

---

## 2. Gaps vs full Italian

| Area | Gap |
|------|-----|
| **Default locale** | `baseLocale` is **`en`**; anonymous / cookie-less users get English from Paraglide even though much living copy is already Italian in config/fixtures. |
| **HTML `lang` / `dir`** | `app.html` uses `%paraglide.lang%` / `%paraglide.dir%`; substitution depends on SK handle — **not** on Go static. No client-side sync observed. |
| **`setLocale` usage** | **No app imports** of `setLocale` / `getLocale`; only hidden `<a href={localizeHref(...)}>` links (no click handler calling `setLocale`). Locale switching is incomplete vs cookie-only strategy. |
| **`m.*` consumption** | **Zero** production imports of `#lib/paraglide/messages.js` — only stub `hello_world` exists. All chrome strings remain hardcoded (English in e2e expectations: "Available", "Can start", "Home"). |
| **Router vs `localizeHref`** | `navigateLivingTarget` / `hrefLivingTarget` use **sv-router** `navigate` / `p()` with **no** Paraglide — locale won't follow in-app navigation; URL-prefix i18n conflicts with **`/app`** + default `/:locale` pattern. |
| **Accept-Language** | Not in strategy — browser language ignored. |
| **Dates / numbers** | shadcn calendar components default **`locale = "en-US"`** — independent of Paraglide. |
| **Config / API copy** | Worklist titles, task titles, labels come from **living HTTP/config** (often Italian) — outside Paraglide catalogs (see sibling scratch notes). |

---

## 3. Generated `app/src/lib/paraglide` (present locally; gitignored)

Regenerated by `@inlang/paraglide-js` on dev/build. Do not edit by hand.

### Layout

- `messages.js` — re-exports **`m`** namespace (`hello_world` only).
- `messages/{en,it}.js`, `messages/_index.js` — per-locale message functions.
- `runtime.js` / `runtime.d.ts` — locale detection, URL helpers, cookies.
- `server.js` / `server.d.ts` — **`paraglideMiddleware`**.
- `registry.js` — ICU/plural/datetime formatters (unused by app today).

### Runtime exports **used by app source today**

| Export | Where |
|--------|--------|
| `deLocalizeUrl` | `app/src/hooks.ts` (`reroute`) |
| `paraglideMiddleware` | `app/src/hooks.server.ts` |
| `getTextDirection` | `app/src/hooks.server.ts` (`transformPageChunk`) |
| `locales`, `localizeHref` | `app/src/routes/+layout.svelte` (hidden locale links) |

### Available but unused (representative)

`getLocale`, `setLocale`, `m` / `hello_world`, `deLocalizeHref`, `localizeUrl`, `extractLocaleFromUrl`, `extractLocaleFromCookie`, `extractLocaleFromNavigator`, `extractLocaleFromHeader`, `generateStaticLocalizedUrls`, registry formatters, custom strategy hooks.

---

## 4. Recommended default-locale strategy (Italian-first plant, English for engineering)

**Primary recommendation (fits current architecture: cookie strategy, no URL prefix, `/app` mount, sv-router):**

1. **Set `baseLocale` to `it`** in `project.inlang/settings.json` and recompile — cookie-less plant tablets default to Italian for `m.*` and Paraglide fallbacks.
2. **Keep strategy `["cookie", "globalVariable", "baseLocale"]`** for predictable kiosk behavior; **avoid `preferredLanguage`** on shared plant devices (engineers’ laptop browsers would otherwise flip locale on first visit).
3. **Engineering English:** expose an in-app locale control that calls **`setLocale('en')`** (sets `PARAGLIDE_LOCALE`, reloads by default) rather than relying on **`localizeHref`** until URL patterns include **`/app`** correctly.
4. **Document `lang` on `<html>`:** in root layout (client), set `document.documentElement.lang = getLocale()` (and `dir` if needed) so Go-served static HTML stays accessible even without SK middleware.
5. **Defer URL-prefix i18n** unless product needs shareable `/app/it/...` links; if added later, configure Paraglide **`urlPatterns`** with **`/app/it/...` vs `/app/...`**, add **`"url"`** to strategy, and teach **`hooks.ts` reroute** + **sv-router** to strip **both** base and locale — default `/:locale` is incompatible with `paths.base: '/app'`.

**Optional:** role- or org-based default via `overwriteGetLocale()` (e.g. engineering role → `en`) — only if cookie override is insufficient.

---

## 5. E2E / test implications

| Topic | Implication |
|-------|-------------|
| **Playwright** | `playwright.config.ts` runs **`build && preview`** on port 5180; **`baseURL` is site root**, tests use full paths **`/app/...`**. Middleware may replace `%paraglide.*%` on first document load — **differs from Go production**. |
| **Assertions** | `page.svelte.e2e.ts` expects **English** UI ("Available", "Can start", "Home", role chip `administration`) while stubs use **Italian** task titles — tests encode **English chrome**, not Paraglide. |
| **Locale isolation** | No tests set `PARAGLIDE_LOCALE`, Accept-Language, or `/it/` URLs. Changing `baseLocale` to `it` will **fail e2e** until strings move to `m.*` or tests assert localized copy. |
| **Vitest** | Router/nav tests use **base-stripped** pathnames (`/intake/...`); no Paraglide coverage. |
| **CI** | Generated `src/lib/paraglide` is gitignored — **`bun run build` must compile messages** on CI (plugin runs on build). |

---

## Deliverable summary

**Path:** `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta/.scratch/app-i18n/03-paraglide-wiring.md`

### Five-bullet gap list

1. **`baseLocale` is `en`** while product chrome is largely hardcoded English and config copy is often Italian — Paraglide defaults disagree with operator reality.
2. **No `m.*` usage** beyond scaffold; full Italian requires a large catalog + living/router migration, not settings alone.
3. **Production Go static serve bypasses `paraglideMiddleware`** — `%paraglide.lang%` / accurate `lang` attribute not guaranteed; CSR-only (`ssr = false`).
4. **`localizeHref` + default `/:locale` pattern conflict with `paths.base: '/app'`** and sv-router base-stripped paths; hidden locale links are not a safe production switcher.
5. **In-app navigation ignores Paraglide** (`navigate` / `p()`); **`setLocale` is never called** — cookie strategy is only partially wired.

### Recommended default-locale strategy (short)

**Italian-first:** set **`baseLocale: it`**, keep **cookie > globalVariable > baseLocale**, switch engineers with **`setLocale('en')` + cookie**, sync **`<html lang>` on the client** for Go static, and **postpone URL-prefix locales** until `/app`-aware `urlPatterns` and reroute/router integration are designed.
