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
_Avoid_: Substep (legacy experiment name); Form (when meaning this unit); composite/spawn-many nodes (see Task group); leaf Job; multi-pass reopen after Submission; task instance

**child Task**:
A **Task** authored under a **Task group**, completed on one occurrence of that group under a **Job instance**. Still a Task for Commit and Cerbos.
_Avoid_: Subtask; task instance; treating child Tasks as a different Commit kind

**Task group**:
A composite node under a Job: ordered **child Tasks**, may be repeatable; **no** Commit of its own (e.g. lots under a truck). Each occurrence under a **Job instance** is one group instance (opened by **Spawn**, ended when the group is **Close**d). Operators **Close** (and may **Spawn**) from that Job instance’s overview; home **Available** may also offer provisional first-leaf Spawn when the Worklist projects `canSpawn`.
_Avoid_: Calling a Task group a Task; jobs all the way down; faking spawn-many with map/table Domain reads; treating Close as a Task Commit; a separate glossary peer for “Task group instance”

**Spawn**:
Open a new occurrence of a **Task group** under a **Job instance** by Committing its first **child Task** with spawn mint (`spawningNew`). That occurrence holds the child Task statuses until **Close**. Offered on overview and, when `canSpawn`, as a provisional first-leaf **Available** row on home (group instance id `new`)—not a separate home Spawn band. Distinct from starting a Job. **Close** stays overview-only.
_Avoid_: Empty-open without the first leaf Submission as the normal operator path; calling Job start “Spawn”; inventing a home Spawn peer of Can start; Task instance

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
Record that a Task succeeded on a Job instance: time, actor, facts snapshot; keyed with Task path. Unlock order, startsWhen, and Worklist done status read Submissions only (one successful Submission finishes the Task on that instance). Post-Commit HTTP **links** are not facts: they are stored in a separate step after Commit (+ HttpPort), and may appear on overview recap without replaying HTTP.
_Avoid_: Treating entity fields alone as completion; putting the submitter only inside form facts; row-id reopen as done law; folding link persistence into the facts Commit write; HTTP replay to rebuild links

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

**fullyProcessed** (entity field):
Plant Domain fact on a **load**: container registration for that load is finished; register load enums omit it. Set by Org Config transform on successful register Commit—not a Worklist gate, not progress enum vocabulary, and not passport closure.
_Avoid_: boolean `processed` as Worklist status; using fullyProcessed as unlock grammar; synonym “complete” without naming this fact; using fullyProcessed as “Passport done”

**incomplete** (entity field):
Plant Domain fact on a **container**: booked weight for the current load is frozen while the physical bin may continue on a later register. Distinct from Task-group Close.
_Avoid_: Treating incomplete as Worklist/progress; engine special-case on the name (Config transform + form only)

**continuesContainerId** (Config form fact):
Hidden row field on plant register: links a continuation row to a still-open incomplete Domain container. Load seeds it via Task `initialData`; Commit transform books delta weight and supersedes the prior incomplete. Not a Domain entity field. Operator visibility uses generic `ui:options.presenceMark` on `hiddenField` (Config title such as Continua)—not a second schema fact.
_Avoid_: Engine special-case on the name; free-picked secondary load enum for continuity; storing continuesContainerId on minted containers; SPA branches on this field name beyond the generic hidden+presenceMark contract

### Passport (living)

**Passport**:
The issued Digital Product Passport for one Domain subject (in Feragame: one **load**). Public credential and human page are projections of a **Passport snapshot**, not a live Domain walk. **Passport production** is a core living platform capability. Org Config declares scheme in a root **`passport`** block (subject binding, identity, snapshot projection); a Lock leaf opts in with thin **`onSubmit.passport`** (not HttpPort).
_Avoid_: Stream Process DPP as the living issue path; SPA-assembled passport payloads; treating Feragame YAML as the feature itself; gather graph only on the Task with no root scheme; Job-level “produces DPP” with no Commit leaf

**Passport lock**:
The operator Task that approves a subject’s gathered facts and issues its **Passport**: Load shows eligibility + recap; one Commit runs transform (Domain locked fact) then the Passport producer. Dedicated repeatable Job; Job instance id is the subject id. Distinct from Worklist status `locked` and from form seal. Post-Commit Digital Link / DPP links come from the producer, not **HttpPort** / `onSubmit.http`.
_Avoid_: Auto-issue on Domain predicate alone; overloading `fullyProcessed`; Lock as a new HTTP Seam; conflating with Unlock-order `locked`; minting passport URLs via outbound HTTP; producer as sole writer of eligibility Domain facts

**Passport snapshot**:
Immutable projection of related Domain facts (and named Submissions) from the root `passport.snapshot` declaration. Evaluated live on **Passport lock** Load for the operator recap; frozen into the platform Passport record (org + subject) on Commit. Later Domain edits do not rewrite what the Passport claims.
_Avoid_: Live Domain as the passport body; a second hand-authored recap expr that can drift from freeze; mutable “current view” credentials; “passport” as just another Feragame entity row

### Operator and seams

**OperatorIdentity**:
Host-injected `{ userId, org, roles }` for role gates, chrome, and Submission actor. Not the Org Config document.
_Avoid_: Principal; forging actor via schema fields; Operator (catalog role label) for this bag

**Commit**:
Authoritative accept of a sealed **Task** into Domain (cmd-bag), addressed by Task path + Job instance id. Production allow/deny is Cerbos; HTTP handlers are Adapters only. The Commit response includes an updated viewer-scoped **Worklist**, an explicit `next` destination (`task` | `overview` | `home`), and, when auto-advance applies, the next **Task form bundle**, so the SPA navigates without a blind re-Load or Worklist re-walk for overview-vs-home. If Commit returns download links, the operator stays on a read-only view of the submitted form with those links; for an open **Task group** that screen also offers **Spawn** again and **Close** (confirm). Without links, an open **child Task** Commit prefers **Job instance overview**. After Close, the usual next-Task / overview / home rule applies.
_Avoid_: Treating the SPA route handler as the Seam; folding Drafts into Commit; Committing a Task group; SPA re-deriving post-Commit destination from Worklist

**Load**:
Hydrate what the operator UI needs for the org: viewer-scoped **Worklist**, viewer Config **roles** (for Available affordances), and Active Config version metadata for the home board; not a dump of full Domain + full Org Config. **Task form bundle**s load when opening a Task route via addressed fetch (`GET /api/living/task-form`; provisional `jobInstanceId` empty / `new`) and may return on Commit as `nextTaskFormBundle` when auto-advancing — home Load omits bulk bundles (ADR-0006). SPA post-Commit routing consumes Commit `next` (+ bundle when `task`).
_Avoid_: Hydrate by whole-config execution id; `LoadRunPort` / `/api/runs/...` as product shape; treating Kit load functions as the Seam; shipping full Domain/Org Config so the SPA re-derives Unlock order / startsWhen / FormPlan / dataReport; treating bulk Available bundles as the lasting home-Load or Commit shape

**Worklist**:
Go-projected operator board for one org: which Jobs may start (`canStart`); per Job, authored `taskIds`, `startLabel`, `repeatable`, and Config-free Task title + roles chrome; and per Job instance, each Task’s status as a viewer-scoped enum (`locked` | `unavailable` | `available` | `actionable` | `done`). Unlock/Submission law yields locked / unlocked / done; unlocked-not-done folds Task **readyWhen** (false ⇒ `unavailable`, skip role split); else unlocked splits by Config-role eligibility into `available` (not for this viewer) vs `actionable` (Per te). A Task is `done` after Submission on the instance. Home **Available** and mount gates read `actionable` (plus provisional Spawn-as-first-leaf when `canSpawn`); home **In progress** lists unfinished Job instances (open work still short of Config task completion) and deep-links to **Job instance overview**; `unavailable` is overview chrome only. The SPA does not re-derive eligibility or readyWhen from a parallel id list. For **Task groups**, home Available includes minted nested ready **child Tasks** and, when group chrome has `canSpawn`, a provisional first-leaf deep link (`new`). Living `/app` landing for the affiliated operator is this board (not legacy Operator home at `/my`): Available + Can start + In progress bands, not a full instance tree or `locked`/`done`/`available` rows as the home surface.
_Avoid_: Job surface; Chrome (FormPlan display catalog); SPA copies of Unlock / startsWhen; `viewerAvailable[]` as a second eligibility channel; row-id reopen for Task status; inventing Task order/labels from a client Org Config snapshot; Operator home (the `/my` Stream landing); client-side role filtering from a full Org Config; inventing Unlock client-side to fake `canSpawn`; treating In progress as the full instance tree on home

**Job instance overview**:
Operator screen for one Job instance: numbered accordion chrome for Tasks and **Task group** cards (group expand shows occurrences + Spawn/Close; occurrence expand shows **child Task** rows). `actionable` rows navigate to the Task form; `done` rows expand to a Submission facts recap plus any stored post-Commit links; handoff when work is `available` (not actionable) for this viewer. Used after Commit when auto-advance does not apply, after **child Task** Commit while the group stays open, and as an exit target from Task/Available and from home **In progress**. Not the Worklist and not an authoring surface. Home may deep-link Spawn via provisional first-leaf Available; Close remains here only.
_Avoid_: Stream timeline as the home board; process page; embedding the full instance tree on Worklist home (In progress is unfinished instances only, not the tree); putting Spawn on Worklist home as a peer of Can start Job; inline live forms on overview; re-running onSubmit.http to rebuild recap links

**Task form bundle**:
Minimum payload to mount one Task form: compiled schema + uiSchema, initial/seed data, projected read ids/columns when needed, Chrome and Derived *rules*, seal/chrome field ids, title, and a DomainSnapshot-shaped bindings bag (including `submissions`) for live Chrome/Derived — not the full Domain or authored Org Config Task document. On Load-of-Task and Commit, server-evaluated `dataReport` results land in `bindings.chromeDisplay` for SPA paint-only; draft-driven Chrome stays live on the client. Fetched addressed by `jobId` + `taskId` + `jobInstanceId` (provisional empty / `new` when starting); Commit may include the next bundle as `nextTaskFormBundle` when auto-advance applies. Home Load and Commit omit bulk Available bundles; Task routes mount from addressed fetch.
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
