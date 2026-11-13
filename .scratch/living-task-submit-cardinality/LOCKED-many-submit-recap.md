# Locked — many-submit overview Submission recap (2026-11-13)

Grill refinement. **Do not re-litigate** without human. Design only until
implementation is asked.

Depends on [`LOCKED.md`](./LOCKED.md) (cardinality / lot session — do not reopen).
Scout: [`SCOUT-many-submit-recap.md`](./SCOUT-many-submit-recap.md).
Plan: [`IMPLEMENTATION-many-submit-recap.md`](./IMPLEMENTATION-many-submit-recap.md).

## Product / UX

1. **Count on Load, detail lazy.** `GET /api/living/domain` (Worklist) carries
   per-instance **`submissionCounts`** (`taskId` → N). Facts and links stay off
   the main response; list-recap loads when the operator expands history.
2. **Chrome for `submit: many`.** Top leaf row stays like other Tasks
   (actionable → open form). **Below** it: an accordion **only when N > 0**.
   - **Closed:** short count text (e.g. N invii) from Load counts — no facts/links.
   - **Open:** lazy list of Submissions for that leaf address.
3. **Once Tasks.** Unchanged chrome (done accordion). Same list-recap wire
   (0–1 `items`).
4. **Expanded row UI.** Occurrence-list density: `<dl>` facts + `HttpLinkStrip`
   per Submission (not stringify-only lines). Complex values may still use
   existing `formatFactValue` / JSON stringify inside the `<dl>`.
5. **Order.** Expanded `items` **oldest first** (`timestamp` ascending).
6. **Links preserved per Submission.** LinkStore keyed by Submission address **+**
   `Submission.Timestamp`. Pre-prod: wipe stores; **no** legacy address-only
   fallback path.
7. **Dumb executor.** Gate chrome on `WorklistTask.submit === 'many'` and counts;
   no Feragame-named SPA/Go branches.

## Rejected

- Latest-only LinkStore for v1 (overwrites prior download URLs).
- Split chevron on the primary row (expand vs Apri on one control).
- Always-visible history strip under the row.
- Dual wire (single-object recap for once + array for many).
- Embedding links into Domain `Submission` documents.
- Shipping full facts/links on Load.

## Out of scope here

- Phase 7 drop Task-group engine / `unlockFrom`.
- Committing Phases 1–6 of the parent board (unless asked separately).
