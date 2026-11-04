# 01: Prefactor — extract Digital Link / UNTP helpers from Stream

**What to build:** Move shared GS1 Digital Link resolution, UNTP credential/HTML+JSON shaping, and related utilities out of Stream DPP code into a neutral package both Stream and living Passport can call. Stream DPP issue and serve paths keep working unchanged; no living Passport store, Org Config `passport` block, or `onSubmit.passport` in this ticket.

**Blocked by:** None (can start immediately).

**Status:** done (`a83f91a` + `d1f3b25`; Spec+Standards APPROVE 2026-11-04)

- [x] Shared helpers cover what Stream DPP needs today (Digital Link + UNTP serve/issue adjacencies) without importing living engine types
- [x] Stream regression tests / existing DPP behavior green; no change to when Stream issues or finalizes processes
- [x] Living code does not yet mint Passports; extraction only
- [x] No Feragame- or Task-id branches introduced for extraction
