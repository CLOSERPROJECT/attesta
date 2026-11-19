# Feragame

First **Org Config** demo plant: one Organization’s Domain shape and Jobs for a recycling intake / lot-processing line. Nouns here are **payload** authored in Config—not Attesta platform product law.

Org platform vocabulary: [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md). Shared membership: [`GLOSSARY.md`](../../GLOSSARY.md). Map: [`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md). Fixture edit checklist: [`docs/agents/feragame-fixture-authoring.md`](../agents/feragame-fixture-authoring.md).

## Language

### Domain entities

**truck**: Intake truck row.\
_Avoid_: Treating truck as the Passport subject; conflating with Job instance

**lot**: One FIR/intake lot. Owns pallets; is the Passport subject.\
_Avoid_: load (former name); conflating with Job instance

**pallet**: Bancale under a **lot**, with weigh fields and **progress**.

**container**: Booked material weight for a source lot; may stay **incomplete** while the physical bin continues on a later register.

### Lot session and row facts

**lot session status**: Lifecycle state of a lot session on a **lot**: `open` | `fullyProcessed`.\
_Avoid_: `parked`; open/close working day or shift as Domain peers; separate boolean fullyProcessed beside open; parallel open lots

**fullyProcessed**: Terminal **lot session status**: the lot is closed for line processing. The lot’s DPP still needs the final **lot_processing** `lock` Task — see [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md) (Passport lock).\
_Avoid_: boolean `processed`; container register as the writer of this status

**progress**: Plant row fact on some entities (e.g. pallet): unset, `partial`, or `done`.

**incomplete**: Plant Domain fact on a **container**: booked weight for the current lot is frozen while the physical bin may continue on a later register. Distinct from lot session status.\
_Avoid_: Treating incomplete as operator-board status or progress

### Jobs and roles

**intake** (Job): Truck arrival chapter.\
_Avoid_: Treating intake as the Passport path

**lot_processing** (Job): Lot session chapter. Ends with once-only `lock`, which issues a Passport whose subject is that **lot** after **fullyProcessed** — see [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md) (Passport lock).\
_Avoid_: A separate passport Job; treating closeLot as DPP issue

**administration** / **line** / **logistic**: Plant Config role slugs (office, line floor, logistics).\
_Avoid_: Treating these as platform standing (Org admin / Member)

## Relationships

- A **lot** owns zero or more **pallets** and zero or more **containers**.
- At most one **lot** has **lot session status** `open` at a time.
- **Passport** subject for this plant is a **lot**, not a **truck**.
- An **incomplete** **container** may be continued by a later register on another lot.

Open/close and Job sequencing: [`feragame-fixture-authoring.md`](../agents/feragame-fixture-authoring.md).

## Historical / rejected

Do not revive as Feragame nouns.

**load**: Former name for **lot**.

**working day** / **shift** / whole-day Domain peers: Former Domain calendar layer; not plant Domain entities.

**`parked`**: Former lot session status; not in the living enum (`open` | `fullyProcessed`).

**Passport Lock Job**: Former separate Job (`passport` / `select`+`lock`) after close; folded into **lot_processing** as final once-only `lock`.
