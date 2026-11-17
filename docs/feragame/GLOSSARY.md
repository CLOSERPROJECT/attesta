# Feragame

First **Org Config** demo plant: one Organization’s Domain shape and Jobs for a
recycling intake / lot-processing line. Nouns here are **payload** authored in
Config—not Attesta platform product law.

Platform vocabulary: [`GLOSSARY.md`](../../GLOSSARY.md). Map:
[`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md). Fixture edit checklist:
[`docs/agents/feragame-fixture-authoring.md`](../agents/feragame-fixture-authoring.md).

## Language

### Domain entities

**truck**:
Intake truck row (gross weight, scale readings, weight-check fields).
_Avoid_: Treating truck as the Passport subject; conflating with Job instance

**lot**:
One FIR/intake lot. Owns pallets; is the Passport subject. Lot-processing work
runs under a **lot session status** on this entity.
_Avoid_: load (former name); conflating with Job instance; treating lot as a
Task group occurrence

**pallet**:
Bancale under a **lot**, with weigh fields and **progress**.
_Avoid_: Engine special-case on the name

**container**:
Booked material weight for a source lot; may stay **incomplete** while the
physical bin continues on a later register.
_Avoid_: Engine special-case on the name

### Lot session and row facts

**lot session status**:
Enum on a **lot**: `open` | `fullyProcessed`. `startLot` opens; `closeLot` sets
`fullyProcessed` when every pallet on that lot has weight and `progress = done`.
At most one lot is `open`; the next `startLot` waits until that lot is
`fullyProcessed`.
_Avoid_: `parked`; open/close working day or shift as Domain peers; separate
boolean fullyProcessed beside open; parallel open lots; putting this table in
Attesta ADRs

**fullyProcessed**:
Terminal **lot session status**: the lot is closed for line processing. Set only
by **closeLot**. Passport eligibility still requires this status (and not
locked).
_Avoid_: boolean `processed` as Worklist status; using fullyProcessed as unlock
grammar; using fullyProcessed as “Passport done” alone without the lock fact;
container register as the writer of this status

**progress**:
Plant row fact on some entities (e.g. pallet): unset, `partial`, or `done`.
Draft ≠ progress; Worklist done is Submission law.
_Avoid_: boolean `processed` as a Worklist gate; treating progress enums as
unlock grammar

**incomplete**:
Plant Domain fact on a **container**: booked weight for the current lot is
frozen while the physical bin may continue on a later register. Distinct from
lot session status.
_Avoid_: Treating incomplete as Worklist or progress; Task-group Close

### Jobs and roles

**intake** (Job):
Truck arrival chapter: arrive → register → weight check. Job instance id matches
the **truck** id.
_Avoid_: Treating intake as the Passport path

**lot_processing** (Job):
Lot session chapter: startLot → weigh / markProgress / containerReg → closeLot.
_Avoid_: Modeling multi-pass weigh/register as Task groups

**Passport Lock Job**:
Issues a Passport whose subject is a **lot**, eligible when
`fullyProcessed` ∧ not locked.
_Avoid_: Auto-issue when the lot becomes fullyProcessed

**administration** / **line** / **logistic** (Config roles):
Fixture role slugs: administration covers intake + passport Tasks; line covers
weigh + markProgress; logistic covers lot_processing bookends and other
non-line plant Tasks.
_Avoid_: Plant-named role branches in Go or SPA executors

## Relationships

- A **lot** owns zero or more **pallets** and zero or more **containers**.
- At most one **lot** has **lot session status** `open` at a time.
- **Passport** subject for this plant is a **lot**, not a **truck**.
- An **incomplete** **container** may be continued by a later register on another
  lot; that link is authored in Config, not a Domain entity field.

## Example dialogue

> **Dev:** “When every pallet on a **lot** has **progress** `done`, is the
> Passport issued?”
> **Domain expert:** “No — **closeLot** sets **fullyProcessed**; Passport still
> needs an explicit Lock Commit on that **lot**.”

## Historical plant names

Do not revive: **load** (now **lot**); working day / shift / whole-day as Domain;
**`parked`** lot session status.
