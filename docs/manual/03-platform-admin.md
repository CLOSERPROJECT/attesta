# Platform Admin

## The Superuser Of Attesta

The platform admin is the Attesta-wide superuser.

This persona has access to:

- the platform admin dashboard at `/admin/orgs`
- organization creation, editing, and deletion
- org admin invitation flows
- platform-level stream deletion
- all stream composer capabilities available to org admins

In the current product, platform admin access is distinct from normal organization membership. It is protected through Cerbos as the `platform_admin` capability.

## Main Responsibilities

The platform admin owns cross-organization setup and control tasks, especially when they cannot be safely delegated to a single organization.

The platform admin can:

- create organizations
- edit organization names and logos
- delete organizations
- invite first or additional org admins
- inspect org-admin status per organization
- create and save streams using the composer
- delete streams, including streams that already have started instances

## How To Create Organizations

Use the `Platform admin dashboard` at `/admin/orgs`.

### Available Inputs

When creating an organization, the platform admin can provide:

- organization name
- optional organization logo
- optional first org admin email

### Important Rules

- organization slugs are derived from the organization name
- slug collisions are rejected with an explicit `organization slug already exists` error
- logo uploads must be image files

### Accepted Logo Types

The current UI and backend accept:

- PNG
- JPG or JPEG
- WEBP
- SVG

## How To Invite Org Admins

Platform admins invite org admins from the same dashboard.

For each organization, the invite dialog shows:

- current accepted org admins
- pending org admin invites

### Invite Flow

1. Open the organization invite action.
2. Enter the org admin email.
3. Submit the invite.
4. The invited person accepts the link and completes account setup if needed.

### Important Rules

- if the email already belongs to the organization and is not yet an org admin, Attesta upgrades the membership
- if the email belongs to another organization, Attesta rejects the invite
- org admin state is tracked through organization membership information

## How To Edit Organizations

The platform admin can update:

- organization name
- organization logo

### Constraints

- renaming may change the slug
- the target slug must remain unique
- replacing the logo stores a new file and Attesta removes the old file reference

## How To Delete Organizations

Organizations can be deleted from the platform admin dashboard.

This is a destructive operation. The current UI presents it as a danger-zone action because it removes:

- the organization
- associated memberships
- organization-level access in Attesta

## How To Delete Streams

Stream deletion is visible from the home stream list when the logged-in user is allowed to delete that stream.

### Platform Admin Behavior

A platform admin may delete any saved stream.

If one or more stream instances have already been started, Attesta can also purge workflow data as part of the delete path. This is the main difference between platform-admin deletion and creator-only deletion.

### Typical Use Cases

- removing obsolete test streams
- removing a wrongly published stream
- resetting a preview or demo environment

## How To Set Up A New Stream

Platform admins can use the same composer used by org admins:

- open `/org-admin/formata-builder`
- define the workflow, organizations, roles, and DPP settings
- validate the configuration
- save the stream

## Other Useful Platform Tasks

In practice, the platform admin also acts as the operator who can:

- search organizations by name
- confirm whether an organization has accepted org admins
- verify whether there are pending org admin invites
- fix organization naming and branding issues
- remove stale streams that ordinary org-admin creators can no longer delete

## Platform Admin Constraints

The platform admin is powerful, but still operates inside product rules:

- only saved streams can be deleted
- missing streams return `Stream not found`
- composer save and console access still go through Cerbos checks
- organization names and stream references must remain valid for downstream user flows

