# Org (`/app`)

Single-organization operating picture: a versioned **Org Config** and one
accumulating **Domain** per org. Operators repeat **Jobs** inside the config;
they do not run the whole config as an instance. Product UI under `/app`.

**Org** and **Stream** are different product surfaces—see root
[`GLOSSARY.md`](../../GLOSSARY.md). Stream:
[`docs/stream/GLOSSARY.md`](../stream/GLOSSARY.md). Shared membership: root
glossary. Map: [`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md). Decisions:
[`docs/adr/`](../adr/) (0005–0013). Config authoring:
[`org-config-authoring.md`](../agents/org-config-authoring.md).

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
The persistence of Domain (entities + submit entries).
_Avoid_: Forms writing entities directly; free-form query language as the sole
mutator

### Jobs and tasks

**Job**:
A top-level authored chapter in the Org Config. Groups a Task tree; may be
**repeatable** (many Job instances). Jobs are not nested inside Jobs.
_Avoid_: Step (legacy experiment name); Stream; phase; nesting Jobs; treating a
Job as a single submit leaf

**Task**:
A leaf submit unit under a Job: gated work that yields a **Submission** via
**Commit**. Authored **submit cardinality** is `once` (default) or `many`.
Names which **Config roles** may complete it.
_Avoid_: Substep (legacy experiment name); Form (when meaning this unit);
modeling multi-pass work as a Task group; leaf Job; task instance

**submit cardinality**:
Authored Task choice of `once` or `many`. **Once:** one successful Submission
finishes the Task on that Job instance. **Many:** each Commit appends a
Submission without finishing the Task from that Commit alone. See
[ADR-0008](../adr/0008-org-task-submit-cardinality.md).
_Avoid_: Repeatable Task groups for multi-submit; calling cardinality
“repeatable” (that word remains Job-level)

**Job instance**:
One occurrence of a repeatable Job within the org Domain. Distinct from Domain
entities the Config may mint for the same real-world beat.
_Avoid_: Step instance (legacy experiment name); whole-config execution
instance; Stream instance; Node (domain docs); treating the Job instance as a
substitute for a Domain entity

**Task path**:
Stable address of a Task under a Job (Job key + Task key).
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
Authored projections of current Domain entity ids into Task form maps. Not a
separate subsystem.
_Avoid_: Completeness or Leftovers as product grammar; conflating map reads with
Task group; Domain clear via Commit omit

**Submission**:
Record that a Task succeeded on a Job instance: time, actor, facts snapshot;
keyed with Task path.
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
([`GLOSSARY.md`](../../GLOSSARY.md)).
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

### Authoring vocabulary

Config transform and form nouns. Mechanics:
[`org-config-authoring.md`](../agents/org-config-authoring.md).

**bind**:
Authored transform step that rebinds the working value for later steps in that
list. Does not itself write Domain.
_Avoid_: Conflating with form bindings, **reads**, or Org **catalogs**

**reads**:
Authored Domain query bindings that project into a Task form. Never bake
instance ids into the Org Config.
_Avoid_: Ref / `$refs` / `optionsFor` (retired kernel names)

**Derived value**:
Task field value from an expression under `derived`.
_Avoid_: Putting GUI-only chrome under `derived`

**Chrome**:
Task-root display catalog (not submit facts). Not the operator board (see
**Worklist**).
_Avoid_: Encoding live readouts as schema fields meant for submit; naming the
Worklist “Chrome”

**dynamicProperties**:
Authored map whose keys come from Domain **reads**.
_Avoid_: form-level `propertyNamesFrom` (rejected); authored `completeness`

### Passport / DPP

**Passport**:
The issued Digital Product Passport for one Domain subject. Public credential
and human page are projections of a **Passport snapshot**, not a live Domain
walk. See [ADR-0007](../adr/0007-org-dpp-production.md).
_Avoid_: Stream Process DPP as the Org `/app` issue path; SPA-assembled passport
payloads; treating one plant’s YAML as the feature itself

**Passport lock**:
The operator Task that approves a subject’s gathered facts and issues its
**Passport**. Distinct from Worklist status `locked` and from form seal. See
[ADR-0007](../adr/0007-org-dpp-production.md).
_Avoid_: Auto-issue on Domain predicate alone; conflating with Unlock-order
`locked`

**Passport snapshot**:
Immutable projection of related Domain facts frozen into the Passport record on
Commit. Later Domain edits do not rewrite what the Passport claims. See
[ADR-0007](../adr/0007-org-dpp-production.md).
_Avoid_: Live Domain as the passport body; mutable “current view” credentials;
“passport” as just another Domain entity row

### Operator surfaces

**Dumb executor**:
Platform code only evaluates Org Config; plant business rules live in Config.
See [`AGENTS.md`](../../AGENTS.md).
_Avoid_: Plant- or domain-named branches in executors; exclusivity law outside
Org Config

**OperatorIdentity**:
The actor context for role gates and Submission actor. Not the Org Config
document.
_Avoid_: Principal; Operator (catalog role label) for this bag

**Commit navigation**:
Where the operator goes after a successful Task Commit. See
[ADR-0006](../adr/0006-org-operator-worklist-home.md).
_Avoid_: Client-side auto-advance; overview as Commit destination

**Advance target eligibility**:
Whether a Task Commit may auto-advance into another Task on that Job instance.
See [ADR-0006](../adr/0006-org-operator-worklist-home.md).
_Avoid_: Treating “next Task in Config order” as always correct after Commit

**Commit**:
Authoritative accept of a sealed **Task** into Domain. See
[ADR-0005](../adr/0005-org-worklist-server-projection.md) and
[ADR-0006](../adr/0006-org-operator-worklist-home.md).
_Avoid_: Folding Drafts into Commit; SPA re-deriving post-Commit destination

**Load**:
Hydration of what the operator UI needs for the org (not a full Domain + Config
dump). See [ADR-0005](../adr/0005-org-worklist-server-projection.md).
_Avoid_: Hydrate by whole-config execution id

**Worklist**:
Operator board for one org: act-now Jobs and Task statuses. Finished rediscovery
is **Archive**. See [ADR-0005](../adr/0005-org-worklist-server-projection.md) and
[ADR-0006](../adr/0006-org-operator-worklist-home.md).
_Avoid_: Job surface; Chrome; Operator home (the `/my` Stream landing); peer home
bands for In progress / Completed

**Archive**:
Operator surface listing past/completed **Job instances**. Each row opens the
**Job instance overview**. Distinct from **Worklist** home act-now.
_Avoid_: day→shift→lot Archive hierarchy; Archive bands on home

**Job instance overview**:
Read-only Job-instance surface reached from **Archive**. Not a Commit, Continue,
home, or Task-back target. See
[ADR-0006](../adr/0006-org-operator-worklist-home.md).
_Avoid_: Treating overview as the operator board

**Task form bundle**:
Payload to mount one Task form. See
[ADR-0005](../adr/0005-org-worklist-server-projection.md).
_Avoid_: Calling this Chrome

**AuthPort** / **AuthorizerPort**:
Seams for who is calling vs who may complete a Task. See
[ADR-0005](../adr/0005-org-worklist-server-projection.md).
_Avoid_: Forging actor from sealed facts

**HttpPort**:
Config-declared HTTP after a successful Domain mutate. See
[ADR-0010](../adr/0010-org-http-side-effects.md).
_Avoid_: `onSubmit.http` as the product noun (that is the Config key)

## Historical / rejected

Do not revive as Org nouns. Living multi-pass work is **submit cardinality**
`many` + **readyWhen** (see [ADR-0008](../adr/0008-org-task-submit-cardinality.md)).

**child Task**:
Former name for a Task under a Task group; replaced by **submit cardinality**
`many` + **readyWhen**.
_Avoid_: Reviving child Tasks or group nesting; Subtask; task instance

**Task group**:
Former composite node under a Job; removed.
_Avoid_: Calling a Task group a Task; jobs all the way down

**Spawn** / **Close** / **spawningNew**:
Former Task-group affordances and Commit mint kind; removed with groups. See
[ADR-0008](../adr/0008-org-task-submit-cardinality.md) and
[ADR-0012](../adr/0012-org-commit-mint-intent.md).

**Whole-config execution** (historical name `Config run`):
One “run of the config” with its own Domain — replaced by one org Domain +
repeated Job instances.

**LoadRunPort**:
Retired load seam name; living nouns are Domain **Load** / **Commit**.

**Step** / **Substep** / **Step instance**:
Experiment names for Job / Task / Job instance.
