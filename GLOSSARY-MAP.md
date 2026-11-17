# Glossary Map

Attesta has three vocabularies. Pick the glossary that matches the work; do not
mix plant nouns into platform ADRs or the Org platform glossary.

## Contexts

- [Org product](./GLOSSARY.md): platform Org Config, Domain, Jobs/Tasks,
  Worklist, Commit, Archive, Passport-as-capability under `/app`. Source of
  truth for Attesta product law and Org ADRs (0005–0013).
- [Feragame](./docs/feragame/GLOSSARY.md): first **Org Config** use case
  (demo plant). Lot session, Jobs `intake` / `lot_processing` / passport—
  **payload**, not platform Decisions. Fixture checklist:
  [`docs/agents/feragame-fixture-authoring.md`](docs/agents/feragame-fixture-authoring.md).
- [Legacy Stream UI](./CONTEXT.md): multi-org Streams, `/my`, HTMX. Historical
  and still accurate for that surface.

## Relationships

- **Org product ↛ Legacy runtime:** `/app` does not load Streams or Stream
  instances. Org Config ≠ Stream blueprint.
- **Feragame → Org product:** Feragame is one Organization’s Config + Domain shape
  evaluated by the dumb executor (`server/internal/org`). Plant entity names and open/close laws
  belong in the Feragame glossary / fixture, not in Attesta ADRs.
- **Shared org/people words:** Organization, Affiliation, and membership roles
  still matter for session → who may act; **Config role** slugs should
  align with organization role slugs where Cerbos checks membership.
- **Merge later (Org ⟷ Legacy):** one glossary; Stream terms become
  historical where the Org product wins (see
  `docs/research/attesta-app-living-product-decisions.md`). Feragame stays a
  plant glossary as long as it is the demo Config.