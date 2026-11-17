# Feragame

First **Org Config** demo plant: one Organization’s Domain shape and Jobs for a
recycling intake / lot-processing line. Nouns here are **payload** authored in
Config—not Attesta platform product law.

Org platform vocabulary: [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md). Shared
membership: [`GLOSSARY.md`](../../GLOSSARY.md). Map:
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
_Avoid_: load (former name); conflating with Job instance

**pallet**:
Bancale under a **lot**, with weigh fields and **progress**.

**container**:
Booked material weight for a source lot; may stay **incomplete** while the
physical bin continues on a later register.

### Lot session and row facts

**lot session status**:
Enum on a **lot**: `open` | `fullyProcessed`. Opening a lot session sets `open`;
closing it sets `fullyProcessed` when every pallet on that lot has weight and
`progress = done`. At most one lot is `open`; the next open waits until that
lot is `fullyProcessed`.
_Avoid_: `parked`; open/close working day or shift as Domain peers; separate
boolean fullyProcessed beside open; parallel open lots

**fullyProcessed**:
Terminal **lot session status**: the lot is closed for line processing. Set by
close; Passport still needs Lock on that lot (see root glossary / ADRs).
_Avoid_: boolean `processed`; container register as the writer of this status

**progress**:
Plant row fact on some entities (e.g. pallet): unset, `partial`, or `done`.

**incomplete**:
Plant Domain fact on a **container**: booked weight for the current lot is
frozen while the physical bin may continue on a later register. Distinct from
lot session status.
_Avoid_: Treating incomplete as operator-board status or progress

### Jobs and roles

**intake** (Job):
Truck arrival chapter: arrive → register → weight check.
_Avoid_: Treating intake as the Passport path

**lot_processing** (Job):
Lot session chapter (open through close for a lot).

**Passport Lock Job**:
Issues a Passport whose subject is a **lot**. Still needs Lock after the lot is
**fullyProcessed** — see [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md)
(Passport lock).

**administration** / **line** / **logistic**:
Plant Config role slugs (office, line floor, logistics). Which Tasks each may
run: [`feragame-fixture-authoring.md`](../agents/feragame-fixture-authoring.md).
_Avoid_: Plant-named role branches in executors

## Relationships

- A **lot** owns zero or more **pallets** and zero or more **containers**.
- At most one **lot** has **lot session status** `open` at a time.
- **Passport** subject for this plant is a **lot**, not a **truck**.
- An **intake** Job instance id matches the **truck** id.
- An **incomplete** **container** may be continued by a later register on another
  lot.

## Example dialogue

> **Dev:** “When every pallet on a **lot** has **progress** `done`, is the
> Passport issued?”
> **Domain expert:** “No — **closeLot** sets **fullyProcessed**; Passport still
> needs an explicit Lock Commit on that **lot**.”

## Historical plant names

Do not revive: **load** (now **lot**); working day / shift / whole-day as Domain;
**`parked`** lot session status.
