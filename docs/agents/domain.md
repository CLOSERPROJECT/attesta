# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Before exploring, read these

- **`GLOSSARY-MAP.md`** at the repo root: which glossary applies (shared
  membership, Org `/app`, Stream `/my`, Feragame). Then read that glossary.
- **`docs/adr/`**: read ADRs that touch the area you're about to work in.
  Org ADRs (0005–0013) lock **platform** Decisions only—not plant YAML.
  When writing or editing an ADR, follow [`../adr/README.md`](../adr/README.md).
- When adding or changing a glossary entry, follow
  [`glossary.md`](glossary.md).
- Org Config authoring (gates, `onCommit`, transforms):
  [`org-config-authoring.md`](org-config-authoring.md).
- Feragame fixture edits: also
  [`feragame-fixture-authoring.md`](feragame-fixture-authoring.md) and
  [`docs/feragame/GLOSSARY.md`](../feragame/GLOSSARY.md).

If the map or a glossary is missing, **proceed silently**. Don't flag their
absence; don't suggest creating them upfront. The `/domain-modeling` skill
creates them lazily when terms or decisions actually get resolved; new entries
still follow [`glossary.md`](glossary.md).

## File structure (this repo)

```
/
├── GLOSSARY-MAP.md                   ← which glossary
├── GLOSSARY.md                       ← Org vs Stream + shared membership
├── docs/
│   ├── org/GLOSSARY.md               ← Org (`/app`) product vocabulary
│   ├── stream/GLOSSARY.md            ← Stream (`/my`)
│   ├── feragame/GLOSSARY.md          ← first plant / Org Config payload
│   └── adr/                          ← platform Decisions; format in README.md
├── docs/agents/
│   ├── glossary.md                   ← how to edit glossary entries
│   ├── domain.md                     ← this file (consume while exploring)
│   ├── org-config-authoring.md       ← platform Config authoring
│   └── feragame-fixture-authoring.md ← Feragame fixture checklist
```

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal,
a hypothesis, a test name), use the term as defined in the glossary
`GLOSSARY-MAP.md` selected for that work. Don't drift to synonyms the glossary
explicitly avoids. Don't promote Feragame plant nouns into Org docs or
Attesta ADRs.

If the concept you need isn't in the right glossary yet, that's a signal:
either you're inventing language the project doesn't use (reconsider) or
there's a real gap — resolve it with `/domain-modeling` and
[`glossary.md`](glossary.md).

## Flag ADR conflicts

If your output contradicts an existing ADR, surface it explicitly rather than silently overriding:

> _Contradicts ADR-0007 (event-sourced orders), but worth reopening because…_
