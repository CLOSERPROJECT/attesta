# Org Config labels — Italian audit (audit-only)

**Worktree:** `i18n-d9256456/attesta-714729698316`  
**Decision:** All Org Config **display labels** must be Italian (`schema.title`, property `title`, `enumNames`, `startLabel`, `closeLabel`, `uiSchema` option/column titles, `chrome.title`, catalog `label`, `dataReport` row `"label"` strings in JSONata, `onSubmit.http` action `label`, QR `messages.*` shown in UI).  
**Out of scope:** JSONata logic, enum **keys**, tone tokens (`green` / `warning` / `danger`), role slugs, entity field names, HTTP body payloads.

**No YAML edits in this pass.**

---

## Files in scope

| Path | Serves `/app`? | Notes |
| --- | --- | --- |
| `server/internal/living/fixtures/feragame-config.yaml` | **Yes** | Embedded in Go (`LoadFeragameConfigFixture`); seeded as **Active** Org Config; SPA tests import same file via `$living-fixtures`. |
| `server/internal/living/fixtures/parked-loaded-extracted.yaml` | **No (live)** | Parked task chrome for `data_report` tests only; not on Feragame worklist. Audited for parity. |
| Other repo YAML (`server/config/*`, `cerbos/*`, …) | No | Not Living Org Config; not projected into `/app` living UI. |

Inline `OrgConfig` snippets in Go tests (e.g. `startLabel: Start`) are harness-only and **excluded** from this audit.

---

## Summary

| File | Non-Italian label count (strict) | Already clean? |
| --- | ---: | --- |
| `server/internal/living/fixtures/feragame-config.yaml` | **2** | **No** |
| `server/internal/living/fixtures/parked-loaded-extracted.yaml` | **0** | **Yes** |

**Feragame is not clean:** two authored strings are English or embed an English product term in operator-facing copy.

---

## Feragame — inventory checked

Rough coverage (all Italian unless noted):

- **Catalogs:** `materials[].label`, client map `Nord` / `Sud`, turni display via `enumNames` and `$catalogs.turni`.
- **Job / group chrome:** `startLabel`, `closeLabel` (intake, lots, line, palletWeight, containerReg).
- **Schema / uiSchema:** property and root `title`, `enumNames` (turni, clienti, CER, tara, avanzamento), `lineLoad` `columns` values (`Bancale`, `Avanzamento`), `continuesContainerId` chrome `title: Continua`.
- **Chrome:** readonly / dataReport `title`; HTTP sidecar `label` (`Stampa etichette`, `Scarica CSV registrazione`).
- **QR:** `messages.onZero`, `onIneligible` (Italian prose).

---

## Feragame — non-Italian labels (patch list)

| # | YAML path | Current | Proposed Italian |
| --- | --- | --- | --- |
| 1 | `steps.line.substeps[weightCheck path].chrome.registeredData.expr` → dataReport row | `"Match %"` | `"Corrispondenza %"` or `"Allineamento %"` |
| 2 | `qr.rules[id=line-pallet-weigh].messages.onMany` | `Più destinazioni possibili: apri il turno giusto dal Worklist` | `Più destinazioni possibili: apri il turno giusto dalla lista lavori` (or keep product name **Lista lavori** if branding differs) |

**Path detail for #1:** under `steps.intake.substeps` → task `weightCheck` → `chrome.registeredData` → JSONata `rows` array, fourth row object `"label": "Match %"`.

**Path detail for #2:** `qr.rules[0].messages.onMany` (rule id `line-pallet-weigh`).

---

## Feragame — borderline (not counted in strict total)

Document for implementers; no change required unless policy says “no Latin/English tokens at all”:

| Location | String | Rationale |
| --- | --- | --- |
| `weigh` schema / uiSchema | `IR / NIR`, `enumNames: [NIR, IR]` | Industry abbreviations; not English prose. |
| `weigh` uiSchema tare `enumNames` | `euro (25 kg)`, … | “euro” = pallet type name; `kg` = unit symbol. |
| `register` http label | `Scarica CSV registrazione` | “CSV” is a universal format acronym. |
| `weightCheck` dataReport | `Σ netti lotti` | Greek sigma as math notation; rest Italian. |
| Catalog keys | `1st`, `2nd`, `NO`, `SU`, … | Keys for exprs; display uses Italian `enumNames` / catalog values. |

---

## Feragame — visible English without a label field (informational)

Worklist uses **task id** when a group node has no `schema.title` (Go `taskTitle()` fallback). Group ids shown to operators today:

- `lots`, `palletWeight`, `containerReg` (and leaf ids like `truckArrive`, `lineLoad` only where no title — most leaves have Italian `schema.title`).

These are **identifiers**, not mis-authored labels. Fixing Italian worklist titles would mean adding `schema.title` (or future job-level titles), not translating existing label keys.

---

## Parked `loadedExtracted` fixture

File: `server/internal/living/fixtures/parked-loaded-extracted.yaml`

- **Schema / chrome titles:** Italian (`Controllo caricato vs estratto`, `Risultati finali`, property titles).
- **dataReport JSONata row labels:** Italian (`Netto lotto`, `Σ bancali lordi`, `Totale estratto`, `Pannelli vs estratto`, …).

**Count: 0** non-Italian display labels.

Duplicate scratch copy at `.scratch/living-container-reg/parked-loaded-extracted.yaml` is documentation-only; canonical body is the Go-embedded fixture above.

---

## Method

Manual read of both living fixture YAML files plus targeted ripgrep for `title`, `startLabel`, `closeLabel`, `enumNames`, catalog `label`, QR `messages`, JSONata `"label":`, and `onSubmit.http` `label`. English tokens inside JSONata **logic** (e.g. `progress = 'done'`, `$tone := "green"`) were ignored per decision.
