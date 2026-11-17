# Org Config authoring

Platform authoring reference for **any** Org Config: gates, transforms, and
post-Commit navigation. Not a plant fixture checklist.

Vocabulary: [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md) (**Dumb executor**,
**readyWhen**, **startsWhen**, **Unlock order**, **Commit navigation**,
**submit cardinality**, **bind**, **reads**, **Chrome**, **dynamicProperties**).
Decisions: [`docs/adr/`](../adr/) (especially 0005–0008, 0010–0013). Feragame
fixture edits: [`feragame-fixture-authoring.md`](feragame-fixture-authoring.md).

## Rules

- **Dumb executor:** plant business rules live in Config (`domain.entities`,
  transforms, `readyWhen`, `onCommit`). Go and `/app` only evaluate—no
  plant-named branches in executors. Only config-declared transforms write
  Domain entities.
- **Transforms:** `bind` rebinds `$` for later ops in that list (no Domain
  mutate by itself). `reads` project Domain at open time—never bake instance
  ids into Config. `derived` may persist on successful submit when configured.
  Chrome is display-only and stripped before Submission facts.
  `dynamicProperties` keys come from Domain reads.
- **Domain gates:** exclusivity and open/close use Task **`readyWhen`** over
  Domain (and `$domain.submissions` when bind-once matters). Encode those facts
  in **`readyWhen`**, not in **`startsWhen`**. When Domain already holds the
  prerequisite, author **`readyWhen`** and omit **`unlockFrom`**.
- **Submit cardinality:** multi-pass same-form work uses leaf Tasks with
  `submit: many` + Domain **`readyWhen`**, not Task groups
  ([ADR-0008](../adr/0008-org-task-submit-cardinality.md)).
- **JSONata:** chrome / transform / QR / HttpPort bodies use jsonata-go — see
  [`jsonata-org.md`](jsonata-org.md) before blaming the executor.
- **Copy tracks:** SPA chrome vs Config product copy vs English engine errors —
  [ADR-0013](../adr/0013-org-app-locale-tracks.md) and [`app-i18n.md`](app-i18n.md).
  Plant Italian labels stay in YAML.

## Commit navigation (`onCommit`)

Per Task, optional `onCommit` keys control **server-side** post-Commit routing
([ADR-0006](../adr/0006-org-operator-worklist-home.md); glossary **Commit
navigation**).

| Key | Values | Default |
|-----|--------|---------|
| `navigation` | `home` \| `overview` \| `advance` \| `stay` | `advance` |
| `advanceTarget` | bool | `true` |

- **`navigation`:** `advance` keeps role-eligible auto-advance on the Job
  instance; `home` / `overview` force that destination (`overview` maps to
  **home** on the wire); `stay` parks on the submitted Task (download links;
  many-submit loops typically stay) and Commit ships deferred **Continue**.
- **`advanceTarget: false`:** never auto-advance to the next Task after this
  Commit. When nothing else is authored, land on **Worklist** home.

The SPA follows Commit `navigation` only—it must not infer stay from links.
