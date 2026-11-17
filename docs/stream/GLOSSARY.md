# Legacy Stream

Multi-party **Stream** blueprints and instances on Legacy `/my` (Formata,
discovery, notarized substeps, optional Stream DPP). Historical product surface;
still accurate for that runtime. Not Org Config / `/app`.

Shared membership: [`GLOSSARY.md`](../../GLOSSARY.md). Org (`/app`):
[`docs/org/GLOSSARY.md`](../org/GLOSSARY.md). Map:
[`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md).

## Language

**Stream**:
A configured blueprint that defines steps, substeps, roles, and forms (what YAML
and code today call a workflow). Starting a stream creates a new stream
instance.
_Avoid_: Workflow, stream type, template; Org Config

**Import-blocking issue**:
A Stream document problem that prevents loading a draft in Composer (e.g.
invalid YAML or missing workflow shape).
_Avoid_: Critical error (vague), validation error (when meaning only this tier)

**Composer-fixable issue**:
A Stream document problem that can load into Composer and be corrected there
before Save (schema/taxonomy and similar).
_Avoid_: Warning (alone), soft error, non-critical

**Stream instance**:
A single running occurrence of a stream — one tracked batch or case from start
through completion or early termination (what code today calls a process).
_Avoid_: Process, stream (when meaning an instance in domain docs — see UI
shorthand below); Domain; Job instance

**UI shorthand**: In operator-facing copy, "stream" may mean stream instance when
context is obvious (lists, status, "End stream"). Use **stream instance** in the
glossary and when ambiguity matters. For the blueprint, prefer the stream's name
or "stream picker" — not bare "stream".

**Step**:
An organization-scoped phase in a stream blueprint. Groups one or more substeps
(e.g. "Incoming intake").
_Avoid_: Using "step" when meaning substep completion state; Job

**Substep**:
The smallest completable unit in a stream — a role-gated form or input that a
participant submits (e.g. `1.1 Record formata`).
_Avoid_: Action; Task

**Stream instance detail**:
The full page for one stream instance: its timeline, completion controls, and
post-completion resources (DPP, downloads, termination summary).
_Avoid_: Process page, action list

**Operator home**:
The authenticated Legacy `/my` landing for an affiliated user. Surfaces Attention
items, Upcoming stream instances, and Stream discovery; Org admin and Onboarding
meanings live in [`GLOSSARY.md`](../../GLOSSARY.md).
_Avoid_: Home (alone), Dashboard (vague), Stream picker (as the name of Operator
home), `/app` Worklist home

**Stream discovery**:
The Operator-home region listing Streams the user’s Organization can take part
in, with Category and Sub-category browsing.
_Avoid_: Stream picker (when meaning the entire Operator home), Home

**Startable stream**:
A Stream the current affiliated user may start a new Stream instance for: their
Organization owns the first Step, and either they hold at least one Stream role
required by that Step’s first Substep or they have Org admin standing in that
Organization. Distinct from merely appearing in Stream discovery as a
participant.
_Avoid_: Accessible stream, visible stream, can open dashboard, requiring every
listed first-Substep role

**Public stream card**:
A public-homepage presentation of a **Stream** (blueprint): name, description,
optional passport badge when DPP is enabled, live stream-instance metrics,
step/role counts, and participating **organizations**. Distinct presentation
from the Operator-home discovery card.
_Avoid_: Stream card (when meaning the Operator home discovery card alone),
process card, landing stream tile, showcase stream

### Stream discovery taxonomy

**Category**:
A platform-global discovery bucket for browsing **Streams** (blueprints). Has a
display name, icon, slug, and sort order; no color or description.
_Avoid_: Topic, tag, theme; treating icon key or slug as free-form labels

**Sub-category**:
A finer discovery leaf under exactly one **Category**. Has a display name, icon,
slug unique within its parent, sort order, and optional description.
_Avoid_: Tag, topic, Category (when meaning the leaf)

**Stream categorization**:
A **Stream** may reference zero or one **Sub-category**. It never attaches to a
**Category** alone.
_Avoid_: Required category, category-only membership

**Stream presentation fields**:
The blueprint label and discovery path on a **Stream**: name, description, and
**Stream categorization**. See
[ADR-0001](../adr/0001-stream-presentation-edit-without-purge.md).
_Avoid_: Metadata (vague), tags, executable config

**Stream role**:
A role named on a Stream blueprint to gate Substeps. Matched to **Organization
roles** by slug; not itself a Role catalog entry. See
[`GLOSSARY.md`](../../GLOSSARY.md).
_Avoid_: Catalog role, workflow role (in domain docs), Organization role (when
meaning the blueprint gate), Config role

**Upcoming stream instance**:
An active Stream instance that involves the user’s Organization where the next
actionable Substep is not theirs. A watchlist, not an Attention item.
_Avoid_: Pending (vague), scheduled, upcoming (when meaning a calendar event)

**Stream dashboard**:
The screen listing stream instances for one stream, with status navigation and a
read-only timeline preview.
_Avoid_: Home, workflow home

**Platform stream catalog**:
The platform-admin screen for browsing and managing Streams across the platform.
Distinct from affiliated Stream discovery on Operator home.
_Avoid_: Operator home, Stream discovery

### Stream instance detail UI

**Stream timeline**:
The accordion tree of steps and substeps on stream instance detail (and preview
on the stream dashboard).
_Avoid_: Workflow timeline, traceability timeline, action list

**My tasks filter**:
A Stream instance detail switch that shows only Substeps the viewer owns. Org
admin visibility is defined in [`GLOSSARY.md`](../../GLOSSARY.md).
_Avoid_: My roles filter, Role filter (vague)

**Substep summary**:
The always-visible row for one substep in the Stream timeline (identity, title,
status, optional completion meta).
_Avoid_: Action card header

**Substep body**:
The expandable content below a substep summary.
_Avoid_: Action detail, action content

## Relationships

- A **Stream** blueprint is started as many **Stream instances**.
- A **Step** belongs to one **Organization**; **Substeps** are gated by **Stream
  roles** matched to **Organization roles**
  ([`GLOSSARY.md`](../../GLOSSARY.md)).
- Org Config / Domain / Job / Task are the other product surface—see
  [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md).
