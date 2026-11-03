# 17: Overview accordion chrome (shadcn)

**What to build:** Job instance overview restores experiment-style chrome with **shadcn Accordion**: boxed Config-order number on Job-level Tasks and Task group cards only; status tags Bloccato / Disponibile / Per te / Fatto; chevron. `actionable` navigates to Task form (not expand). `done` expands to recap (facts + links). Task group card is an accordion (body = occurrences + Spawn/Close); each occurrence accordion expands to child Tasks (no numbers inside). Done rows collapsed by default; multi-open OK. Add shadcn-svelte accordion if missing.

**Blocked by:** 14, 16

**Status:** done

- [x] shadcn Accordion used for expandable rows
- [x] Numbers on Job-level Tasks and Task group headers only
- [x] Status tags for four-value status (IT labels)
- [x] `actionable` → Task route; locked/available display-only
- [x] `done` → lazy recap body with facts + HttpLinkStrip-style links
- [x] Group card accordion → occurrences → child Task rows
- [x] Spawn/Close remain on expanded group card
