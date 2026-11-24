# Glossary Map

Attesta has multiple vocabularies. **Org** and **Stream** name different product nouns—do not mix them. `/my` can host both Formata Streams and **Org-backed streams** ([ADR-0014](docs/adr/0014-org-config-as-stream.md)). Shared membership lives in the root glossary.

Every vocabulary file is named `GLOSSARY.md`. This map is `GLOSSARY-MAP.md`. How to edit an entry: [`docs/agents/glossary.md`](docs/agents/glossary.md).

## Shared (both product surfaces)

- [Root glossary](./GLOSSARY.md): **Org** vs **Stream** framing, then Organization, Affiliation, standing (Org admin / Member), Organization roles, Registration, verification, Invitation / Join / creation requests. Session → who may act for both surfaces.

## Product surfaces

- [Org](./docs/org/GLOSSARY.md) (`/app`, Org-backed `/my`): platform Org Config, Domain, **Org stream** / **Org stream instance**, Jobs/Tasks, Worklist, Commit, Passport-as-capability. Source of truth for Attesta product law and Org ADRs (0005–0015).
- [Stream](./docs/stream/GLOSSARY.md) (`/my`, Formata): multi-org Formata Streams, Stream instances, Steps/Substeps, HTMX operator UI. Historical Formata runtime; `/my` also lists Org-backed streams (nouns for those live in the Org glossary).

## Plant payloads (Org Config instances)

- [Feragame](./docs/feragame/GLOSSARY.md): first Org Config demo plant. Lot session, Jobs `intake` / `lot_processing` (Passport via final `lock`)—**payload**, not platform Decisions. Fixture checklist: [`docs/agents/feragame-fixture-authoring.md`](docs/agents/feragame-fixture-authoring.md).

## Relationships

- **`/my` hosts both:** Formata Streams and Org-backed streams (Org Config + Domain via the `org` package). `/app` is the instance-scoped Org operator surface, not a separate runtime from those Org streams.
- **Feragame → Org:** Feragame is one Organization’s Config + Domain shape evaluated by the dumb executor. Plant entity names and open/close laws belong in the Feragame glossary / fixture, not in Attesta ADRs.
- **Shared → both surfaces:** Organization / Affiliation / Organization roles gate who may act; **Config role** slugs (Org) should align with **Organization role** slugs where Cerbos checks membership.
- **Role catalog vs Role catalog:** Organization Role catalog lives in root [`GLOSSARY.md`](GLOSSARY.md); Config Role catalog lives in [`docs/org/GLOSSARY.md`](docs/org/GLOSSARY.md)—do not conflate.
- **Stream nouns stay Stream:** Formata Stream / Stream instance / Step / Substep stay in the Stream glossary; **Org stream** / **Org stream instance** / Job / Task live in the Org glossary. Feragame stays a plant glossary while it is the demo Config.
