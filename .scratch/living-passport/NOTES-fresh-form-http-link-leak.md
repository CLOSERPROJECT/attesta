# Tribal — DPP link on a *fresh* TaskForm

**Date:** 2026-11-04  
**Human report:** Passport URL appeared on a fresh form (not only as post-lock flash).

## Root cause (session, not Passport UI)

1. Commit writes `LivingSession.lastHttpLinks` from `CommitResult.Links`.
2. `TaskForm` always renders that session field via `HttpLinkStrip` — not scoped to the Task that committed.
3. Stay-for-links only arms for `action: download`. Passport used `action: link`.
4. Non-stay `onCommitted` (`app/src/lib/router/task.svelte`) navigates **without** `clearLinksStay()`.
5. Next mounted TaskForm therefore still shows prior Commit links.

So the UX felt like a patch because it was **session leakage onto the next form**, hitchhiking the HttpPort strip.

## Ticket 10 (`6a2f835`)

Omits Passport from Commit `Links` (keeps LinkStore → overview). That stops Passport from entering `lastHttpLinks`. It does **not** fix the general leak for other non-download Commit links.

## Verified

[Confirm fresh-form link leak](2f936818-4de8-41af-891f-f298ab7d1ab6): all four hypotheses confirmed.

## Optional follow-up

Ticket `issues/11-clear-session-links-on-nonstay-navigate.md` — clear on non-stay navigate / inactive routeKey change. Prefer that over Passport-named SPA branches.
