# `/app` i18n (wiring)

How to implement locale on the Org SvelteKit app. **Tracks and v1 policy:** [ADR-0013](../adr/0013-org-app-locale-tracks.md) — do not restate that decision here.

## Paraglide catalogs

| Piece | Path |
| --- | --- |
| Inlang project | [`app/project.inlang/settings.json`](../../app/project.inlang/settings.json) — `locales` `en` / `it`, **`baseLocale: it`** |
| Human-edited strings | [`app/messages/en.json`](../../app/messages/en.json), [`app/messages/it.json`](../../app/messages/it.json) |
| Generated API | `app/src/lib/paraglide/` (Vite `paraglideVitePlugin`; gitignored — compile on `bun run build`) |

**Strategy:** `cookie` → `globalVariable` → `baseLocale`. Cookie name **`PARAGLIDE_LOCALE`**. No Accept-Language in v1.

- Shell chrome: `import * as m from '#lib/paraglide/messages.js'` in `app/src/lib/org/` and `app/src/lib/router/`.
- **Engineering English:** hidden control in root layout calls **`setLocale('en')`** (persists cookie; avoid `localizeHref` / URL-prefix locales per ADR-0013).

## Production static `/app`

[`server/cmd/server/app_spa.go`](../../server/cmd/server/app_spa.go) serves `app/build` under `/app` — **no** `paraglideMiddleware`, no edge cookie negotiation, no `%paraglide.lang%` substitution on that path. App is CSR (`ssr = false`); locale is entirely client-side after the bundle loads.

In **`+layout.svelte`**, sync **`document.documentElement.lang`** from **`getLocale()`** (and `dir` if needed) on load and after `setLocale`, so Go-static HTML stays correct for a11y and tests.

## SJSF form chrome (not Config field titles)

Config `schema` / uiSchema titles remain pass-through (ADR track 2). Only `@sjsf/form` label ids (submit, array add/remove, yes/no, …) are translated.

1. **`getSjsfTranslation(locale)`** — map Paraglide `form_*` keys to a `Translation` (`fromRecord` / `overrideByRecord`).
2. **`chain(itPack, en)`** with `@sjsf/form/translations/en` — no upstream `it` pack; chain fills gaps until every label id is in the catalog.
3. Pass into **`createConfigForm`** ([`create-config-form.svelte.ts`](../../app/src/lib/org/form/create-config-form.svelte.ts)): `translation: getSjsfTranslation(getLocale())`. Remount when projection or locale changes.

Widget-only strings (combobox empty, date placeholders) use `extraUiOptions` / uiSchema, not the label table.

## Ajv

[`ajv-form-validator.ts`](../../app/src/lib/org/form/ajv-form-validator.ts): add **`ajv-i18n`**, set **`messages: false`** on Ajv options, and wire `localize` with `ajv-i18n/localize/it` or `/en` matching `getLocale()`. Submit banners: Paraglide for framing; error bodies from localized Ajv.

## Agent rules (from ADR consequences)

| Track | Code behavior |
| --- | --- |
| SPA shell | `m.*` only; no new EN/IT literals outside `app/messages/`. |
| Plant copy | Display Worklist / form bundle / QR messages verbatim — edit YAML via [`plant-config.md`](plant-config.md), not Paraglide. |
| Platform errors | Show Go HTTP bodies as English v1; Paraglide prefixes only. Role slugs are ids, not copy. |

**Verify:** e2e expects Italian chrome by default; CI must run `app` build so Paraglide compiles. Go Commit/Load/QR catalogs and dual-locale Config are out of scope for this wiring.

## Known post-v1 gaps

| Gap | Notes |
| --- | --- |
| Calendar / number locale | shadcn calendars still hardcode `en-US` (deferred in ADR-0013). |
| Go platform errors | English bodies by policy (ADR track 3). |
| shadcn dialog `Close` | Default/sr-only English in dialog chrome. |
| SJSF upstream fallbacks | Untranslated label ids still chain to English pack. |
| Icon / primitive a11y | bits-ui / lucide may still surface English aria strings. |
| Slice 5 guardrail | Optional lint: no new hardcoded EN/IT under `org/` / `router/` outside catalogs. |