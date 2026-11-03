# Slice: containerReg compact uiSchema (explicit A) + recursive Go placeholders

**Date:** 2026-11-02  
**Branch:** `feat/attesta-rework` @ `773113d`  
**Status:** implementation done — await human single commit (exclude unrelated `.scratch` dirt)

## Settled decisions (do not re-litigate)

1. **Go / Org Config authoritative.** SPA must not paint or rewrite bundle `uiSchema` (scout: pass-through; only `$today` schema sentinel at mount).
2. **Explicit A:** restore compact materials chrome by **authoring** it in `containerReg.uiSchema` (twin fixtures). Not SPA `paintContainerRegCompactArrayUi`.
3. **Defaults:** deepen Go `withDefaultPlaceholders` to recurse nested `properties` / `items`. Authored placeholders win. Optional: `Seleziona {title}…` when property has `enum`.
4. **No plant Task-id branches** in `BuildTaskFormBundle` (existing comment).
5. After YAML owns chrome: delete orphan SPA painter (`compact-array-item-ui.ts` and dead imports); keep theme `compactArrayItemField` for render.

## Pointers

- Lost paint semantics: `app/src/lib/living/form/compact-array-item-ui.ts` (orphan; source of chrome shape)
- Theme registration: `app/src/lib/living/form/form-theme.ts` (`compactArrayItemField`, `checkboxOnlyField`)
- Go defaults today (root-only): `server/internal/living/form_compile.go` → `withDefaultPlaceholders`; called from `form_bundle.go`
- Fixture (single source): `server/internal/living/fixtures/feragame-config.yaml` (`containerReg` uiSchema is order-only today); SPA tests import via `$living-fixtures`.
- Authored `ui:components` pattern (radio): same fixture `idoneitaRiutilizzo`
- Nested `ui:field` is **not** recursively painted by Go today — prefer resolved `ui:components` in YAML (like radio), e.g. `booleanField: checkboxOnlyField`, `arrayItemField: compactArrayItemField`

## Delivery order

1. Go recursive placeholders (+ test)
2. YAML compact chrome (explicit A); rely on Go for generic placeholders; optional material-specific weight copy only if needed
3. Delete dead SPA painter module / unused imports
4. Verify: Go tests; mount containerReg in UI if stack up

## Out of scope

- SPA re-wire of `paintContainerRegCompactArrayUi` in `planFromTaskFormBundle`
- Twin-fixture DRY / JobChrome delete
- dataReport / finalResults work (already landed on branch)

## Git

**Do not commit** until the whole slice is done. One commit at the end (human asks).
