# Future capture and Domain — salvage notes

> **Not product law.** Living Org vocabulary is [`docs/org/GLOSSARY.md`](../org/GLOSSARY.md); shared membership is root [`GLOSSARY.md`](../../GLOSSARY.md); decisions in [`docs/adr/`](../adr/) (through ADR-0013). This note records north-stars and backlog kept from the interpretation-package / resource-model exploration after that plant model was simplified (act-now home, `submit: many`, no day/shift Domain calendar). Do not treat it as an ADR or roadmap.

Offline support is **eventual**. Until then, Commit may keep mutating the Domain; new work should not make a later offline cut harder than it needs to be.

Exploration branch kept for archaeology only — do not merge its vision / interpretation-package / sketch / resource ADR stubs onto master.

---

## North-stars

**Three layers of truth.** Separate what was reported (Observation), what we believe happened (Reconstruction / world model), and what has been accepted (Review + Seal). A Digital Product Passport is an output of the sealed layer, not the purpose of the system.

**Split Commit (design constraint, not a project).** Today's Commit both accepts a report and updates the believed world. Longer term, capture stores an Observation and reconstruction derives Domain; `readyWhen` and sequence become guidance and findings, not the only place truth can enter. Do not implement a two-phase engine now. When touching Commit or Submission, prefer shapes that could become “append Observation + derive” without a second identity rewrite.

**Replayable Domain.** Domain should eventually be rebuildable from capture history. Snapshots are an optimization. Avoid APIs that only make sense if Domain is irrevocably “whatever Commit last wrote” with no usable input history.

**Corrections.** Fix bad reports with a new record that supersedes an earlier one, then recompute — not reverse state-machine edges or silent rewrites of history. No corrections API until an observation envelope exists; still prefer append-only facts over “edit the entity as if the old submit never happened” when changing many-submit UX.

**Device-minted IDs (with offline).** Entity and instance ids created on the device and preserved by the server; display labels stay separate. Defer the cutover until offline is real; avoid new create paths that assume server-only mint forever.

---

## Explicitly not kept

- Interpretation package as near-term replacement for Org Config Jobs/Tasks
- Resource / event / scope ADR stubs (numbers collide with living ADRs)
- `sketch.yaml` and board UI assuming working day → shift scopes
- Authoring rewrite (“tables as views”) as committed direction

---

## Backlog

| Issue | Topic |
| --- | --- |
| [#291](https://github.com/CLOSERPROJECT/attesta/issues/291) | Observation envelope under today's Commit |
| [#292](https://github.com/CLOSERPROJECT/attesta/issues/292) | Optimistic concurrency on Domain Put |
| [#293](https://github.com/CLOSERPROJECT/attesta/issues/293) | Measure Domain document size vs 16 MB |

Envelope work makes Submissions larger; size measurement (#293) should stay in view when #291 lands. Concurrency (#292) is independent and valid for online-only use.
