# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Before exploring, read these

- **`GLOSSARY-MAP.md`** at the repo root: which glossary applies (Living
  platform, Feragame plant, Legacy Stream). Then read that glossary.
- **`docs/adr/`**: read ADRs that touch the area you're about to work in.
  Living ADRs lock **platform** Decisions only—not plant YAML.
- Plant fixture edits: also
  [`plant-config.md`](plant-config.md) and
  [`docs/feragame/GLOSSARY.md`](../feragame/GLOSSARY.md).

If the map or a glossary is missing, **proceed silently**. Don't flag their
absence; don't suggest creating them upfront. The `/domain-modeling` skill
creates them lazily when terms or decisions actually get resolved.

## File structure (this repo)

```
/
├── GLOSSARY-MAP.md              ← which glossary
├── GLOSSARY.md                  ← Living Attesta (platform)
├── CONTEXT.md                   ← Legacy Stream
├── docs/
│   ├── feragame/GLOSSARY.md     ← first plant / Org Config payload
│   └── adr/                     ← platform Decisions only
```

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal,
a hypothesis, a test name), use the term as defined in the glossary
`GLOSSARY-MAP.md` selected for that work. Don't drift to synonyms the glossary
explicitly avoids. Don't promote Feragame plant nouns into Living platform
docs or Attesta ADRs.

If the concept you need isn't in the right glossary yet, that's a signal:
either you're inventing language the project doesn't use (reconsider) or
there's a real gap (note it for `/domain-modeling`).

## Flag ADR conflicts

If your output contradicts an existing ADR, surface it explicitly rather than silently overriding:

> _Contradicts ADR-0007 (event-sourced orders), but worth reopening because…_
