# 12: Stay on TaskForm when Commit returns any links

**What to build:** Post-Commit SPA stay is currently download-only (`action === 'download'`). Change to **stay whenever `CommitResult.Links` is non-empty** (link and/or download), so Passport DPP / other `action:link` items keep the read-only form + HttpLinkStrip until Continue (or group Start another / Close). Empty links → existing ADR-0006 navigate. Non-stay clear from ticket 11 remains for empty/no-links navigate.

**Blocked by:** None.

**Status:** done

- [x] `resolveCommitFollowUp` / stay detection: any non-empty links → stay
- [x] Download stay + Continue / Start another / Close still work
- [x] Link-only (Passport) Commit stays and shows HttpLinkStrip
- [x] Unit tests updated (worklist-nav, session as needed)
- [x] No plant-named SPA branches

**Pointers:** `app/src/lib/living/worklist-nav.ts` `commitHasLinks` / `resolveCommitFollowUp`; `app/src/lib/router/task.svelte`; session `linksStayActive`. Worktree: `/Users/giovanniabbatepaolo/.cursor/worktrees/living-passport-a3f7c2b1/attesta-714729698316`. Conventional Commits; comments = why only. Do not push.
