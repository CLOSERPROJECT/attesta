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
A **leaf** submit unit: schema, gates, leftovers, and one successful **Submission** via **Commit**. Names which **Config roles** may complete it. Cerbos Commit targets Tasks only.
_Avoid_: Substep (legacy experiment name); Form (when meaning this unit); composite/spawn-many nodes (see Task group); leaf Job

**Task group**:
_Deferred product name (working label)._ A composite node under a Job: children, may be repeatable with parent-scoped instances; **no** Commit of its own. Spawn-many (e.g. lot under truck). Not implemented in the flat port.
_Avoid_: Calling a Task group a Task; jobs all the way down; faking spawn-many with leftovers `completeness: some`

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

**Leftovers**:
Map/table-style Task progress over many Domain entities in one Task. Authored **completeness**: `all` (every eligible entity; may unlock later work / be a `startsWhen` target) or `some` (batch progress only; must not unlock next or be a `startsWhen` target). Not spawn-many of full child registers.
_Avoid_: Using `completeness: some` for unknown-N full lots/registers; conflating leftovers with Task group

**Submission**:
Record that a Task succeeded on a Job instance: time, actor, facts snapshot; keyed with Task path. Unlock order and startsWhen read Submissions.
_Avoid_: Treating entity fields alone as completion; putting the submitter only inside form facts

**Draft**:
Working data of an in-progress Task before Submission. Not part of Domain until Commit succeeds.
_Avoid_: Submission; Domain

### Roles

**Role catalog** (living):
The set of **Config roles** declared on the Org Config.
_Avoid_: Putting platform-only standing in the catalog; Stream role (legacy blueprint gate name)

**Config role**:
One named role in the Org Config catalog (slug). Tasks list which Config roles may complete them; Cerbos allow/deny uses principal roles matched by slug.
_Avoid_: Stream role; conflating with Org admin standing; Operator (the default membership role name) for every Config role

### Authoring details

**reads**:
Authored Domain query bindings used at open time to project enums / property names into a Task — never bake instance ids into the Org Config.
_Avoid_: Ref / `$refs` / `optionsFor` (retired kernel names)

**Derived value**:
Task field value from an expression under `derived`; may persist into Domain on successful submit when configured.
_Avoid_: Putting GUI-only chrome under `derived`

**Chrome**:
Task-root display catalog (print, readonly, recap, …). Sibling to `derived`; stripped before Submission facts. Not the operator board (see **Worklist**).
_Avoid_: Encoding live readouts as schema fields meant for submit; naming the Worklist “Chrome”

**dynamicProperties**:
Field-keyed leftover/map config for leftover-style Tasks (see Leftovers).
_Avoid_: form-level `propertyNamesFrom` (rejected)

**progress** (entity field):
Plant-style availability on some entities: unset = available; `partial`; `done`. Draft ≠ partial.
_Avoid_: boolean `processed` as the living leftover gate

### Operator and seams

**OperatorIdentity**:
Host-injected `{ userId, org, roles }` for role gates, chrome, and Submission actor. Not the Org Config document.
_Avoid_: Principal; forging actor via schema fields; Operator (catalog role label) for this bag

**Commit**:
Authoritative accept of a sealed **Task** into Domain (cmd-bag), addressed by Task path + Job instance id. Production allow/deny is Cerbos; HTTP handlers are Adapters only.
_Avoid_: Treating the SPA route handler as the Seam; folding Drafts into Commit; Committing a Task group

**Load**:
Hydrate what the operator UI needs for the org: **Worklist**, **Task form bundle**s for available work, and Active Config version metadata — not a dump of full Domain + full Org Config for client-side gate or FormPlan compile.
_Avoid_: Hydrate by whole-config execution id; `LoadRunPort` / `/api/runs/...` as product shape; treating Kit load functions as the Seam; shipping full Domain/Org Config so the SPA re-derives Unlock order / startsWhen / FormPlan

**Worklist**:
Go-projected operator board for one org: which Jobs may start (`canStart`); per Job, authored `taskIds`, `startLabel`, and `repeatable` for Config-free chrome; and per Job instance, each Task’s status (`locked` | `available` | `done`). Unlock order, startsWhen, and Leftovers openness are evaluated only on the server; the UI reads the projection and does not re-implement those rules.
_Avoid_: Job surface; Chrome (FormPlan display catalog); SPA copies of Unlock / startsWhen; a second boolean visibility matrix beside status; inventing Task order/labels from a client Org Config snapshot

**Task form bundle**:
Minimum payload to mount one Task form: compiled schema + uiSchema, initial/seed data, leftover ids/columns when needed, Chrome and Derived *rules*, seal/chrome field ids, title, and a DomainSnapshot-shaped bindings bag (including `submissions`) for live Chrome/Derived — not the full Domain or authored Org Config Task document. Load/Commit ship a set for every Worklist-`available` Task plus a provisional bundle (`jobInstanceId` empty) when `canStart`.
_Avoid_: Client compile from full authored Task + full Domain; calling this Chrome; baking live Chrome results as static strings when the draft must still drive them

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
