# Rework delta (MAIN) vs i18n worktree — strings needing Paraglide / Config

**Generated:** 2026-11-04  
**MAIN:** `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta` — `feat/attesta-rework` @ `0e691794be37472463e2d427fc5783ba3a17a940`  
**WT:** `/Users/giovanniabbatepaolo/.cursor/worktrees/i18n-d9256456/attesta-714729698316` — `feat/i18n` @ `e881d719b9358e9faf403539ff48d7dd7ae079c8`  
**Audit-only:** no app source edits.

---

## Git relationship

| Item | Value |
| --- | --- |
| **Merge-base** | `e881d719b9358e9faf403539ff48d7dd7ae079c8` (= WT `HEAD`) |
| **Interpretation** | WT has not diverged; MAIN is **3 commits ahead** with no WT-only commits in scope. |
| **MAIN-only commits (all repo)** | `af70822`, `c6e481b`, `0e69179` |
| **MAIN-only commits (scoped paths)** | Same 3 (all touch living home/nav/drafts chrome) |

### Commits on MAIN not in WT

1. `af70822` — feat(living): home In progress band and small operator UI fixes  
2. `c6e481b` — feat(living): job titles, accent colors, and task topbar chrome  
3. `0e69179` — feat(living): persist dirty Drafts with Bozza marks  

### Files changed on MAIN since merge-base (scoped)

| Path |
| --- |
| `app/src/lib/living/DraftBozzaMark.svelte` *(new)* |
| `app/src/lib/living/LivingShell.svelte` |
| `app/src/lib/living/OverviewRowChrome.svelte` |
| `app/src/lib/living/TaskForm.svelte` |
| `app/src/lib/living/WorklistJobLine.svelte` *(new)* |
| `app/src/lib/living/WorklistRow.svelte` |
| `app/src/lib/living/draft-storage.test.ts` |
| `app/src/lib/living/draft-storage.ts` |
| `app/src/lib/living/drafts.ts` |
| `app/src/lib/living/living-app.svelte.ts` |
| `app/src/lib/living/nav-place.test.ts` |
| `app/src/lib/living/nav-place.ts` |
| `app/src/lib/living/session.svelte.ts` |
| `app/src/lib/living/session.test.ts` |
| `app/src/lib/living/types.ts` |
| `app/src/lib/living/worklist-home.test.ts` |
| `app/src/lib/living/worklist-home.ts` |
| `app/src/lib/living/worklist-overview.test.ts` |
| `app/src/lib/living/worklist-overview.ts` |
| `app/src/lib/router/layout.svelte` |
| `app/src/lib/router/overview.svelte` |
| `server/internal/living/fixtures/feragame-config.yaml` |

**Not changed on MAIN in this delta:** `app/messages/*`, `app/src/routes/*` (beyond living/router above).

---

## Executive counts (this delta only)

| Metric | Count |
| ---: | ---: |
| **(a) New Paraglide candidates** | **3** |
| **(b) Config label Italian gaps (new on MAIN)** | **0** |

Suggested Paraglide keys (not yet in WT `app/messages/en.json` / `it.json`):

| Proposed key | EN (catalog) | IT (catalog) | Source |
| --- | --- | --- | --- |
| `in_progress` | In progress | In corso | `LivingShell.svelte` section `<h2>` |
| `no_jobs_in_progress` | No jobs in progress. | Nessun lavoro in corso. | `LivingShell.svelte` empty row |
| `draft_bozza` | Draft | Bozza | `DraftBozzaMark.svelte` |

---

## (a) Must Paraglide — **new** user-facing copy on MAIN

Strings **introduced or newly surfaced** in the rework delta that are **absent** from WT Paraglide catalogs.

### 1. Home “In progress” band

**File:** `app/src/lib/living/LivingShell.svelte`

| String | Lang | UI role |
| --- | --- | --- |
| `In progress` | EN | Section heading (`data-testid="worklist-in-progress"`) |
| `No jobs in progress.` | EN | Empty-state copy (`data-testid="in-progress-empty"`) |

WT already has parallel keys for **Available** / **Can start** bands (`available`, `can_start`, `no_available_tasks`, `no_jobs_to_start`) but **not** for this new third band.

### 2. Draft dirty mark “Bozza”

**File:** `app/src/lib/living/DraftBozzaMark.svelte` *(new component)*

| String | Lang | UI role |
| --- | --- | --- |
| `Bozza` | IT | Visible mark on home available rows, overview rows (`data-testid="draft-bozza-mark"`) |

Align with existing mixed-locale pattern (`taskStatusLabel` Italian in TS, English elsewhere). For full `it` locale, treat as SPA chrome: one Paraglide key with IT **Bozza** and EN **Draft** (or product term TBD).

---

## (b) Config label Italian — feragame delta on MAIN

**File:** `server/internal/living/fixtures/feragame-config.yaml`

| Change | Assessment |
| --- | --- |
| `steps.intake.title: Ingresso` | Italian — **no gap** |
| `steps.intake.color: '#ea580c'` | Not display copy |
| `steps.line.title: Lavorazione` | Italian — **no gap** |
| `steps.line.color: '#0284c7'` | Not display copy |
| `steps.line.label` expr: `$.date & ' Turno: ' & $lookup(...)` | Italian **Turno:** — format tweak only; **no new English** |

**Config label gaps introduced by this delta:** **0**  
(Pre-existing Feragame English/Match % / “Worklist” items remain per `05-config-labels-italian-audit.md`; unchanged by rework.)

New Config **job titles** (`Ingresso`, `Lavorazione`) flow to SPA via Worklist `job.title` → home rows / `WorklistJobLine` / `nav-place` `jobTitleLine` — **display pass-through**, not Paraglide.

---

## (c) Already covered / ignore (rework delta)

| Area | Why ignore for *new* i18n work |
| --- | --- |
| **`WorklistJobLine.svelte`** `aria-label="{done} of {total} tasks done"` | Same English as pre-rework `LivingShell`; WT catalog has `tasks_done_of_total`. Refactor only — wire Paraglide when merging rework into i18n branch, not a new string. |
| **`LivingShell` / home row layout** | Task title, instance label, `startLabel`, progress counts — **Config / Worklist projection** (including new `jobColor` border). |
| **`nav-place.ts` + `layout.svelte`** | Swaps primary vs muted lines (`primaryLabel` = task title; `jobTitleLine` = `Job.title · instance`). Copy is Config/instance labels; fallbacks `Home` / `Job instance` already in WT (`fallback_home`, `fallback_job_instance`). |
| **`layout.svelte` role chips** | Raw role slugs from domain — identifiers per `04-config-vs-spa-copy.md`. |
| **`overview.svelte` draft / spawn UX** | Uses existing Config `startLabel`, task titles; no new literals. |
| **`OverviewRowChrome.svelte`** | ChevronRight vs ChevronDown — no text. |
| **`TaskForm.svelte`, draft-storage, session** | Behavior/tests only; no new operator copy. |
| **`worklist-home.ts` / `worklist-overview.ts`** | Logic for in-progress rows and provisional spawn row; no new UI strings. |

**Note:** MAIN rework still **hardcodes** many strings WT already catalogued (`Available`, `Can start`, `Loading…`, `Scan`, etc.). Those are **pre-existing debt** on MAIN, not introduced by this delta — merge i18n wiring separately.

---

## Merge guidance (when integrating rework → feat/i18n)

1. Add the **3** catalog keys above; replace literals in `LivingShell.svelte` and `DraftBozzaMark.svelte`.
2. Port structural changes (Draft persistence, `WorklistJobLine`, in-progress band, nav chrome order, feragame `title`/`color`) without re-introducing English section headers.
3. Feragame YAML delta on MAIN is already Italian-friendly; no Config label patch required for rework-only fields.

---

## Return summary

**Report path:**  
`/Users/giovanniabbatepaolo/.cursor/worktrees/i18n-d9256456/attesta-714729698316/.scratch/app-i18n/06-rework-delta-needs-i18n.md`

| | Count |
| --- | ---: |
| New Paraglide candidates | **3** |
| Config label gaps (delta) | **0** |
