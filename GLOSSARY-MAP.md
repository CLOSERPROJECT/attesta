# Glossary Map

Attesta has three vocabularies. Pick the glossary that matches the work; do not
mix plant nouns into platform ADRs or the living platform glossary.

## Contexts

- [Living Attesta](./GLOSSARY.md): platform Org Config, Domain, Jobs/Tasks,
  Worklist, Commit, Archive, Passport-as-capability under `/app`. Source of
  truth for Attesta product law and living ADRs.
- [Feragame](./docs/feragame/GLOSSARY.md): first living **Org Config** use case
  (demo plant fixture). Lot session, Jobs `intake` / `lot_processing` /
  passport, Italian labels—**payload**, not platform Decisions.
- [Legacy Stream UI](./CONTEXT.md): multi-org Streams, `/my`, HTMX. Historical
  and still accurate for that surface.

## Relationships

- **Living ↛ Legacy runtime:** Living `/app` does not load Streams or Stream
  instances. Org Config ≠ Stream blueprint.
- **Feragame → Living:** Feragame is one Organization’s Config + Domain shape
  evaluated by the living dumb executor. Plant entity names and open/close laws
  belong in the Feragame glossary / fixture, not in Attesta ADRs.
- **Shared org/people words:** Organization, Affiliation, and membership roles
  still matter for session → who may act; living **Config role** slugs should
  align with organization role slugs where Cerbos checks membership.
- **Merge later (Living ⟷ Legacy):** one glossary; Stream terms become
  historical where living wins (see
  `docs/research/attesta-app-living-product-decisions.md`). Feragame stays a
  plant glossary as long as it is the demo Config.
