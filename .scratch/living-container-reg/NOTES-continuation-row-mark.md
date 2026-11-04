# Architecture note: continuation-row display mark

**Status:** implemented — `1c45f08bcdcd8a7cdebb5bdd859f69d5cf018591` (`feat(living): paint presence mark on hidden continuation fields`)

**Scope (original):** Design only. Prefill of prior weight + hidden `continuesContainerId` already works. Operators still cannot *see* that a compact row continues a prior incomplete.

**Constraint:** Go/SPA stay dumb Org Config executors — no Feragame-/containerReg-named engine or booking logic.

---

## 1. Recommended approach

**Config-authored presence mark on the existing hidden link field**, painted by generic form theme.

Keep `continuesContainerId` as the only continuation signal. On its `uiSchema`, author a generic option (e.g. `presenceMark: true`) plus a plant title (`Continua`). Extend the living theme so a `hiddenField` with a non-empty value *and* that option paints a compact text mark instead of collapsing to nothing.

No new schema property. No Go. No SPA branch on plant Task ids or the name `continuesContainerId` beyond the generic “hidden string + presenceMark” contract.

**Seam:** small interface = “hidden string may show a Config title when set”; plant behavior stays in fixture uiSchema + existing seed.

---

## 2. Alternatives considered (rejected)

| Approach | Why not |
| --- | --- |
| New schema field (`continuationLabel` / `isContinuation`) | Duplicates the link; enters seal facts (Chrome strip is task-root only); transform must ignore. |
| Task-root `chrome` / `dataReport` | Wrong granularity — Chrome is Task-root, not per compact array item. |
| `derived` / `derivedLeaf` | Glossary: GUI-only chrome must not live under `derived`. |
| SPA hardcode on `continuesContainerId` | Plant-named SPA — violates dumb-executor constraint. |
| Extra readonly weight clone | Noise; confuses full-scale vs freeze; delta UX already uses one weight field. |

---

## 3. Seam ownership

| Layer | Owns |
| --- | --- |
| **Org Config** | uiSchema options + Italian title on `continuesContainerId`; optional `order`; no transform/seed change for the mark. |
| **Generic SPA theme** | `hiddenField` / `compactArrayItemField`: presence mark, flex hug (peer of `checkboxOnlyField`), collapse when empty. |
| **Go** | Nothing. Prefer zero engine change. |

---

## 4. Data model

- **No new form field.** Display is chrome-only paint from non-empty `continuesContainerId`.
- **i18n/title:** Config uiSchema (e.g. `Continua`), not hardcoded Italian in theme.
- Optional tooltip may surface the prior id; Domain must still not store `continuesContainerId` on minted containers.

---

## 5. UX

Compact row today: `[weight | incomplete ☐ | (hidden continues)]`.

Add short muted text (not a card/pill) hugging left of weight or between weight and incomplete — same flex language as `checkboxOnlyField`. Visible only on continuation rows; new rows unchanged. Prior freeze stays as editable weight prefill.

---

## 6. Risks

- **Commit of display fields:** Avoided by not adding a schema property.
- **Delta math:** Mark must not change seed weight, strip the link, or touch transform. Keep existing Commit delta tests; assert mark iff seed has `continuesContainerId`.
- **a11y:** Text label (not color alone); `aria-label` from Config title; mark must not be `aria-hidden`.

---

## 7. Implementable next steps

1. Spec generic `ui:options.presenceMark` (+ title) on `hiddenField`; note in glossary near Chrome / `continuesContainerId`.
2. Adjust compact-array CSS so presence-marked wrappers hug like checkbox-only.
3. Plant fixture: enable option + `Continua` on register `continuesContainerId`.
4. SPA unit: paints when value set, omitted when empty; no plant Task-id assertions.
5. Keep Go Commit/Load delta + seed tests green; optional fixture uiSchema assertion.
6. Manual check: continuation Spawn shows mark + prefilled weight; new “+” rows have neither.
