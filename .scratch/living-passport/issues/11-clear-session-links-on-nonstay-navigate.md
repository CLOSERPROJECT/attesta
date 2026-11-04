# 11: Clear session HttpLinkStrip on non-stay post-Commit navigate

**What to build:** When Commit follow-up is not stay-for-downloads, clear `LivingSession.lastHttpLinks` / `lastHttpErrors` before navigating (and/or when task `routeKey` changes while stay is inactive), so prior Commit links do not appear on unrelated fresh TaskForms. Preserve download stay + Continue/Start another/Close chrome.

**Also:** **Revert ticket 10** (`6a2f835`). Human confirmed Passport on `CommitResult.Links` was correct; the bug was session leak onto the next form, not Commit returning the URL. After revert, Passport links again append to Commit response **and** LinkStore (pre-10 behavior). Do not leave the omit split.

**Blocked by:** None.

**Status:** done

- [x] `6a2f835` / omit-Passport-from-Commit-Links reverted (tests expect Commit.Links include DPP again)
- [x] Non-stay `onCommitted` clears session links before navigate
- [x] Fresh TaskForm after link-only Commit shows no prior strip
- [x] Download stay behavior + unit tests unchanged / extended for clear-on-navigate
- [x] No plant-named SPA branches

**Pointers:** `.scratch/living-passport/NOTES-fresh-form-http-link-leak.md`; seam `app/src/lib/router/task.svelte` `onCommitted`; `session.clearLinksStay`. Prefer clear in `onCommitted` when follow-up is not stay. Worktree only: `/Users/giovanniabbatepaolo/.cursor/worktrees/living-passport-a3f7c2b1/attesta-714729698316` on `feat/living-passport`. Conventional Commits; comments = why only.
