# 02: localStorage Draft persist + reload + Bozza marks

**What to build:** Dirty Drafts survive full page reload via debounced localStorage (org + userId + draftKey), rehydrate into the LivingSession Draft map when identity becomes known (not on every `refreshKeepDrafts`), and flush pending writes on page hide. Home Available rows and Job instance overview leaf / Task-group child rows show a muted **Bozza** mark beside the existing status tag when `hasDirtyDraft` — including provisional spawn (`new`) — without inventing a sixth Worklist TaskStatus. Successful Commit clears memory and storage for that key; identity switch resets the in-memory map (other users’ keys left alone); no orphan GC in this ticket.

**Blocked by:** 01 — Prefactor Draft write intents on LivingSession

**Status:** done

- [x] Dirty Drafts persist to localStorage under org+userId-scoped keys; seed writes never persist
- [x] Rehydrate from storage only when org/user becomes known or changes — not on `refreshKeepDrafts` / later Loads (in-memory dirty wins)
- [x] Pending debounced writes flush on `beforeunload` / page hide
- [x] `hasDirtyDraft` paints muted Bozza beside TaskStatusTag on home Available and overview / group children (incl. provisional); status vocabulary unchanged
- [x] Commit clears that key in memory and storage; identity switch clears in-memory map; single-tab only (no `storage` events)
- [x] Vitest with memory StorageLike covers persist / rehydrate / clear / mark binding; no Go or Worklist projection changes

**Decisions (grilled):**

- Storage presence = dirty; absence = not dirty
- Mark is client overlay paint-only; keep worklist-home / worklist-overview projections pure
- No multi-tab sync; last debounced write wins if both tabs edit
