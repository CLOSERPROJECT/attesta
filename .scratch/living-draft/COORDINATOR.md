# Living Draft Phase 1 — coordinator

**Worktree:** `feat-draft-d8e59123` · branch `feat/draft`  
**Merge-back:** `/apply-worktree` · **Cleanup:** `/delete-worktree`

## Tickets

| # | File | Gate |
| --- | --- | --- |
| 01 | `issues/01-prefactor-draft-write-intents.md` | done |
| 02 | `issues/02-localstorage-draft-persist-and-bozza-marks.md` | done |

## Settled law (do not re-grill)

- Deepen **LivingSession** Draft map — not DraftStore, not per-form DraftSession
- **Dirty** only for persist/mark; seed is non-dirty; dirty sticks until `clearDraft`
- Storage keys: `org + userId + draftKey`; rehydrate only on identity known/change — **not** on `refreshKeepDrafts`
- Persist dirty only; flush on page hide; single-tab
- GUI: muted **Bozza** beside status tag; **not** a sixth Worklist TaskStatus
- Draft ≠ progress ≠ Worklist status (glossary + ADR-0005)

## Pit of success / traps

- No dual writers (bag + DraftSession)
- Do not fold `hasDraft` into Worklist projections or Go
- Do not thread StorageLike through TaskForm / createConfigForm
- Do not mark on non-empty bag (schema defaults)

## Out of scope (Phase 1)

Server DraftPort, multi-tab `storage` events, orphan GC, leftover sanitize/rebind from experiment.
