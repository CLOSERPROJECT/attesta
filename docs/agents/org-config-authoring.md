# Org Config authoring

Platform authoring reference for **any** Org Config: gates, transforms, and post-Commit navigation. Not a plant fixture checklist.

Product nouns: [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md) (**Dumb executor**, **readyWhen**, **startsWhen**, **Commit navigation**, **submit cardinality**, **Chrome**, **Org stream**, **anchor Job**, **each**, **blockedReason**). Transform keys (`bind`, `reads`, `derived`, `dynamicProperties`) are authored here—not glossary entries. Decisions: [`docs/adr/`](../adr/) (especially 0005–0008, 0010–0014). Feragame fixture edits: [`feragame-fixture-authoring.md`](feragame-fixture-authoring.md).

## Rules

- **Dumb executor:** plant business rules live in Config (`domain.entities`, transforms, `readyWhen`, `onCommit`). Go and `/app` only evaluate—no plant-named branches in executors. Only config-declared transforms write Domain entities.
- **Transforms:** `bind` rebinds `$` for later ops in that list (no Domain mutate by itself). `reads` (Domain reads) project Domain at open time—never bake instance ids into Config. `derived` may persist on successful submit when configured. Chrome is display-only and stripped before Submission facts. Chrome entries may set `hideIfNull: true` (default false); compile stamps `ui:options.hideIfNull` so the SPA readonly chrome field omits itself when the live display value is null. `dynamicProperties` keys come from Domain reads.
- **Form columns:** author `ui:options.layout: cols-2` or `cols-3` on an object uiSchema; on a property, `layoutSpan: true` (full-bleed) or `layoutNewRow: true` (start a new grid row so the previous row can stay short). Same shortcuts are accepted on chrome entries (`placeholder`, `layoutNewRow`) and stamped onto the chrome field uiSchema. Task form compile expands these into `layouts` + `form-cols-*` classes and strips the shortcuts so they never collide with sjsf’s attribute `layout` object. Do not put Tailwind classes in YAML `layouts` (not `@source`d).
- **Domain gates:** exclusivity and open/close use Task **`readyWhen`** over Domain (and `$domain.submissions` when bind-once matters). Encode those facts in **`readyWhen`**, not in **`startsWhen`**. Do not author **`unlockFrom`** (Config rejects it; sibling/Domain gates belong in **`readyWhen`**).
- **Submit cardinality:** multi-pass same-form work uses leaf Tasks with `submit: many` + Domain **`readyWhen`**, not Task groups ([ADR-0008](../adr/0008-org-task-submit-cardinality.md)).
- **JSONata:** chrome / transform / QR / HttpPort bodies use jsonata-go — see [`jsonata-org.md`](jsonata-org.md) before blaming the executor.
- **Copy tracks:** SPA chrome vs Config product copy vs English engine errors — [ADR-0013](../adr/0013-org-app-locale-tracks.md) and [`app-i18n.md`](app-i18n.md). Plant Italian labels stay in YAML.

## Roles catalog (`roles`)

Optional top-level authored catalog for Identity seed and timeline role badges. Membership remains Identity/Appwrite at runtime (ADR-0002/0003)—this block is catalog shape only (`slug`, `name`, palette key such as `blue` / `cyan` / `emerald`, not CSS vars). Task `roles: [slug]` gates stay slug lists and must align with this catalog when it is present. Gate slug list `OrgConfig.Roles` is derived from the authored block (order preserved) plus any task-only slugs.

```yaml
roles:
  - slug: administration
    name: Administration
    palette: blue
```

## Stream section (`stream`)

Every Org Config has a required `stream` block ([ADR-0014](../adr/0014-org-config-as-stream.md)). It names the Org stream for `/my` and lists timeline steps over Jobs.

```yaml
stream:
  key: recycling              # slug; unique per org; not reserved /app segments i or new
  title: Lavorazione rifiuti
  description: …             # optional
  categorySlug: …             # optional /my discovery
  subCategorySlug: …
  instance:
    job: intake               # anchor Job (repeatable); Play starts it
  steps:
    - job: intake             # once per Org stream instance
    - job: lot_processing
      each:                   # one timeline row per matching Domain entity
        entity: lot
        where: truckId = $stream.id
```

Validation (`ValidateOrgConfig` at Config load and Commit prep — stream, DPP, and Task `recap`): `stream` present; `key` is a slug and not `i`/`new`; `instance.job` is a repeatable Job listed in `steps`; every `steps[].job` exists; `each.entity` is in `domain.entities`; `each.where` compiles. `$stream` is accepted in expressions here and bound when evaluating under an Org stream instance.

**Membership caveat:** a Job instance belongs to Org stream instance S when it is the anchor instance with id S, or when its Job step has `each` and its id is one of the entity ids that `each` selects with `$stream.id = S`. That only works when the row Job’s `instanceId` equals the selected entity id (Feragame: `instanceId: $.lotId`). Validation cannot check this statically—author it that way.

### Seed-bound provisional start

Provisional `each` timeline rows open the Job start Task with query seed `?seed.<field>=` (Feragame: `seed.lotId` from the row entity id when `instanceId` is `$.lotId`). Fields present in that URL/QR seed for the form open are operator-locked (readonly or hidden)—not re-picked. Unseeded Worklist `startLabel` / Play start without seed keeps the Config enum or select. Plant law stays in Config; this is platform seed UX, not a plant-named executor branch. Do not narrow `each.where` to hide provisional rows: accordion select-in-row is fixed by seeding start, not by shrinking membership.

**Finished rule (no new field):** a Job instance is finished when every `submit: once` Task on it is done. End a Job with a once-only closing Task when “finished” should mean that beat is complete.

### `blockedReason` (Task, optional)

JSONata string evaluated when `readyWhen` is false; shown next to the unavailable Task on the `/my` timeline. Compile-checked at Config load; evaluation happens with Worklist projection.

### `recap` (Task, optional)

Optional overview table over a Domain entity kind. Field names are validated against `domain.entities.<kind>.fields` (allowlist of names only — not JSON Schema).

```yaml
recap:
  entity: lot
  columns:
    - field: numeroFir
      title: FIR          # optional display override
    - field: netWeight
```

Rules when authored: `entity` must be a declared `domain.entities` kind that declares `fields`; at least one column; each `columns[].field` must appear in that list; no duplicate fields. Omit `recap` (or leave it empty) when the Task has no overview table.

## Commit navigation (`onCommit`)

Per Task, optional `onCommit` keys control **server-side** post-Commit routing ([ADR-0006](../adr/0006-org-operator-worklist-home.md); glossary **Commit navigation**).

| Key             | Values                                      | Default   |
| --------------- | ------------------------------------------- | --------- |
| `navigation`    | `home` \| `overview` \| `advance` \| `stay` | `advance` |
| `advanceTarget` | bool                                        | `true`    |

- **`navigation`:** `advance` keeps role-eligible auto-advance on the Job instance; `home` / `overview` force that destination (`overview` maps to **home** on the wire); `stay` parks on the submitted Task (download links; many-submit loops typically stay) and Commit ships deferred **Continue**.
- **`advanceTarget: false`:** never auto-advance to the next Task after this Commit. When nothing else is authored, land on **Worklist** home.

The SPA follows Commit `navigation` only—it must not infer stay from links.
