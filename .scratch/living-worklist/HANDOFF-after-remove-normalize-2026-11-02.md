# Handoff: after remove-normalize (cell maps + mapTable select law)

> **SUPERSEDED (2026-11-02 evening):** Human law is now **stock clearable only** — never unset Domain facts; no `enum: [null,…]`. Active brief: `BRIEF-stock-clearable-2026-11-02.md` / `COORDINATOR-stock-clearable-2026-11-02.md`. Ignore “Clearable cell maps (select)” enum-null section below.

**Date:** 2026-11-02  
**Repo:** `/Users/giovanniabbatepaolo/Documents/GitHub/forkbomb/attesta`  
**Worktree:** `/Users/giovanniabbatepaolo/.cursor/worktrees/fix-remove-normalize-36fadf53/attesta-714729698316`  
**Branch:** `fix/remove-normalize` @ `9b8046e` (from `1b64fcc` mapTable unify)  
**Prior handoff:** `HANDOFF-remove-normalize-2026-11-02.md` (goals largely landed)  
**Audience:** coordinator — conventional commits; **do not push** unless asked; leave dirty `.scratch/**` unstaged.  
**Policy:** **no patches** — schema/projection/bind law only. Prefer reactive classes over `$effect`.

**Durable twin (optional):** copy this file + `COORDINATOR-remove-normalize-2026-11-02.md` into main checkout `.scratch/living-worklist/` uncommitted.

**Dev stack (this worktree):** `task start` / `task dev` → http://localhost:10108/app/  
Demo login: `living@attesta.com` / `living@attesta.com`  
After config/Domain seed changes: `task seed -- --replace`.

---

## Recap — landed on this branch (`1b64fcc..9b8046e`)

| SHA | Summary |
| --- | --- |
| `502ced2` | Go: materialize dynamic maps → concrete `properties[id]`, `additionalProperties: false`, drop `propertyNames` |
| `3c58e3c` + `0c528d1` | Go: stop first-enum densify for object-map seed |
| `7959c5a` | SPA: delete `form-cell-map` normalize; stock Ajv; identity seal |
| `8fce806` | GLOSSARY: materialize / template |
| `9cca74f` + `17340f3` | Early mapTable bind patches (partially superseded) |
| `89a79e6` | Cell maps: `enum: [null, partial, done]` (not `anyOf`); enumNames |
| `e9d1992` | Projection: enum contains `null` ⇒ `clearable: false` |
| `ef41b5f` | Null option + omit placeholder both **–** (U+2013) |
| `9b8046e` | SeedDemo WipeDomain: Commit intake→palletWeight + line/shift `1st` so lineLoad is available |

Human smoke: Avanzamento select OK with – / Parziale / Fatto; demo seed lands on **Turno: 1st → lineLoad**.

---

## Settled law (do not re-litigate)

### Projection / seed
- Authored YAML keeps compact `additionalProperties` **template**.
- Load/bundle project: `properties[id] = clone(template)`, `additionalProperties: false`, drop `propertyNames`, `required: ids` iff full-map.
- Object-map seed: authored `default` only — never invent first-enum.
- Cell-map seed: Domain enum rehydrate only.
- Replace seed builds **lineLoad-ready** Domain via Committer (`demo_commits.go`), not EmptyDomain.

### Clearable cell maps (select)
- Schema: **`enum: [null, …]`** — not `anyOf: [null, string enum]` (that mounts `anyOfField` → Option N, no clear).
- **`null` in enum IS Domain clear.** Stock sjsf clearable stays **OFF** (projection stamps `ui:options.clearable: false` when enum contains null).
- Labels: `ui:enumNames` with **–** for null, then value titles (e.g. Parziale / Fatto).
- Omit/empty trigger uses the same **–** placeholder (projection stamps `placeholder` + `shadcn4Select.placeholder`).
- **Do not** set schema `default: null` on concrete cell properties — sjsf merger densifies every id to null → partial Commit would clear all rows.
- Partial maps: **omit** = untouched; **selecting – (null)** = Domain clear (`filterFilledCells` keeps null).
- Never form-level `""` as untouched sentinel.

### SPA
- No `normalizeFormCellMaps` / Ajv wrap / seal strip.
- `MapTableCells` (`.svelte.ts`): omit↔omit quiet bind; null writes via whole-map `setValue` (sjsf `objCtx.set(null)` can `removeObject`). Not undefined→null (clearable off). Prefer shrinking further only with a real object-context law — not more coercion patches.
- `$effect` only as escape hatch; coordination → reactive class.

### Go prep
- `filterFilledCells` / object-row filters remain Commit authority.
- `additionalPropertiesIsCellMap` treats string enum (incl. null-in-enum) as cell map.

---

## Pointers

| Area | Path |
| --- | --- |
| Projection | `server/internal/living/form_project.go` (`materializeDynamicMapSchema`, `stampClearableOffWhenEnumNull`) |
| Demo seed Domain | `server/internal/living/demo_commits.go`, `seed.go` |
| Fixture lineLoad | `server/internal/living/fixtures/feragame-config.yaml` (~lineLoad progress) |
| mapTable | `app/src/lib/living/form/map-table-field.svelte`, `map-table-cells.svelte.ts` |
| Glossary | `GLOSSARY.md` — Domain reads / clearable cell maps |
| Prior peels | tip `9b8046e`; unify tip `1b64fcc` |

**Svelte:** MCP autofixer / svelte-file-editor for `.svelte`.

---

## Suggested next peels

### 1) Post-Commit `jobId` null (pre-existing UX)
Commit succeeds → worklist updates → TaskForm unmounts → `submitSealed` still reads `taskBundle.jobId`. Snapshot ids **before** `await`. Not introduced by this branch; smoke hits it on happy path.

### 2) MapTableCells / sjsf object `removeObject`
Null clear still bypasses `objCtx.set` via whole-map `setValue`. Sound follow-up only if there is a principled object-context / mapTable write model — **not** more special cases.

### 3) Human smoke leftovers
- palletWeight empty cells (no pre-selected euro/NIR) after honest seed.
- lineLoad: set / clear (–) / Commit; confirm Domain unset on clear.
- Confirm cold hard-refresh still mounts (no `effect_update_depth_exceeded`).

### 4) Merge-back
When human OK: `/apply-worktree` into `feat/attesta-rework` (or instructed base); then `/delete-worktree`. Do not push unless asked.

### 5) Optional docs
Fold `.scratch` coordinator/handoff into a docs commit only if asked.

---

## Out of scope
- FakeHttp / HttpPort  
- Completo / multi-Commit redesign  
- Renaming Feragame entity field `progress`  
- Ticket 13 package rename  
- Re-litigating mapTable unify  
- `default: null` densify for cell maps  
- Stock clearable ON together with enum-null  
- Push / force  

---

## Done when (branch merge-ready)
- [x] Concrete map projection + honest seed + no SPA normalize  
- [x] Clearable cells: enum-null + clearable false + – labels/placeholder  
- [x] Demo seed opens lineLoad  
- [ ] Post-Commit jobId crash fixed (or explicitly deferred)  
- [ ] Human smoke lineLoad clear + palletWeight quiet defaults signed off  
- [ ] Merge-back / delete-worktree when asked  

## Process
- Match `AGENTS.md`; comments = why only; conventional commits.  
- Review: Standards ∥ Spec if multi-commit — not Bugbot unless asked.  
- Human said **no patches**; escalate hard calls on empty/clear/select law.
