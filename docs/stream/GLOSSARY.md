# Legacy Stream

Multi-party **Stream** blueprints and instances on `/my` (Formata, discovery, notarized substeps, optional Stream DPP). Historical Formata runtime; still accurate for that path. `/my` can also host **Org-backed streams** (Org Config + Domain)—those nouns live in [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md) ([ADR-0014](../adr/0014-org-config-as-stream.md)).

Shared membership: [`GLOSSARY.md`](../../GLOSSARY.md). Org (`/app` and Org-backed `/my`): [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md). Map: [`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md).

## Language

**Stream**: A configured Formata blueprint of steps, substeps, roles, and forms. Starting a stream creates a new **Stream instance**. Distinct from an **Org stream**.\
_Avoid_: Workflow, stream type, template; Org Config; Org stream

**Import-blocking issue**: A Stream document problem that prevents loading a draft in Composer.\
_Avoid_: Critical error (vague), validation error (when meaning only this tier)

**Composer-fixable issue**: A Stream document problem that can load into Composer and be corrected there before Save.\
_Avoid_: Warning (alone), soft error, non-critical

**Stream instance**: A single running occurrence of a Formata Stream — one tracked batch or case from start through completion or early termination. Distinct from an **Org stream instance**.\
_Avoid_: Process, stream (when meaning an instance in domain docs); Domain; Job instance; Org stream instance

**Step**: An organization-scoped phase in a stream blueprint. Groups one or more substeps.\
_Avoid_: Using "step" when meaning substep completion state; Job

**Substep**: The smallest completable unit in a stream — a role-gated form or input that a participant submits.\
_Avoid_: Action; Task

**Stream instance detail**: The full page for one stream instance.\
_Avoid_: Process page, action list

**Operator home**: The authenticated `/my` landing for an affiliated user. Surfaces Attention items, Upcoming stream instances, and Stream discovery (Formata and Org-backed); Org admin and Onboarding meanings live in [`GLOSSARY.md`](../../GLOSSARY.md). Everyone lands here ([ADR-0014](../adr/0014-org-config-as-stream.md)).\
_Avoid_: Home (alone), Dashboard (vague), Stream picker (as the name of Operator home), `/app` Worklist home as the affiliated landing

**Stream discovery**: The Operator-home region listing Streams the user’s Organization can take part in, with Category and Sub-category browsing.\
_Avoid_: Stream picker (when meaning the entire Operator home), Home

**Startable stream**: A Stream the current affiliated user may start a new Stream instance for. Distinct from merely appearing in Stream discovery as a participant.\
_Avoid_: Accessible stream, visible stream, can open dashboard

**Public stream card**: A public-homepage presentation of a **Stream** (blueprint). Distinct from the Operator-home discovery card.\
_Avoid_: Stream card (when meaning the Operator home discovery card alone), process card, landing stream tile, showcase stream

### Stream discovery taxonomy

**Category**: A platform-global discovery bucket for browsing **Streams** (blueprints).\
_Avoid_: Topic, tag, theme

**Sub-category**: A finer discovery leaf under exactly one **Category**.\
_Avoid_: Tag, topic, Category (when meaning the leaf)

**Stream categorization**: A **Stream** may reference zero or one **Sub-category**. It never attaches to a **Category** alone.\
_Avoid_: Required category, category-only membership

**Stream presentation fields**: The blueprint label and discovery path on a **Stream**. See [ADR-0001](../adr/0001-stream-presentation-edit-without-purge.md).\
_Avoid_: Metadata (vague), tags, executable config

**Stream role**: A role named on a Stream blueprint to gate Substeps. Not itself a Role catalog entry. See [`GLOSSARY.md`](../../GLOSSARY.md).\
_Avoid_: Catalog role, workflow role (in domain docs), Organization role (when meaning the blueprint gate), Config role

**Upcoming stream instance**: An active Stream instance that involves the user’s Organization where the next actionable Substep is not theirs. A watchlist, not an Attention item.\
_Avoid_: Pending (vague), scheduled, upcoming (when meaning a calendar event)

**Stream dashboard**: The screen listing stream instances for one stream.\
_Avoid_: Home, workflow home

**Platform stream catalog**: The platform-admin screen for browsing and managing Streams across the platform. Distinct from affiliated Stream discovery on Operator home.\
_Avoid_: Operator home, Stream discovery

### Stream instance detail UI

**Stream timeline**: The tree of steps and substeps on stream instance detail (and preview on the stream dashboard).\
_Avoid_: Workflow timeline, traceability timeline, action list

**My tasks filter**: A Stream instance detail switch that shows only Substeps the viewer owns.\
_Avoid_: My roles filter, Role filter (vague)

**Substep summary**: The always-visible row for one substep in the Stream timeline.\
_Avoid_: Action card header

**Substep body**: The expandable content below a substep summary.\
_Avoid_: Action detail, action content

## Relationships

- A **Stream** blueprint is started as many **Stream instances**.
- A **Step** belongs to one **Organization**; **Substeps** are gated by **Stream roles** (see Organization roles in [`GLOSSARY.md`](../../GLOSSARY.md)).
- Org Config / Domain / **Org stream** / Job / Task are the Org product nouns—see [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md). `/my` may list Org-backed streams beside Formata Streams.

## Historical / rejected

**Process**: Former code name for **Stream instance**.

In operator-facing copy, bare "stream" may mean stream instance when context is obvious; use **stream instance** in domain docs when ambiguity matters.
