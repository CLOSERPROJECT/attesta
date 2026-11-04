# SPA chrome string inventory — `living` + `router`

**Scope:** `app/src/lib/living/**` and `app/src/lib/router/**` (production UI).  
**Excluded:** `app/src/lib/shadcn/**` primitives except where living theme surfaces user-facing English (noted below).  
**Paraglide today:** stub `hello_world` only (`app/messages/en.json`, `it.json`).

## Mixed locale (finding)

The SPA already ships **Italian** in some high-visibility chrome and **English** elsewhere. Full `it` locale requires normalizing both, not only translating English leftovers.

| Locale | Examples |
|--------|----------|
| **IT** | Status tags (`Bloccato`, `Per te`, …), toast dismiss `Chiudi`, `Modulo non disponibile`, `Validazione fallita`, checkbox default `Fonti doppie`, data-report prefix `anche:` |
| **EN** | Home sections `Available` / `Can start`, nav `Back` / `Scan`, handoff banner, post-commit `Continue` / `Start another`, scan dialog, most load/commit error prefixes, fallbacks `Spawn` / `Close` / `Start` / `Home` / `Job instance` |
| **Neutral** | `…` (ellipsis busy states), em dash `—` in recap/formula empty states |

---

## (a) SPA chrome → Paraglide catalogs

Strings **authored in the SPA** (or SPA-owned fallbacks when Config omits labels). These should become `m.*` keys (or shared helpers wrapping Paraglide).

### `app/src/lib/router/layout.svelte`

| String | Lang | Context |
|--------|------|---------|
| `Back` | EN | Icon back button `aria-label` |
| `Scan` | EN | Nav button visible label + `aria-label` |
| `Attesta` | EN | Logo `alt` on home mark |

`place.primaryLabel` / `place.jobTitleLine` are **mostly Config** (see §b); only nav chrome above is fixed copy.

### `app/src/lib/router/task.svelte`

| String | Lang | Context |
|--------|------|---------|
| `Missing Task address` | EN | Invalid route / missing `jobId` or `taskId` |
| `Close denied` / `Commit denied` | EN | Prefix before `{living.commitError}` |
| `Redirecting…` | EN | Deep-link gate redirect |
| `Loading…` | EN | Task form bundle fetch |
| `Confirm close` | EN | Inline close confirm `aria-label` |
| `{groupLoop.closeLabel}?` | Config* | Title uses Config close label when present |
| `This finishes the group and unlocks the next step.` | EN | Close confirm helper (inline + matches overview dialog) |
| `Cancel` | EN | Close confirm cancel |
| `{groupLoop.closeLabel}` | Config* | Confirm submit / close button when Config provides label |
| `Start another` | EN | Post-commit group spawn loop |
| `Continue` | EN | Post-commit when no group chrome |

\*Button text is Config when set; English **fallback** for missing close label comes from `worklist-overview.ts` (`Close`).

Dynamic: `{error}` from load/commit (often **server body** — display as-is, do not duplicate in catalog).

### `app/src/lib/router/overview.svelte`

| String | Lang | Context |
|--------|------|---------|
| `Loading…` | EN | Initial domain load |
| `Close denied:` | EN | Prefix + `{living.commitError}` |
| `{formatHandoffCopy(...)}` | EN | Handoff banner (templates in TS, § below) |
| `{row.group.startLabel}` / `{row.group.closeLabel}` | Config* | Spawn link / close button |
| `{confirmCloseGroup?.closeLabel ?? 'Close'}?` | EN fallback | Dialog title |
| `This finishes the group and unlocks the next step.` | EN | Dialog description |
| `Cancel` | EN | Dialog footer |
| `{confirmCloseGroup?.closeLabel ?? 'Close'}` | EN fallback | Dialog confirm button |

Task row **titles** and group **displayId** come from Worklist projection (§b).

### `app/src/lib/router/home.svelte` / `router.svelte`

No user-facing literals (shell only).

### `app/src/lib/living/LivingShell.svelte`

| String | Lang | Context |
|--------|------|---------|
| `Loading…` | EN | Domain hydrate |
| `{living.error}` | server | Load failure message |
| `Not loaded yet.` | EN | Idle before hydrate |
| `Commit denied:` | EN | + `{living.commitError}` |
| `Committing…` | EN | In-flight commit |
| `Available` | EN | Section heading |
| `{row.instanceLabel}`, `{row.jobTitle}`, `{row.title}` | Config | Row primary/secondary lines |
| `{row.tasksDone} of {row.tasksTotal} tasks done` | EN | Progress ring `aria-label` |
| `No available Tasks.` | EN | Empty available list |
| `Can start` | EN | Section heading |
| `{row.startLabel}` | Config | Can-start row title |
| `No Jobs to start.` | EN | Empty can-start list |

### `app/src/lib/living/WorklistRow.svelte`

| String | Lang | Context |
|--------|------|---------|
| `Available` | EN | Ready dot `aria-label` |

### `app/src/lib/living/ScanDialog.svelte`

| String | Lang | Context |
|--------|------|---------|
| `Camera permission denied — paste the code below.` | EN | Camera error hint |
| `Camera not available — paste the code below.` | EN | Camera error hint |
| `Scan` | EN | Dialog title |
| `Point at a label QR, or paste the link / id.` | EN | Dialog description |
| `Paste entityId or URL` | EN | Manual input placeholder |
| `Go` | EN | Submit button (non-busy) |
| `…` | neutral | Busy submit (icon replaced) |

### `app/src/lib/living/LivingToast.svelte`

| String | Lang | Context |
|--------|------|---------|
| `{livingToast.message}` | mixed | QR/server/toast pipeline (§b for server) |
| `Chiudi` | IT | Dismiss control |

### `app/src/lib/living/TaskRecapPanel.svelte`

| String | Lang | Context |
|--------|------|---------|
| `Loading recap…` | EN | Fetch in progress |
| `Could not load recap` | EN | Generic catch fallback |
| `{error}` | server | `TaskRecapError.message` or transport |
| `{key}` / `{formatFactValue(value)}` | server/Config | Recap fact grid (keys from server facts) |

### `app/src/lib/living/TaskForm.svelte`

| String | Lang | Context |
|--------|------|---------|
| `Modulo non disponibile` | IT | Live form failed to mount |
| `{submitError}` | mixed | From `formatSubmitValidationError` (AJV messages often EN from SJSF; wrapper IT) |

SJSF field **titles/descriptions** come from Task bundle schema/uiSchema (§b / `02-sjsf-i18n.md`).

### `app/src/lib/living/TaskStatusTag.svelte`

| String | Lang | Context |
|--------|------|---------|
| `{taskStatusLabel(status)}` | IT | Via `worklist-overview.ts` |

### `app/src/lib/living/nav-place.ts`

| String | Lang | Context |
|--------|------|---------|
| `Home` | EN | `primaryLabel` on home route |
| `Job instance` | EN | Overview fallback when instance label missing |
| `{overview?.label}`, `{taskTitle}`, `{instanceLabel} · {taskTitle}` | Config | Composed nav titles |

### `app/src/lib/living/worklist-home.ts`

| String | Lang | Context |
|--------|------|---------|
| `'Start'` | EN | Fallback when `job.startLabel` empty |

All other row strings in this module are projected from Worklist (§b).

### `app/src/lib/living/worklist-overview.ts`

| String | Lang | Context |
|--------|------|---------|
| `'Spawn'` | EN | Group spawn fallback when no `startLabel` / title |
| `'Close'` | EN | Group close fallback when no `closeLabel` |
| `Bloccato` | IT | Status tag |
| `Non disponibile` | IT | Status tag |
| `Disponibile` | IT | Status tag |
| `Per te` | IT | Status tag |
| `Fatto` | IT | Status tag |
| `—` | neutral | `formatFactValue` null/undefined |
| `Next work is waiting for another role.` | EN | Handoff, no role names |
| `Next work waits for role: ${role}.` | EN | Handoff, one role (role id from Config) |
| `Next work waits for roles: ${roles}.` | EN | Handoff, comma-separated role ids |

Task/group **titles** in overview model: `chrome?.title || taskId` (§b).

### `app/src/lib/living/qr-scan-flow.ts`

| String | Lang | Context |
|--------|------|---------|
| `Empty scan` | EN | Toast when trimmed raw empty |

Other toasts: server `error.message`, HTTP exception text, or `QR_RESOLVE_FAILED` (below).

### `app/src/lib/living/qr-resolve-port.ts`

| String | Lang | Context |
|--------|------|---------|
| `QR resolve failed` | EN | Normalized invalid resolve body |

### `app/src/lib/living/http-qr-resolve-port.ts` (+ sibling HTTP ports)

| String | Lang | Context |
|--------|------|---------|
| `living qr-resolve failed (${status})` | EN | Non-200/422 transport |
| `living domain load failed (${status})` | EN | `http-load-port.ts` |
| `living commit failed (${status})` | EN | `http-commit-port.ts` |
| `living task-form failed (${status})` | EN | `http-task-form-port.ts` |
| `living task-recap failed (${status})` | EN | `http-task-recap-port.ts` |
| `living /api/living/close failed (${status})` | EN | `http-group-close-port.ts` |
| `invalid task-recap body` | EN | `parseTaskRecap` guard |

When response body is non-empty, **server `detail` text** is shown instead (§b).

### `app/src/lib/living/form/submit-validation.ts`

| String | Lang | Context |
|--------|------|---------|
| `Validazione fallita` | IT | No errors array |
| `` `${path}: ${message}` `` | mixed | Per-field; `message` from AJV/SJSF |

### `app/src/lib/living/form/data-report-field.svelte`

| String | Lang | Context |
|--------|------|---------|
| `{title}` | Config/server | From chrome display bag / YAML |
| `{chromeError}` | technical | e.g. `chrome-display-missing` (§ below) |
| `No primary load` | EN | Empty sections state |
| `anche: {ids}` | IT prefix | Secondary load ids |
| `{row.label}` / `{row.value}` / `{row.error}` | Config/server | Chrome rows |
| `{section.id}` | Config/domain | Section heading (often load id) |
| `…` | neutral | Empty row value placeholder |

### `app/src/lib/living/form/live-task-chrome.svelte.ts` + `jsonata-safe.ts`

User-visible **error tokens** (readonly/data-report):

| String | Lang | Context |
|--------|------|---------|
| `chrome-failed` | EN | JSONata eval undefined / bad number |
| `chrome-display-missing` | EN | Missing server chromeDisplay entry |
| `jsonata denied: $${name}` | EN | Blocked builtin |
| `jsonata compile: …` / `jsonata evaluate: …` | EN | JSONata failures |

### `app/src/lib/living/form/read-columns.ts`

| String | Lang | Context |
|--------|------|---------|
| `Id` | EN | Default map-table id column header |
| `Value` | EN | Default value column header |

Overridden when uiSchema `columns.id` / `columns.value` set (Config).

### `app/src/lib/living/form/checkbox-only-field.svelte`

| String | Lang | Context |
|--------|------|---------|
| `Fonti doppie` | IT | Default `title` / `aria-label` / `title` attr when schema title absent |

### `app/src/lib/living/form/derived-leaf-field.svelte`

| String | Lang | Context |
|--------|------|---------|
| `—` | neutral | Empty formula display |
| `override` | EN | Start manual override |
| `restore` | EN | Revert override |

### `app/src/lib/living/form/HttpLinkStrip.svelte`

| String | Lang | Context |
|--------|------|---------|
| `{item.label}` | Config/server | Commit/recap HTTP links |
| `{err}` | server | `httpErrors` strings |
| `download` | EN | Fallback download filename only (not shown as label) |

### `app/src/lib/living/form/hidden-field.svelte`

| String | Lang | Context |
|--------|------|---------|
| `{label}` | Config | `uiOption('title') ?? config.title` → presence mark `aria-label` |

### `app/src/lib/living/form/compact-array-item-field.svelte`

Remove control uses SJSF `<Text id="remove-array-item" {translate} />` — **theme/SJSF i18n**, not a living literal.

### `app/src/lib/living/form/map-table-field.svelte`

Headers: `columns.id`, column titles from schema/uiSchema — **Config** (defaults `Id`/`Value` from `read-columns.ts` above).

---

## (b) Config / server-provided — do **not** duplicate in Paraglide

Display in UI but **source of truth is Org Config, domain projection, or API**:

| Source | Where it appears |
|--------|------------------|
| Worklist `job.title`, `job.startLabel`, task `title`, instance `label` | Home rows, overview rows, nav place, spawn/close buttons when authored |
| Group `startLabel`, `closeLabel` | Overview spawn link, close buttons, task post-commit chrome |
| Role ids in `roles[]` | Nav badges, handoff **parameter** values (banner template is SPA §a) |
| Commit/recap `links[].label`, `httpErrors[]` | `HttpLinkStrip`, toasts indirectly |
| HTTP error **response bodies** | `living.error`, `commitError`, task load `error`, recap errors |
| QR resolve `error.message` (422) | Toast (often Italian in feragame tests) |
| Task form bundle `schema` / `uiSchema` titles, enum labels | SJSF `@sjsf/form` render path |
| Recap `facts` keys/values | `TaskRecapPanel` dl grid |
| `chromeDisplay` bag (titles, rows, sections) | `data-report-field`, readonly chrome |
| Map-table row keys (`<code>{rowKey}</code>`) | Domain ids |
| `taskId` / `jobId` fallbacks | When Config title missing (nav/overview) |

**Boundary note:** SPA fallbacks `'Spawn'`, `'Close'`, `'Start'`, `'Home'`, `'Job instance'` are **SPA-owned** (§a) even though they mirror Config semantics.

---

## (c) Test-only strings

Files matching `*.test.ts`, `line-smoke.test.ts`, fixture Worklists (e.g. `'Nuovo camion'`, `'Registrazione'`, `'Più destinazioni possibili'`), and vitest expectations on English/Italian tags — **not** product catalogs.

Also: `load-feragame-config.ts` / YAML probes — test harness only.

---

## Estimated Paraglide key count (category **a** only)

| Bucket | Approx. keys |
|--------|----------------|
| Nav + shell (home sections, empty states, progress aria) | 12–14 |
| Router task/overview (load, commit, close dialog, post-commit actions) | 14–16 |
| Scan dialog | 6–7 |
| Status tags (5 IT → migrate to keys) | 5 |
| Handoff templates (3) | 3 |
| Config-label fallbacks (Spawn/Close/Start/Home/Job instance) | 5 |
| QR + HTTP transport fallbacks (shared pattern + status param) | 6–8 |
| Form chrome (validation header, module missing, data-report, derived, map defaults, checkbox default, chrome/jsonata errors) | 18–22 |
| Shared actions (Cancel, Continue, …) | 4–6 |
| Toast dismiss, recap loading/errors | 3–4 |

**Total estimate: ~70–85 message keys** (≈ **75** mid-point if one parameterized template replaces six `living … failed ({status})` variants).

Does **not** include: SJSF/AJV field copy, Config task titles, server error bodies, or link labels.

---

## Top 10 highest-traffic string clusters

1. **Load / wait states** — `Loading…`, `Loading recap…`, `Redirecting…`, `Committing…`, `Not loaded yet.` (home, task, overview, recap).
2. **Commit / close failure chrome** — `Commit denied` / `Close denied` prefixes plus server `commitError` body.
3. **Home worklist** — section titles `Available` / `Can start`, empty lists, progress `aria-label`.
4. **Global nav** — `Back`, `Scan`, `Attesta`, composed place line (Config + `Home` fallback).
5. **Post-submit task strip** — `Continue`, `Start another`, close confirm copy, `Cancel`.
6. **Overview group actions + dialog** — Config spawn/close labels with English fallbacks and shared confirm description.
7. **Task status tags** — five Italian labels (every overview/home status chip).
8. **Scan flow** — dialog title/body, camera hints, manual paste, `Go`, plus QR toasts (`Empty scan`, resolve failures).
9. **Role handoff banner** — three English sentence templates on overview.
10. **Living form adjuncts** — `Modulo non disponibile`, validation summary, data-report empty/`anche:`, derived override/restore, map `Id`/`Value` defaults.

---

## Files with no user-facing literals (reference)

Pure ports/types/routing logic (non-exhaustive): `types.ts`, `drafts.ts`, `worklist-nav.ts` (logic only), `navigate-target.ts`, `qr-resolve-apply.ts`, `session.svelte.ts` (errors passthrough), `living-app.svelte.ts`, most `http-*-port.ts` beyond fallback strings listed above.
