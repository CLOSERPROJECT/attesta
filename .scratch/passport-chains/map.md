# Wayfinder map: legacy Streams → Passport chains

Label: wayfinder:map

## Destination

A handoff-ready spec for **Passport chains** (Org Configs of different Organizations linked because each downstream Config takes upstream Passports as inputs), plus a decided conversion target for every legacy Stream: which Org Configs it becomes and how they link. Writing the converted Configs and building the feature happen after this map.

## Notes

- **Domain:** Org surface (`/app`, `server/internal/org`). Read [`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md), [`docs/org/GLOSSARY.md`](../../docs/org/GLOSSARY.md), [`docs/stream/GLOSSARY.md`](../../docs/stream/GLOSSARY.md), [ADR-0007](../../docs/adr/0007-org-dpp-production.md), [`docs/untp.md`](../../docs/untp.md), [`docs/agents/org-config-authoring.md`](../../docs/agents/org-config-authoring.md), and the Feragame fixture (`server/internal/org/fixtures/feragame-config.yaml`).
- **Skills:** grilling + domain-modeling for every grilling ticket; research for research tickets; prototype for prototype tickets.
- **Tracker:** local Markdown. Tickets live in `issues/NN-<slug>.md` with `Type:`, `Status:` (`open` / `claimed` / `resolved`) and `Blocked by:` lines. Research findings go to `research/` in this folder; there are no research branches.
- **Standing preferences:**
  - Prototype phase: breaking changes to routes, Config shape and persistence are allowed for this effort, so don't contort the design to preserve them.
  - Dumb executor: chaining is declared in Org Config, with no Feragame- or org-named branches in Go or `/app`.
- **Settled while charting:**
  - The domain term is **Passport chain**. "Stream" becomes its user-facing label only once legacy Streams are retired; until then the UI avoids "Stream" for chains.
  - A Config declares an upstream source at the type level (for example, "this Job takes containers from Feragame lot Passports"). A Task binds the concrete upstream Passports to a Job instance at runtime. A Config never names one Passport instance.
  - Downstream orgs inherit the upstream **Passport snapshot** only, never the upstream org's live Domain.
  - Every org-to-org hand-off in a chain is an issued upstream Passport. Passports stay public-by-link.
  - One Org Config per org: an org's legacy Steps, across all Streams, become Jobs in that org's single Config.
  - Converted Configs are YAML files in the repo, one per org, loaded by seed. An authoring UI is not part of this effort.
  - Chains run on one Attesta instance first, but upstream Passports are identified by Digital Link so external sources stay possible.
  - UNTP has no name for the chain. It calls it the value chain, traversed as a linked-data graph; upstream identifiers are **inputs** (`inputEPCList`) of Digital Traceability Events, mainly `TransformationEvent`. How the DPP links them depends on the UNTP version (see the UNTP research ticket).

## Decisions so far

<!-- one line per resolved ticket: [title](issues/NN-slug.md): gist -->
- [UNTP linking of input Passports to outputs](issues/01-untp-input-passport-linking.md): each org issues a `TransformationEvent` for its own work, naming upstream identifiers in `inputEPCList`. Partial lots go through identified containers. Since 0.7.0 the DPP links events and input DPPs through `relatedDocument` and the resolver; `traceabilityInformation` is 0.6-only. The latest release is 0.7.0; 1.0 is a draft built on unmodified EPCIS 2.0 events.
- [Export the legacy Stream catalog](issues/02-export-legacy-stream-catalog.md): production has 23 Streams (20 distinct; three are byte-identical re-imports), all hand-authored, with 55 instances and 7 issued DPPs. Families: a six-org Indium/Gallium recycling chain, demo organization-1/2/3 Streams, many single-org Streams, and one non-product onboarding flow (PID/EAA).

## Not yet specified

- **One review ticket per kept legacy Stream.** These are created from the triage's kept list once the Step-to-Config mapping is decided. Each answers "what does Stream X become?"
- **Issuance order:** must upstream Passports be issued before the downstream Job can bind them? What happens when an upstream Passport is reissued or a new Config version changes its snapshot shape?
- **User-facing Stream surface:** a chain view walking from a final Passport back to its inputs, and whether the legacy discovery features (categories, sub-categories, `/streams/public`) carry over to chains.
- **Cross-instance and external upstream Passports:** trust, verification and fetching when the input's Digital Link resolves elsewhere.
- **Private hand-offs:** inputs that shouldn't be public-by-link (UNTP Decentralised Access Control).
- **DTE emission from Org:** Org Passports emit no traceability events today. The Stream side emits 0.7.0-shaped `ModifyEvent`s under the 0.6 event link-type URI. What Org emits follows from the UNTP version ticket and the mechanism lock.
- **Config naming drift:** ADRs and the glossary call the root block `passport`, the code reads `dpp:`, and Tasks opt in with `onSubmit.passport`. Settle this before the chain declaration adds another key.
- **Mongo key round-trip:** `decodeMongoKey` turns every `_` into `.`, so authored keys containing `_` that refer to upstream Passports may not round-trip.

## Out of scope

- Removing the legacy Stream runtime (`/my` Stream routes, Formata builder, Stream DPP fallback on `/01/…`).
- Migrating Stream instances (in-flight processes and their progress). Already-issued Stream Passports keep resolving.
- Third-party JSON ingestion: the second half of [Derived DPP and integration with 3rd parties](https://github.com/CLOSERPROJECT/attesta/issues/201). The derived-DPP half is this effort.
- An Org Config authoring UI or builder.
