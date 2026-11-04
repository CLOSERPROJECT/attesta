# 16: Lazy Task recap (facts + links)

**What to build:** Addressed GET returns a done Task’s Submission facts (plus actor/time) and any stored post-Commit links for that address. Auth matches Load (org operator who can hydrate Worklist), not Cerbos Commit and not “submitter only.” Payload is thin — no schema/uiSchema/full Domain.

**Blocked by:** 15

**Status:** done (local; not committed)

- [x] GET endpoint (or Loader method + HTTP) addressed by job / Job instance / Task [+ group occurrence]
- [x] Returns Submission facts + timestamp + userId when done
- [x] Includes stored links from ticket 15 when present
- [x] Rejects or empty when Task not done / unknown address
- [x] SPA can fetch for overview expand (client helper OK in this ticket or 17)
