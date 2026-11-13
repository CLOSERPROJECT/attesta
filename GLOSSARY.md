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
A **leaf** submit unit: schema, gates, Domain **reads** into the form, and **Submission** via **Commit**. Authored **submit cardinality** is `once` (default) or `many`. Names which **Config roles** may complete it. Cerbos Commit targets Tasks only.
_Avoid_: Substep (legacy experiment name); Form (when meaning this unit); modeling multi-pass work as a Task group; leaf Job; task instance

**submit cardinality**:
Authored Task key `submit: once | many` (omit ⇒ `once`). **Once:** one successful Submission finishes the Task on that Job instance (Worklist `done`). **Many:** each Commit appends a Submission; Worklist status stays Domain-gated (`readyWhen` + roles) and does not become `done` from a single Commit. Feragame multi-pass forms (lot register, weigh, progress, container register, start/close lot) are `many`.
_Avoid_: Repeatable Task groups for multi-submit; Close chrome on a many-submit Task; calling cardinality “repeatable” (that word remains Job-level)

**child Task**:
A **Task** authored under a **Task group**, completed on one occurrence of that group under a **Job instance**. Still a Task for Commit and Cerbos. Retained for engine compatibility; Feragame plant Config does not author groups for multi-pass work.
_Avoid_: Subtask; task instance; treating child Tasks as a different Commit kind; using child Tasks instead of `submit: many`

**Task group**:
A composite node under a Job: ordered **child Tasks**, may be repeatable; **no** Commit of its own. Engine may still project Spawn/Close; **not** the Feragame model for N submits of the same form (use **submit cardinality** `many` + **readyWhen**).
_Avoid_: Calling a Task group a Task; jobs all the way down; faking multi-pass with groups; treating Close as required plant law for many-submit leaves

**Spawn**:
Open a new occurrence of a **Task group** under a **Job instance** by Committing its first **child Task** with spawn mint (`spawningNew`). Engine affordance for configs that still author groups; Feragame multi-pass does not use Spawn.
_Avoid_: Empty-open without the first leaf Submission as the normal operator path; calling Job start “Spawn”; inventing a home Spawn peer of Can start; Task instance; using Spawn for many-submit leaf Tasks

**Job instance**:
One occurrence of a repeatable Job within the org Domain (e.g. one truck, one line shift). Often created on first Task submit; id/label from config and/or submitted data. Distinct from Domain entities such as **working day** and **shift** (a line Job instance may share id with its shift entity).
_Avoid_: Step instance (legacy experiment name); whole-config execution instance; Stream instance; Node (domain docs); treating the Job instance as a substitute for a Domain entity

**Task path**:
Stable address of a Task (or later a node) under a Job, e.g. `intake/palletWeight`. Commit and Submission keys use Task path plus Job instance id so deeper trees stay additive.
_Avoid_: Hard one-level child index as the only public address; bare substep id without Job context when paths matter

**Unlock order**:
In-Job rule for which Tasks become completable after prior Submissions on the same Job instance (optional `unlockFrom` names a predecessor). Distinct from **startsWhen** (Job start) and **readyWhen** (Domain readiness). Engine keeps the mechanic; Feragame plant Config prefers **readyWhen** whenever Domain already holds the prerequisite and does not author `unlockFrom` (likely removed later).
_Avoid_: Calling unlock “startsWhen”; Seal-gated unlock; encoding plant open/close or lot-session law in Unlock when Domain status exists

**startsWhen**:
Job-level start gate: quantifier (`some` | `all`) over Submissions of a named Task. Distinct from Unlock order and from Domain readiness. Unused for plant working-day / shift exclusivity (those are Domain facts via **readyWhen**).
_Avoid_: Locked (as the gate state word); Seal-gated start; encoding entity open/closed or “previous closed” law; availableWhen

**readyWhen**:
Task-level Domain readiness (boolean expression over Domain). Combined with the Task’s role gate. When false: visible but unavailable. Primary plant gate for Domain entity state (open working day / shift, lot session, eligible rows) and for which actions are available without a fixed Unlock pipeline.
_Avoid_: availableWhen; hiding unavailable Tasks; binding operator identity into the expression; overloading startsWhen or Unlock for Domain facts

**Domain reads** (map/table Tasks):
Authored `reads` + `dynamicProperties` that project current entity ids into a form map (row projection / `rowIds` on the Task form bundle). Projection **materializes** those ids into concrete `schema.properties`; authored `additionalProperties` is the per-id **template** only (`additionalProperties: false` after project). Full-map Tasks list the map property in `schema.required` so every current read id is required; partial maps omit that and leave untouched keys unforced. Not a separate subsystem; not spawn-many of full child registers. Two shapes share one authored `ui:field: mapTable`: a **cell map** (template scalar — one value column, sjsf field per cell schema) and an **object map** (template object — property columns, sjsf field per property). Cell enums are **real values only** (`partial`, `done`, …) — no `null` in enum. Stock sjsf clearable is for draft fix only: emptying a cell drops the key (`omit`/`undefined`); Commit treats omit as untouched (set-or-skip). Never unset Domain facts via Commit.
_Avoid_: Completeness `all`/`some` as product grammar; Leftovers as product grammar; SPA twin Domain-read id helpers; SPA cell-map normalize; densify first-enum without `default`; densify `default: null` into Draft cell maps; form-level `""` as untouched sentinel; conflating map reads with Task group; dual markers (`rowTable`); table-owned enum paint; `enum: [null,…]` Domain clear; stamping `clearable: false` for enum-null law; `anyOf` null|enum as product clear; Commit nil as Domain unset

**Submission**:
Record that a Task succeeded on a Job instance: time, actor, facts snapshot; keyed with Task path. Unlock order and startsWhen read Submissions. Worklist `done` applies after one successful Submission only for **submit** `once`; for `many`, Submissions append while status stays readyWhen-gated. Post-Commit HTTP **links** are not facts: they are stored in a separate step after Commit (+ HttpPort), and may appear on overview recap without replaying HTTP.
_Avoid_: Treating entity fields alone as completion; putting the submitter only inside form facts; row-id reopen as done law for once Tasks; folding link persistence into the facts Commit write; HTTP replay to rebuild links

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

**lot** (entity):
Plant Domain entity for one FIR/intake lot (formerly named **load**). Owns pallets and is the passport subject. Line work runs inside a **lot session** status on this entity.
_Avoid_: load (living plant name); conflating with Job instance; treating lot as a Task group occurrence

**lot session status** (entity field):
Single enum on a **lot**: `open` | `parked` | `fullyProcessed`. `startLot` → `open`; `closeTurno` while `open` → `parked`; next `openTurno` resumes `parked` → `open`; `closeLot` → `fullyProcessed` when every pallet on that lot has weight and `progress = done`. At most one lot is `open` or `parked` under the open working day. `parked` freezes line Tasks for that lot until resume.
_Avoid_: Separate boolean fullyProcessed beside open/parked; forcing lot completion before shift close; parallel open lots; engine special-case on the field name

**fullyProcessed** (lot session status value):
Terminal **lot session status**: the lot is closed for line processing. Set only by **closeLot** Commit transform—not by container register. Passport eligibility still requires this status (and not locked). Enums/filters that previously meant “register closure” now mean this terminal status.
_Avoid_: boolean `processed` as Worklist status; using fullyProcessed as unlock grammar; using fullyProcessed as “Passport done” alone without the lock fact; containerReg as the writer of this status

**incomplete** (entity field):
Plant Domain fact on a **container**: booked weight for the current lot is frozen while the physical bin may continue on a later register. Distinct from lot session status.
_Avoid_: Treating incomplete as Worklist/progress; engine special-case on the name (Config transform + form only); Task-group Close

**working day** (entity):
Plant Domain entity for one calendar date of plant operation. Id is that date. Mode is multi-shift or **whole-day**; status is open or closed; closed is terminal for that date. At most one working day is open; intake binds to the open working day, not to a shift.
_Avoid_: Date form fields alone as the day; conflating with Job instance; giornata intera as a fourth turno peer of 1°/2°/3° without this entity; engine special-case on the name

**shift** (entity):
Plant Domain entity nested under a **working day**. Line work references the open shift; line Job instance id matches the shift id. Status is open or closed; Config allows at most one open shift at a time (catalog order 1°→2°→3° not required yet).
_Avoid_: Equating shift with Job instance with no Domain row; binding intake to shift; engine special-case on the name

**whole-day** (working day mode):
Working day mode that mints exactly one shift on the same open Commit; further shift opens are gated off in Config. Multi-shift days mint only the day until a later shift open.
_Avoid_: whole-day as only a turno enum beside 1st/2nd/3rd; auto-creating a line Job instance outside Org Config transforms

**continuesContainerId** (Config form fact):
Hidden row field on plant register: links a continuation row to a still-open incomplete Domain container. Load seeds it via Task `initialData`; Commit transform books delta weight and supersedes the prior incomplete. Not a Domain entity field. Operator visibility uses generic `ui:options.presenceMark` on `hiddenField` (Config title such as Continua)—not a second schema fact.
_Avoid_: Engine special-case on the name; free-picked secondary lot enum for continuity; storing continuesContainerId on minted containers; SPA branches on this field name beyond the generic hidden+presenceMark contract

### Passport (living)

**Passport**:
The issued Digital Product Passport for one Domain subject (in Feragame: one **lot**). Public credential and human page are projections of a **Passport snapshot**, not a live Domain walk. **Passport production** is a core living platform capability. Org Config declares scheme in a root **`passport`** block (subject binding, identity, snapshot projection); a Lock leaf opts in with thin **`onSubmit.passport`** (not HttpPort).
_Avoid_: Stream Process DPP as the living issue path; SPA-assembled passport payloads; treating Feragame YAML as the feature itself; gather graph only on the Task with no root scheme; Job-level “produces DPP” with no Commit leaf; naming the subject load

**Passport lock**:
The operator Task that approves a subject’s gathered facts and issues its **Passport**: Load shows eligibility + recap; one Commit runs transform (Domain locked fact) then the Passport producer. Dedicated repeatable Job; Job instance id is the subject id. Distinct from Worklist status `locked` and from form seal. Post-Commit Digital Link / DPP links come from the producer, not **HttpPort** / `onSubmit.http`.
_Avoid_: Auto-issue on Domain predicate alone; treating lot session `fullyProcessed` as passport issued; Lock as a new HTTP Seam; conflating with Unlock-order `locked`; minting passport URLs via outbound HTTP; producer as sole writer of eligibility Domain facts

**Passport snapshot**:
Immutable projection of related Domain facts (and named Submissions) from the root `passport.snapshot` declaration. Evaluated live on **Passport lock** Load for the operator recap; frozen into the platform Passport record (org + subject) on Commit. Later Domain edits do not rewrite what the Passport claims.
_Avoid_: Live Domain as the passport body; a second hand-authored recap expr that can drift from freeze; mutable “current view” credentials; “passport” as just another Feragame entity row

### Operator and seams

**Dumb executor**:
Go and the living SPA only evaluate Org Config and project Worklist/forms; plant business rules (entity shapes, open/close transforms, readyWhen) are authored in Config, not hard-coded in executors.
_Avoid_: Feragame-named branches in Go or SPA; day/shift exclusivity outside Org Config; “the engine knows what a turno is”

**OperatorIdentity**:
Host-injected `{ userId, org, roles }` for role gates, chrome, and Submission actor. Not the Org Config document.
_Avoid_: Principal; forging actor via schema fields; Operator (catalog role label) for this bag

**Commit navigation**:
Server-authored instruction in the Commit response telling the SPA where to go after a successful Task Commit: advance to another Task on the same Job instance, hand off to **Job instance overview**, return to **Worklist** home, or **stay** on the submitted form. On stay, Commit also carries a deferred **Continue** target (overview, home, or next Task) so the operator can leave the parked screen without the client guessing. Evaluated in Go from Org Config and Worklist eligibility; the SPA follows Commit only.
_Avoid_: Re-walking Worklist after Commit to pick overview vs home; inferring stay from HTTP links in the SPA; client-side auto-advance

**Advance target eligibility**:
Whether a given Task Commit may auto-advance the operator into the next role-eligible Task on that Job instance. Close and similar leaves disable it so linear sibling order does not chain into the wrong screen; open-style leaves may force **Worklist** home instead. When advance is disallowed and no stronger navigation is authored, hand off to **Job instance overview**.
_Avoid_: Treating “next Task in Config order” as always correct after Commit; plant-named routing branches in executors

**Commit**:
Authoritative accept of a sealed **Task** into Domain (cmd-bag), addressed by Task path + Job instance id. Production allow/deny is Cerbos; HTTP handlers are Adapters only. The Commit response includes an updated viewer-scoped **Worklist**, explicit **Commit navigation** (including stay with Continue when authored), and when advancing to a Task, the next **Task form bundle**, so the SPA navigates without a blind re-Load or Worklist re-walk. If navigation is stay, the operator remains on a read-only view of the submitted form with any download links (many-submit loops typically stay). Configs that still author **Task groups** may also offer Spawn/Close on stay; Feragame multi-pass uses stay without group chrome.
_Avoid_: Treating the SPA route handler as the Seam; folding Drafts into Commit; Committing a Task group; SPA re-deriving post-Commit destination from Worklist or from link presence

**Load**:
Hydrate what the operator UI needs for the org: viewer-scoped **Worklist**, viewer Config **roles** (for Available affordances), and Active Config version metadata for the home board; not a dump of full Domain + full Org Config. **Task form bundle**s load when opening a Task route via addressed fetch (`GET /api/living/task-form`; provisional `jobInstanceId` empty / `new`) and may return on Commit when **Commit navigation** advances to a Task — home Load omits bulk bundles (ADR-0006). SPA post-Commit routing consumes Commit navigation (+ bundle when advancing to a Task).
_Avoid_: Hydrate by whole-config execution id; `LoadRunPort` / `/api/runs/...` as product shape; treating Kit load functions as the Seam; shipping full Domain/Org Config so the SPA re-derives Unlock order / startsWhen / FormPlan / dataReport; treating bulk Available bundles as the lasting home-Load or Commit shape

**Worklist**:
Go-projected operator board for one org: which Jobs may start (`canStart`); per Job, authored `taskIds`, `startLabel`, `repeatable`, and Config-free Task title + roles chrome; and per Job instance, each Task’s status as a viewer-scoped enum (`locked` | `unavailable` | `available` | `actionable` | `done`). Unlock/Submission law yields locked / unlocked; **submit once** may become `done` after Submission; **submit many** never uses Submission alone for `done`. Unlocked-not-done folds Task **readyWhen** (false ⇒ `unavailable`, skip role split); else unlocked splits by Config-role eligibility into `available` (not for this viewer) vs `actionable` (Per te). Home **Available** and mount gates read `actionable`; home **In progress** lists unfinished Job instances and deep-links to **Job instance overview**; `unavailable` is overview chrome only. The SPA does not re-derive eligibility or readyWhen from a parallel id list. Living `/app` landing for the affiliated operator is this board (not legacy Operator home at `/my`): Available + Can start + In progress bands, not a full instance tree.
_Avoid_: Job surface; Chrome (FormPlan display catalog); SPA copies of Unlock / startsWhen / readyWhen; `viewerAvailable[]` as a second eligibility channel; inventing Task order/labels from a client Org Config snapshot; Operator home (the `/my` Stream landing); treating In progress as the full instance tree on home

**Job instance overview**:
Operator screen for one Job instance: numbered accordion chrome for leaf Tasks (and, where a config still authors them, **Task group** cards). `actionable` rows navigate to the Task form; `done` rows expand to a Submission facts recap plus any stored post-Commit links; handoff when work is `available` (not actionable) for this viewer. Used after Commit when auto-advance does not apply and as an exit target from Task/Available and from home **In progress**. Not the Worklist and not an authoring surface. Feragame flattened Jobs show leaf Tasks only.
_Avoid_: Stream timeline as the home board; process page; embedding the full instance tree on Worklist home; requiring Spawn/Close for many-submit loops; inline live forms on overview; re-running onSubmit.http to rebuild recap links

**Task form bundle**:
Minimum payload to mount one Task form: compiled schema + uiSchema, initial/seed data, projected read ids/columns when needed, Chrome and Derived *rules*, seal/chrome field ids, title, and a DomainSnapshot-shaped bindings bag (including `submissions`) for live Chrome/Derived — not the full Domain or authored Org Config Task document. On Load-of-Task and Commit, server-evaluated `dataReport` results land in `bindings.chromeDisplay` for SPA paint-only; draft-driven Chrome stays live on the client. Fetched addressed by `jobId` + `taskId` + `jobInstanceId` (provisional empty / `new` when starting); Commit may include the next bundle when **Commit navigation** advances or when Continue targets a Task. Home Load and Commit omit bulk Available bundles; Task routes mount from addressed fetch.
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
- **load** (plant entity): renamed to **lot**; status enum replaces boolean fullyProcessed (ADR-0008).
- **Repeatable Task groups for Feragame multi-pass**: superseded by **submit cardinality** `many` + lot session (ADR-0008).
