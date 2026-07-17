# Attesta

Attesta tracks multi-party stream instances with notarized substeps, role-based completion, and optional Digital Product Passport output.

## Language

**Stream**:
A configured blueprint that defines steps, substeps, roles, and forms (what YAML and code today call a workflow). Starting a stream creates a new stream instance.
_Avoid_: Workflow, stream type, template

**Stream instance**:
A single running occurrence of a stream — one tracked batch or case from start through completion or early termination (what code today calls a process).
_Avoid_: Process, stream (when meaning an instance in domain docs — see UI shorthand below)

**UI shorthand**: In operator-facing copy, "stream" may mean stream instance when context is obvious (lists, status, "End stream"). Use **stream instance** in the glossary and when ambiguity matters. For the blueprint, prefer the stream's name or "stream picker" — not bare "stream".

**Step**:
An organization-scoped phase in a stream blueprint. Groups one or more substeps (e.g. "Incoming intake").
_Avoid_: Using "step" when meaning substep completion state (`ProcessStep` in code)

**Substep**:
The smallest completable unit in a stream — a role-gated form or input that a participant submits (e.g. `1.1 Record formata`).
_Avoid_: Action

**Stream instance detail**:
The view of one stream instance: its timeline (steps and substeps), completion controls, and post-completion resources (DPP, downloads, termination summary).
_Avoid_: Process page, action list

**Stream picker**:
The screen at `/` where an operator chooses which stream (blueprint) to open.
_Avoid_: Home, workflow picker

**Stream dashboard**:
The screen at `/w/:key/` listing stream instances for one stream, with status navigation and a read-only timeline preview.
_Avoid_: Home, workflow home

**Stream instance detail page**:
The full page at `/w/:key/process/:id`. Comprises a stable outer shell (process metadata, SSE hooks) and an inner **content partial** swapped via HTMX/SSE after substep completion or live updates.
_Avoid_: Process page

### Stream instance detail UI

**Stream timeline**:
The accordion tree on stream instance detail (and preview on the stream dashboard): steps, each containing expandable substeps.
_Avoid_: Workflow timeline, traceability timeline, action list

**Substep summary**:
The always-visible row for one substep: ID, title, status, and optional completion meta in the accordion header.
_Avoid_: Action card header

**Substep body**:
The expandable content below a substep summary. Renders in one of four modes: **preview** (read-only form shell, includes locked), **actionable** (fillable form), **result** (submitted values), or **message** (text-only, e.g. terminated/skipped). Go builders set an explicit `SubstepBodyView.Mode` field; the template dispatches via `effectiveSubstepBodyMode`.
_Avoid_: Action detail, action content
