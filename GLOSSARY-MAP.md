# Glossary Map

Attesta has multiple vocabularies. **Org** and **Stream** name different product surfaces—do not mix their nouns. Shared membership lives in the root glossary.

Every vocabulary file is named `GLOSSARY.md`. This map is `GLOSSARY-MAP.md`. How to edit an entry: [`docs/agents/glossary.md`](docs/agents/glossary.md).

## Shared (both product surfaces)

- [Root glossary](./GLOSSARY.md): **Org** vs **Stream** framing, then Organization, Affiliation, standing (Org admin / Member), Organization roles, Registration, verification, Invitation / Join / creation requests. Session → who may act for both surfaces.

## Product surfaces

- [Org](./docs/org/GLOSSARY.md) (`/app`): platform Org Config, Domain, Jobs/Tasks, Worklist, Commit, Archive, Passport-as-capability. Source of truth for Attesta product law and Org ADRs (0005–0013).
- [Stream](./docs/stream/GLOSSARY.md) (`/my`, Formata): multi-org Streams, Stream instances, Steps/Substeps, HTMX operator UI. Historical surface; still accurate for that runtime.

## Plant payloads (Org Config instances)

- [Feragame](./docs/feragame/GLOSSARY.md): first Org Config demo plant. Lot session, Jobs `intake` / `lot_processing` / passport—**payload**, not platform Decisions. Fixture checklist: [`docs/agents/feragame-fixture-authoring.md`](docs/agents/feragame-fixture-authoring.md).

## Relationships

- **Org ↛ Stream runtime:** `/app` does not load Streams or Stream instances. Org Config ≠ Stream blueprint.
- **Stream → Passport chain:** legacy multi-org Streams convert into **Passport chains** (one Org Config per org, linked by Passports); the Stream runtime retires after.
- **Feragame → Org:** Feragame is one Organization’s Config + Domain shape evaluated by the dumb executor. Plant entity names and open/close laws belong in the Feragame glossary / fixture, not in Attesta ADRs.
- **Shared → both surfaces:** Organization / Affiliation / Organization roles gate who may act; **Config role** slugs (Org) should align with **Organization role** slugs where Cerbos checks membership.
- **Role catalog vs Role catalog:** Organization Role catalog lives in root [`GLOSSARY.md`](GLOSSARY.md); Config Role catalog lives in [`docs/org/GLOSSARY.md`](docs/org/GLOSSARY.md)—do not conflate.
- **Stream nouns stay Stream:** when Org wins a concept, Stream terms become historical for that concept; Feragame stays a plant glossary while it is the demo Config.
