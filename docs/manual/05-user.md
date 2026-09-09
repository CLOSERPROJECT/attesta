# User

## What A User Can Do

In the current Attesta product, a normal user can:

- sign in
- open the stream list
- start a new stream instance
- open an existing stream instance
- complete substeps that match the user's organization and role
- view completed work and exported outputs

This page should be read together with [Streams](./02-streams.md), because most user actions happen inside stream pages.

## Where Users Work

Users mainly work in three places:

- the home stream picker
- the selected stream page
- the stream instance page

## Starting A New Stream Instance

The user starts by:

1. opening the home page
2. choosing a stream
3. clicking `New instance`

This creates a new process from the saved stream template.

## Working On Substeps

Inside a stream instance, the user:

1. selects a substep from the timeline
2. reviews the required input
3. submits the requested form or evidence

Attesta records:

- the submitted payload
- the completion time
- the user identity
- the active role used for the action
- a notarization digest

## Role-Based Completion Rules

Users do not complete arbitrary actions. A user may complete a substep only when all of the following are true:

- the user belongs to the organization assigned to that step
- the user holds at least one allowed role for the substep
- the user chooses an allowed active role if more than one applies
- all earlier required substeps are already complete

If one of these conditions fails, Attesta blocks the action.

## Users With Multiple Roles

Some users carry more than one role.

When more than one of those roles matches a substep, Attesta asks the user to choose the active role for that submission. This matters because Attesta stores the role used for the completed action.

## What Users See After Submission

Once a substep is completed, Attesta shows it as submitted and displays:

- recorded values
- attachment links where applicable
- completion timestamp
- who completed it
- which role was used

## What Happens At The End Of A Stream Instance

When all required substeps are completed, the stream instance reaches `done`.

At that point users can access final outputs such as:

- downloaded attachments bundle
- notarized JSON export
- Merkle tree export
- DPP link, if the stream was configured for DPP

## DPP For End Users

If DPP is enabled, Attesta generates the DPP automatically when the stream instance first becomes complete.

The user can then:

- open the GS1 Digital Link page
- share the DPP link
- use the DPP page as the public-facing traceability view

## What Users Cannot Do

A normal user cannot:

- create or edit organizations
- create or edit roles
- invite or remove users
- access the platform admin dashboard
- access the org admin dashboard unless they are also an org admin
- create or save streams unless they are also an org admin or platform admin

## Related Pages

- [Streams](./02-streams.md)
- [Org Admin](./04-org-admin.md)
- [Platform Admin](./03-platform-admin.md)
