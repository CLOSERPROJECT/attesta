# 01: Prefactor Draft write intents on LivingSession

**What to build:** Deepen the LivingSession Draft map so seed vs operator/QR writes are distinct intents: non-dirty seed does not count as a Draft for marks or persistence; dirty `setDraft` does. Expose `hasDirtyDraft` for GUI. TaskForm seeds via the non-dirty write; QR and form edits stay dirty. Successful Commit still clears that Draft. Still in-memory only in this ticket — no localStorage yet — but the dirty/mark contract is real and tested.

**Blocked by:** None (can start immediately).

**Status:** done

- [x] LivingSession Draft interface distinguishes non-dirty seed writes from dirty operator/QR writes
- [x] `hasDirtyDraft(taskPath, draftBagInstanceKey)` is true only after a dirty write, false after seed-only or `clearDraft`
- [x] TaskForm `seedBundleDraft` uses the non-dirty write; existing empty-bag seed skip still holds when a Draft is already present
- [x] QR / form paths continue to dirty-set; successful Commit clears that key; `refreshKeepDrafts` still keeps in-memory Drafts
- [x] Vitest covers seed vs dirty vs clear through the LivingSession Draft interface (no Worklist status change)

**Decisions (grilled):**

- Draft ≠ Worklist TaskStatus ≠ entity progress
- Dirty sticks until clear (no undirty-by-equality)
- Phase 1 client-only; no server DraftPort
