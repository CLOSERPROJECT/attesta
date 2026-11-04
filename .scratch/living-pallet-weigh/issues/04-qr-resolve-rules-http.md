# 04: QR resolve — qr.rules + HTTP seam

**What to build:** Authenticated client sends `{ raw }` (scanned string or cold-link href). Go resolves using Org Config `qr.rules` from `LOCKED.md`: fixed decode to `entityId`, then when / match / eligible / seed / messages. Exactly one Worklist candidate → response with navigate coords + seed (`palletId`); zero, many, or ineligible → error payload with Config message. No `$place`. SPA not required to verify.

**Blocked by:** 02 (One-pallet palletWeight weigh Config + Commit).

**Status:** implement-done

- [x] Org Config accepts frozen `qr.rules` shape; Feragame fixture includes the locked rule
- [x] HTTP resolve accepts `{ raw }`, returns one destination + seed or toastable error
- [x] Decode: `entityId` query param if present, else bare raw as id (no Config parse)
- [x] Go tests cover when / eligible / zero / one / many without Feragame hardcoding in engine source beyond reading Config
