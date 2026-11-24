# Attesta Manual

This manual describes Attesta as it behaves in the current product codebase.

## Contents

- [Introduction](./01-intro.md)
- [Streams](./02-streams.md)
- [Platform Admin](./03-platform-admin.md)
- [Org Admin](./04-org-admin.md)
- [User](./05-user.md)

## Audience

This is an end-user product manual written with enough technical depth to explain how Attesta works in practice.

## Key Terms

### Stream

In Attesta, a stream is the saved workflow definition that describes:

- the ordered steps
- the organizations responsible for those steps
- the roles allowed to complete each substep
- the input schema for every action
- optional DPP generation settings

### Stream Instance

A stream instance is a started execution of a stream. In the backend this is stored as a process. A single stream can have many instances.

### User

A user is a real account that can sign in to Attesta. Users belong to one organization context at a time.

### Role

A role is a responsibility assigned to a user inside an organization. Roles are used to decide which substeps a user may complete.

### Org Admin

An org admin is the owner of an organization inside Attesta. This person can change the organization name and logo, manage roles, invite users, edit user roles, and remove users.

### Platform Admin

A platform admin is the Attesta-wide superuser. This person can create, update, and delete organizations, invite org admins, and delete streams at platform level.

## Product Rules To Keep In Mind

- Streams are sequential: later substeps stay locked until earlier required substeps are completed.
- A substep can only be completed by a user whose organization matches the step organization and whose active role matches one of the substep roles.
- Users may hold multiple roles.
- If a user has multiple matching roles for a substep, Attesta asks for the active role used for that action.
- When DPP is enabled on a stream, the DPP is generated when the stream instance first reaches `done`.

