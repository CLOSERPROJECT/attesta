# 01: Prefactor — extract Digital Link / UNTP helpers from Stream

**What to build:** Move shared GS1 Digital Link resolution, UNTP credential/HTML+JSON shaping, and related utilities out of Stream DPP code into a neutral package both Stream and living Passport can call. Stream DPP issue and serve paths keep working unchanged; no living Passport store, Org Config `passport` block, or `onSubmit.passport` in this ticket.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] Shared helpers cover what Stream DPP needs today (Digital Link + UNTP serve/issue adjacencies) without importing living engine types
- [ ] Stream regression tests / existing DPP behavior green; no change to when Stream issues or finalizes processes
- [ ] Living code does not yet mint Passports; extraction only
- [ ] No Feragame- or Task-id branches introduced for extraction
