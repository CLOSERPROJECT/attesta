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
The authenticated screen at `/my` where an affiliated operator chooses which stream (blueprint) to open. Public marketing content lives at `/` and does not redirect logged-in users to `/my`.
_Avoid_: Home, workflow picker

**Onboarding**:
The unaffiliated-user flow to obtain an Affiliation: join an organization, request a new organization, or wait on / withdraw a pending request. Narrow reading-width layout.
_Avoid_: Signup (account creation), registration wizard

**Public stream card**:
A public-homepage presentation of a **Stream** (blueprint): name, description, a static “Stream” badge in markup plus an optional passport badge when DPP is enabled, step preview (titles and substep counts), live stream-instance metrics (total count plus active-or-completed activity, or that none exist yet), and participating **organizations**. Distinct UI from the authenticated stream picker card. Not clickable in v1.
_Avoid_: Stream card (when meaning the `/my` picker card), process card, landing stream tile, showcase stream

### Stream discovery taxonomy

**Category**:
A platform-global discovery bucket for browsing **Streams** (blueprints). Has a display name, an icon key from a shared allowlist, an immutable slug, and a sort order. Has no color and no description. A Category cannot be removed while it still has Sub-categories.
_Avoid_: Topic, tag, theme, CSS `data-category` / palette key (presentation only — not this concept)

**Sub-category**:
A finer discovery leaf under exactly one **Category**. Has a display name, the same icon allowlist key, an immutable slug unique within its parent, a sort order, and an optional description. Parent Category is immutable after create. A Sub-category cannot be removed while any **Stream** still references its path.
_Avoid_: Tag, topic, Category (when meaning the leaf)

**Stream categorization**:
A **Stream** may reference zero or one **Sub-category**, identified by `(categorySlug, subCategorySlug)`. It never attaches to a **Category** alone. Both slugs are required together; a missing or unknown path means the Stream is uncategorized. Display names need not be unique; slugs are identity.
_Avoid_: Required category, category-only membership, document id as the Stream’s taxonomy reference

**Stream presentation fields**:
The blueprint label and discovery path on a **Stream**: name, description, and **Stream categorization**. Changing only these does not redefine the executable stream (steps, roles, forms, organizations, DPP identifiers).
_Avoid_: Metadata (vague), tags, executable config

**Organization**:
A party that owns steps in a stream blueprint (team in Appwrite; listed under the stream’s organizations).
_Avoid_: Department, company, team (in domain docs — Appwrite may still say team)

### Auth & affiliation

**Registration**:
Open self-serve account creation (email + password). A registered user may have zero organizations until a join or creation path completes.
_Avoid_: Closed signup, invite-only registration (as the product default)

**Invitation**:
An Org-admin-initiated (or platform-admin-initiated) outbound membership offer to an email. The inviter may grant Org admin standing and/or one or more Organization roles — those are separate choices. Distinct from a Join request. An invitee who already has an Affiliation may only accept (or be invited into) that same Organization.
_Avoid_: Join request, membership request (when meaning outbound invite)

**Join request**:
A registered user’s inbound ask to join an existing organization as a Member, naming one or more Organization roles. Org admins approve or reject; approval grants Member standing with those Organization roles (never Org admin standing). While pending, the requester may withdraw the request (it is removed, as if never submitted).
_Avoid_: Invite, application (vague), membership (the lasting affiliation), requesting Org admin

**Organization creation request**:
A registered user’s ask to create a new organization. Platform admins approve or reject; approval creates the organization and grants the requester Org admin standing. Platform admins may still create organizations directly without a request. While pending, the requester may withdraw the request (it is removed, as if never submitted).
_Avoid_: Org signup, self-serve org (when meaning live create without approval)

**Affiliation**:
A user’s membership in at most one organization at a time. Unaffiliated means zero organizations. Switching requires leaving (or being removed) before a new join or creation request can complete. The stream picker is for affiliated users only; unaffiliated users (including those with a pending join or creation request) go through onboarding instead. Within an Affiliation, standing (Org admin vs Member) is independent of which Organization roles the user holds.
_Avoid_: Multi-org membership, active org switcher, workspace

**Org admin**:
Privileged standing within an Affiliation: can manage organization profile, Role catalog, and members. Orthogonal to Organization roles — an Org admin may hold zero or more Organization roles. Never an entry in the Role catalog. An Org admin may grant or remove Org admin standing for any affiliated user (including themselves) when at least one other Org admin would remain; the sole Org admin’s standing cannot be removed. An Org admin cannot remove their own Affiliation via member delete — that exit is Leave.
_Avoid_: Treating Org admin as a catalog / Organization role, “org-admin” as one more role among peers (in domain docs)

**Member**:
An affiliated user without Org admin standing. May hold zero or more Organization roles.
_Avoid_: Regular member, non-admin (as primary terms), conflating Member with Organization role

**Role catalog**:
The set of Organization roles an Organization defines (name, slug, palette). Does not include Org admin.
_Avoid_: Catalog (alone — also means stream catalog), putting Org admin in the catalog

**Organization role**:
One named entry in a Role catalog. Affiliated users (Org admin or Member) may each hold zero or more. Chosen on Join request and (alongside standing) on Invitation; matched to Stream substep gates by slug.
_Avoid_: Catalog role, org role (vague), membership role, Org admin

**Stream role**:
A role named on a Stream blueprint to gate Substeps. Matched to Organization roles by slug; not itself a Role catalog entry.
_Avoid_: Catalog role, workflow role (in domain docs), Organization role (when meaning the blueprint gate)

**Account settings**:
The affiliated user’s modal for account-level actions on their current Affiliation (notably Leave). Until other account-level options exist, the account-menu entry appears only when the user has an Affiliation. Distinct from the Org admin console.
_Avoid_: Organization home, account page, My Org settings (when meaning admin console), org admin console

**Leave**:
The affiliated user’s action of ending their own Affiliation. Distinct from an Org admin removing another member. A sole Org admin cannot Leave until another Org admin exists.
_Avoid_: Delete self, resign, unlink, soft-delete (when meaning self-serve exit)

**Operator**:
The default Organization role created with every new organization so Members have a joinable role. An organization must keep at least one Organization role in its Role catalog.
_Avoid_: Default member, basic role, Org admin, catalog role

**Stream dashboard**:
The screen at `/my/streams/:key/` listing stream instances for one stream, with status navigation and a read-only timeline preview.
_Avoid_: Home, workflow home

**Stream instance detail page**:
The full page at `/my/streams/:key/instance/:id`. Comprises a stable outer shell (process metadata, SSE hooks) and an inner **content partial** swapped via HTMX/SSE after substep completion or live updates.
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
