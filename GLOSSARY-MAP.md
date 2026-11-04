# Glossary Map

Attesta currently has two product languages until they merge.

## Contexts

- [Living Attesta](./GLOSSARY.md): single-org Org Config, living domain, jobs/tasks under `/app`. Source of truth for new product work.
- [Legacy Stream UI](./CONTEXT.md): multi-org Streams, `/my`, HTMX. Historical and still accurate for that surface.

## Relationships

- **Living ↛ Legacy runtime:** Living `/app` does not load Streams or Stream instances. Org Config ≠ Stream blueprint.
- **Shared org/people words:** Organization, Affiliation, and membership roles still matter for session → who may act; living **Config role** slugs should align with organization role slugs where Cerbos checks membership.
- **Merge later:** one glossary; Stream terms become historical where living wins (see `docs/research/attesta-app-living-product-decisions.md`).
