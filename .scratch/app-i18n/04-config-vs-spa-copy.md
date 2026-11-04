# Living / Worklist: Org Config vs SPA copy boundary

Research for **full Italian i18n of `/app`** (Paraglide). Sources: `server/internal/living/fixtures/feragame-config.yaml`, Go projection (`worklist.go`, `form_bundle.go`, `qr_resolve.go`, `http.go`, `commit.go`, `lifecycle.go`), SPA (`app/src/lib/living/*`, `app/src/lib/router/*`), `.scratch/living-container-reg/NOTES-continuation-row-mark.md`, `.scratch/living-pallet-weigh/LOCKED.md`.

---

## 1. Boundary: Config / server projection vs SPA chrome

| Layer | Who owns user-facing copy | Locale today (Feragame) | i18n project treatment |
| --- | --- | --- | --- |
| **Org Config YAML** | Plant / org authors | Italian for demo plant | **Out of Paraglide.** Treat as operational content in the active Config version. Plants localize by authoring (or future per-org Config versions), not by SPA catalogs. |
| **Worklist wire (`ProjectWorklist`)** | Server projects Config into JSON; SPA only displays | Italian where Config defines it | **Display-only pass-through.** No SPA translation of `worklist.jobs[].title`, `startLabel`, `tasks[].title`, `groups.*.startLabel` / `closeLabel`, instance `label`. |
| **Task form bundle** | Server builds from Config (`schema` / `uiSchema` / `chrome` / `derived`) | Italian field & section titles, enumNames, dataReport row labels in JSONata | **SJSF renders Config strings as-is.** Paraglide applies to theme chrome (submit, add/remove, generic widgets), not to replacing `schema.title`. |
| **QR resolve (422 body)** | Config `qr.rules[].messages` evaluated in Go (`ResolveQR`) | Italian in fixture (`onZero` / `onMany` / `onIneligible`) | **Config-owned.** SPA toasts `error.message` verbatim. |
| **Commit / Close denial body** | Go `CommitError.Message` (+ domain/group errors surfaced as text) | English engine messages | **Server-owned English** today; SPA prefixes + shows raw body (`Commit denied:` / `Close denied:`). Not Config. |
| **Load / QR HTTP failures** | Go `http.Error` plain text or SPA port fallbacks | English | **Platform / transport**; outside plant Config. |
| **Living SPA shell** | Svelte routes + living components | Mixed English UI + some hardcoded Italian helpers | **Must Paraglide** (`m.*`). |
| **Role chips (`nav-roles`)** | Config role slugs on the wire | English ids (`administration`, `logistic`, …) | **Identifiers**, not product copy unless plants add display names later (would be Config or a mapping layer). |

### Org Config (leave alone / plant responsibility)

Authored in YAML (Feragame fixture is the reference plant):

- **Job / group chrome:** `startLabel`, `closeLabel` (e.g. `Nuovo camion`, `Nuovo lotto`, `Chiudi scarico`, `Nuova pesata`).
- **Task titles on Worklist:** leaf `schema.title` (and group nodes without title fall back to **task id** in Go — see fallbacks below). Example: `Apri turno`, `Pesa bancale`, `Registrazione contenitori`.
- **Form labels:** JSON Schema `properties.*.title`, root `schema.title`, `uiSchema` `enumNames`, chrome `title` (e.g. `Progressivo cliente`, `Continua` on `presenceMark` hidden fields — per continuation-row note: *title in Config uiSchema, not hardcoded Italian in theme*).
- **Catalogs / enums:** `catalogs.*`, CER enumNames, material array `label`s.
- **Live / static reports:** `chrome` / `dataReport` JSONata row `label` strings (Italian in fixture).
- **QR:** `qr.rules[].messages.*` (Italian in fixture; one message still says “Worklist” as product term).
- **Optional `jobs.*.title`:** human Job title on wire when set; Feragame intake/line rely on `label` exprs for instance labels, not necessarily `job.title`.

Go **does not translate** these; it copies or evaluates them onto the wire (`taskTitle()` reads `schema.title`, else task id).

### SPA chrome (must Paraglide)

Generic executor UI — must not embed plant Italian or English literals long term:

| Area | Examples (current) |
| --- | --- |
| **Home (`LivingShell.svelte`)** | `Available`, `Can start`, `Loading…`, `No available Tasks.`, `No Jobs to start.`, `Commit denied:`, `Committing…`, progress `aria-label` (`N of M tasks done`) |
| **Nav (`layout.svelte`, `nav-place.ts`)** | `Home`, `Back`, `Scan`, `Job instance` fallback |
| **Scan (`ScanDialog.svelte`)** | `Scan`, camera hints, placeholder `Paste entityId or URL`, `Go` |
| **Overview (`overview.svelte`)** | `Loading…`, `Close denied:`, close confirm (`Cancel`, description *This finishes the group…*), `Close` fallback in dialog |
| **Task route (`task.svelte`)** | `Commit denied` / `Close denied`, `Redirecting…`, `Loading…`, `Start another`, `Continue`, close confirm copy |
| **Helpers (`worklist-overview.ts`, `worklist-home.ts`)** | `formatHandoffCopy` (English), `taskStatusLabel` (**Italian hardcoded**: Bloccato, Per te, …), fallbacks `Spawn` / `Close` / `Start` |
| **Task form shell (`TaskForm.svelte`)** | `Modulo non disponibile`; client validation wrapper (`formatSubmitValidationError` — Italian default, AJV messages often English) |
| **Ports / constants** | `QR_RESOLVE_FAILED`, `living domain load failed`, `living commit failed`, `living qr-resolve failed` |
| **Misc** | `WorklistRow` `aria-label="Available"`; shadcn dialog `Close` sr-only where shown |

Config-sourced strings **shown** in SPA (not translated by SPA): row titles, spawn/close button labels when Config set, nav `primaryLabel` when built from Worklist task title / instance label, toast text from QR 422.

---

## 2. Server error messages & QR resolve — locale ownership

### QR resolve (`POST /api/living/qr-resolve`)

| Outcome | Message source | Locale owner |
| --- | --- | --- |
| **422 JSON `error.message`** (zero / many / ineligible) | Config `messages.onZero` / `onMany` / `onIneligible`, else Go fallback (`No destination`, `Multiple destinations`, `Ineligible`) | **Plant Config** when authored; **Go English fallback** if omitted |
| **422** `no-rule`, `seed` | Go: `No QR rule configured`, `No QR rule for this code`, `QR seed not evaluable` | **Platform (English)** |
| **Non-422 HTTP** | Go: `invalid json`, `active config not found`, `failed to resolve qr`, `organization required`, …; SPA: `QrResolveHttpError` / `QR_RESOLVE_FAILED` | **Platform (English)** |

SPA path: `runQrScan` → toast shows server `error.message` or port/constant English on network failure.

### Commit / Close (`POST /api/living/commit`, close endpoint)

| Outcome | Message source | Locale owner |
| --- | --- | --- |
| **4xx body text** | `CommitError.Message` (prep/mutate/close), e.g. `value required`, `cannot close group "lots" on intake/T1`, `organization required`, parse/unlock errors | **Platform Go (English)** — not Config-authored today |
| **401/403** | Often bare `forbidden` | **Platform (English)** |
| **5xx / generic** | `commit failed`, `failed to commit` | **Platform (English)** |
| **Success `httpErrors[]`** | Sidecar / `onSubmit.http` failures (strings on wire) | **Whatever upstream returns** (often English); not Paraglide |

SPA: prefixes **English** (`Commit denied:` / `Close denied:`) + raw server text in `living.commitError`.

### Load (`GET /api/living/domain`)

| Outcome | Locale owner |
| --- | --- |
| Non-OK response body or `living domain load failed (${status})` | **Platform (English)** |

---

## 3. Mixed English fallbacks in SPA when Config omits labels

These appear **even when Feragame Config is Italian**, if a label is missing or a code path uses shell copy:

| Location | Fallback / shell copy | When it surfaces |
| --- | --- | --- |
| `worklist-home.ts` | `'Start'` | `job.startLabel` empty |
| `worklist-overview.ts` | `'Spawn'`, `'Close'` | group `startLabel` / `closeLabel` empty (else group title or task id) |
| `overview.svelte` / `task.svelte` | `'Close'` | close confirm when `closeLabel` missing |
| `nav-place.ts` | `'Home'`, `'Job instance'` | home nav; overview before instance label |
| `formatHandoffCopy` | English sentences with role slugs | overview handoff banner |
| `taskStatusLabel` | Italian in **source** (not Config) | every status tag — inconsistent with English shell |
| Group **task row title** | task id (e.g. `lots`, `palletWeight`) | group nodes without `schema.title` in Config |
| Nested child in tests | `"Register"` | synthetic fixtures; real plant uses Italian schema titles on leaves |
| AJV / `@sjsf` validation | English (typical) | submit errors unless catalog configured (see `02-sjsf-i18n.md`) |
| Feragame QR `onMany` | contains word **Worklist** | intentional product term in Italian sentence |

Operators on Feragame mostly see Italian for **actions** (Config `startLabel` / `closeLabel`) and **task/form labels** (schema/uiSchema). English leaks concentrate in **home section headers**, **nav**, **scan dialog**, **load/commit error framing**, **handoff**, **post-submit loop** (`Continue`, `Start another`), and **engine errors**.

---

## 4. What “FULL i18n for `/app`” means when Config is already Italian

**Recommended scope (this project):**

1. **Paraglide all SPA chrome** under `app/src/lib/living/` and `app/src/lib/router/` (including migrating hardcoded Italian in `taskStatusLabel` and `formatSubmitValidationError` defaults into `it.json` / `en.json` so `en` base locale stays coherent).
2. **SJSF/AJV layer** for generic validation/widget strings (companion note `02-sjsf-i18n.md`) — still **not** replacing Config field titles.
3. **Do not** move Feragame YAML titles or QR messages into Paraglide for v1; document that **plant locale = Config content**.
4. **Server `CommitError` / HTTP / QR engine fallbacks** stay **English** unless a separate platform i18n effort adds localized error catalogs or Config-driven denial messages — **out of “SPA full Italian”** but worth listing in operator-facing gap analysis.

**Not required for “Italian `/app`” with Italian Feragame Config:**

- Dual-locale Org Config (e.g. `title_it` / `title_en`) — only needed for **multi-language plants** or **same org switching UI language independently of Config**. That is a **Config product** decision, not Paraglide.

**Dual track summary:**

| Track | Tooling | Delivers |
| --- | --- | --- |
| **A — SPA shell** | Paraglide | Italian (and English) for generic Living UI regardless of plant |
| **B — Plant copy** | Org Config YAML | Italian task/form/QR/button labels for Feragame today |
| **C — Engine errors** | Go (+ optional future design) | English until localized server messages exist |

For Feragame-only Italian operators, **Track A closes most visible gaps**; Track B is already largely done; Track C remains English for denial/toast edge cases when Config messages are absent or Commit fails.

---

## Scope recommendation (one paragraph)

Treat **Org Config (YAML) and server-projected Worklist/form/QR toast text** as **plant-authored locale**: Feragame’s Italian `startLabel`/`closeLabel`, schema/uiSchema/chrome titles, catalogs, and `qr.messages` should **not** be duplicated into Paraglide—only maintained in Config. The **FULL `/app` i18n project** should **Paraglide every generic Living SPA string** (home, nav, scan, overview/task chrome, handoff, status tags, commit/close prefixes, loading/empty states, port fallbacks, and SJSF generic validation chrome) while **passing through Config strings unchanged**; **server Commit/Close/Load/QR infrastructure errors** remain **English platform copy** unless a follow-up adds localized Go messages or Config-owned denial text. With Config already Italian, success means **eliminating English (and scattered hardcoded Italian) from the executor UI**, not re-translating the plant fixture—**dual-locale Config is a later, org-authoring concern**, not a prerequisite for Italian `/app` against Feragame.
