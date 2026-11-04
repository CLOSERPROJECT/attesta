# 13: Render HttpLinkStrip items as button chrome

**What to build:** `HttpLinkStrip` already uses `<button>` but styles them as underlined text links (`text-sky-800 underline`). Human: render Commit/recap links as **buttons** — match living SPA primary/secondary button chrome used on the task route (e.g. Continue: `rounded-md bg-zinc-900 px-3 py-1.5 text-sm text-white hover:bg-zinc-800`). Keep open/download behavior and `data-testid`s. Applies to TaskForm stay strip and overview TaskRecap (same component).

**Blocked by:** None.

**Status:** done

- [x] HttpLinkStrip items look like buttons, not underline links
- [x] Layout readable with multiple items (stack or wrap gap)
- [x] No plant-named branches; reuse existing task-route button classes if possible
- [x] Svelte MCP autofixer clean if available

**Pointers:** `app/src/lib/living/form/HttpLinkStrip.svelte`; compare `app/src/lib/router/task.svelte` Continue button. Worktree: `/Users/giovanniabbatepaolo/.cursor/worktrees/living-passport-a3f7c2b1/attesta-714729698316`. Conventional Commits; comments = why only. Do not push.
