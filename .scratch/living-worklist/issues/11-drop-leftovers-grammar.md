# 11: Drop leftovers grammar (one submit + draft)

**What to build:** Retire Leftovers as product grammar (`completeness`, leftover Worklist reopen, leftover completeness prep). Rows still come from Domain reads into the form. Full vs partial Commit is JSON Schema `required` (projected from current read ids) or not. One successful Submission finishes the Task on that instance; unlock/startsWhen use Submission relations only. Operators keep WIP in draft and Invia once — no multi-pass reopen. Silent progress enum fallback and SPA leftover-id twins are deleted.

**Blocked by:** 10 Domain kinds map.

**Status:** done (squashed into `feat/attesta-rework`)

**Execution:** sequential subagents only (A → B → C).

### Subagent 11A — Workflow law

- [x] Worklist marks a Task `done` after Submission on the instance (no “open while leftover ids remain”)
- [x] Unlock / startsWhen no longer infer gate eligibility from `completeness`
- [x] Go worklist/unlock tests match Submission-only law; Feragame still unlocks as authored by relations

### Subagent 11B — Submit shape

- [x] Full-map Tasks get projected `required` for current read ids; validation is schema, not `completeness: all` prep
- [x] Partial maps omit that full `required`; untouched keys are not forced
- [x] Leftover completeness prep / silent `partial`/`done` fallback are gone; prep tests updated

### Subagent 11C — Plant + SPA

- [x] Feragame lineLoad / palletWeight are one-Commit flows (draft until Invia); no multi-pass leftover reopen
- [x] SPA leftover-id helpers removed; tests use server leftover ids / Worklist only where still needed mid-migration
- [x] Glossary/ADR wording drops Leftovers-as-subsystem (Domain reads + Submission law)

**Decisions (grill 2026-11-02):** Leftovers noun retired; Domain + reads are the feature; drop `completeness`; one Submission; no lineLoad multi-pass; Completo one-shot absorbed by draft→single submit.

**Out of scope:** Completo-as-separate-product; catalogs/bind/domain peels (08–10); inventing server-side draft persistence (client draft is enough unless already present).

**Landed:** after review+fix — Worklist JSDoc; partial-map `required` omit test. Residual: server `leftover*` wire/identifiers rename (mid-migration).
