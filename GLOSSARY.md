# Living Attesta

Single-organization operating picture: a versioned **Org Config** (roles + jobs/tasks) and one accumulating **Domain** per org. Operators repeat **Jobs** inside the config; they do not run the whole config as an instance. Product UI for this context lives under `/app`.

Decisions: [`docs/research/attesta-app-living-product-decisions.md`](docs/research/attesta-app-living-product-decisions.md). Legacy Stream language: [`CONTEXT.md`](CONTEXT.md) via [`GLOSSARY-MAP.md`](GLOSSARY-MAP.md).

## Language

### Org Config and domain

**Org Config**:
The versioned authored document for one Organization: role catalog plus keyed Jobs, each with an ordered Task array, for that org’s whole operation.
_Avoid_: Stream; blueprint (when meaning this document); running the whole config as an instance; multiple unrelated configs per org as the default story

**Config version**:
One immutable revision of an Org Config.
_Avoid_: Blueprint executable hash; treating every save as a new org

**Active Config version**:
The Config version the Organization uses for new work and for Load unless a pin policy says otherwise. Pin-in-flight policy is open.
_Avoid_: Implied “latest file on disk” without an org pointer

**Domain** (living):
The Organization’s accumulating facts: entities with stable ids plus append-only submission history. One Domain per org.
_Avoid_: One Domain per whole-config execution; Stream instance; Process progress bag

**Domain store**:
The Module that holds Domain (entities + submit entries). Only config-declared transforms write entities.
_Avoid_: Forms writing entities directly; free-form query language as the sole mutator

### Jobs and tasks

**Job**:
A top-level authored chapter in the Org Config. Groups a Task tree; may be **repeatable** (many Job instances). Jobs are not nested inside Jobs.
_Avoid_: Step (legacy experiment name); Stream; phase; nesting Jobs; treating a Job as a single submit leaf

**Task**:
A **leaf** submit unit: schema, gates, Domain **reads** into the form, and one successful **Submission** via **Commit**. Names which **Config roles** may complete it. Cerbos Commit targets Tasks only.
_Avoid_: Substep (legacy experiment name); Form (when meaning this unit); composite/spawn-many nodes (see Task group); leaf Job; multi-pass reopen after Submission

**Task group**:
_Deferred product name (working label)._ A composite node under a Job: children, may be repeatable with parent-scoped instances; **no** Commit of its own. Spawn-many (e.g. lot under truck). Not implemented in the flat port.
_Avoid_: Calling a Task group a Task; jobs all the way down; faking spawn-many with map/table Domain reads

**Job instance**:
One occurrence of a repeatable Job within the org Domain (e.g. one truck, one line). Often created on first Task submit; id/label from config and/or submitted data.
_Avoid_: Step instance (legacy experiment name); whole-config execution instance; Stream instance; Node (domain docs)

**Task path**:
Stable address of a Task (or later a node) under a Job, e.g. `intake/palletWeight`. Commit and Submission keys use Task path plus Job instance id so deeper trees stay additive.
_Avoid_: Hard one-level child index as the only public address; bare substep id without Job context when paths matter

**Unlock order**:
In-Job rule for which Tasks become completable after prior Submissions on the same Job instance. Distinct from **startsWhen** (Job start) and **readyWhen** (Domain readiness).
_Avoid_: Calling unlock “startsWhen”; Seal-gated unlock

**startsWhen**:
Job-level start gate: quantifier (`some` | `all`) over Submissions of a named Task. Distinct from Unlock order.
_Avoid_: Locked (as the gate state word); Seal-gated start

**readyWhen**:
Task-level Domain readiness (boolean expression over Domain). Combined with the Task’s role gate. When false: visible but unavailable.
_Avoid_: availableWhen; hiding unavailable Tasks; binding operator identity into the expression

**Domain reads** (map/table Tasks):
Authored `reads` + `dynamicProperties` that project current entity ids into a form map (row projection / `rowIds` on the Task form bundle). Projection **materializes** those ids into concrete `schema.properties`; authored `additionalProperties` is the per-id **template** only (`additionalProperties: false` after project). Full-map Tasks list the map property in `schema.required` so every current read id is required; partial maps omit that and leave untouched keys unforced. Not a separate subsystem; not spawn-many of full child registers. Two shapes share one authored `ui:field: mapTable`: a **cell map** (template scalar — one value column, sjsf field per cell schema) and an **object map** (template object — property columns, sjsf field per property). Cell enums are **real values only** (`partial`, `done`, …) — no `null` in enum. Stock sjsf clearable is for draft fix only: emptying a cell drops the key (`omit`/`undefined`); Commit treats omit as untouched (set-or-skip). Never unset Domain facts via Commit.
_Avoid_: Completeness `all`/`some` as product grammar; Leftovers as product grammar; SPA twin Domain-read id helpers; SPA cell-map normalize; densify first-enum without `default`; densify `default: null` into Draft cell maps; form-level `""` as untouched sentinel; conflating map reads with Task group; dual markers (`rowTable`); table-owned enum paint; `enum: [null,…]` Domain clear; stamping `clearable: false` for enum-null law; `anyOf` null|enum as product clear; Commit nil as Domain unset

**Submission**:
Record that a Task succeeded on a Job instance: time, actor, facts snapshot; keyed with Task path. Unlock order, startsWhen, and Worklist done status read Submissions only (one successful Submission finishes the Task on that instance).
_Avoid_: Treating entity fields alone as completion; putting the submitter only inside form facts; row-id reopen as done law

**Draft**:
Working data of an in-progress Task before Submission. Operators keep WIP in draft and Invia once. Not part of Domain until Commit succeeds.
_Avoid_: Submission; Domain; multi-pass Task reopen while “row ids remain”

### Roles

**Role catalog** (living):
The set of **Config roles** declared on the Org Config.
_Avoid_: Putting platform-only standing in the catalog; Stream role (legacy blueprint gate name)

**Config role**:
One named role in the Org Config catalog (slug). Tasks list which Config roles may complete them; Cerbos allow/deny uses principal roles matched by slug.
_Avoid_: Stream role; conflating with Org admin standing; Operator (the default membership role name) for every Config role

**Org catalogs** / `$catalogs`:
Authored bag on Org Config for stable plant knowledge (materials, tareKinds, clients, …). Bound as `$catalogs` wherever Load/Commit exprs see `$domain` (transform, readyWhen, reads.where, onSubmit.http, form Derived/Chrome). Not Domain entities, not the Role catalog, not Chrome.
_Avoid_: Treating catalogs as Domain kinds; conflating with Role catalog or Chrome display catalog

### Authoring details

**bind** (transform op):
Authored JSONata step in `onSubmit.transform` whose result replaces `$` for later ops in that list. Does not emit Domain mutations itself.
_Avoid_: Conflating with form **bindings** bag, **reads** bindings, or Org **catalogs** (`$catalogs`)

**reads**:
Authored Domain query bindings used at open time to project enums / property names into a Task — never bake instance ids into the Org Config.
_Avoid_: Ref / `$refs` / `optionsFor` (retired kernel names)

**Derived value**:
Task field value from an expression under `derived`; may persist into Domain on successful submit when configured.
_Avoid_: Putting GUI-only chrome under `derived`

**Chrome**:
Task-root display catalog (readonly, dataReport, …). Sibling to `derived`; stripped before Submission facts. Not the operator board (see **Worklist**). `dataReport` is evaluated on Load/Commit into `bindings.chromeDisplay` (Domain-only; flat or sections DTO); live kinds that still need draft/operator (`readonly`) stay client-side.
_Avoid_: Encoding live readouts as schema fields meant for submit; naming the Worklist “Chrome”; SPA Domain aggregation for dataReport

**dynamicProperties**:
Field-keyed map config: each property names a Domain **read** whose ids fill that map. Those ids are materialized into `properties`; authored `additionalProperties` is template only. Full vs partial Commit is `schema.required` projection, not completeness. Drives both cell maps and object maps under `mapTable`.
_Avoid_: form-level `propertyNamesFrom` (rejected); authored `completeness`; `ui:field: rowTable`

**progress** (entity field):
Plant-style entity fact on some rows: unset; `partial`; `done`. Draft ≠ progress; Worklist done is Submission law. Not a form/map type — cell enums live on the JSON Schema cell (`additionalProperties`), rendered by sjsf.
_Avoid_: boolean `processed` as a Worklist gate; treating progress enums as unlock grammar; form `progressEnums` / `valueType: "progress"` / table-owned enum paint

### Operator and seams

**OperatorIdentity**:
Host-injected `{ userId, org, roles }` for role gates, chrome, and Submission actor. Not the Org Config document.
_Avoid_: Principal; forging actor via schema fields; Operator (catalog role label) for this bag

**Commit**:
Authoritative accept of a sealed **Task** into Domain (cmd-bag), addressed by Task path + Job instance id. Production allow/deny is Cerbos; HTTP handlers are Adapters only. The Commit response includes an updated viewer-scoped **Worklist** and, when auto-advance applies, the next **Task form bundle**, so the SPA can route to the next Task, **Job instance overview**, or Worklist home without a blind re-Load.
_Avoid_: Treating the SPA route handler as the Seam; folding Drafts into Commit; Committing a Task group

**Load**:
Hydrate what the operator UI needs for the org: viewer-scoped **Worklist**, viewer Config **roles** (for Available affordances), and Active Config version metadata for the home board; not a dump of full Domain + full Org Config. **Task form bundle**s load when opening a Task route via addressed fetch (`GET /api/living/task-form`; provisional `jobInstanceId` empty / `new`) and may return on Commit as `nextTaskFormBundle` when auto-advancing — home Load omits bulk bundles (ADR-0006). Commit may still ship interim bulk until post-Commit routing drops it.
_Avoid_: Hydrate by whole-config execution id; `LoadRunPort` / `/api/runs/...` as product shape; treating Kit load functions as the Seam; shipping full Domain/Org Config so the SPA re-derives Unlock order / startsWhen / FormPlan / dataReport; treating bulk Available bundles as the lasting home-Load shape

**Worklist**:
Go-projected operator board for one org: which Jobs may start (`canStart`); per Job, authored `taskIds`, `startLabel`, `repeatable`, and Config-free Task title + roles chrome; and per Job instance, each Task’s status (`locked` | `available` | `done`). Unlock order and startsWhen are evaluated only on the server from **Submission** relations; a Task is `done` after Submission on the instance. Status stays that Submission/unlock law for handoff/overview; **Available** affordances and **Can start** are separately scoped to the current **OperatorIdentity**’s Config roles (first Task for Can start). The UI reads the projection and does not re-implement those rules. Living `/app` landing for the affiliated operator is this board (not legacy Operator home at `/my`): two actionable bands, not `locked`/`done` rows.
_Avoid_: Job surface; Chrome (FormPlan display catalog); SPA copies of Unlock / startsWhen; row-id reopen for Task status; remapping instance Task status by viewer role; inventing Task order/labels from a client Org Config snapshot; Operator home (the `/my` Stream landing); client-side role filtering from a full Org Config

**Job instance overview**:
Operator screen for one Job instance at `/app/{jobId}/{jobInstanceId}`: Task statuses on that instance plus handoff copy when the next work is gated to Config roles the viewer lacks. Used after Commit when auto-advance does not apply, and as an exit target from Task/Available. Not the Worklist and not an authoring surface.
_Avoid_: Stream timeline; process page; embedding the full instance tree on Worklist home

**Task form bundle**:
Minimum payload to mount one Task form: compiled schema + uiSchema, initial/seed data, projected read ids/columns when needed, Chrome and Derived *rules*, seal/chrome field ids, title, and a DomainSnapshot-shaped bindings bag (including `submissions`) for live Chrome/Derived — not the full Domain or authored Org Config Task document. On Load-of-Task and Commit, server-evaluated `dataReport` results land in `bindings.chromeDisplay` for SPA paint-only; draft-driven Chrome stays live on the client. Fetched addressed by `jobId` + `taskId` + `jobInstanceId` (provisional empty / `new` when starting); Commit may include the next bundle as `nextTaskFormBundle` when auto-advance applies. Home Load omits bulk Available bundles; Task routes mount from addressed fetch.
_Avoid_: Client compile from full authored Task + full Domain; calling this Chrome; baking live draft-driven Chrome results as static strings; SPA Domain walk for dataReport or row ids

**AuthPort / AuthorizerPort**:
Who is calling (`OperatorIdentity`) vs who may complete a Task; Commit uses both. Production AuthorizerPort Adapter is Cerbos.
_Avoid_: Forging actor from sealed facts; IdentityStore as AuthPort; bypassing Cerbos on product Commit

**HttpPort / onSubmit.http**:
Config-declared HTTP after successful Domain mutate; failures do not roll back Domain.
_Avoid_: Rolling back Domain on side-effect failure

### Legacy (other context — do not use for `/app` product prose)

**Stream / Stream instance / Process**:
Multi-party linear flow vocabulary for the HTMX `/my` surface. See [`CONTEXT.md`](CONTEXT.md).
_Avoid_: Using these as names for Org Config, Domain, Job, or Task

### Rejected / historical code names

Do not revive as living product nouns. Experiment/code may still say them until the port lands.

- **Whole-config execution** (historical name `Config run`): one “run of the config” with its own Domain. Living Attesta uses one org Domain and repeated Job instances instead.
- **`LoadRunPort` / `/api/runs` / `RunStore`**: run-id shaped load/persist. Replace with org Domain Load/Commit.
- **Step / Substep / Step instance**: experiment YAML/code names for Job / Task / Job instance.
