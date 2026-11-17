# Glossary edit

How to add or change Attesta glossary entries. Layer routing lives only in [`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md). Decisions live in [`docs/adr/`](../adr/) (format: [`README.md`](../adr/README.md)). Config / plant mechanics live in authoring docs, not glossary bodies.

## Steps

1. **Pick the layer** — open [`GLOSSARY-MAP.md`](../../GLOSSARY-MAP.md) and choose exactly one vocabulary file (root membership, Org, Stream, or plant). Done when the term’s home file is named and it matches the map.
2. **Write or shrink the entry** — apply **Entry shape** and **What belongs** below. Prefer tightening an existing term over adding a near-synonym. Done when the definition states what the term _is_ in ≤2 sentences and rival names sit under `_Avoid_`.
3. **Place rejected names** — retired product or code nouns go under `## Historical / rejected` at the end of that file (not inline in Language). Done when living Language has no historical/rejected markers.
4. **Point at Decision or authoring** — if meaning depends on a Decision, link the ADR; if the detail is authoring/fixture procedure, link [`org-config-authoring.md`](org-config-authoring.md) or [`feragame-fixture-authoring.md`](feragame-fixture-authoring.md). Done when the living entry is vocabulary-only and every mechanism sits behind those links.

## Entry shape

```md
**Term**: One or two sentences: what it _is_. Optional ADR or authoring pointer.\
_Avoid_: Rival names; conflations with other layers
```

Group under `###` subheadings when a cluster is clear. Keep `## Language` for living vocabulary; keep `## Historical / rejected` last.

## What belongs

**Vocabulary** (glossary): product or plant nouns for that layer; short distinctions that define the noun; `_Avoid_` rivals; ADR pointers when a Decision locks meaning.

**Decision** ([`docs/adr/`](../adr/)): hard-to-reverse trade-offs, seams, and platform law — see [`adr/README.md`](../adr/README.md).

**Authoring** ([`org-config-authoring.md`](org-config-authoring.md), [`feragame-fixture-authoring.md`](feragame-fixture-authoring.md)): transform keys, open/close procedures, YAML sequencing, fixture checklists.

Cross-layer: define a noun once in its home glossary; other files only link or `_Avoid_`. Plant nouns stay in the plant glossary; Org ADRs use platform nouns only.

## Done checklist

Every changed living term:

- [ ] Correct layer file per the map
- [ ] _Is_-definition ≤2 sentences
- [ ] Rivals only under `_Avoid_`
- [ ] Mechanisms live in ADR or authoring links
- [ ] Rejected names live under `## Historical / rejected`
