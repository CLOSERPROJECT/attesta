# 05: Topbar Scan + cold entityId entry

**What to build:** Operator can Scan from an always-visible topbar control anywhere in `/app`: camera/decoder yields a string, client POSTs it to QR resolve, then navigates to the weigh form with seed or shows a toast. Opening `/app?entityId=` (cold https from the label QR) uses the same resolve seam. Thin SPA — no plant match logic in the client.

**Blocked by:** 04 (QR resolve — qr.rules + HTTP seam). Ticket 03 improves real-label demos but does not block this.

**Status:** ready-for-agent

- [ ] Topbar Scan is reachable from home, overview, and task surfaces
- [ ] Successful resolve navigates to nested weigh with `palletId` seeded
- [ ] Failed resolve shows the server message as a toast (zero / many / ineligible)
- [ ] Cold `/app?entityId=` triggers the same resolve path once
