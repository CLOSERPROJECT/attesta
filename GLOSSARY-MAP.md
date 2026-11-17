# Glossary Map

Attesta has multiple vocabularies. Pick the glossary that matches the work; do
not mix plant nouns into platform ADRs or the Org product glossary.

Every vocabulary file is named `GLOSSARY.md`. This map is `GLOSSARY-MAP.md`.

## Shared (both product surfaces)

- [Organization & affiliation](./docs/org/GLOSSARY.md): Organization,
  Affiliation, standing (Org admin / Member), Organization roles, Registration,
  verification, Invitation / Join / creation requests. Session → who may act for
  Legacy Stream and Org product.

## Product surfaces

- [Org product](./GLOSSARY.md) (`/app`): platform Org Config, Domain, Jobs/Tasks,
  Worklist, Commit, Archive, Passport-as-capability. Source of truth for Attesta
  product law and Org ADRs (0005–0013).
- [Legacy Stream](./docs/stream/GLOSSARY.md) (`/my`, Formata): multi-org Streams,
  Stream instances, Steps/Substeps, HTMX operator UI. Historical surface; still
  accurate for that runtime.

## Plant payloads (Org Config instances)

- [Feragame](./docs/feragame/GLOSSARY.md): first Org Config demo plant. Lot
  session, Jobs `intake` / `lot_processing` / passport—**payload**, not platform
  Decisions. Fixture checklist:
  [`docs/agents/feragame-fixture-authoring.md`](docs/agents/feragame-fixture-authoring.md).

## Relationships

- **Org product ↛ Legacy runtime:** `/app` does not load Streams or Stream
  instances. Org Config ≠ Stream blueprint.
- **Feragame → Org product:** Feragame is one Organization’s Config + Domain
  shape evaluated by the dumb executor. Plant entity names and open/close laws
  belong in the Feragame glossary / fixture, not in Attesta ADRs.
- **Shared → both surfaces:** Organization / Affiliation / Organization roles
  gate who may act; **Config role** slugs (Org product) should align with
  **Organization role** slugs where Cerbos checks membership.
- **Role catalog vs Role catalog:** Organization Role catalog lives in
  [`docs/org/GLOSSARY.md`](docs/org/GLOSSARY.md); Config Role catalog lives in
  root [`GLOSSARY.md`](GLOSSARY.md)—do not conflate.
- **Stream nouns stay Stream:** when Org product wins a concept, Stream terms
  become historical for that concept; Feragame stays a plant glossary while it
  is the demo Config.
