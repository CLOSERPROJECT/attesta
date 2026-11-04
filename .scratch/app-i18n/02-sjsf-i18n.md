# @sjsf/form i18n — research for Attesta `/app`

**Package versions (installed):** `@sjsf/form@3.9.0`, `@sjsf/ajv8-validator`, `@sjsf/shadcn4-theme`, `@sjsf/lucide-icons`.

**Scope:** Form *chrome* (buttons, boolean labels, validation plumbing, widget empty states) — not JSON Schema / uiSchema field titles from Org Config.

---

## 1. How sjsf translation / i18n works

### 1.1 Core types (`@sjsf/form` → `dist/form/translation.d.ts`)

| Type | Role |
|------|------|
| `Labels` | Map of **label id → parameter object** (empty `{}` for static strings). |
| `Label` | `keyof Labels` |
| `Translator<P>` | `string` **or** `(params: P) => string` |
| `TranslatorDefinitions` | `{ [K in keyof Labels]: Translator<Labels[K]> }` |
| `Translation` | `Resolver<Partial<Labels>, Partial<TranslatorDefinitions>>` — lookup by label id |
| `Translate` | `(label, params) => string` — runtime function passed into fields |
| `createTranslate(translation)` | Builds `Translate`; missing keys → `` `Label "${label}" is not translated` `` |

**Label ids (form chrome)** — from `Labels` + module augmentation in `@sjsf/form/extra-labels/*`:

| Label id | Params | Typical UI |
|----------|--------|------------|
| `submit` | — | Submit button (`form/submit-button.svelte`) |
| `add-array-item` | — | Array “Add item” |
| `remove-array-item` | — | Array row delete |
| `copy-array-item` | — | Array row copy |
| `move-array-item-up` / `move-array-item-down` | — | Reorder |
| `add-object-property` / `remove-object-property` | — | Additional properties |
| `yes` / `no` | — | Boolean select enum titles |
| `multi-schema-option-label` | `{ index }` | oneOf/anyOf selector |
| `multi-schema-option-label-with-title` | `{ title, index }` | oneOf/anyOf with schema title |
| `key-input-title` | `{ name }` | Dynamic object key input |
| `additional-property` | — | Additional property row |
| `expand` | — | Expand control |
| `edit` / `clear` | — | Field actions |
| `array-schema-missing-items` | — | Broken array schema |
| `validation-process-error` | `{ error: FailedTask }` | Submit/field validation task failure |
| `component-not-found` | `{ type }` | Theme resolution failure |
| `unknown-field-error` | `{ schema }` | Unknown field fallback (long English template + JSON) |

Rendered via **`Text`** (`dist/form/text.svelte`): calls `translate(id, args)`, optionally replaces text with an **icon** from `icons` resolver when `ctx[FORM_ICONS]` defines that label.

### 1.2 Wiring at form creation (`createForm`)

`translation` is **required** on `FormOptions` (`dist/form/create-form.svelte.d.ts`).

Flow (`dist/form/create-form.svelte.js`):

1. `options.translation` stored on form state as `FORM_TRANSLATION`.
2. `translate = createTranslate(options.translation)` exposed as `FORM_TRANSLATE`.
3. Root field receives `translate={retrieveTranslate(ctx, config)}` (`form/content.svelte`).

**Built-in bundles** (`@sjsf/form/translations/*`):

- `@sjsf/form/translations/en` — `fromRecord(definitions)` (default English).
- `@sjsf/form/translations/ru` — Russian reference locale.

No other locales ship in `dist/translations/` for 3.9.0.

### 1.3 Injecting / overriding translations

**Resolver helpers** (`@sjsf/form/lib/resolver`):

- `fromRecord({ ... })` — plain object → `Translation`.
- `overrideByRecord(base, partial)` — shallow override (Attesta already uses this pattern in `form-theme.ts` for components).
- `chain(primary, fallback)` — try primary resolver, then fallback.

**Recommended patterns:**

```ts
import { translation as en } from '@sjsf/form/translations/en';
import { overrideByRecord, chain, fromRecord } from '@sjsf/form/lib/resolver';

// Full custom table
const it = fromRecord({ submit: 'Invia', 'add-array-item': 'Aggiungi elemento', /* … */ });

// Override only deltas on English
const itPartial = overrideByRecord(en, { submit: 'Invia', 'remove-array-item': 'Rimuovi' });

// Fallback chain (custom → en)
const translation = chain(itPartial, en);
```

**Per-field overrides** (`UiOptions.translations` on uiSchema):

- `retrieveTranslate(ctx, config)` merges, in order: form-level `FORM_TRANSLATION` → `ui:options.translations` on that field’s uiSchema → `extraUiOptions('translations', config)` (`dist/form/state/ui-schema.js`).
- Replaces deprecated `submitButtonText` / similar (upstream commit f30da6d).

**Icons vs text:** `@sjsf/lucide-icons` defines icons only for move up/down, remove item/property, copy — those labels render as **icons without visible text** (`Text` + `icons`). Other labels (e.g. `add-array-item`, `submit`) still show translated strings unless you add icons or hide via CSS (Attesta’s `form-theme-button.svelte` adds a Plus icon for `array-item-add` but still renders `{@render children()}`, so English “Add item” can remain visible unless styled away).

### 1.4 Validation message localization (separate from `Translation`)

Ajv errors are **not** passed through the sjsf label table.

**Pipeline** (`@sjsf/ajv8-validator`):

- `createFormValidator({ ... localize?, ajvOptions, ajvPlugins })`
- `localize?: (errors: ErrorObject[]) => ErrorObject[]` — run **before** path/title rewriting (`withLocalize` in `dist/errors.js`).
- `createFormErrorsTransformer` / `createFieldErrorsTransformer` map Ajv `message` to field paths; for `required` + `missingProperty`, substitute **property name → uiSchema title or schema `title`** (Config-authored titles can appear in messages without translating Ajv templates).

**Official pattern** (upstream `examples/basic-starter/src/routes/ajv-i18n/+page.svelte`):

```ts
import localize from 'ajv-i18n/localize/ru'; // or /it

createFormValidator({
  ajvOptions: { ...DEFAULT_AJV_CONFIG, messages: false },
  ajvPlugins: (ajv) => addFormats(addFormComponents(ajv)),
  localize: (errors) => {
    localize(errors);
    return errors;
  },
});
```

Attesta today (`app/src/lib/living/form/ajv-form-validator.ts`): `createFormValidator({ ajvOptions: { ...DEFAULT_AJV_CONFIG, $data: true } })` — **no `localize`**, **`messages` not disabled** → English Ajv strings (`verbose: true` in `DEFAULT_AJV_CONFIG`).

**Submit banner:** `TaskForm.svelte` → `formatSubmitValidationError` (`submit-validation.ts`) joins raw `err.message` (Ajv) with path prefixes; fallback string is already hardcoded `'Validazione fallita'` when no errors — mixed locale.

### 1.5 Array / object button labels (where they come from)

| Control | Source component | Label id |
|---------|------------------|----------|
| Add row | `fields/array/array-base.svelte` | `add-array-item` |
| Remove / copy / move | `fields/array/array-item-field.svelte` | `remove-array-item`, etc. |
| Compact remove | Attesta `compact-array-item-field.svelte` | `remove-array-item` via `<Text id="remove-array-item" {translate} />` |
| Add property | `fields/object/object-field.svelte` | `add-object-property` |
| Remove property | `fields/object/object-property-field.svelte` | `remove-object-property` |
| Submit | `form/submit-button.svelte` | `submit` (uses `retrieveTranslate` for root config) |

Attesta overrides **`button`** and **`compactArrayItemField`** in `form-theme.ts`; array add still uses stock `array-base` + custom `FormThemeButton`.

### 1.6 Theme / widget strings *outside* `Translation`

`@sjsf/shadcn4-theme` exposes **uiSchema uiOptions**, not label ids, for widget chrome, e.g.:

- `shadcn4ComboboxEmptyText`
- `shadcn4DateRangePickerPlaceholder`
- Select/multi-select **placeholder** from schema/ui attributes

These are **not** covered by `@sjsf/form/translations/*`; localize via uiSchema / `extraUiOptions` on `createForm`, or Config uiSchema if plant-specific.

Date display: Attesta uses `useDatePickerForDateFormat` in `form-theme.ts` (widget swap only; locale for formatting is separate).

---

## 2. Italian locale packs upstream

| Asset | Italian? |
|-------|----------|
| `@sjsf/form/translations/it` | **No** — only `en` and `ru` in installed package |
| `@sjsf/form/translations/ru` | Yes (reference for structure) |
| `ajv-i18n` | **Not** in `app` dependencies; package publishes `localize/it` (standard ajv-i18n) — must add dep + wire `localize` |
| `@sjsf/shadcn4-theme` | No locale packs |

**Conclusion:** Custom Italian `Translation` (new module or Paraglide-backed factory) is **required** for form chrome. Validation needs **`ajv-i18n`** (or custom `localize`) for Italian Ajv messages.

---

## 3. How Attesta wires sjsf today

```
live-task-form.svelte.ts
  └─ createConfigForm()          app/src/lib/living/form/create-config-form.svelte.ts
       ├─ theme                  form-theme.ts → shadcn4 + authored fields + FormThemeButton
       ├─ translation            static import @sjsf/form/translations/en  (both branches)
       ├─ validator              ajv-form-validator.ts → createFormValidator (no localize)
       ├─ icons                  @sjsf/lucide-icons
       └─ resolver/merger/id     compat / modern

ConfigForm.svelte
  └─ setShadcnContext() + BasicForm {form}
```

**`compact-array-item-field.svelte`:**

- Receives `translate` from sjsf field props (same as stock array item field).
- Remove button: `<Text {config} id="remove-array-item" {translate} />` inside custom `Button` — **correct** pattern; will pick up Italian once form-level `translation` changes.
- Passes `{translate}` through to nested `Field`.

**`map-table-field.svelte`:** forwards `translate` / `retrieveTranslate(ctx, cellCfg)` to cell fields (same mechanism).

**Not sjsf-translated (Attesta-owned chrome in form tree):**

- `data-report-field.svelte` — e.g. `"No primary load"` (English literal).
- `submit-validation.ts` / `TaskForm.svelte` — banner text composition.
- `HttpLinkStrip.svelte` (if user-facing strings) — separate from sjsf label table.

---

## 4. Paraglide (`m.*`) vs sjsf `Translation` — recommendation

### Primary approach: **Paraglide as the single human-edited catalog**, with a thin **sjsf adapter factory**

Implement `getSjsfTranslation(locale): Translation` in app code that returns `fromRecord({ submit: m.form_submit({}, { locale }), … })` (or `overrideByRecord(en, …)` for partial migration). Pass result into `createForm({ translation: getSjsfTranslation(getLocale()) })`.

**Why this over a standalone `@sjsf/form/translations/it` fork:**

- One review surface for Italian (`app/messages/it.json`) alongside living/router chrome.
- Translators never hunt sjsf-specific files in `node_modules` patterns.
- Parameterized labels (`multi-schema-option-label`, `validation-process-error`) map cleanly to Paraglide messages with placeholders.

**Tradeoffs:**

| | Paraglide-backed sjsf adapter | Pure sjsf `fromRecord` / upstream-style `translations/it.ts` |
|--|-------------------------------|-------------------------------------------------------------|
| Catalog unity | Good — one Paraglide project | Split — form chrome duplicated from SPA messages |
| Type safety | Must keep adapter keys in sync with `Label` (const object + satisfies or codegen) | Native `TranslatorDefinitions` typing via copy of `en.ts` |
| Runtime / SSR | Messages resolved at factory time; form recreated or translation must be reactive if locale switches mid-session | Static import; locale switch requires remounting form (already happens on projection key in `TaskForm`) |
| sjsf-internal strings | Still need **ajv-i18n** separately — not Paraglide | Same |
| shadcn widget strings | Still uiSchema / `extraUiOptions` — neither approach covers them | Same |

**Avoid:** Calling `m.*` from inside every custom field for stock labels — duplicates sjsf’s `Text`/`translate` pipeline.

**Secondary:** Keep `@sjsf/form/translations/en` as **`chain(it, en)` fallback** until every label id is translated (prevents ``Label "…" is not translated`` in dev).

---

## 5. Concrete files / APIs to touch for FULL form chrome i18n

### Must change

| File | Change |
|------|--------|
| `app/src/lib/living/form/create-config-form.svelte.ts` | Replace static `translation` from `en` with locale-aware `getSjsfTranslation(...)`; import from new adapter module. |
| **New** `app/src/lib/living/form/sjsf-translation.ts` (or `app/src/lib/i18n/sjsf-translation.ts`) | Build `Translation` from Paraglide messages; export label key checklist mirroring `en.js` + `edit`/`clear`. |
| `app/messages/en.json` + `app/messages/it.json` | Add ~20 `form_*` (or `sjsf_*`) keys for all label ids + validation-process-error reasons. |
| `app/src/lib/living/form/ajv-form-validator.ts` | Accept optional locale or import localize fn; set `messages: false`, `localize: ajv-i18n/localize/it`, keep `addFormComponents` / formats as in upstream example. |
| `app/package.json` | Add `ajv-i18n` dependency (peer to Ajv 8). |

### Should change (Attesta form UX / mixed locale)

| File | Change |
|------|--------|
| `app/src/lib/living/form/submit-validation.ts` | Paraglide for fallback/prefix formatting; messages already localized via Ajv after step above. |
| `app/src/lib/living/TaskForm.svelte` | Banner strings via Paraglide if any wrappers beyond `formatSubmitValidationError`. |
| `app/src/lib/living/form/data-report-field.svelte` | Replace hardcoded empty-state copy with Paraglide. |
| `app/src/lib/living/form/form-theme-button.svelte` | Decide add-button UX: icon-only + `aria-label={translate('add-array-item', {})}` (needs `translate` prop) vs show Italian text from `Text` child. |
| `app/src/lib/living/form/compact-array-item-field.svelte` | Optional: `aria-label` on remove when icon-only (`Text` + lucide icons). |

### Optional / schema-driven (widgets + plant-specific)

| Mechanism | Use for |
|-----------|---------|
| `createForm({ extraUiOptions })` | Global Italian `shadcn4ComboboxEmptyText`, date placeholders, etc. |
| uiSchema `ui:globalOptions.translations` | Rare per-form overrides |
| uiSchema `ui:options.translations` | Per-field button overrides |
| Config uiSchema (out of scope for “chrome”) | Field titles/descriptions — already Italian in plant configs |

### No change required for stock label wiring

| File | Notes |
|------|-------|
| `app/src/lib/living/form/ConfigForm.svelte` | Passes `form` through; translation is on `FormState`. |
| `app/src/lib/living/form/form-theme.ts` | Theme/components only unless adding `extraUiOptions` helper. |
| `app/src/lib/shadcn/set-shadcn-context.ts` | Unrelated to sjsf labels. |

### Reference only (do not edit)

- `app/node_modules/@sjsf/form/dist/translations/en.js` — complete key list / English defaults.
- `app/node_modules/@sjsf/form/dist/translations/ru.js` — non-English sample.
- Upstream example: `ajv-i18n` route in svelte-jsonschema-form repo.

---

## Blockers / gaps

1. **No upstream Italian sjsf bundle** — custom adapter required.
2. **`ajv-i18n` not installed** — field/submit validation messages stay English until added and `messages: false` set.
3. **Icon-only actions** (`@sjsf/lucide-icons` + `Text`) may lack accessible Italian names unless buttons get `aria-label` from `translate(...)`.
4. **shadcn extra-widget strings** (combobox empty, placeholders) bypass sjsf `Translation` — need uiSchema/`extraUiOptions` strategy.
5. **`unknown-field-error`** is a long developer-facing English template — likely low priority for operators but still English if triggered.
6. **Locale switch:** ensure `createConfigForm` runs with new `translation` when Paraglide locale changes (TaskForm remount on `projectionKey` helps; verify non-remount paths).
7. **Mixed locale today:** `formatSubmitValidationError` uses Italian fallback with English Ajv bodies.

---

## Quick reference: English defaults (`@sjsf/form/translations/en.js`)

```
submit: Submit
add-array-item: Add item
remove-array-item / remove-object-property: Del
copy-array-item: Copy
move-array-item-up/down: Up / Down
add-object-property: Add property
yes/no: Yes / No
expand: Expand
edit/clear: Edit / Clear
multi-schema-option-label: Option ${index+1}
validation-process-error: aborted | timeout | error → English failure strings
```

Russian pack uses fuller remove strings (`Удалить`) — better UX reference for Italian remove labels than English “Del”.
