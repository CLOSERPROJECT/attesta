# Org Admin

## Role Of The Org Admin

The org admin is the owner of one organization inside Attesta.

This persona can change anything inside that organization, including:

- organization name
- organization logo
- organization roles
- organization users
- stream creation through the composer

The org admin dashboard is available at:

- `/org-admin/users`
- `/org-admin/roles`

In the current UI both user and role management are centered in the organization admin area.

## Organization Profile Management

Org admins can:

- create the organization if their invite is not yet attached to one
- update the organization name
- update the logo

### Organization Creation

This is relevant when an org-admin account exists before the organization context is fully attached.

### Constraints

- organization name is required
- slug is generated from the name
- duplicate slugs are rejected
- logos must be PNG, JPG, WEBP, or SVG

## Users And Roles: The Operational Difference

This distinction is critical for org admins.

### User

A user is an account that can log in and belong to the organization.

### Role

A role is a responsibility assigned to the user and used to unlock stream actions.

One user can have:

- zero business roles
- one business role
- many business roles

In addition, a user may also be an org admin.

## How To Create And Edit Roles

Org admins manage business roles from the roles panel.

### What A Role Contains

Each role contains:

- a display name
- a slug derived from the name
- a chosen color palette

### Constraints When Creating Roles

- role name is required
- the generated role slug must be valid
- the role slug must be unique inside the organization
- the special `org-admin` capability is reserved and always handled as the organization-owner role

### Constraints When Editing Roles

Org admins can edit a role only if that role is not currently in use.

Attesta blocks editing when the role is assigned to:

- current users
- pending invites

In that case the UI requires the org admin to remove the role from those users or invites first.

### Constraints When Deleting Roles

Org admins can delete a role only if it is not in use.

If the role is still assigned anywhere, Attesta blocks deletion with an explicit error.

### Practical Meaning

This design prevents silent breakage of active permissions in existing streams and user assignments.

## How To Invite Users

Org admins invite users from the user management area.

### Invite Inputs

The invite flow accepts:

- user email
- zero, one, or many selected roles
- optional org-admin ownership

### What Happens On Invite

Depending on the email state, Attesta behaves differently.

#### New Email

Attesta creates an organization invite.

#### Existing User In The Same Organization

Attesta updates the user's role labels directly instead of creating a new organization membership.

#### Existing Pending Invite In The Same Organization

Attesta updates the pending membership if the selected roles changed.

#### Existing User In Another Organization

Attesta rejects the action.

### Invite Statuses

The organization admin page shows invite statuses derived from membership state:

- `pending`
- `accepted`
- `expired`

`expired` is used when an invite is still unconfirmed after the invite lifetime window.

## How To Edit User Roles

Org admins can update the role set of an existing user.

### Constraints

- the target user must exist in the organization
- every selected role must exist in the organization
- org-admin ownership can be added or removed from other users
- an org admin cannot remove org-admin ownership from their own account

That self-protection rule prevents accidental lockout of the only organization owner.

## How To Remove Users

Org admins can remove users through the `Delete user` action.

### What Removal Means

Attesta removes:

- the organization membership
- Attesta-managed role labels for that organization

### What Removal Does Not Mean

Attesta does not delete the person's global account from the identity system.

This is organization removal, not platform-wide account destruction.

## How To Set Up A New Stream

Org admins create streams through the Attesta composer:

- open `/org-admin/formata-builder`
- define the workflow
- assign organizations and roles
- configure DPP if required
- validate the stream
- save it

### Important Composer Dependency

The composer reads live organization and role data from Attesta. That means:

- organizations should already exist
- required roles should already exist before you assign them in the stream

## Org Admin Constraints In Stream Design

When creating streams, org admins should remember:

- every step must point to an organization
- every actionable substep should point to valid roles
- stream execution is sequential
- users can complete only the substeps allowed by their organization and active role

If these references are wrong, the stream may appear in Attesta but new instances can be blocked until the references are fixed.

## Recommended Operating Pattern For Org Admins

1. Set up the organization profile.
2. Define the business roles.
3. Invite users and assign roles.
4. Create the stream in the composer.
5. Verify the stream appears on the home page.
6. Let users start instances and complete work.

