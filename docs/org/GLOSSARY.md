# Organization & affiliation

Shared Attesta vocabulary for people, membership, and standing. Used by both
Legacy Stream (`/my`) and Org product (`/app`) for session → who may act.

Which glossary applies: [`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md).

## Language

**Organization**:
A party a user may affiliate with. Legacy Streams list Organizations as step
owners; Org product gives each Organization one **Org Config** and **Domain**.
_Avoid_: Department, company, team (infra wording)

**Registration**:
Open self-serve account creation. A registered user may have zero organizations
until a join or creation path completes.
_Avoid_: Closed signup, invite-only registration (as the product default);
restating email-control rules—see [ADR-0003](../adr/0003-email-verification-gate.md)

**Email verification**:
Proof that a registered user controls their account email. Until verified,
authenticated app use is limited to the **verification waiting path**. See
[ADR-0003](../adr/0003-email-verification-gate.md).
_Avoid_: Confirmed account, activated, email confirmation (as the glossary name),
conflating with Affiliation or Onboarding

**Verification waiting path**:
The authenticated-only path for an unverified user after Registration or login
while still unverified. Distinct from Onboarding.
_Avoid_: Onboarding, signup success page, confirm page (when meaning the
email-link landing)

**Invitation**:
An Org-admin-initiated (or platform-admin-initiated) outbound membership offer
to an email. The inviter may grant Org admin standing and/or one or more
**Organization roles**—those are separate choices. Distinct from a Join request.
An invitee who already has an Affiliation may only accept (or be invited into)
that same Organization.
_Avoid_: Join request, membership request (when meaning outbound invite)

**Join request**:
A registered user’s inbound ask to join an existing organization as a Member,
naming one or more Organization roles. Org admins approve or reject; approval
grants Member standing with those Organization roles (never Org admin standing).
While pending, the requester may withdraw the request.
_Avoid_: Invite, application (vague), membership (the lasting affiliation),
requesting Org admin

**Organization creation request**:
A registered user’s ask to create a new organization. Platform admins approve or
reject; approval creates the organization and grants the requester Org admin
standing. Platform admins may still create organizations directly without a
request. While pending, the requester may withdraw the request.
_Avoid_: Org signup, self-serve org (when meaning live create without approval)

**Affiliation**:
A user’s membership in at most one organization at a time. Unaffiliated means
zero organizations. Switching requires leaving (or being removed) before a new
join or creation request can complete. Within an Affiliation, standing (Org
admin vs Member) is independent of which Organization roles the user holds.
_Avoid_: Multi-org membership, active org switcher, workspace

**Onboarding**:
The unaffiliated-user surface to obtain an Affiliation: accept or decline
Invitations, join an organization, request a new organization, or wait on /
withdraw a pending request. Only for verified users. Distinct from Legacy
Operator home and from the Org product.
_Avoid_: Signup (account creation), registration wizard, folding Onboarding into
Operator home, Email verification waiting path

**Waiting**:
The user’s own pending Join request or Organization creation request while
another party decides. Visible to the requester; not an Attention item.
_Avoid_: Notification, Attention item, pending (when meaning actionable work for
the current user)

**Attention item**:
Something that requires the current user to act: pending Join request (Org
admin), pending Organization creation request (platform admin), open Invitation
(invitee), or—on Legacy Stream—an actionable Substep waiting on them. Distinct
from Waiting (requester’s own pending request).
_Avoid_: Notification, alert, badge, task (vague), pending (when meaning
“waiting on others”)

**Org admin**:
Privileged standing within an Affiliation: can manage organization profile, Role
catalog, and members. Orthogonal to Organization roles—an Org admin may hold
zero or more. Never an entry in the Role catalog. A sole Org admin cannot lose
that standing (including via Leave) until another Org admin exists.
_Avoid_: Treating Org admin as a catalog / Organization role

**Member**:
An affiliated user without Org admin standing. May hold zero or more
Organization roles.
_Avoid_: Regular member, non-admin (as primary terms), conflating Member with
Organization role

**Role catalog**:
The set of **Organization roles** an Organization defines. Does not include Org
admin. Distinct from the Org Config **Role catalog** of **Config roles**
([`GLOSSARY.md`](../../GLOSSARY.md)).
_Avoid_: Catalog (alone), putting Org admin in the catalog, conflating with
Config roles

**Organization role**:
One named entry in an Organization’s Role catalog. Affiliated users may each
hold zero or more. Chosen on Join request and (alongside standing) on
Invitation. Slugs should align with Stream roles
([`docs/stream/GLOSSARY.md`](../stream/GLOSSARY.md)) and Config roles
([`GLOSSARY.md`](../../GLOSSARY.md)).
_Avoid_: Catalog role, org role (vague), membership role, Org admin, Config role

**Operator**:
The default Organization role created with every new organization so Members
have a joinable role.
_Avoid_: Default member, basic role, Org admin; conflating with
**OperatorIdentity** (Org product); treating an empty Role catalog as valid

**Account settings**:
The affiliated user’s account-level actions on their current Affiliation
(notably Leave). Distinct from the Org admin console.
_Avoid_: Organization home, account page, My Org settings (when meaning admin
console)

**Leave**:
The affiliated user’s action of ending their own Affiliation. Distinct from an
Org admin removing another member. Sole-admin constraint: see **Org admin**.
_Avoid_: Delete self, resign, unlink, soft-delete (when meaning self-serve exit)

## Relationships

- An **Affiliation** binds one user to at most one **Organization**.
- **Org admin** / **Member** is standing; **Organization roles** are catalog
  entries—orthogonal.
- **Organization role** slugs should align with Stream roles and Config roles
  (see those glossaries).
