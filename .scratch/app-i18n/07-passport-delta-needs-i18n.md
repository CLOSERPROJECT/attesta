# Passport delta (MAIN) vs i18n worktree — strings needing Paraglide / Config

**Generated:** 2026-11-04  
**MAIN tip:** `8a5d89ad151eb0a37c2949683cff87ba26271c56` (`feat(living): passport production`)  
**Prior WT base:** `0e691794be37472463e2d427fc5783ba3a17a940`  
**WT:** `/Users/giovanniabbatepaolo/.cursor/worktrees/i18n-d9256456/attesta-714729698316` — `feat/i18n` @ `0e69179` + uncommitted i18n WIP  
**Audit-only for this file:** scan before integrate; no app source edits in this step.

---

## Git relationship

| Item | Value |
| --- | --- |
| **Range** | `0e69179..8a5d89a` |
| **Interpretation** | MAIN is **2 commits ahead**; WT has no unique commits beyond shared base (WIP is uncommitted). |
| **MAIN-only commits** | `6839070`, `8a5d89a` |
| **MAIN-only (scoped living/router/routes/fixtures)** | `8a5d89a` only (`6839070` is Go `cover_gate_test.go`) |

### Commits on MAIN not in WT

1. `6839070` — test(living): raise coverage for cover gate  
2. `8a5d89a` — feat(living): passport production  

### Files changed on MAIN since base (scoped)

| Path |
| --- |
| `app/src/lib/living/LivingShell.svelte` |
| `app/src/lib/living/form/HttpLinkStrip.svelte` |
| `app/src/lib/living/session.svelte.ts` |
| `app/src/lib/living/session.test.ts` |
| `app/src/lib/living/worklist-home.test.ts` |
| `app/src/lib/living/worklist-home.ts` |
| `app/src/lib/living/worklist-nav.test.ts` |
| `app/src/lib/living/worklist-nav.ts` |
| `app/src/lib/living/worklist-overview.test.ts` |
| `app/src/lib/router/overview.svelte` |
| `app/src/lib/router/task.svelte` |
| `app/src/routes/[...path]/page.svelte.e2e.ts` |
| `server/internal/living/fixtures/feragame-config.yaml` |

**Not changed on MAIN in this delta:** `app/messages/*`.

---

## Executive counts (this delta only)

| Metric | Count |
| ---: | ---: |
| **(a) New Paraglide candidates** | **2** |
| **(b) Config label Italian gaps (new on MAIN)** | **0** |

Suggested Paraglide keys (not yet in WT `app/messages/en.json` / `it.json`):

| Proposed key | EN (catalog) | IT (catalog) | Source |
| --- | --- | --- | --- |
| `completed` | Completed | Completati | `LivingShell.svelte` section `<h2>` |
| `no_completed_jobs` | No completed jobs. | Nessun lavoro completato. | `LivingShell.svelte` empty row |

---

## (a) Must Paraglide — **new** user-facing copy on MAIN

### 1. Home “Completed” band

**File:** `app/src/lib/living/LivingShell.svelte`

| String | Lang | UI role |
| --- | --- | --- |
| `Completed` | EN | Section heading (`data-testid="worklist-completed"`) |
| `No completed jobs.` | EN | Empty-state copy (`data-testid="completed-empty"`) |

Parallel keys already exist for Available / Can start / In progress bands; this fourth band needs matching catalog entries.

---

## (b) Config label Italian — feragame delta on MAIN

**File:** `server/internal/living/fixtures/feragame-config.yaml`

| Change | Assessment |
| --- | --- |
| `passport.identity.productName: "Carico Feragame"` | Italian — **no gap** |
| Snapshot row labels (`Passaporto carico`, `Carico`, `Cliente`, `FIR`, …) | Italian — **no gap** |
| `steps.passport.title: Verifica dati e DPP` | Italian — **no gap** |
| `startLabel: Emetti DPP` | Italian — **no gap** |
| `label: 'Passaporto: ' & $.loadId` | Italian — **no gap** |
| Schema titles (`Seleziona carico`, `Carico`, `Emetti passaporto`, `Confermo i dati del passaporto`) | Italian — **no gap** |
| Domain field adds `fullyProcessed` / `locked` / `issued` | Field ids, not display labels |

**Config label gaps introduced by this delta:** **0**

New Config job title / startLabel flow to SPA via Worklist as pass-through (not Paraglide).

---

## (c) Already covered / ignore (passport delta)

| Area | Why ignore for *new* i18n work |
| --- | --- |
| **`HttpLinkStrip.svelte`** | Button chrome only; `item.label` is server/Config-authored. No new SPA literal. |
| **`worklist-home.ts` Completed helpers** | Projection logic; labels from Config / instance. |
| **`worklist-nav.ts` `commitHasLinks`** | Behavior rename; no UI strings. |
| **`session.svelte.ts` / `task.svelte` clearLinksStay** | Navigation hygiene; no copy. |
| **`overview.svelte` keep finished overview** | No new literals. |
| **e2e `Finished` expectation** | Stub Worklist instance `label: 'Finished'` — fixture data, not SPA chrome. |
| **Go `issuePassport` link label `Digital Product Passport`** | Surfaces in SPA strip but is **Go platform** copy (v1 English per locked decisions / `04-config-vs-spa-copy.md`). Not Paraglide; not Config YAML. |
| **`living_passport.html` / public DPP page** | Go templates outside `/app` SPA i18n scope. |

**Note:** MAIN still hardcodes many strings WT already catalogued (`Available`, `Completed` band peers, etc.). Pre-existing debt on MAIN — merge keeps WT `m.*` wiring.

---

## Merge guidance (when integrating passport → feat/i18n)

1. Add the **2** catalog keys above; replace literals in `LivingShell.svelte` Completed band.
2. Port structural changes (Completed band, stay-for-any-links, finished overview stay, feragame passport job + root `passport:`) without re-introducing English section headers.
3. Keep prior feragame Italian patches (`Corrispondenza %`, `lista lavori`) + new passport Config labels as authored (already Italian).
4. HttpLinkStrip conflict: keep MAIN button chrome + any WT wiring if present (labels remain pass-through).

---

## Return summary

**Report path:**  
`/Users/giovanniabbatepaolo/.cursor/worktrees/i18n-d9256456/attesta-714729698316/.scratch/app-i18n/07-passport-delta-needs-i18n.md`

| | Count |
| --- | ---: |
| New Paraglide candidates | **2** |
| Config label gaps (delta) | **0** |
