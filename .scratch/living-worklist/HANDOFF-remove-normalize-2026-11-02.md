# Handoff: remove SPA cell-map normalize + project concrete map properties

**Date:** 2026-11-02  
**Repo:** `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta`  
**Worktree:** `/Users/giovanniabbatepaolo/.cursor/worktrees/fix-remove-normalize-36fadf53/attesta-714729698316`  
**Branch:** `fix/remove-normalize` (from `1b64fcc` — mapTable unify already merged)  
**Audience:** coordinator agent (one or few peels; conventional commits; do **not** push unless asked). Leave dirty `.scratch/**` unstaged unless asked.

**Durable twin (optional):** copy into main checkout `.scratch/living-worklist/` uncommitted.

---

## Recap (settled — do not re-litigate)

### Problem chain
1. Dynamic maps authored as `additionalProperties` + Go only stamps `propertyNames` / `required`.
2. Rows did not reliably appear from schema alone → densified `initialData` (object maps: every id → template with **first enum** even without authored `default`).
3. SPA `normalizeFormCellMaps` / Ajv wrapper strip `""` / undefined before validate+seal — a form-junk sanitizer twin of Go prep.
4. Plant **progress** had leaked into form infrastructure; already cleaned by mapTable unify (`1b64fcc`). GLOSSARY progress = entity fact only.

### Goals
- **Minimum SPA custom code.** Ajv validates Draft as-is. Go owns projection + Commit prep.
- **No SPA normalize** (`form-cell-map.ts` / validator wrap / `sealSubmission` identity).
- **Seed only with a good reason:** Domain facts and authored `default` — never invent first-enum / empty junk.
- **Quiet cell default:** `undefined` / omit = untouched; `null` = explicit clear where schema allows; **never `""` as law**.

### Stock select (verified in `@sjsf/form`)
- Clearable placeholder uses DOM `EMPTY_VALUE = ""`, but `singleOption.toValue("")` → **`undefined`** (not form-level `""`).
- Do **not** plan a “paint select to stop emitting `""`” peel for stock select.
- Stock clearable → omit/`undefined`, **not** `null`. Explicit clear→`null` is a separate product affordance if still required; Go prep already keeps `null`. Prefer omit for untouched; keep `null` only when operator means clear and schema allows it.

### Projection law (backend)
Authored YAML stays compact (`additionalProperties` = **template**).

On Load/bundle project, for each dynamic map property with resolved read ids, **materialize concrete schema**:

```text
properties: { [id]: clone(template) for each read id }
required: ids                    # only if full-map (map property in schema.required)
additionalProperties: false
# drop propertyNames (ids are real properties now)
```

Same pattern as Derived map-star → concrete keys.

| Template (`additionalProperties`) | Each concrete property | UI |
| --- | --- | --- |
| scalar / anyOf null\|enum | that schema | mapTable one value column; sjsf per cell |
| object | that object schema | mapTable property columns; sjsf per field |

- Rows come from **schema.properties**, not densified formData.
- Unknown ids rejected by Ajv (`additionalProperties: false`).
- Partial map: concrete properties, **no** forcing every id in `required` → omit still = untouched.
- Full map: `required: all ids` → keys must exist; values still follow cell/row schema (do not invent enums).

### Seed law (backend `seedInitialData`)
| Case | Seed |
| --- | --- |
| Cell map | Only ids with existing Domain enum facts (rehydrate). No `""`. |
| Object map | **Stop** `seedRowFromSchema` first-enum densify. Prefer empty `initialData` for the map, **or** only Domain-backed / authored-`default` values. No empty shells unless a test proves full-map Ajv needs key presence — then `id → {}` only, never invented enums. |
| Root fields | Authored `default` / `$today` via existing SPA merger or Go defaults — unchanged intent. |

### SPA after peel
- Delete or gut `form-cell-map.ts`; remove imports from `ajv-form-validator.ts` and `task-form-bundle.ts` (`sealSubmission` = identity or drop).
- Update cascade #6 / submit-validation tests: assert **no** normalize; Ajv fails on `""` if somehow present; happy path = omit/`undefined`/`null`+enum only.
- `map-table-field.svelte`: rows from object context over **concrete properties** (may simplify vs additionalProperties detection). Keep one `mapTable` component.
- Do not reintroduce `rowTable` / `progressEnums` / `TableFormShell`.

### Go prep
Keep `filterFilledCells` / object-row filters as Commit authority (omit untouched, keep `null` clear, enum check). Rename vocabulary already done on unify tip. No SPA twin.

---

## Pointers

| Area | Path |
| --- | --- |
| Projection today | `server/internal/living/form_project.go` (`propertyNames` + `required`) |
| Seed today (bad densify) | `server/internal/living/form_bundle.go` `seedInitialData` / `seedRowFromSchema` |
| Prep | `server/internal/living/prep.go`, `rows.go` |
| SPA normalize | `app/src/lib/living/form/form-cell-map.ts`, `ajv-form-validator.ts`, `task-form-bundle.ts` |
| mapTable | `app/src/lib/living/form/map-table-field.svelte` |
| Fixture | `server/internal/living/fixtures/feragame-config.yaml` (lineLoad cell; palletWeight object) |
| Tests | `form_bundle_test.go` (weights densify assert), `progress-cascade6-probe.test.ts`, living unit + `go test ./internal/living/...` |
| Glossary | `GLOSSARY.md` — Domain reads / dynamicProperties / cell vs object map |
| Prior peel | tip `1b64fcc` mapTable unify; prior handoff `HANDOFF-map-table-unify-2026-11-02.md` |

**Svelte:** MCP autofixer / svelte-file-editor when editing `.svelte`.

---

## Suggested peels (coordinator)

Prefer 1–3 conventional commits; or one peel.

### 1) Go: materialize concrete map properties
- In `projectTaskForm` (or helper): for each dynamic map, clone `additionalProperties` template into `properties[id]` for each resolved id; set `additionalProperties: false`; set `required` to ids iff full-map; remove `propertyNames`.
- Paint uiSchema per concrete id as needed (clone additionalProperties ui).
- Fix tests that assume `propertyNames` / densified weights count.
- Assert lineLoad + palletWeight bundles: concrete keys, no `propertyNames`, `additionalProperties: false`.

### 2) Go: honest seed
- Remove first-enum densify for object maps.
- Keep cell-map Domain rehydrate.
- Adjust `TestLoadJSONTaskFormBundlePalletWeightRows` (and kin) — do not require `len(weights)==len(rowIds)` unless empty shells are proven necessary; prefer asserting schema.properties keys instead.

### 3) SPA: remove normalize
- Delete `normalizeFormCellMaps` path; validator = stock Ajv; seal = pass-through.
- Rewrite cascade #6 tests around omit/`undefined`/`null` without sanitizer.
- Confirm mapTable still lists all projected ids from schema with empty Draft.
- Manual if stack up: lineLoad partial Commit + clear; palletWeight open with empty cells (no pre-selected euro/NIR).

### 4) Glossary
- Document: dynamic map projection **materializes** read ids into `properties`; authored `additionalProperties` is template only.
- `_Avoid_`: SPA cell-map normalize; densify first-enum without `default`; form-level `""` as untouched sentinel.

---

## Out of scope
- FakeHttp / HttpPort
- Completo / multi-Commit redesign
- Renaming Feragame entity field `progress`
- Ticket 13 package rename
- Re-litigating mapTable unify
- Push / force
- Committing unrelated `.scratch/**`

---

## Done when
- [ ] Projected map schemas use concrete `properties[id]` + `additionalProperties: false` (no `propertyNames` for those maps)
- [ ] No SPA `form-cell-map` normalize / Ajv wrap / seal strip
- [ ] Object-map seed does not invent first enum without authored `default`
- [ ] Cell-map seed still rehydrates Domain facts only
- [ ] `go test ./internal/living/...` and `bun run test:unit -- --run src/lib/living/` green
- [ ] GLOSSARY updated
- [ ] lineLoad + palletWeight smoke reasonable (schema-driven rows; quiet defaults)

## Process
- Match `AGENTS.md`; comments = why only; conventional commits.
- Merge-back: `/apply-worktree` from parent chat on `feat/attesta-rework` (or instructed base).
- Cleanup: `/delete-worktree`.
- Review: code-review skill (Standards ∥ Spec) if multi-commit — not Bugbot unless asked.
