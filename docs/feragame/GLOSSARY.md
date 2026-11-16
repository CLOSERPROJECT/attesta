# Feragame

First **Org Config** demo plant fixture:
[`server/internal/org/fixtures/feragame-config.yaml`](../../server/internal/org/fixtures/feragame-config.yaml).
These nouns are **payload** authored in Config (entities, transforms, `readyWhen`,
Italian labels)—not Attesta platform product law. Platform vocabulary:
[`GLOSSARY.md`](../../GLOSSARY.md). Which glossary applies:
[`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md).

## Language

### Domain entities

**truck** (entity):
Intake truck row (gross weight, scale readings, weight-check fields). Job
instance id for **intake** matches the truck id.
_Avoid_: Treating truck as the Passport subject; conflating with Job instance

**lot** (entity):
Domain entity for one FIR/intake lot (formerly named **load**). Owns pallets and
is the Passport subject. Lot-processing work runs inside a **lot session**
status on this entity.
_Avoid_: load (living plant name); conflating with Job instance; treating lot as
a Task group occurrence

**pallet** (entity):
Bancale under a **lot**: weigh fields (weight, tare, IR/NIR, panel count) plus
**progress**.
_Avoid_: Engine special-case on the name

**container** (entity):
Booked material weight for a source lot; may carry **incomplete** while the
physical bin continues on a later register.
_Avoid_: Engine special-case on the name

### Lot session and row facts

**lot session status** (entity field):
Single enum on a **lot**: `open` | `fullyProcessed`. `startLot` → `open`; an
open lot stays open until `closeLot` sets `fullyProcessed` when every pallet on
that lot has weight and `progress = done`. At most one lot is `open`; next
`startLot` waits until that lot is `fullyProcessed`. No `parked` and no
park/resume via turno Tasks.
_Avoid_: `parked`; open/close working day or shift as Domain peers; separate
boolean fullyProcessed beside open; parallel open lots; engine special-case on
the field name; putting this table in Attesta ADRs

**fullyProcessed** (lot session status value):
Terminal **lot session status**: the lot is closed for line processing. Set only
by **closeLot** Commit transform—not by container register. Passport eligibility
still requires this status (and not locked).
_Avoid_: boolean `processed` as Worklist status; using fullyProcessed as unlock
grammar; using fullyProcessed as “Passport done” alone without the lock fact;
containerReg as the writer of this status

**progress** (entity field):
Plant row fact on some entities (e.g. pallet): unset; `partial`; `done`. Draft ≠
progress; Worklist done is Submission law. Cell enums live on the JSON Schema
cell (`additionalProperties`), rendered by sjsf.
_Avoid_: boolean `processed` as a Worklist gate; treating progress enums as
unlock grammar; form `progressEnums` / `valueType: "progress"` / table-owned
enum paint

**incomplete** (entity field):
Plant Domain fact on a **container**: booked weight for the current lot is frozen
while the physical bin may continue on a later register. Distinct from lot
session status.
_Avoid_: Treating incomplete as Worklist/progress; engine special-case on the
name (Config transform + form only); Task-group Close

**continuesContainerId** (Config form fact):
Hidden row field on plant register: links a continuation row to a still-open
incomplete Domain container. Load seeds it via Task `initialData`; Commit
transform books delta weight and supersedes the prior incomplete. Not a Domain
entity field. Operator visibility uses generic `ui:options.presenceMark` on
`hiddenField` (Config title such as Continua)—not a second schema fact.
_Avoid_: Engine special-case on the name; free-picked secondary lot enum for
continuity; storing continuesContainerId on minted containers; SPA branches on
this field name beyond the generic hidden+presenceMark contract

### Jobs and roles (fixture)

Authored Jobs (titles are Italian Config copy):

| Job id | Role | Notes |
|--------|------|--------|
| `intake` | Ingresso | truckArrive → register (`many`) → weightCheck |
| `lot_processing` | Lavorazione | startLot → weigh / markProgress / containerReg → closeLot; leaf-only `submit: many` where multi-pass |
| Passport Lock Job | Passport | subject = **lot**; eligibility `fullyProcessed` ∧ not locked |

Config role slugs in this fixture:

| Slug | Tasks |
|------|--------|
| `administration` | intake + passport Tasks |
| `line` | weigh + markProgress only |
| `logistic` | lot_processing bookends: startLot, containerReg, closeLot, and other non-line non-admin plant Tasks |

Multi-pass leaves use platform **submit cardinality** `many` + Domain
**readyWhen**—not Task groups / `unlockFrom`. See
[`docs/adr/0008-org-task-submit-cardinality.md`](../adr/0008-org-task-submit-cardinality.md).

### Authoring conventions

- **Italian labels** (titles, enumNames, QR messages) stay in the YAML as plant
  product copy (ADR-0013 track 2)—not Paraglide.
- **Catalogs** in this fixture: clients, tareKinds, materials (bound as
  `$catalogs`). No turni catalog (no shift Domain).
- **Seed / smoke:** `task seed:feragame-config`, `seed:feragame-domain`,
  `seed:feragame-identity`, `seed:feragame-full`; checklist in
  [`docs/agents/plant-config.md`](../agents/plant-config.md).

### Historical plant names (do not revive)

- **load** (entity): renamed to **lot**.
- **working day / shift / whole-day** (Domain): removed — calendar is not Domain
  machinery; no open/close day or turno Jobs.
- **`parked`** (lot session status): removed — session is `open | fullyProcessed`
  only.
