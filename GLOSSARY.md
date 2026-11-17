# Org product (`/app`)

Single-organization operating picture: a versioned **Org Config** (roles +
jobs/tasks) and one accumulating **Domain** per org. Operators repeat **Jobs**
inside the config; they do not run the whole config as an instance. Product UI
for this context lives under `/app`.

Which glossary applies: [`GLOSSARY-MAP.md`](GLOSSARY-MAP.md) (shared org,
Org product, Legacy Stream, Feragame). Org product ADRs: [`docs/adr/`](docs/adr/)
(0005–0013). Shared membership:
[`docs/org/GLOSSARY.md`](docs/org/GLOSSARY.md).

## Language

### Org Config and domain

**Org Config**:
The versioned authored document for one Organization: role catalog plus keyed
Jobs, each with an ordered Task array, for that org’s whole operation.
_Avoid_: Stream; blueprint (when meaning this document); running the whole
config as an instance; multiple unrelated configs per org as the default story

**Config version**:
One immutable revision of an Org Config.
_Avoid_: Blueprint executable hash; treating every save as a new org

**Active Config version**:
The Config version the Organization uses for new work and for Load. Pinning
policy is separate.
_Avoid_: Implied “latest file on disk” without an org pointer

**Domain**:
The Organization’s accumulating facts: entities with stable ids plus append-only
submission history. One Domain per org.
_Avoid_: One Domain per whole-config execution; Stream instance; Process
progress bag

**Domain store**:
The store that holds Domain (entities + submit entries). Only config-declared
transforms write entities.
_Avoid_: Forms writing entities directly; free-form query language as the sole
mutator

### Jobs and tasks

**Job**:
A top-level authored chapter in the Org Config. Groups a Task tree; may be
**repeatable** (many Job instances). Jobs are not nested inside Jobs.
_Avoid_: Step (legacy experiment name); Stream; phase; nesting Jobs; treating a
Job as a single submit leaf

**Task**:
A **leaf** submit unit: schema, gates, Domain **reads** into the form, and
**Submission** via **Commit**. Authored **submit cardinality** is `once`
(default) or `many`. Names which **Config roles** may complete it.
_Avoid_: Substep (legacy experiment name); Form (when meaning this unit);
modeling multi-pass work as a Task group; leaf Job; task instance

**submit cardinality**:
Authored Task choice of `once` or `many`. **Once:** one successful Submission
finishes the Task on that Job instance. **Many:** each Commit appends a
Submission without finishing the Task from that Commit alone. See
[ADR-0008](docs/adr/0008-org-task-submit-cardinality.md).
_Avoid_: Repeatable Task groups for multi-submit; calling cardinality
“repeatable” (that word remains Job-level)

**child Task** *(historical / rejected)*:
Former name for a Task under a Task group; replaced by **submit cardinality**
`many` + **readyWhen**. See [ADR-0008](docs/adr/0008-org-task-submit-cardinality.md).
_Avoid_: Reviving child Tasks or group nesting; Subtask; task instance

**Task group** *(historical / rejected)*:
Former composite node under a Job; removed. Multi-pass same-form work is
**submit cardinality** `many` + **readyWhen**. See
[ADR-0008](docs/adr/0008-org-task-submit-cardinality.md).
_Avoid_: Calling a Task group a Task; jobs all the way down

**Spawn** *(historical / rejected)*:
Former affordance to open a Task group occurrence; removed with groups. See
[ADR-0012](docs/adr/0012-org-commit-mint-intent.md).
_Avoid_: Calling Job start “Spawn”; inventing Spawn chrome

**Close** *(historical / rejected)*:
Former affordance to end a Task group occurrence; removed with groups. See
[ADR-0008](docs/adr/0008-org-task-submit-cardinality.md).
_Avoid_: Reviving Close chrome; treating Close as entity open/closed law

**spawningNew** *(historical / rejected)*:
Former Commit mint kind for a Task group occurrence; removed with groups. See
[ADR-0012](docs/adr/0012-org-commit-mint-intent.md).
_Avoid_: Sending `spawningNew` on Commit

**Job instance**:
One occurrence of a repeatable Job within the org Domain. Often created on first
Task submit; id/label from config and/or submitted data. Distinct from Domain
entities the Config may mint for the same real-world beat.
_Avoid_: Step instance (legacy experiment name); whole-config execution
instance; Stream instance; Node (domain docs); treating the Job instance as a
substitute for a Domain entity

**Task path**:
Stable address of a Task under a Job (for example `intake/register`).
_Avoid_: Hard one-level child index as the only public address; bare substep id
without Job context when paths matter

**Unlock order**:
In-Job rule for which Tasks become completable after prior Submissions on the
same Job instance (optional predecessor). Distinct from **startsWhen** and
**readyWhen**.
_Avoid_: Calling unlock “startsWhen”; encoding entity open/close law in Unlock
when Domain status exists

**startsWhen**:
Job-level start gate over Submissions of a named Task. Distinct from Unlock
order and from Domain readiness (**readyWhen**).
_Avoid_: Locked (as the gate state word); encoding entity open/closed law;
availableWhen

**readyWhen**:
Task-level Domain readiness (boolean expression over Domain), combined with the
Task’s role gate. Distinct from Unlock order and **startsWhen**.
_Avoid_: availableWhen; hiding unavailable Tasks; binding operator identity into
the expression; overloading startsWhen or Unlock for Domain facts

**Domain reads**:
Authored projections that map current Domain entity ids into Task form maps.
Not a separate subsystem.
_Avoid_: Completeness or Leftovers as product grammar; conflating map reads with
Task group; Domain clear via Commit omit

**Submission**:
Record that a Task succeeded on a Job instance: time, actor, facts snapshot;
keyed with Task path. Unlock order and startsWhen read Submissions.
_Avoid_: Treating entity fields alone as completion; putting the submitter only
inside form facts

**Draft**:
Working data of an in-progress Task before Submission. Not part of Domain until
Commit succeeds.
_Avoid_: Submission; Domain

### Roles

**Role catalog**:
The set of **Config roles** declared on the Org Config. Distinct from the
Organization **Role catalog** of membership roles
([`docs/org/GLOSSARY.md`](docs/org/GLOSSARY.md)).
_Avoid_: Putting platform-only standing in the catalog; Stream role (legacy
blueprint gate name); conflating with Organization roles

**Config role**:
One named role in the Org Config catalog (slug). Tasks list which Config roles
may complete them.
_Avoid_: Stream role; conflating with Org admin standing; Operator (the default
membership role name) for every Config role; plant-named role branches in
executors

**Org catalogs** / `$catalogs`:
Authored bag on Org Config for stable org knowledge (lookups, enums, labels).
Not Domain entities, not the Role catalog, not Chrome.
_Avoid_: Treating catalogs as Domain kinds; conflating with Role catalog or
Chrome display catalog

### Authoring details

**bind**:
Authored transform step whose result replaces `$` for later ops in that list.
Does not emit Domain mutations itself.
_Avoid_: Conflating with form bindings, **reads**, or Org **catalogs**
(`$catalogs`)

**reads**:
Authored Domain query bindings used at open time to project into a Task — never
bake instance ids into the Org Config.
_Avoid_: Ref / `$refs` / `optionsFor` (retired kernel names)

**Derived value**:
Task field value from an expression under `derived`; may persist into Domain on
successful submit when configured.
_Avoid_: Putting GUI-only chrome under `derived`

**Chrome**:
Task-root display catalog (readonly, dataReport, …). Sibling to `derived`;
stripped before Submission facts. Not the operator board (see **Worklist**).
_Avoid_: Encoding live readouts as schema fields meant for submit; naming the
Worklist “Chrome”

**dynamicProperties**:
Field-keyed map config: each property names a Domain **read** whose ids fill
that map.
_Avoid_: form-level `propertyNamesFrom` (rejected); authored `completeness`

### Passport / DPP

**Passport**:
The issued Digital Product Passport for one Domain subject. Public credential
and human page are projections of a **Passport snapshot**, not a live Domain
walk. Org Config declares scheme; a Lock leaf opts in. See
[ADR-0007](docs/adr/0007-org-dpp-production.md).
_Avoid_: Stream Process DPP as the Org `/app` issue path; SPA-assembled passport
payloads; treating one plant’s YAML as the feature itself

**Passport lock**:
The operator Task that approves a subject’s gathered facts and issues its
**Passport**. Distinct from Worklist status `locked` and from form seal. See
[ADR-0007](docs/adr/0007-org-dpp-production.md).
_Avoid_: Auto-issue on Domain predicate alone; conflating with Unlock-order
`locked`; minting passport URLs via outbound HTTP

**Passport snapshot**:
Immutable projection of related Domain facts (and named Submissions) frozen
into the platform Passport record on Commit. Later Domain edits do not rewrite
what the Passport claims. See [ADR-0007](docs/adr/0007-org-dpp-production.md).
_Avoid_: Live Domain as the passport body; mutable “current view” credentials;
“passport” as just another Domain entity row

### Operator and seams

**Dumb executor**:
Platform executors only evaluate Org Config; plant business rules live in
Config, not hard-coded in executors. See [`AGENTS.md`](AGENTS.md).
_Avoid_: Plant- or domain-named branches in executors; exclusivity law outside
Org Config

**OperatorIdentity**:
Host-injected actor context for role gates, chrome, and Submission actor. Not
the Org Config document.
_Avoid_: Principal; forging actor via schema fields; Operator (catalog role
label) for this bag

**Commit navigation**:
Server-authored instruction after a successful Task Commit: advance to another
Task, return to **Worklist** home, or stay. See
[ADR-0006](docs/adr/0006-org-operator-worklist-home.md).
_Avoid_: Re-walking Worklist after Commit to pick destination; client-side
auto-advance; overview as Commit destination

**Advance target eligibility**:
Whether a given Task Commit may auto-advance into the next role-eligible Task
on that Job instance. See
[ADR-0006](docs/adr/0006-org-operator-worklist-home.md).
_Avoid_: Treating “next Task in Config order” as always correct after Commit;
plant-named routing branches in executors

**Commit**:
Authoritative accept of a sealed **Task** into Domain, addressed by Task path +
Job instance id. See [ADR-0005](docs/adr/0005-org-worklist-server-projection.md)
and [ADR-0006](docs/adr/0006-org-operator-worklist-home.md).
_Avoid_: Treating the SPA route handler as the Seam; folding Drafts into Commit;
SPA re-deriving post-Commit destination

**Load**:
Hydrate what the operator UI needs for the org: viewer-scoped **Worklist**,
viewer Config **roles**, and Active Config version metadata — not a dump of
full Domain + full Org Config. See
[ADR-0005](docs/adr/0005-org-worklist-server-projection.md) and
[ADR-0006](docs/adr/0006-org-operator-worklist-home.md).
_Avoid_: Hydrate by whole-config execution id; shipping full Domain/Org Config
so the SPA re-derives gates

**Worklist**:
Go-projected operator board for one org: which Jobs may start; per Job instance,
each Task’s viewer-scoped status (`locked` | `unavailable` | `available` |
`actionable` | `done`). Home is act-now; finished rediscovery is **Archive**.
See [ADR-0005](docs/adr/0005-org-worklist-server-projection.md) and
[ADR-0006](docs/adr/0006-org-operator-worklist-home.md).
_Avoid_: Job surface; Chrome; SPA copies of Unlock / startsWhen / readyWhen;
Operator home (the `/my` Stream landing); peer home bands for In progress /
Completed

**Archive**:
Operator surface listing past/completed **Job instances**. Each row opens the
**Job instance overview**. Distinct from **Worklist** home act-now.
_Avoid_: day→shift→lot Archive hierarchy; Archive bands on home

**Job instance overview**:
Read-only Job-instance surface reached from **Archive**. Not a **Commit
navigation**, Continue, home, or Task-back target. See
[ADR-0006](docs/adr/0006-org-operator-worklist-home.md).
_Avoid_: Commit or Continue navigation to overview; treating overview as the
operator board

**Task form bundle**:
Minimum payload to mount one Task form — not the full Domain or authored Org
Config Task document. See
[ADR-0005](docs/adr/0005-org-worklist-server-projection.md) and
[ADR-0006](docs/adr/0006-org-operator-worklist-home.md).
_Avoid_: Client compile from full authored Task + full Domain; calling this
Chrome

**AuthPort / AuthorizerPort**:
Who is calling (`OperatorIdentity`) vs who may complete a Task; Commit uses
both.
_Avoid_: Forging actor from sealed facts; treating an identity store as AuthPort

**HttpPort**:
Config-declared HTTP after successful Domain mutate; failures do not roll back
Domain. See [ADR-0010](docs/adr/0010-org-http-side-effects.md).
_Avoid_: Rolling back Domain on side-effect failure; `onSubmit.http` as the
product noun (that is the Config key)

### Legacy (other context — do not use for `/app` product prose)

**Stream / Stream instance / Process**:
Multi-party linear flow vocabulary for the Legacy `/my` surface. See
[`docs/stream/GLOSSARY.md`](docs/stream/GLOSSARY.md).
_Avoid_: Using these as names for Org Config, Domain, Job, or Task

### Rejected / historical code names

Do not revive as Org product nouns.

- **Whole-config execution** (historical name `Config run`): one “run of the
  config” with its own Domain — replaced by one org Domain + repeated Job
  instances.
- **`LoadRunPort` / run-id load shape**: replaced by org Domain Load/Commit
  ([ADR-0005](docs/adr/0005-org-worklist-server-projection.md)).
- **Step / Substep / Step instance**: experiment names for Job / Task / Job
  instance.
- **Task group / child Task / Spawn / Close / `spawningNew`**: removed; see
  [ADR-0008](docs/adr/0008-org-task-submit-cardinality.md) and
  [ADR-0012](docs/adr/0012-org-commit-mint-intent.md).
