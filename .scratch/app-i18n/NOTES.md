# Full Italian i18n for `/app` — research notes

**Goal:** Full Italian locale for the SvelteKit SPA at `base: /app`.

**Status:** slices 1–4 done in worktree `feat/i18n` (not committed) — **integrated onto** `feat/attesta-rework` @ `8a5d89a` (passport production FF + stash `i18n-wip-2` pop); see [`07-passport-delta-needs-i18n.md`](./07-passport-delta-needs-i18n.md); v1 Italian `/app` ready for review; optional slice 5 / leftover gaps below

**Slice 1 (locale foundation):** done — `baseLocale: it`, client `document.documentElement.lang` sync in `+layout.svelte`, hidden eng switch calls `setLocale` (cookie; no `preferredLanguage` / URL prefixes).

**Slice 2 (SPA chrome catalog):** done — 63 Paraglide keys in `messages/{en,it}.json` (stub `hello_world` removed); living/router chrome + SPA fallbacks/helpers (`taskStatusLabel`, `formatHandoffCopy`, port transport strings) use `m.*`. Config/server bodies untouched. SJSF/Ajv still slice 3.

**Slice 3 (SJSF + Ajv):** done — `form_*` Paraglide keys + `getSjsfTranslation(locale)` (`chain(it|en pack, upstream en)`); wired in `create-config-form`; `ajv-i18n` with `messages: false` + `localize/it|en`; shadcn combobox empty + date-range placeholder via `extraUiOptions`. Calendar number/format locale still deferred.

**Slice 4 (tests):** done — e2e expects Italian chrome by default (`Disponibili` / `Avviabili`), `html[lang=it]`, and `PARAGLIDE_LOCALE` not `en` (first `getLocale()` seeds cookie=`it`). `Home` stays `Home` (same in both catalogs). Unit tests already go through `m.*` / explicit `it`|`en` locale args. **Verified 2026-11-04:** `bun run test:unit -- --run` (relevant living i18n files) 40 passed; `APP_PORT=5181 CI=1 bun run test:e2e` 7 passed. Run: `cd app && bun run test:e2e` (Playwright browsers; builds preview on `APP_PORT`/`5180`; kill stale preview if `reuseExistingServer` serves an old `lang=en` build).

## Config vs SPA i18n boundary

Plant **Config YAML** labels (titles, `startLabel` / `closeLabel`, enumNames, QR messages) stay plant-authored Italian — **not** Paraglide. SPA shell + SJSF/Ajv chrome use Paraglide (`m.*`). See [`04-config-vs-spa-copy.md`](./04-config-vs-spa-copy.md).

## Locked decisions

1. **`baseLocale: it`** + cookie strategy; **no** Accept-Language on shared plant devices.
2. **Config labels** stay out of Paraglide; plant YAML must have **all labels in Italian** (titles / `startLabel` / `closeLabel` / enumNames / uiSchema titles — labels only, not every Config string).
3. **Go platform errors** stay English in v1.
4. Engineering locale: **hidden** `setLocale('en')` (cookie), not a visible plant UI.
5. **Calendar / number locale** deferred past v1.

## Remaining gaps for FULL Italian (post-slice-4)

| Gap | Notes |
|-----|-------|
| **Calendar / number locale** | shadcn calendar / range-calendar hardcode `locale = "en-US"`; deferred by decision 5. |
| **Go platform errors** | Commit/Close/Load/QR HTTP bodies stay English (decision 3); SPA only prefixes via Paraglide. |
| **shadcn dialog Close** | `dialog-content.svelte` / `dialog-footer.svelte` still have English `Close` (sr-only / default footer). |
| **SJSF upstream fallbacks** | `unknown-field-error` and other untranslated keys still chain to English upstream pack. |
| **Icon / primitive aria-labels** | bits-ui / lucide calendar nav, etc. may still surface English a11y strings until wired to locale. |
| **Role chips** | `administration` / `logistic` are Config role ids, not product copy. |
| **`chrome_failed` / `chrome_display_missing`** | Catalog values still English-ish diagnostic tokens. |
| **Slice 5 guardrail** | Optional lint: no new hardcoded EN/IT in `living/` / `router/` outside catalogs — not started. |

## Artifacts

| File | Agent | Verdict |
|------|-------|---------|
| [`01-spa-chrome-strings.md`](./01-spa-chrome-strings.md) | [SPA chrome strings](899f9078-cab9-470e-bf8a-919f33e685c0) | ~70–85 Paraglide keys; mixed IT/EN chrome today |
| [`02-sjsf-i18n.md`](./02-sjsf-i18n.md) | [SJSF i18n](06a0cb73-fffc-426a-9047-d26991a2472a) | Paraglide → `createForm({ translation })`; Ajv via `ajv-i18n` |
| [`03-paraglide-wiring.md`](./03-paraglide-wiring.md) | [Paraglide wiring](637ffbe3-0a57-4327-ba1a-b46d7dee2eb6) | cookie → en; Go static bypasses middleware; defer URL prefixes |
| [`04-config-vs-spa-copy.md`](./04-config-vs-spa-copy.md) | [Config vs SPA](33fb7de0-c1bb-4db3-b46d-1d8c9b0b5d7b) | Config Italian stays; SPA chrome + SJSF chrome in Paraglide |
| [`05-config-labels-italian-audit.md`](./05-config-labels-italian-audit.md) | [Config label audit](23a1f4c9-aea8-4632-a552-ba09ceecc5dc) | 2 Feragame gaps — **patched** (`Corrispondenza %`, `lista lavori`) |
| [`06-rework-delta-needs-i18n.md`](./06-rework-delta-needs-i18n.md) | [Rework delta i18n](d9544aec-ae29-453d-9655-09471321c61e) | MAIN +3 commits; **3** keys added (`in_progress`, `no_jobs_in_progress`, `draft_bozza`) |
| [`07-passport-delta-needs-i18n.md`](./07-passport-delta-needs-i18n.md) | passport delta scan | MAIN +2 (`6839070`, `8a5d89a`); **2** keys (`completed`, `no_completed_jobs`); Config gaps **0** |

**Integrate (done 2026-11-04 @ 0e69179):** FF → `0e69179`; stash `i18n-wip` pop. Conflicts: `LivingShell.svelte` + `worklist-home.ts`. Wired `WorklistJobLine` `tasks_done_of_total`. Verify then: build OK; unit 53; e2e `APP_PORT=5181` 7 passed.

**Passport update (done 2026-11-04):** FF `feat/i18n` → `8a5d89a` (`6839070` + `8a5d89a`); stash `i18n-wip-2` pop. Overlap 3-way clean (no conflict markers): `LivingShell.svelte`, `worklist-home.ts`, `worklist-overview.test.ts`, `overview.svelte`, `task.svelte`, e2e, `feragame-config.yaml` — kept passport Completed band / finished-overview stay / `commitHasLinks` + prior `m.*` wiring. Added Paraglide `completed` / `no_completed_jobs`. Feragame keeps Italian patches + new passport job labels (`Verifica dati e DPP`, `Emetti DPP`, …). **Verify:** `bun run build` OK; unit subset **80 passed** (7 files); **e2e:** `APP_PORT=5182 CI=1 bun run test:e2e` — **7 passed, 0 failed**. Delta notes: [`07-passport-delta-needs-i18n.md`](./07-passport-delta-needs-i18n.md).

**Remaining after integrate:** same post-slice-4 gaps table above (calendar, Go errors, dialog Close, SJSF upstream fallbacks, a11y primitives, slice 5 guardrail). Go `issuePassport` link label `Digital Product Passport` still English platform copy (v1). No unresolved merge conflicts.

## Scope recommendation (synthesis)

**In scope for FULL `/app` i18n (Feragame Italian plant):**

1. **SPA shell chrome** → Paraglide `m.*` (~75 keys): nav, home, overview/task shells, scan, handoffs, status tags, load/commit prefixes, empty/loading, fallbacks (`Spawn`/`Close`/`Start`/`Home`).
2. **SJSF form chrome** → Paraglide-backed `getSjsfTranslation(locale)` into `createForm({ translation })`; `chain(it, en)` until complete. No upstream `it` pack (only `en`/`ru`).
3. **Ajv validation messages** → add `ajv-i18n`, `messages: false`, `localize/it` in `ajv-form-validator.ts`.
4. **Locale default** → `baseLocale: it`, cookie strategy, client `document.documentElement.lang`; engineering via `setLocale('en')`. **Defer** URL-prefix i18n (`/it/...`) — conflicts with `paths.base: '/app'` + Go static.

**Out of scope (separate tracks):**

- Org Config YAML titles / `startLabel` / `closeLabel` / enumNames / QR `messages` / dataReport labels (already Italian for Feragame; plant-authored).
- Go Commit/Close/Load/QR **platform** English errors (unless later Config-owned denial text or Go i18n).
- Dual-locale Config authoring.
- shadcn calendar `en-US` formatters (follow-on once Paraglide locale is live).

## Proposed implementation slices

1. **Locale foundation:** `baseLocale: it`, client `lang` sync, tiny locale switch for eng; fix/remove unsafe hidden `localizeHref` links. ✅
2. **SPA chrome catalog:** migrate living/router strings from `01` inventory to `messages/{en,it}.json`. ✅
3. **SJSF + Ajv:** translation adapter + `ajv-i18n`; wire in `create-config-form` / validator. ✅
4. **Tests:** update e2e expectations (`Available` → Italian); assert cookie/`lang`. ✅
5. **Guardrail (env):** optional lint/check — no new hardcoded EN/IT in `living/`/`router/` outside catalogs.

## Open decisions (need user)

- [x] Confirm **`baseLocale: it`** + cookie (no Accept-Language on shared devices)?
- [x] Confirm **Config copy stays plant-authored** (no Paraglide for schema titles)?
- [x] Confirm **Go platform errors stay English** for v1?
- [x] Engineering locale switch: hidden cookie toggle vs visible UI?
- [x] Include **calendar/number locale** in v1 or defer?

## Environment improvements

- Single catalog (`m.*`) + SJSF adapter — no parallel translation systems.
- Document Config-vs-SPA boundary (this folder + eventually AGENTS/app note).
- Prefer a check that living UI literals go through Paraglide once catalogs exist.
