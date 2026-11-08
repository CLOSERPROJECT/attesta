# Streams

## What Is A Stream

A stream is the reusable process definition that Attesta executes.

Each stream contains:

- a name and description
- ordered steps
- one organization assigned to each step
- ordered substeps inside each step
- one or more allowed roles for each substep
- an input type and schema for each substep
- optional DPP settings

From a product point of view, the stream is the template. When a user clicks `New instance`, Attesta starts a new stream instance from that template.

## Stream Structure

Each stream is made of:

### Step

A step is a major stage in the workflow. Each step belongs to one organization.

### Substep

A substep is the unit that users actually complete. Each substep defines:

- title
- order
- allowed roles
- input key
- input type
- schema and UI schema when the input is Formata-based

### Input Types

In the current Attesta product, stream actions can collect:

- structured Formata forms
- file-like inputs represented through Formata form components
- scalar values in simpler legacy cases

The dominant user-facing mode is Formata-driven structured forms.

## How To Operate A Stream

Operating a stream usually follows this flow:

1. Select a stream from the home page.
2. Open the stream page and start a new instance with `New instance`.
3. Open the stream instance.
4. Select an available substep from the timeline.
5. Complete the form or upload the required evidence.
6. Repeat until all substeps are completed in sequence.
7. When the instance reaches `done`, export outputs and DPP materials.

## Stream Timeline And Action Area

Each stream instance page has two main areas:

- the timeline, which shows the ordered steps and substeps
- the action area, which shows the selected substep details or final outputs

Attesta updates this page live using SSE and partial refreshes.

## Stream Rules And Constraints

Attesta applies the following constraints when a user tries to complete a substep.

### Sequence Constraint

Substeps are sequence-locked. If earlier required work is not complete, Attesta blocks the action with a locked-step message.

### Organization Constraint

The user organization must match the organization assigned to the step.

### Role Constraint

The active role chosen by the user must:

- belong to the user
- be one of the substep's allowed roles

### Workflow Constraint

The action must belong to the currently selected stream.

### Form Constraint

Submitted values must match the form requirements defined in the stream configuration.

## DPP From A Stream

If DPP is enabled for the stream, Attesta generates the DPP when the stream instance first becomes complete.

The generated DPP includes:

- GS1 Digital Link URL
- GS1 element string values
- traceability content from completed substeps
- displayed values and available attachment links

The DPP route is public:

`/01/{gtin}/10/{lot}/21/{serial}`

This means the DPP can be shared externally if that is part of the operating model.

## Final Outputs Of A Completed Stream Instance

When a stream instance is done, the process page exposes:

- a zip file with all attachments
- notarized JSON export
- Merkle tree JSON export
- DPP link, if DPP is configured

## How To Create A Stream: Attesta Composer

Attesta stream creation is done through the embedded composer, based on Formata Arch.

The composer is available at:

- `/org-admin/formata-builder`

Only org admins and platform admins can access it. It is the supported UI path for creating and saving streams.

## What The Composer Does

The composer provides:

- a stream and workflow editor
- step and substep editing
- organization and role selection using the live Attesta catalog
- DPP configuration
- import and export flows
- validation before save

Its navigation is centered around:

- `Stream / Workflow`
- `DPP`
- `Export`

## Composer Workflow

The normal composer workflow is:

1. Open the composer from the `Create new stream` card.
2. Define the workflow name and description.
3. Add steps in sequence.
4. Assign each step to an organization.
5. Add substeps under each step.
6. Assign one or more roles to each substep.
7. Define the input schema for each substep.
8. Configure DPP settings if needed.
9. Review validation errors.
10. Save the stream.

Once saved, the new stream appears in the Attesta home stream list.

## Validity Of A Stream

A stream is operational only if its organization and role references resolve correctly in Attesta.

If the stream references organizations or roles that are missing or inconsistent, Attesta still shows the stream but blocks new instance creation until the references are fixed.

## Stream Deletion

Stream deletion depends on who created the stream and whether any stream instances already exist.

### Org Admin Deletion

The stream creator may delete the stream only if no stream instances have been started yet.

### Platform Admin Deletion

A platform admin may delete any stream. If stream instances already exist, Attesta can also purge the stream history as part of the deletion flow.

See [Platform Admin](./03-platform-admin.md) and [Org Admin](./04-org-admin.md).

