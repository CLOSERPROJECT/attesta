# Passport chain vocabulary and the user-facing Stream label

Type: grilling
Status: open
Map: [map.md](../map.md)

## Question

What are the canonical nouns for the parts of a Passport chain, and when may the UI call a chain a "Stream"?

To settle:

- The name for the Config-level declaration ("this Job takes inputs from Org A's lot Passports").
- The name for the runtime binding of concrete upstream Passports to a Job instance. Candidate: **Passport input**, following UNTP `inputEPCList`.
- Upstream versus downstream wording, and whether "derived DPP" from [Derived DPP and integration with 3rd parties](https://github.com/CLOSERPROJECT/attesta/issues/201) is the same concept as a Passport chain or a narrower case (DPP1 plus fields, within a single org).
- What exactly happens when "Stream" switches from the legacy noun to the user-facing label for Passport chains: when legacy Streams are retired, or earlier with a distinguishing qualifier.

Resolution updates [`docs/org/GLOSSARY.md`](../../../docs/org/GLOSSARY.md) and [`GLOSSARY-MAP.md`](../../../GLOSSARY-MAP.md) per [`docs/agents/glossary.md`](../../../docs/agents/glossary.md).
