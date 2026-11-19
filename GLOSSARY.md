# Attesta

**Org** and **Stream** are different product nouns—do not mix them. `/my` hosts Formata Streams and may host **Org-backed streams**; `/app` is the instance-scoped Org operator surface.

- **Org** (`/app`, Org-backed `/my`): [`docs/org/GLOSSARY.md`](docs/org/GLOSSARY.md)
- **Stream** (Formata `/my`): [`docs/stream/GLOSSARY.md`](docs/stream/GLOSSARY.md)

Plant Config payloads (demo fixture nouns): [`docs/feragame/GLOSSARY.md`](docs/feragame/GLOSSARY.md). Which file to open: [`GLOSSARY-MAP.md`](GLOSSARY-MAP.md).

Below: vocabulary shared by both Org and Stream — people, membership, and standing (session → who may act).

## Language

**Organization**: A party a user may affiliate with. Formata Streams list Organizations as step owners; Org gives each Organization one or more **Org streams**, each with its own **Org Config** and **Domain**.\
_Avoid_: Department, company, team (infra wording)

**Registration**: Open self-serve account creation. A registered user may have zero organizations.\
_Avoid_: Closed signup, invite-only registration (as the product default); restating email-control rules—see [ADR-0003](docs/adr/0003-email-verification-gate.md)

**Email verification**: Proof that a registered user controls their account email. See [ADR-0003](docs/adr/0003-email-verification-gate.md).\
_Avoid_: Confirmed account, activated, email confirmation (as the glossary name), conflating with Affiliation or Onboarding

**Verification waiting path**: The authenticated-only path for an unverified user. Distinct from Onboarding.\
_Avoid_: Onboarding, signup success page, confirm page (when meaning the email-link landing)

**Invitation**: An outbound membership offer to an email (Org admin or platform admin). Distinct from a Join request. May grant Org admin standing and/or **Organization roles**.\
_Avoid_: Join request, membership request (when meaning outbound invite)

**Join request**: A registered user’s inbound ask to join an existing organization as a Member, naming one or more Organization roles.\
_Avoid_: Invite, application (vague), membership (the lasting affiliation), requesting Org admin

**Organization creation request**: A registered user’s ask to create a new organization (platform-admin approval).\
_Avoid_: Org signup, self-serve org (when meaning live create without approval)

**Affiliation**: A user’s membership in at most one organization at a time. Unaffiliated means zero organizations. Within an Affiliation, standing (Org admin vs Member) is independent of which Organization roles the user holds.\
_Avoid_: Multi-org membership, active org switcher, workspace

**Onboarding**: The unaffiliated-user surface to obtain an Affiliation. Only for verified users. Distinct from Operator home (`/my`) and from the Org Worklist.\
_Avoid_: Signup (account creation), registration wizard, folding Onboarding into Operator home, Email verification waiting path

**Waiting**: The user’s own pending Join request or Organization creation request while another party decides. Visible to the requester; not an Attention item.\
_Avoid_: Notification, Attention item, pending (when meaning actionable work for the current user)

**Attention item**: Something that requires the current user to act (e.g. pending Join request, open Invitation, or—on Legacy Stream—an actionable Substep waiting on them). Distinct from Waiting.\
_Avoid_: Notification, alert, badge, task (vague), pending (when meaning “waiting on others”)

**Org admin**: Privileged standing within an Affiliation: can manage organization profile, Role catalog, and members. Orthogonal to Organization roles. Never an entry in the Role catalog. A sole Org admin cannot lose that standing (including via Leave) until another Org admin exists.\
_Avoid_: Treating Org admin as a catalog / Organization role

**Member**: An affiliated user without Org admin standing. May hold zero or more Organization roles.\
_Avoid_: Regular member, non-admin (as primary terms), conflating Member with Organization role

**Role catalog**: The set of **Organization roles** an Organization defines. Does not include Org admin. Distinct from the Org Config **Role catalog** of **Config roles** ([`docs/org/GLOSSARY.md`](docs/org/GLOSSARY.md)).\
_Avoid_: Catalog (alone), putting Org admin in the catalog, conflating with Config roles

**Organization role**: One named entry in an Organization’s Role catalog. Affiliated users may each hold zero or more.\
_Avoid_: Catalog role, org role (vague), membership role, Org admin, Config role

**Operator**: The default Organization role created with every new organization so Members have a joinable role.\
_Avoid_: Default member, basic role, Org admin; conflating with Org actor identity (**OperatorIdentity**); treating an empty Role catalog as valid

**Account settings**: The affiliated user’s account-level actions on their current Affiliation (notably Leave). Distinct from the Org admin console.\
_Avoid_: Organization home, account page, My Org settings (when meaning admin console)

**Leave**: The affiliated user’s action of ending their own Affiliation. Distinct from an Org admin removing another member. Sole-admin constraint: see **Org admin**.\
_Avoid_: Delete self, resign, unlink, soft-delete (when meaning self-serve exit)

## Relationships

- An **Affiliation** binds one user to at most one **Organization**.
- **Org admin** / **Member** is standing; **Organization roles** are catalog entries—orthogonal.
- **Org** and **Stream** share this membership layer; their operating nouns stay in [`docs/org/GLOSSARY.md`](docs/org/GLOSSARY.md) and [`docs/stream/GLOSSARY.md`](docs/stream/GLOSSARY.md). Slug alignment across Organization / Stream / Config roles: [`GLOSSARY-MAP.md`](GLOSSARY-MAP.md).
