# Locked — living `/app` home hard cut (2026-11-13)

Grill with docs. **Do not re-litigate** without human. Design locked; implement on
`feat/living-home-hard-cut` only when asked.

Glossary: `GLOSSARY.md` (Worklist, Job instance overview, Commit navigation,
Advance target eligibility). ADR: `docs/adr/0006-living-operator-worklist-home.md`
(rewritten), `docs/adr/0008-living-task-submit-cardinality.md` (overview sentence).

## Product

1. **Home = one act-now list.** Viewer `actionable` Tasks ∪ Can start (peer row
   kinds, shared list). No In progress, no Completed, no Archive this cut.
2. **Many-submit underbelly.** For `submit: many` act-now rows, when
   `submissionCounts` N > 0: accordion under the primary row (closed = count;
   open = lazy list recap + links). Primary row still opens the Task. Not a full
   instance accordion; not once-done recap on home.
3. **Overview unlinked, route kept.** `/app/{jobId}/{jobInstanceId}` may still
   render if hit directly; no Commit / Continue / home / Task-back entry.
   Wire/Config `overview` → **home**. Default when advance disallowed → **home**.
   Commit `navigation`: `task` | `home` | `stay` (+ `continueNavigation` without
   overview).
4. **Task back → home.**
5. **Task groups:** out of scope; full removal soon. No Spawn/Close hub work.
6. **Once-recap gap:** accepted until Archive.
7. **Archive:** later Domain hierarchy (e.g. working day → shift → lot); not now.

## Keep / reuse

- LinkStore address + timestamp, `submissionCounts`, list `task-recap` `items`,
  `TaskRecapList` / facts — re-host under home many-rows.
- Can start mint path (`new` / first Task) unchanged in law.

## Rejected

- Peer home bands for In progress / Completed.
- Archive stub this cut.
- Deleting the overview route.
- Overview as Commit/Continue/home deep-link target.
- Full overview accordion chrome on home (locked/`available` siblings, etc.).
