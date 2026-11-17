# ADR format

Architecture Decision Records in this directory. Attesta follows Matt Pocock’s
short form: record *that* a decision was made and *why*.

## Template

```md
# {Short title of the decision}

{1–3 sentences: context, decision, why.}
```

An ADR may be a single paragraph. Optional sections only when they earn keep:

- **Status** (`proposed` | `accepted` | `deprecated` | `superseded by ADR-NNNN`)
  when a decision is revisited
- **Considered options** when rejected alternatives are worth remembering
- **Consequences** when non-obvious downstream effects need calling out
- **Amended by / Amends** when a later ADR changes current law (rewrite the
  amended ADR to **current** decision text; link the amend—do not keep a
  changelog of wire shapes)

## Numbering

Filenames: `NNNN-slug.md`. Scan this directory for the highest `NNNN` and
increment by one.

## When to add an ADR

All three must hold:

1. **Hard to reverse** — changing later has meaningful cost
2. **Surprising without context** — a future reader will wonder why the code
   looks this way
3. **Real trade-off** — genuine alternatives, picked for stated reasons

## What belongs here

Platform **Decisions**: architectural shape, context boundaries, integration
seams, lock-in technology choices, deliberate deviations, constraints invisible
in code, non-obvious rejections.

Org product ADRs lock **platform** law only. Plant Config YAML (entity names,
predicates, Italian labels, demo fixture shape) is payload—glossary / fixture,
not an ADR.

## Decision vs mechanism

Keep: the choice, the boundary, the rationale, rejected alternatives that
matter.

Leave out (ops docs, agent how-tos, or the code): runbooks, host script paths,
wire/DTO field inventories, API handler names, library pins, test checklists,
build-out task lists, and reversible Phase-N storage protocols unless the
protocol *is* the decision.

When an ADR names a seam, state what is locked vs what may change without a new
decision. Prefer rewriting superseded text to current law over appending
“Later:” history blocks.
