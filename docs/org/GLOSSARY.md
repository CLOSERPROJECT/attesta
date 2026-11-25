# Org (`/app`)

Single-organization operating picture: one or more **Org streams**, each a versioned **Org Config** with its own accumulating **Domain**. Operators work **Org stream instances** (filtered views over Domain) in `/app`; discovery and timelines live on `/my`. See [ADR-0014](../adr/0014-org-config-as-stream.md).

**Org** and **Stream** remain different product nouns—see root [`GLOSSARY.md`](../../GLOSSARY.md). Formata Stream runtime: [`docs/stream/GLOSSARY.md`](../stream/GLOSSARY.md). Shared membership: root glossary. Map: [`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md). Decisions: [`docs/adr/`](../adr/) (0005–0016). Config authoring: [`org-config-authoring.md`](../agents/org-config-authoring.md).

## Language

### Org Config and domain

**Org Config**: The versioned authored document for one **Org stream**: role catalog plus keyed Jobs, each with an ordered Task array, plus a `stream` section for `/my` presentation.\
_Avoid_: Formata Stream blueprint (when meaning this document); running the whole config as an instance

**Config version**: One immutable revision of an Org Config.\
_Avoid_: Blueprint executable hash; treating every save as a new org

**Active Config version**: The Config version the Org stream uses for new work and for Load. Pinning policy is separate.\
_Avoid_: Implied “latest file on disk” without an (org, stream) pointer

**Domain**: The Org stream’s accumulating facts: entities with stable ids plus append-only submission history. One Domain per Org stream.\
_Avoid_: One Domain per Org stream instance; Formata Stream instance; Process progress bag

**Domain store**: The persistence of Domain (entities + submit entries). Commit writes are optimistic: concurrent writers retry the Commit's gates against the current **Domain revision** instead of overwriting.\
_Avoid_: Forms writing entities directly

**Domain revision**: The count of writes to one Org stream's Domain, +1 per write; boards compare it to know their projections are stale. See [ADR-0015](../adr/0015-org-domain-change-feed.md).\
_Avoid_: version (Config version is a different noun), etag

### Org stream

**Org stream**: One Org Config file exposed as a Stream on `/my`: its own Active Config version and Domain, isolated from other streams of the same org. See [ADR-0014](../adr/0014-org-config-as-stream.md).\
_Avoid_: Formata Stream blueprint; treating all of an org’s Config as one stream; sharing entities across streams

**Org stream instance**: A filtered view over an Org stream’s Domain, anchored on one **anchor Job** instance plus the Job instances its `each` steps select—not a Domain of its own. See [ADR-0014](../adr/0014-org-config-as-stream.md).\
_Avoid_: Domain per instance; Formata Stream instance; whole-config execution

**anchor Job**: The Job whose instances define Org stream instances (Play starts that Job’s start Task).\
_Avoid_: Treating every Job as an instance boundary; Formata Step as the anchor

**each**: Authored on a stream step: the step repeats once per Domain entity matching the expression, as one timeline row (or group) per match.\
_Avoid_: submit cardinality `many`; nesting Jobs

**blockedReason**: Authored expression that explains why a Task is not ready, projected onto the `/my` timeline when the Task is blocked.\
_Avoid_: Encoding the gate itself (that stays **readyWhen** / **startsWhen**)

### Jobs and tasks

**Job**: A top-level authored chapter in the Org Config. Groups a Task tree; may be **repeatable** (many Job instances). Jobs are not nested inside Jobs.\
_Avoid_: Step (legacy experiment name); Formata Stream; phase; nesting Jobs; treating a Job as a single submit leaf

**Task**: A leaf submit unit under a Job: gated work that yields a **Submission** via **Commit**. Authored **submit cardinality** is `once` (default) or `many`. Names which **Config roles** may complete it.\
_Avoid_: Substep (legacy experiment name); Form (when meaning this unit); modeling multi-pass work as a Task group; leaf Job; task instance

**submit cardinality**: Authored Task choice of `once` or `many`. **Once:** one successful Submission finishes the Task on that Job instance. **Many:** each Commit appends a Submission without finishing the Task from that Commit alone. See [ADR-0008](../adr/0008-org-task-submit-cardinality.md).\
_Avoid_: Repeatable Task groups for multi-submit; calling cardinality “repeatable” (that word remains Job-level)

**Job instance**: One occurrence of a repeatable Job within the Org stream Domain. Distinct from Domain entities the Config may mint for the same real-world beat.\
_Avoid_: Step instance (legacy experiment name); whole-config execution instance; Formata Stream instance; Node (domain docs); treating the Job instance as a substitute for a Domain entity

**finished** (Job instance): Platform rule—a Job instance is finished when every `submit: once` Task on it is `done`. Drives the `/my` timeline and done filter; no separate `doneWhen` expression.\
_Avoid_: Domain-predicate-only finished; treating `submit: many` Tasks as required for finished

**Task path**: Stable address of a Task under a Job (Job key + Task key).\
_Avoid_: Hard one-level child index as the only public address; bare substep id without Job context when paths matter

**startsWhen**: Job-level start gate over Submissions of a named Task. Distinct from Domain readiness (**readyWhen**).\
_Avoid_: Locked (as the gate state word); encoding entity open/closed law; availableWhen; calling startsWhen “unlock”

**readyWhen**: Task-level Domain readiness (boolean expression over Domain), combined with the Task’s role gate. Distinct from **startsWhen**. Primary gate for Domain entity state and sibling/Submission prerequisites.\
_Avoid_: availableWhen; hiding unavailable Tasks; binding operator identity into the expression; overloading startsWhen for Domain facts; calling readyWhen “unlock”

**Submission**: Record that a Task succeeded on a Job instance: time, actor, facts snapshot, and entitiesTouched (mint/patch of Domain entities by kind + id); keyed with Task path.\
_Avoid_: Treating entity fields alone as completion; putting the submitter only inside form facts

**Draft**: Working data of an in-progress Task before Submission. Not part of Domain until Commit succeeds.\
_Avoid_: Submission; Domain

### Roles

**Role catalog**: The set of **Config roles** declared on the Org Config. Distinct from the Organization **Role catalog** of membership roles ([`GLOSSARY.md`](../../GLOSSARY.md)).\
_Avoid_: Putting platform-only standing in the catalog; Stream role (legacy blueprint gate name); conflating with Organization roles

**Config role**: One named role in the Org Config catalog (slug). Tasks list which Config roles may complete them.\
_Avoid_: Stream role; conflating with Org admin standing; Operator (the default membership role name) for every Config role; plant-named role branches in executors

**Org catalogs** / `$catalogs`: Authored bag on Org Config for stable org knowledge (lookups, enums, labels). Not Domain entities, not the Role catalog, not Chrome.\
_Avoid_: Treating catalogs as Domain kinds; conflating with Role catalog or Chrome display catalog

**Chrome**: Task display catalog (not submit facts). Not the operator board (see **Worklist**).\
_Avoid_: Naming the Worklist “Chrome”

### Passport / DPP

**Passport**: The issued Digital Product Passport for one Domain subject. Public credential and human page are projections of a **Passport snapshot**, not a live Domain walk. See [ADR-0007](../adr/0007-org-dpp-production.md).\
_Avoid_: Stream Process DPP as the Org `/app` issue path; SPA-assembled passport payloads; treating one plant’s YAML as the feature itself

**Passport lock**: The operator Task that approves a subject’s gathered facts and issues its **Passport**. Distinct from form seal and from any historical Worklist Unlock status word. See [ADR-0007](../adr/0007-org-dpp-production.md).\
_Avoid_: Auto-issue on Domain predicate alone; conflating Passport Domain `locked` with removed Worklist Unlock status

**Passport snapshot**: Immutable projection of related Domain facts frozen into the Passport record on Commit. Later Domain edits do not rewrite what the Passport claims. See [ADR-0007](../adr/0007-org-dpp-production.md).\
_Avoid_: Live Domain as the passport body; mutable “current view” credentials; “passport” as just another Domain entity row

### Operator surfaces

**Dumb executor**: Platform code only evaluates Org Config; plant business rules live in Config. See [`AGENTS.md`](../../AGENTS.md).\
_Avoid_: Plant- or domain-named branches in executors; exclusivity law outside Org Config

**OperatorIdentity**: The actor context for role gates and Submission actor. Not the Org Config document.\
_Avoid_: Principal; Operator (catalog role label) for this bag

**Commit navigation**: Where the operator goes after a successful Task Commit. Commit `home` means the Org stream instance home. See [ADR-0006](../adr/0006-org-operator-worklist-home.md).\
_Avoid_: Client-side auto-advance; overview as Commit destination; unscoped `/app` as landing

**Advance target eligibility**: Whether a Task Commit may auto-advance into another Task on that Job instance. See [ADR-0006](../adr/0006-org-operator-worklist-home.md).\
_Avoid_: Treating “next Task in Config order” as always correct after Commit

**Commit**: Authoritative accept of a sealed **Task** into Domain. See [ADR-0005](../adr/0005-org-worklist-server-projection.md) and [ADR-0006](../adr/0006-org-operator-worklist-home.md).\
_Avoid_: Folding Drafts into Commit; SPA re-deriving post-Commit destination

**Load**: Bring Worklist and Active Config context into the operator UI for one Org stream instance. Keyed by (org, stream, Org stream instance). See [ADR-0005](../adr/0005-org-worklist-server-projection.md) and [ADR-0014](../adr/0014-org-config-as-stream.md).\
_Avoid_: Whole-config execution id as the load key; org-only Load without stream and instance

**Worklist**: Operator board for one Org stream instance: act-now Jobs and Task statuses. Finished rediscovery is the `/my` dashboard and timeline. See [ADR-0005](../adr/0005-org-worklist-server-projection.md), [ADR-0006](../adr/0006-org-operator-worklist-home.md), and [ADR-0014](../adr/0014-org-config-as-stream.md).\
_Avoid_: Job surface; Chrome; Operator home (the `/my` landing); peer home bands for In progress / Completed; plant-wide Worklist across instances

**Recap**: Board overview table for one Task at a Submission address. Config may project Domain entities via `entitiesTouched`, or fall back to one row per Submission. See [ADR-0016](../adr/0016-org-task-recap-entities.md).\
_Avoid_: Stamping createdBy on entities; mint-time field snapshots; SPA re-deriving the table from Domain

## Historical / rejected

Do not revive as Org nouns. Living multi-pass work is **submit cardinality** `many` + **readyWhen** (see [ADR-0008](../adr/0008-org-task-submit-cardinality.md)). Sibling / Domain prerequisites live in **readyWhen** (and Job **startsWhen** for start)—not Unlock. Finished rediscovery for Org streams is `/my`, not Archive.

**Archive**: Former `/app` surface listing past/completed Job instances into overview. Removed for Org streams ([ADR-0014](../adr/0014-org-config-as-stream.md)); `/my` dashboard done filter replaces it.\
_Avoid_: Reviving Archive on `/app`; day→shift→lot Archive hierarchy

**Job instance overview**: Former read-only Job-instance surface reached from Archive. Removed for Org streams ([ADR-0014](../adr/0014-org-config-as-stream.md)); `/my` instance timeline replaces it.\
_Avoid_: Treating overview as the operator board

**Unlock order**: Former in-Job sibling-order gate (optional `unlockFrom` predecessor) and Worklist status `locked`. Removed from the platform; Config rejects authored `unlockFrom`.\
_Avoid_: Reintroducing Unlock as platform law; calling readyWhen “unlock”

**child Task**: Former name for a Task under a Task group; replaced by **submit cardinality** `many` + **readyWhen**.\
_Avoid_: Reviving child Tasks or group nesting; Subtask; task instance

**Task group**: Former composite node under a Job; removed.\
_Avoid_: Calling a Task group a Task; jobs all the way down

**Spawn** / **Close** / **spawningNew**: Former Task-group affordances and Commit mint kind; removed with groups. See [ADR-0008](../adr/0008-org-task-submit-cardinality.md) and [ADR-0012](../adr/0012-org-commit-mint-intent.md).

**Whole-config execution** (historical name `Config run`): One “run of the config” with its own Domain — replaced by one Domain per Org stream + repeated Job instances / Org stream instances.

**LoadRunPort** / **AuthPort** / **AuthorizerPort** / **HttpPort** / **Task form bundle**: Retired or seam-only names. Living operator nouns are **Load**, **Commit**, **Worklist**, **OperatorIdentity**. Side-effect and projection Decisions: [ADR-0005](../adr/0005-org-worklist-server-projection.md), [ADR-0010](../adr/0010-org-http-side-effects.md).

**Step** / **Substep** / **Step instance**: Experiment names for Job / Task / Job instance.
