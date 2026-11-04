# 03: Public serve living Passport from store

**What to build:** HTTP routes that serve living Passport credentials (UNTP + GS1 Digital Link HTML/JSON) from the Passport store using the shared helpers from 01. Scoped by org + subject identity stored at lock time; behavior aligned with ADR public serve boundaries (separate from Stream issue triggers).

**Blocked by:** 02 (Passport records populated by lock producer).

**Status:** ready-for-agent

- [ ] Public Digital Link / UNTP handlers read frozen **Passport snapshot** from store, not Stream `Process.DPP`
- [ ] Reuses extracted helpers; Stream DPP serve still works
- [ ] Tests: serve 404/410 when not locked; happy path returns expected UNTP/Digital Link payloads for synthetic record
- [ ] No Feragame- or load-id hardcoding in handlers
