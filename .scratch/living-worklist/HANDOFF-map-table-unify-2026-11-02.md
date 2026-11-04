# Handoff: unify map table (schema-driven cells, no plant progress paint)

**Date:** 2026-11-02  
**Repo:** `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta`  
**Branch:** `feat/attesta-rework` (check tip; recent: `d4e5590` progressEnums on bundle — **to reverse/replace**)  
**Job:** implement the settled map-table cleanup. Prefer one peel (or 0→5 commits). Conventional commits. Do **not** push unless asked. Leave dirty `.scratch/**` unstaged unless asked.

**Durable twin (optional):** copy into `.scratch/living-worklist/` if the human wants it in-tree uncommitted.

---

## Problem

Feragame **progress** (`partial`/`done`) leaked into living form infrastructure:

- Bundle/API: `progressEnums`, `valueType: "progress"`, Go `*Progress*` helpers
- SPA: `row-table-field` owns a custom `<select>` + enums; `TableFormShell` + `surface: "row-table"` bypass ConfigForm
- Dual authored markers: `ui:field: rowTable` vs `mapTable` as if two products

That made the **table** own cell widgets. Enums/null/number belong on the **cell schema**, rendered by sjsf.

---

## Settled solution (do not re-litigate)

### Law

A dynamic map is always `Record<ID, T>` from projected `propertyNames` + `additionalProperties`:

| `additionalProperties` | Columns | Each cell |
| --- | --- | --- |
| **scalar** (string / number / …) | `[id, value]` (value header from ui:options.columns / schema) | one **sjsf** field for that cell schema |
| **`object`** | `[id, ...Object.keys(properties)]` | one **sjsf** field per property schema |

- Go compiles full projected JSON Schema + uiSchema (propertyNames, required, enums on cell schema, `ui:field`→`ui:components` paint).
- SPA table = layout + mount fields. **No** table-owned enum lists.
- GLOSSARY **progress** = plant **entity fact** only — not a form/map type.

### Component / authoring (proper cleanup — no aliases)

- **One** SPA field component for both shapes (extend today’s `map-table-field.svelte` pattern; it already does sjsf-per-cell for objects).
- **One** authored marker: `ui:field: mapTable` only.
- **Delete** `rowTable` / `rowTableField` / `row-table-field.svelte` — **no** registry alias, no passthrough rename shim.
- Fixture YAML: change lineLoad `ui:field: rowTable` → `mapTable`.
- Go paint registry: drop `rowTable` entry in `form_project.go`.
- SPA `AUTHORED_FIELD_REGISTRY` / theme: drop rowTable row.
- `dynamic-properties.ts` (test harness): only `mapTable`.

### Mount path

- Prefer **ConfigForm only** for map Tasks (same as object mapTables today).
- **Delete** `surface: "row-table"` fork, `TableFormShell`, `row-context.ts` if unused.
- Go `BuildTaskFormBundle`: stop setting `surface = "row-table"`; stop stamping `ProgressEnums` / `ValueType` / `stampRowTablePaint` (revert `d4e5590` paint API).
- TaskForm: remove row-table branch / shellEpoch special case; always live plan + ConfigForm when bundle mounts.

### Prep / naming (Go)

Keep Commit prep behavior; rename plant-leaky symbols when touching:

| Current | Target |
| --- | --- |
| `additionalPropertiesIsProgress` | detect **cell** (non-object) vs **object** map (name e.g. `additionalPropertiesIsObject` + invert, or `isCellMapAdditional`) |
| `filterFilledProgress` | `filterFilledCells` |
| `progressEnumsFromSchema` | `stringEnumsFromSchema` (internal to prep validation only — **not** on bundle wire) |
| SPA `omitUntouchedProgressCells` / `normalizeFormProgress` | cell-map omit helpers if still needed for `""`; prefer schema/widgets; no enum table paint |

---

## Pointers (read before coding)

| Area | Path |
| --- | --- |
| Object map (keep / extend) | `app/src/lib/living/form/map-table-field.svelte` |
| Scalar outlier (delete after merge) | `app/src/lib/living/form/row-table-field.svelte` |
| Shell to delete | `app/src/lib/living/form/TableFormShell.svelte`, `row-context.ts` |
| Mount fork | `app/src/lib/living/TaskForm.svelte` |
| Bundle wire | `app/src/lib/living/form/task-form-bundle.ts`, `server/internal/living/form_bundle.go` |
| Paint registry | `form_project.go` (`authoredFieldRegistry`), `authored-field-registry.ts`, `form-theme.ts` |
| Fixture | `server/internal/living/fixtures/feragame-config.yaml` (`lineLoad` ~389) |
| Prep | `server/internal/living/prep.go`, `rows.go` |
| Glossary | `GLOSSARY.md` — Domain reads / dynamicProperties / progress entity field |
| Prior audit | agent transcript FE/BE twins; tip had `d4e5590` enums-on-bundle |

**Svelte:** use Svelte MCP autofixer / svelte-file-editor skill when editing `.svelte`.

---

## Implementation steps (ordered)

### 1) Extend `mapTable` for scalar cells

- In `map-table-field.svelte` (or extract shared table module then single export): if `additionalProperties` is **not** object, columns = one value column; each cell = `getFieldComponent` for the **cell schema** (the additionalProperties schema itself), not a custom select.
- Object path: keep existing property-column + per-property fields.
- Id column title: existing `ui:options.columns.id`; value column: `columns.value` or schema title.
- Wire bindable value at `value[rowId]` (scalar) vs `value[rowId][col]` (object).

### 2) Authoring + registry cleanup (no aliases)

- YAML: `rowTable` → `mapTable` (feragame lineLoad; any other hits).
- Delete SPA `row-table-field.svelte`, theme/registry/`rowTable` types.
- Delete Go `rowTable` paint map entry; tests that paint `rowTable` → `mapTable`.
- `DYNAMIC_TABLE_FIELDS` / messages: `mapTable` only.

### 3) Drop row-table surface + bundle progress paint

- Remove `ProgressEnums`, `ValueType`, `stampRowTablePaint` from Go bundle.
- `surface` is only `"live"` (or drop unused row-table from TS union).
- Simplify `TaskForm.svelte`: no `TableFormShell`; delete shell + row-context.
- Update line-smoke / cascade tests that set `surface: 'row-table'` or `progressEnums`.

### 4) Go prep rename + tests

- Rename progress-shaped helpers to cell-map vocabulary; update call sites/tests.
- lineLoad / filterFilled still green; assert bundle has **no** `progressEnums`.

### 5) Glossary

- Document **cell map** vs **object map** under Domain reads / dynamicProperties.
- `_Avoid_`: form `progressEnums`, `valueType: "progress"`, table-owned enum paint, `ui:field: rowTable`.

### 6) Verify

```bash
cd server && go test ./internal/living/...
cd app && bun run test:unit -- --run src/lib/living/
```

`rg` product paths (exclude fixtures narrative / archival research): no `rowTable`, `row-table-field`, `TableFormShell`, `progressEnums`, `valueType: "progress"`, `surface: "row-table"`.

Manual if stack up: lineLoad (enum select from schema) + palletWeight / object mapTable.

---

## Out of scope

- FakeHttp / HttpPort replacement
- Seal omit grill beyond what’s required for sjsf cells / `""`
- Renaming Feragame Domain fact field `progress` in YAML
- Completo / multi-Commit product redesign
- Ticket 13 living package rename
- Committing unrelated dirty `.scratch/**`
- Push / force

---

## Process hints

- Match `AGENTS.md`: minimal diffs; comments = why only; conventional commits.
- Review: Matt Pocock **code-review** skill (Standards ∥ Spec) if multi-commit peel — not Bugbot unless asked.
- Human asked: **proper cleanup, no aliases or passthroughs** — do not leave `rowTable` → `mapTable` shim in registry.

---

## Done when

- [ ] One `mapTable` component renders scalar and object maps via sjsf cells
- [ ] Zero `rowTable` / TableFormShell / progressEnums bundle paint in product
- [ ] lineLoad authored as `mapTable`; mounts through ConfigForm like other Tasks
- [ ] Go tests + SPA living unit green
- [ ] GLOSSARY updated; plant **progress** remains entity-fact vocabulary only
